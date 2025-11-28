package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/eltoncampos/load-balancer/internal/backend"
	"github.com/eltoncampos/load-balancer/internal/handler"
	"github.com/eltoncampos/load-balancer/internal/pool"
	"github.com/eltoncampos/load-balancer/testutil"
)

func createTestBackend(urlStr string) *backend.Backend {
	u, _ := url.Parse(urlStr)
	proxy := httputil.NewSingleHostReverseProxy(u)
	return backend.New(u, proxy)
}

func createTestPool(backends ...*backend.Backend) *pool.ServerPool {
	p := pool.New()
	for _, b := range backends {
		p.AddBackend(b)
	}
	return p
}

func TestCreateProxyErrorHandler_WithRetriesUnderLimit(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d after retry, got %d", http.StatusOK, w.Code)
	}

	retryHeader := req.Header.Get("X-Retry-Count")
	if retryHeader != "1" {
		t.Errorf("expected X-Retry-Count header to be '1', got '%s'", retryHeader)
	}
}

func TestCreateProxyErrorHandler_WithRetriesAtLimit(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), handler.Retry, 2)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d after final retry, got %d", http.StatusOK, w.Code)
	}

	retryHeader := req.Header.Get("X-Retry-Count")
	if retryHeader != "3" {
		t.Errorf("expected X-Retry-Count header to be '3', got '%s'", retryHeader)
	}
}

func TestCreateProxyErrorHandler_WithRetriesOverLimit(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), handler.Retry, 3)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	// Backend should be marked as down after 3 retries
	if b.IsAlive() {
		t.Error("expected backend to be marked as down after retry limit exceeded")
	}

	// Should attempt with next backend (which will also fail since it's the only one)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d when all backends are down, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestCreateProxyErrorHandler_MarksBackendDown(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	if !b.IsAlive() {
		t.Fatal("backend should start as alive")
	}

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), handler.Retry, 3)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	if b.IsAlive() {
		t.Error("expected backend to be marked as down after retry limit exceeded")
	}
}

func TestCreateProxyErrorHandler_IncrementsRetryCount(t *testing.T) {
	testCases := []struct {
		name          string
		initialRetry  int
		expectedRetry string
	}{
		{
			name:          "first retry",
			initialRetry:  0,
			expectedRetry: "1",
		},
		{
			name:          "second retry",
			initialRetry:  1,
			expectedRetry: "2",
		},
		{
			name:          "third retry",
			initialRetry:  2,
			expectedRetry: "3",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := testutil.CreateTestServer("ok", http.StatusOK)
			defer server.Close()

			serverURL, _ := url.Parse(server.URL)
			b := createTestBackend(server.URL)
			serverPool := createTestPool(b)
			lb := handler.New(serverPool)

			errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

			req := httptest.NewRequest("GET", "/test", nil)
			ctx := context.WithValue(req.Context(), handler.Retry, tc.initialRetry)
			req = req.WithContext(ctx)

			w := httptest.NewRecorder()

			errorHandler(w, req, errors.New("test error"))

			retryHeader := req.Header.Get("X-Retry-Count")
			if retryHeader != tc.expectedRetry {
				t.Errorf("expected X-Retry-Count header to be '%s', got '%s'", tc.expectedRetry, retryHeader)
			}
		})
	}
}

func TestCreateProxyErrorHandler_IncrementsAttempts(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	b.SetAlive(false)

	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), handler.Retry, 3)
	ctx = context.WithValue(ctx, handler.Attempts, 1)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	// After retry limit is exceeded, attempts should be incremented
	// Since all backends are down, we should get service unavailable
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestCreateProxyErrorHandler_WithMultipleBackends(t *testing.T) {
	server1 := testutil.CreateTestServer("backend1", http.StatusOK)
	defer server1.Close()

	server2 := testutil.CreateTestServer("backend2", http.StatusOK)
	defer server2.Close()

	serverURL1, _ := url.Parse(server1.URL)
	b1 := createTestBackend(server1.URL)
	b2 := createTestBackend(server2.URL)

	serverPool := createTestPool(b1, b2)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL1)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), handler.Retry, 3)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	// First backend should be marked as down
	if b1.IsAlive() {
		t.Error("expected first backend to be marked as down")
	}

	// Should succeed with second backend
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d with available backend, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if body != "backend2" {
		t.Errorf("expected response from backend2, got '%s'", body)
	}
}

func TestCreateProxyErrorHandler_NoRetryHeader(t *testing.T) {
	server := testutil.CreateTestServer("ok", http.StatusOK)
	defer server.Close()

	serverURL, _ := url.Parse(server.URL)
	b := createTestBackend(server.URL)
	serverPool := createTestPool(b)
	lb := handler.New(serverPool)

	errorHandler := createProxyErrorHandler(lb, serverPool, serverURL)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Initial retry count should be 0
	errorHandler(w, req, errors.New("test error"))

	retryHeader := req.Header.Get("X-Retry-Count")
	if retryHeader != "1" {
		t.Errorf("expected X-Retry-Count header to be '1' for first retry, got '%s'", retryHeader)
	}
}
