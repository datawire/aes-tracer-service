package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNew_CanInitializeServer(t *testing.T) {
	// Test server initialization
	tests := []struct {
		name     string
		id       string
		host     string
		port     int
		tls      bool
		wantPort int
	}{
		{
			name:     "basic server",
			id:       "test-server",
			host:     "localhost",
			port:     8080,
			tls:      false,
			wantPort: 8080,
		},
		{
			name:     "tls server",
			id:       "test-server-tls",
			host:     "localhost",
			port:     443,
			tls:      true,
			wantPort: 443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.id, tt.host, tt.port, tt.tls)
			if s.Port != tt.wantPort {
				t.Errorf("New() port = %v, want %v", s.Port, tt.wantPort)
			}
			if s.TLS != tt.tls {
				t.Errorf("New() TLS = %v, want %v", s.TLS, tt.tls)
			}
			if !s.Ready {
				t.Error("New() server should be ready")
			}
		})
	}
}

func TestServer_CanHitConfiguredRoutes(t *testing.T) {
	s := New("test", "localhost", 8080, false)
	s.ConfigureRouter()

	// Test health endpoint
	t.Run("health check endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check returned wrong status code: got %v want %v",
				w.Code, http.StatusOK)
		}
	})

	// Test debug endpoint
	t.Run("debug endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/debug", nil)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Debug endpoint returned wrong status code: got %v want %v",
				w.Code, http.StatusOK)
		}
	})
}

func TestServer_CanStartAndShutdown(t *testing.T) {
	s := New("test", "localhost", 8888, false) // Use port 0 for testing
	s.ConfigureRouter()

	go func() {
		err := s.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Server.Start() error = %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Make a test request to verify server is running
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/health", s.Host, s.Port))
	if err != nil {
		t.Fatalf("Failed to make test request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK; got %v", resp.StatusCode)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.Server.Shutdown(ctx); err != nil {
		t.Errorf("Server shutdown failed: %v", err)
	}
}
