package healthcheck

import (
	"log"
	"net"
	"net/url"
	"time"

	"github.com/eltoncampos/load-balancer/internal/backend"
)

func IsBackendAlive(u *url.URL) bool {
	timeout := 2 * time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Println("Site unreachable, error: ", err)
		return false
	}
	_ = conn.Close()
	return true
}

func CheckBackends(backends []*backend.Backend) {
	for _, b := range backends {
		status := "up"
		alive := IsBackendAlive(b.URL)
		b.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s [%s]\n", b.URL, status)
	}
}

func StartHealthCheck(backends []*backend.Backend, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()

	for range t.C {
		log.Println("Starting health check...")
		CheckBackends(backends)
		log.Println("Health check completed")
	}
}
