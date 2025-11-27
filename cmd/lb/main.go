package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/eltoncampos/load-balancer/internal/backend"
	"github.com/eltoncampos/load-balancer/internal/config"
	"github.com/eltoncampos/load-balancer/internal/handler"
	"github.com/eltoncampos/load-balancer/internal/healthcheck"
	"github.com/eltoncampos/load-balancer/internal/pool"
)

func main() {
	cfg := config.Load()

	serverPool := pool.New()
	lb := handler.New(serverPool)

	tokens := strings.SplitSeq(cfg.ServerList, ",")
	for tok := range tokens {
		serverURL, err := url.Parse(tok)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(serverURL)
		proxy.ErrorHandler = createProxyErrorHandler(lb, serverPool, serverURL)

		b := backend.New(serverURL, proxy)
		serverPool.AddBackend(b)
		log.Printf("Configured server: %s\n", serverURL)
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: lb,
	}

	go healthcheck.StartHealthCheck(
		serverPool.GetBackends(),
		time.Duration(cfg.HealthCheckInterval)*time.Second,
	)

	log.Printf("Load Balancer started on port %d\n", cfg.Port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func createProxyErrorHandler(lb *handler.LoadBalancer, serverPool *pool.ServerPool, serverURL *url.URL) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, req *http.Request, e error) {
		log.Printf("[%s] %s\n", serverURL.Host, e.Error())

		retries := handler.GetRetryFromContext(req)
		if retries < 3 {
			time.Sleep(10 * time.Millisecond)
			ctx := context.WithValue(req.Context(), handler.Retry, retries+1)
			req.Header.Set("X-Retry-Count", fmt.Sprintf("%d", retries+1))
			lb.ServeHTTP(w, req.WithContext(ctx))
			return
		}

		serverPool.MarkBackendStatus(serverURL, false)

		attempts := handler.GetAttemptsFromContext(req)
		log.Printf("%s(%s) Attempting retry %d\n", req.RemoteAddr, req.URL.Path, attempts)
		ctx := context.WithValue(req.Context(), handler.Attempts, attempts+1)
		lb.ServeHTTP(w, req.WithContext(ctx))
	}
}
