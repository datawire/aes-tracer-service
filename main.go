package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

var port = "8080"
var defaultTraceRoute = "/trace"

const (
	EnvPORT              = "PORT"
	EnvHOST              = "HOST"
	EnvTLS               = "ENABLE_TLS"
	EnvTargetHost        = "TARGET_HOST"  // The host where the request will be sent			  #OPTIONAL - defaults to current host
	EnvTraceHeaderPrefix = "TRACE_PREFIX" // The host where the request will be sent			  #OPTIONAL - defaults to current host
	EnvTraceRoute        = "TRACE_ROUTE"  // The host where the request will be sent			  #OPTIONAL - defaults to current host
)

type Server struct {
	id     string
	host   string
	port   int
	tls    bool
	router *chi.Mux
	ready  bool
}

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if !s.ready {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err := w.Write([]byte("OK"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) Trace(w http.ResponseWriter, r *http.Request) {
	log.Println("Tracing request initiated")

	// Create a new request to forward
	// Read the original body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Restore the body for potential later use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	traceRoute := getEnv(EnvTraceRoute, defaultTraceRoute)

	// Create the forwarded request (replace TARGET_URL with your desired destination)
	updatedUrl := strings.Replace(r.URL.Path, traceRoute, "", 1)

	tls := "https://"

	if r.TLS == nil {
		tls = "http://"
	}

	host := getEnv(EnvTargetHost, r.Host)

	fullUrl := tls + host + updatedUrl

	// Append query parameters to the full URL
	if r.URL.RawQuery != "" {
		fullUrl += "?" + r.URL.RawQuery
	}

	log.Println("Sending request to " + fullUrl)

	forwardReq, err := http.NewRequest(r.Method, fullUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("Error creating forward request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tracingHeaderPrefix := getEnv(EnvTraceHeaderPrefix, "X-B3")

	// Copy all headers from original request, excluding X-B3 headers
	for name, values := range r.Header {
		if !strings.HasPrefix(strings.ToUpper(name), tracingHeaderPrefix) {
			for _, value := range values {
				forwardReq.Header.Add(name, value)
			}
		}
	}

	uuidValue, err := uuid.NewRandom()
	if err != nil {
		log.Println("Error generating UUID: ", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	forwardReq.Header.Add("x-client-trace-id", uuidValue.String())
	forwardReq.Header.Add("x-envoy-force-trace", "true")

	// Send the forwarded request
	client := &http.Client{}
	resp, err := client.Do(forwardReq)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy the response status and headers back to the original client
	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	// Copy the response body back to the original client
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}

func (s *Server) Debug(w http.ResponseWriter, r *http.Request) {
	log.Println("Debug request initiated")

	// Create response structure
	response := struct {
		Headers     map[string][]string `json:"headers"`
		QueryParams map[string][]string `json:"query_parameters"`
		Body        string              `json:"body"`
	}{
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
	}

	// Read Body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// Restore the body for potential later use
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	response.Body = string(bodyBytes)

	// Pretty print the JSON response
	prettyJSON, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Write the JSON response
	w.Header().Set("Content-Type", "application/json")
	w.Write(prettyJSON)
}

func (s *Server) ConfigureRouter() {
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)

	s.router.Post("/health", s.HealthCheck)
	s.router.Get("/health", s.HealthCheck)
	s.router.HandleFunc("/debug", s.Debug)

	traceRoute := getEnv(EnvTraceRoute, defaultTraceRoute)
	s.router.HandleFunc(traceRoute+"/*", s.Trace)
}

func getEnv(name, fallback string) string {
	res := os.Getenv(name)
	if res == "" {
		res = fallback
	}

	return res
}

func (s *Server) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", s.host, s.port)
	log.Printf("listening on %s\n", listenAddr)
	if s.tls {
		return http.ListenAndServeTLS(listenAddr, "/certs/cert.pem", "/certs/key.pem", s.router)
	}
	return http.ListenAndServe(listenAddr, s.router)
}

func main() {
	tls, err := strconv.ParseBool(getEnv(EnvTLS, "true"))
	if err != nil {
		log.Println("ERROR: ENABLE_HTTPS environment variable must be either 'true' or 'false'.")
	}
	defPort := port
	if tls {
		defPort = "8443"
	}
	port, err := strconv.Atoi(getEnv(EnvPORT, defPort))
	if err != nil {
		log.Fatalln(err)
	}

	if port < 1 || port > 65535 {
		log.Fatalln("Server port must be in range 1..65535 (inclusive)")
	}

	s := Server{
		id:     "aes-tracer",
		host:   os.Getenv(EnvHOST),
		port:   port,
		tls:    tls,
		router: chi.NewRouter(),
		ready:  true,
	}

	s.ConfigureRouter()

	// Handle SIGTERM gracefully
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)

	go func(r *bool) {
		<-signals
		*r = false
		fmt.Printf("SIGTERM received. Marked unhealthy and waiting to be killed.\n")
	}(&s.ready)

	log.Fatalln(s.Start())
}
