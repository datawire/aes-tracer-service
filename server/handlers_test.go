package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
)

type mockTransport struct {
	response *http.Response
	err      error
	request  *http.Request
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.request = req

	if t.err != nil {
		return nil, t.err
	}

	resp := *t.response
	resp.Request = req
	if t.response.Body != nil {
		bodyBytes, _ := io.ReadAll(t.response.Body)
		t.response.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	return &resp, nil
}

func TestHealthCheck(t *testing.T) {
	tests := []struct {
		name        string
		serverReady bool
		wantStatus  int
		wantBody    string
	}{
		{
			name:        "server ready",
			serverReady: true,
			wantStatus:  http.StatusOK,
			wantBody:    "OK",
		},
		{
			name:        "server not ready",
			serverReady: false,
			wantStatus:  http.StatusInternalServerError,
			wantBody:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{Ready: tt.serverReady}
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			s.HealthCheck(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HealthCheck() status = %v, want %v", w.Code, tt.wantStatus)
			}
			if got := w.Body.String(); got != tt.wantBody {
				t.Errorf("HealthCheck() body = %v, want %v", got, tt.wantBody)
			}
		})
	}
}

func TestTrace(t *testing.T) {
	tests := []struct {
		name                 string
		method               string
		path                 string
		body                 string
		headers              map[string]string
		envVars              map[string]string
		expectedStatus       int
		mockTargetResp       *http.Response
		wantForwardedHeaders map[string]string
		skipHeaderNames      []string
	}{
		{
			name:   "successful trace with header verification",
			method: http.MethodPost,
			path:   "/trace/test",
			body:   "test body",
			headers: map[string]string{
				"Content-Type":    "application/json",
				"X-Custom-Header": "custom-value",
				"X-B3-TraceId":    "should-not-be-forwarded",
			},
			envVars: map[string]string{
				"TRACE_ROUTE":         "/trace",
				"TARGET_HOST":         "fake-host.net",
				"TRACE_HEADER_PREFIX": "X-B3",
			},
			expectedStatus: http.StatusOK,
			mockTargetResp: &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBufferString("response body")),
				Header:     make(http.Header),
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Status:     "200 OK",
			},
			wantForwardedHeaders: map[string]string{
				"Content-Type":        "application/json",
				"X-Custom-Header":     "custom-value",
				"X-Client-Trace-Id":   "",
				"X-Envoy-Force-Trace": "true",
			},
			skipHeaderNames: []string{
				"X-B3-TraceId",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			mockTransport := &mockTransport{
				response: tt.mockTargetResp,
			}

			s := &Server{
				client: &http.Client{
					Transport: mockTransport,
				},
			}

			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()

			s.Trace(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Trace() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Verify response body
			respBody, _ := io.ReadAll(w.Body)
			expectedBody := "response body"
			if string(respBody) != expectedBody {
				t.Errorf("Trace() body = %v, want %v", string(respBody), expectedBody)
			}

			// Verify the forwarded request headers
			if mockTransport.request != nil {
				// Verify expected headers are present
				for header, expectedValue := range tt.wantForwardedHeaders {
					gotValue := mockTransport.request.Header.Get(header)
					if expectedValue != "" && gotValue != expectedValue {
						t.Errorf("Expected header %s = %s, got %s", header, expectedValue, gotValue)
					}
					if expectedValue == "" && gotValue == "" {
						t.Errorf("Expected header %s to be present but it was not", header)
					}
				}

				// Verify UUID format for X-Client-Trace-Id
				traceID := mockTransport.request.Header.Get("X-Client-Trace-Id")
				if _, err := uuid.Parse(traceID); err != nil {
					t.Errorf("X-Client-Trace-Id is not a valid UUID: %s", traceID)
				}

				// Verify headers that should not be forwarded
				for _, skipHeader := range tt.skipHeaderNames {
					if got := mockTransport.request.Header.Get(skipHeader); got != "" {
						t.Errorf("Header %s should not be forwarded, but got value: %s", skipHeader, got)
					}
				}

				// Verify request path
				expectedPath := "/test"
				if mockTransport.request.URL.Path != expectedPath {
					t.Errorf("Expected request path %s, got %s", expectedPath, mockTransport.request.URL.Path)
				}
			}
		})
	}
}

func TestDebug(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		queryParams    map[string]string
		headers        map[string]string
		body           string
		expectedStatus int
	}{
		{
			name:   "successful debug request",
			method: http.MethodPost,
			path:   "/debug",
			queryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
			headers: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "test-agent",
			},
			body:           `{"test": "data"}`,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{}

			// Construct request URL with query parameters
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Add(k, v)
			}
			req.URL.RawQuery = q.Encode()

			// Add headers
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			s.Debug(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Debug() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			// Verify response structure
			var response struct {
				Headers     map[string][]string `json:"headers"`
				QueryParams map[string][]string `json:"query_parameters"`
				Body        string              `json:"body"`
			}

			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify the response contains expected data
			if response.Body != tt.body {
				t.Errorf("Debug() body = %v, want %v", response.Body, tt.body)
			}

			// Verify query parameters
			for k, v := range tt.queryParams {
				if params, ok := response.QueryParams[k]; !ok || params[0] != v {
					t.Errorf("Debug() query param %s = %v, want %v", k, params, v)
				}
			}

			// Verify headers
			for k, v := range tt.headers {
				if headers, ok := response.Headers[k]; !ok || headers[0] != v {
					t.Errorf("Debug() header %s = %v, want %v", k, headers, v)
				}
			}
		})
	}
}
