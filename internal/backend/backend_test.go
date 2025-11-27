package backend

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"
)

func TestNew(t *testing.T) {
	urlStr := "http://localhost:8080"
	u, _ := url.Parse(urlStr)
	proxy := httputil.NewSingleHostReverseProxy(u)

	b := New(u, proxy)

	if b.URL.String() != urlStr {
		t.Errorf("expected URL %s, got %s", urlStr, b.URL.String())
	}

	if !b.Alive {
		t.Error("expected new backend to be alive")
	}

	if b.ReverseProxy != proxy {
		t.Error("expected reverse proxy to match")
	}
}

func TestSetAlive(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := New(u, proxy)

	b.SetAlive(false)
	if b.Alive {
		t.Error("expected backend to be not alive after SetAlive(false)")
	}

	b.SetAlive(true)
	if !b.Alive {
		t.Error("expected backend to be alive after SetAlive(true)")
	}
}

func TestIsAlive(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := New(u, proxy)

	if !b.IsAlive() {
		t.Error("expected new backend to be alive")
	}

	b.SetAlive(false)
	if b.IsAlive() {
		t.Error("expected backend to be not alive")
	}

	b.SetAlive(true)
	if !b.IsAlive() {
		t.Error("expected backend to be alive")
	}
}

func TestConcurrentAccess(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := New(u, proxy)

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(2)

		go func() {
			defer wg.Done()
			b.SetAlive(true)
		}()

		go func() {
			defer wg.Done()
			_ = b.IsAlive()
		}()
	}

	wg.Wait()
}

func TestSetAliveThreadSafety(t *testing.T) {
	u, _ := url.Parse("http://localhost:8080")
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := New(u, proxy)

	var wg sync.WaitGroup
	goroutines := 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(val bool) {
			defer wg.Done()
			b.SetAlive(val)
		}(i%2 == 0)
	}

	wg.Wait()

	alive := b.IsAlive()
	if alive != true && alive != false {
		t.Error("expected alive to be either true or false")
	}
}
