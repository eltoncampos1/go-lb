package handler

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/eltoncampos/load-balancer/internal/pool"
)

const (
	Attempts int = iota
	Retry
)

type LoadBalancer struct {
	pool *pool.ServerPool
}

func New(p *pool.ServerPool) *LoadBalancer {
	return &LoadBalancer{
		pool: p,
	}
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	attempts := GetAttemptsFromContext(r)
	if attempts > 3 {
		log.Printf("%s(%s) Max attempts reached, terminating\n", r.RemoteAddr, r.URL.Path)
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
		return
	}

	peer := lb.pool.GetNextPeer()
	if peer != nil {
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func (lb *LoadBalancer) CreateErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, e error) {
		log.Printf("[%s] %s\n", req.URL.Host, e.Error())

		retries := GetRetryFromContext(req)
		if retries < 3 {
			time.Sleep(10 * time.Millisecond)
			ctx := context.WithValue(req.Context(), Retry, retries+1)
			req = req.WithContext(ctx)
			lb.ServeHTTP(w, req)
			return
		}

		serverURL := req.URL
		lb.pool.MarkBackendStatus(serverURL, false)

		attempts := GetAttemptsFromContext(req)
		log.Printf("%s(%s) Attempting retry %d\n", req.RemoteAddr, req.URL.Path, attempts)
		ctx := context.WithValue(req.Context(), Attempts, attempts+1)
		lb.ServeHTTP(w, req.WithContext(ctx))
	}
}

func GetAttemptsFromContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

func GetRetryFromContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}
