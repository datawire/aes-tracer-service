package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Server struct {
	ID     string
	Host   string
	Port   int
	TLS    bool
	Router *chi.Mux
	Ready  bool
	Server *http.Server
	client *http.Client
}

func New(id string, host string, port int, tls bool) *Server {
	return &Server{
		ID:     id,
		Host:   host,
		Port:   port,
		TLS:    tls,
		Router: chi.NewRouter(),
		Ready:  true,
	}
}

func (s *Server) ConfigureRouter() {
	s.Router.Use(middleware.Recoverer)
	s.Router.Use(middleware.RequestID)
	s.Router.Use(middleware.RealIP)

	s.Router.Post("/health", s.HealthCheck)
	s.Router.Get("/health", s.HealthCheck)
	s.Router.HandleFunc("/debug", s.Debug)

	initTraceRoute := GetEnv(EnvTraceRoute, defaultTraceRoute)
	s.Router.HandleFunc(initTraceRoute+"/*", s.Trace)
}

func (s *Server) Start() error {
	listenAddr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	log.Printf("listening on %s\n", listenAddr)

	s.Server = &http.Server{
		Addr:    listenAddr,
		Handler: s.Router,
	}

	if s.TLS {
		return s.Server.ListenAndServeTLS("/certs/cert.pem", "/certs/key.pem")
	}
	return s.Server.ListenAndServe()
}
