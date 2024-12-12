package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

func (s *Server) HealthCheck(w http.ResponseWriter, r *http.Request) {
	if !s.Ready {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err := w.Write([]byte("OK"))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) Debug(w http.ResponseWriter, r *http.Request) {
	log.Println("Debug request initiated")

	response := struct {
		Headers     map[string][]string `json:"headers"`
		QueryParams map[string][]string `json:"query_parameters"`
		Body        string              `json:"body"`
	}{
		Headers:     r.Header,
		QueryParams: r.URL.Query(),
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	response.Body = string(bodyBytes)

	prettyJSON, err := json.MarshalIndent(response, "", "    ")
	if err != nil {
		log.Printf("Error encoding response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(prettyJSON)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) Trace(w http.ResponseWriter, r *http.Request) {
	log.Println("Tracing request initiated")

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	traceRoute := GetEnv(EnvTraceRoute, defaultTraceRoute)

	// Create new URL using the original request's URL as base
	targetURL := *r.URL

	// Update the host
	targetURL.Host = GetEnv(EnvTargetHost, r.Host)

	// Update the scheme based on configuration or original request
	if r.TLS != nil {
		targetURL.Scheme = "https"
	} else {
		targetURL.Scheme = "http"
	}

	// Update the path by removing the trace route prefix
	targetURL.Path = strings.TrimPrefix(targetURL.Path, traceRoute)

	fullUrl := targetURL.String()

	log.Println("Sending request to " + fullUrl)

	forwardReq, err := http.NewRequest(r.Method, fullUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Printf("Error creating forward request: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tracingHeaderPrefix := GetEnv(EnvTraceHeaderPrefix, "X-B3")

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

	if s.client == nil {
		s.client = &http.Client{}
	}
	resp, err := s.client.Do(forwardReq)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for name, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("Error copying response body: %v", err)
	}
}
