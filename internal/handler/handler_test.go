package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/eltoncampos/load-balancer/internal/backend"
	"github.com/eltoncampos/load-balancer/internal/pool"
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

func createTestServer(response string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	}))
}

func TestNew(t *testing.T) {
	p := pool.New()
	lb := New(p)

	if lb.pool != p {
		t.Error("expected pool to match")
	}
}

func TestServeHTTP_WithAvailableBackend(t *testing.T) {
	server := createTestServer("ok", http.StatusOK)
	defer server.Close()

	b := createTestBackend(server.URL)
	p := createTestPool(b)
	lb := New(p)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	lb.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", w.Body.String())
	}
}

func TestServeHTTP_WithNoBackends(t *testing.T) {
	p := pool.New()
	lb := New(p)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	lb.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestServeHTTP_WithDeadBackends(t *testing.T) {
	b := createTestBackend("http://localhost:99999")
	b.SetAlive(false)

	p := createTestPool(b)
	lb := New(p)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	lb.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestServeHTTP_MaxAttemptsExceeded(t *testing.T) {
	server := createTestServer("ok", http.StatusOK)
	defer server.Close()

	b := createTestBackend(server.URL)
	p := createTestPool(b)
	lb := New(p)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), Attempts, 4)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	lb.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestServeHTTP_RoundRobin(t *testing.T) {
	server1 := createTestServer("backend1", http.StatusOK)
	defer server1.Close()

	server2 := createTestServer("backend2", http.StatusOK)
	defer server2.Close()

	b1 := createTestBackend(server1.URL)
	b2 := createTestBackend(server2.URL)
	p := createTestPool(b1, b2)
	lb := New(p)

	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	lb.ServeHTTP(w1, req1)

	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	lb.ServeHTTP(w2, req2)

	responses := []string{w1.Body.String(), w2.Body.String()}

	hasBackend1 := false
	hasBackend2 := false

	for _, resp := range responses {
		if resp == "backend1" {
			hasBackend1 = true
		}
		if resp == "backend2" {
			hasBackend2 = true
		}
	}

	if !hasBackend1 || !hasBackend2 {
		t.Error("expected round-robin to distribute requests to both backends")
	}
}

func TestCreateErrorHandler_WithRetriesUnderLimit(t *testing.T) {
	server := createTestServer("ok", http.StatusOK)
	defer server.Close()

	b := createTestBackend(server.URL)
	p := createTestPool(b)
	lb := New(p)

	errorHandler := lb.CreateErrorHandler()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d after retry, got %d", http.StatusOK, w.Code)
	}
}

func TestCreateErrorHandler_WithRetriesOverLimit(t *testing.T) {
	b := createTestBackend("http://localhost:99999")
	b.SetAlive(false)

	p := createTestPool(b)
	lb := New(p)

	errorHandler := lb.CreateErrorHandler()

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), Retry, 3)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	errorHandler(w, req, errors.New("test error"))

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestGetAttemptsFromContext_WithValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), Attempts, 5)
	req = req.WithContext(ctx)

	attempts := GetAttemptsFromContext(req)

	if attempts != 5 {
		t.Errorf("expected 5 attempts, got %d", attempts)
	}
}

func TestGetAttemptsFromContext_WithoutValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	attempts := GetAttemptsFromContext(req)

	if attempts != 1 {
		t.Errorf("expected default 1 attempt, got %d", attempts)
	}
}

func TestGetRetryFromContext_WithValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), Retry, 3)
	req = req.WithContext(ctx)

	retries := GetRetryFromContext(req)

	if retries != 3 {
		t.Errorf("expected 3 retries, got %d", retries)
	}
}

func TestGetRetryFromContext_WithoutValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)

	retries := GetRetryFromContext(req)

	if retries != 0 {
		t.Errorf("expected default 0 retries, got %d", retries)
	}
}
