package healthcheck

import (
	"net"
	"net/http/httputil"
	"net/url"
	"testing"
	"time"

	"github.com/eltoncampos/load-balancer/internal/backend"
	"github.com/eltoncampos/load-balancer/testutil"
)

func createTestBackend(urlStr string) *backend.Backend {
	u, _ := url.Parse(urlStr)
	proxy := httputil.NewSingleHostReverseProxy(u)
	return backend.New(u, proxy)
}

func TestIsBackendAlive_WithRunningServer(t *testing.T) {
	server := testutil.CreateTestServerSimple()
	defer server.Close()

	u, _ := url.Parse(server.URL)
	alive := IsBackendAlive(u)

	if !alive {
		t.Error("expected backend to be alive when server is running")
	}
}

func TestIsBackendAlive_WithStoppedServer(t *testing.T) {
	server := testutil.CreateTestServerSimple()
	serverURL := server.URL
	server.Close()

	u, _ := url.Parse(serverURL)
	alive := IsBackendAlive(u)

	if alive {
		t.Error("expected backend to be dead when server is stopped")
	}
}

func TestIsBackendAlive_WithInvalidHost(t *testing.T) {
	u, _ := url.Parse("http://invalid-host-that-does-not-exist:9999")
	alive := IsBackendAlive(u)

	if alive {
		t.Error("expected backend to be dead with invalid host")
	}
}

func TestCheckBackends_WithMixedStatus(t *testing.T) {
	server := testutil.CreateTestServerSimple()
	defer server.Close()

	backends := []*backend.Backend{
		createTestBackend(server.URL),
		createTestBackend("http://localhost:99999"),
	}

	CheckBackends(backends)

	if !backends[0].IsAlive() {
		t.Error("expected first backend to be alive")
	}

	if backends[1].IsAlive() {
		t.Error("expected second backend to be dead")
	}
}

func TestCheckBackends_WithAllAlive(t *testing.T) {
	server1 := testutil.CreateTestServerSimple()
	defer server1.Close()

	server2 := testutil.CreateTestServerSimple()
	defer server2.Close()

	backends := []*backend.Backend{
		createTestBackend(server1.URL),
		createTestBackend(server2.URL),
	}

	CheckBackends(backends)

	for i, b := range backends {
		if !b.IsAlive() {
			t.Errorf("expected backend %d to be alive", i)
		}
	}
}

func TestCheckBackends_WithAllDead(t *testing.T) {
	backends := []*backend.Backend{
		createTestBackend("http://localhost:99998"),
		createTestBackend("http://localhost:99999"),
	}

	CheckBackends(backends)

	for i, b := range backends {
		if b.IsAlive() {
			t.Errorf("expected backend %d to be dead", i)
		}
	}
}

func TestCheckBackends_UpdatesBackendStatus(t *testing.T) {
	server := testutil.CreateTestServerSimple()
	u, _ := url.Parse(server.URL)

	b := createTestBackend(server.URL)
	b.SetAlive(false)

	backends := []*backend.Backend{b}
	CheckBackends(backends)

	if !backends[0].IsAlive() {
		t.Error("expected backend status to be updated to alive")
	}

	server.Close()
	time.Sleep(10 * time.Millisecond)
	CheckBackends(backends)

	if backends[0].IsAlive() {
		t.Error("expected backend status to be updated to dead after server stopped")
	}

	_ = u
}

func TestStartHealthCheck_ExecutesPeriodically(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	testURL := "http://" + listener.Addr().String()
	b := createTestBackend(testURL)
	backends := []*backend.Backend{b}

	interval := 50 * time.Millisecond

	go StartHealthCheck(backends, interval)

	time.Sleep(interval * 2)

	if !b.IsAlive() {
		t.Error("expected backend to be alive while listener is running")
	}

	listener.Close()
	time.Sleep(interval * 2)

	if b.IsAlive() {
		t.Error("expected backend to be marked as dead after listener closed")
	}
}

func TestCheckBackends_EmptyList(t *testing.T) {
	backends := []*backend.Backend{}
	CheckBackends(backends)
}
