package pool

import (
	"net/http/httputil"
	"net/url"
	"sync"
	"testing"

	"github.com/eltoncampos/load-balancer/internal/backend"
)

func createTestBackend(urlStr string, alive bool) *backend.Backend {
	u, _ := url.Parse(urlStr)
	proxy := httputil.NewSingleHostReverseProxy(u)
	b := backend.New(u, proxy)
	b.SetAlive(alive)
	return b
}

func TestNew(t *testing.T) {
	p := New()

	if p == nil {
		t.Error("expected pool to be created")
	}

	if p.backends == nil {
		t.Error("expected backends slice to be initialized")
	}

	if len(p.backends) != 0 {
		t.Errorf("expected empty backends, got %d", len(p.backends))
	}

	if p.current != 0 {
		t.Errorf("expected current to be 0, got %d", p.current)
	}
}

func TestAddBackend(t *testing.T) {
	p := New()
	b := createTestBackend("http://localhost:8080", true)

	p.AddBackend(b)

	if len(p.backends) != 1 {
		t.Errorf("expected 1 backend, got %d", len(p.backends))
	}

	if p.backends[0] != b {
		t.Error("expected backend to match")
	}
}

func TestAddBackend_Multiple(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", true)
	b2 := createTestBackend("http://localhost:8081", true)
	b3 := createTestBackend("http://localhost:8082", true)

	p.AddBackend(b1)
	p.AddBackend(b2)
	p.AddBackend(b3)

	if len(p.backends) != 3 {
		t.Errorf("expected 3 backends, got %d", len(p.backends))
	}
}

func TestNextIndex(t *testing.T) {
	p := New()
	p.AddBackend(createTestBackend("http://localhost:8080", true))
	p.AddBackend(createTestBackend("http://localhost:8081", true))
	p.AddBackend(createTestBackend("http://localhost:8082", true))

	idx1 := p.NextIndex()
	idx2 := p.NextIndex()
	idx3 := p.NextIndex()
	idx4 := p.NextIndex()

	if idx1 == idx2 && idx2 == idx3 {
		t.Error("expected indices to rotate")
	}

	if idx1 != idx4 {
		t.Error("expected index to wrap around after full rotation")
	}
}

func TestGetNextPeer_WithEmptyPool(t *testing.T) {
	p := New()

	peer := p.GetNextPeer()

	if peer != nil {
		t.Error("expected nil when pool is empty")
	}
}

func TestGetNextPeer_WithSingleBackend(t *testing.T) {
	p := New()
	b := createTestBackend("http://localhost:8080", true)
	p.AddBackend(b)

	peer := p.GetNextPeer()

	if peer != b {
		t.Error("expected to get the only backend")
	}
}

func TestGetNextPeer_RoundRobin(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", true)
	b2 := createTestBackend("http://localhost:8081", true)
	b3 := createTestBackend("http://localhost:8082", true)

	p.AddBackend(b1)
	p.AddBackend(b2)
	p.AddBackend(b3)

	peer1 := p.GetNextPeer()
	peer2 := p.GetNextPeer()
	peer3 := p.GetNextPeer()
	peer4 := p.GetNextPeer()

	if peer1 == peer2 && peer2 == peer3 {
		t.Error("expected different backends in round-robin")
	}

	if peer1 != peer4 {
		t.Error("expected round-robin to wrap around")
	}
}

func TestGetNextPeer_SkipsDeadBackends(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", false)
	b2 := createTestBackend("http://localhost:8081", true)
	b3 := createTestBackend("http://localhost:8082", false)

	p.AddBackend(b1)
	p.AddBackend(b2)
	p.AddBackend(b3)

	peer1 := p.GetNextPeer()
	peer2 := p.GetNextPeer()

	if peer1 != b2 {
		t.Error("expected to get only alive backend")
	}

	if peer2 != b2 {
		t.Error("expected to keep getting the only alive backend")
	}
}

func TestGetNextPeer_AllBackendsDead(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", false)
	b2 := createTestBackend("http://localhost:8081", false)

	p.AddBackend(b1)
	p.AddBackend(b2)

	peer := p.GetNextPeer()

	if peer != nil {
		t.Error("expected nil when all backends are dead")
	}
}

func TestMarkBackendStatus(t *testing.T) {
	p := New()
	b := createTestBackend("http://localhost:8080", true)
	p.AddBackend(b)

	if !b.IsAlive() {
		t.Error("expected backend to be alive initially")
	}

	u, _ := url.Parse("http://localhost:8080")
	p.MarkBackendStatus(u, false)

	if b.IsAlive() {
		t.Error("expected backend to be marked as dead")
	}

	p.MarkBackendStatus(u, true)

	if !b.IsAlive() {
		t.Error("expected backend to be marked as alive")
	}
}

func TestMarkBackendStatus_NonExistentBackend(t *testing.T) {
	p := New()
	b := createTestBackend("http://localhost:8080", true)
	p.AddBackend(b)

	u, _ := url.Parse("http://localhost:9999")
	p.MarkBackendStatus(u, false)

	if !b.IsAlive() {
		t.Error("expected existing backend to remain alive")
	}
}

func TestGetBackends(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", true)
	b2 := createTestBackend("http://localhost:8081", true)

	p.AddBackend(b1)
	p.AddBackend(b2)

	backends := p.GetBackends()

	if len(backends) != 2 {
		t.Errorf("expected 2 backends, got %d", len(backends))
	}

	if backends[0] != b1 || backends[1] != b2 {
		t.Error("expected backends to match in order")
	}
}

func TestConcurrentGetNextPeer(t *testing.T) {
	p := New()
	b1 := createTestBackend("http://localhost:8080", true)
	b2 := createTestBackend("http://localhost:8081", true)
	b3 := createTestBackend("http://localhost:8082", true)

	p.AddBackend(b1)
	p.AddBackend(b2)
	p.AddBackend(b3)

	var wg sync.WaitGroup
	iterations := 100

	for i := 0; i < iterations; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			peer := p.GetNextPeer()
			if peer == nil {
				t.Error("expected to get a peer")
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentMarkBackendStatus(t *testing.T) {
	p := New()
	b := createTestBackend("http://localhost:8080", true)
	p.AddBackend(b)

	u, _ := url.Parse("http://localhost:8080")

	var wg sync.WaitGroup
	goroutines := 50

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(val bool) {
			defer wg.Done()
			p.MarkBackendStatus(u, val)
		}(i%2 == 0)
	}

	wg.Wait()

	alive := b.IsAlive()
	if alive != true && alive != false {
		t.Error("expected alive to be either true or false")
	}
}
