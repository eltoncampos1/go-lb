package pool

import (
	"net/url"
	"sync/atomic"

	"github.com/eltoncampos/load-balancer/internal/backend"
)

type ServerPool struct {
	backends []*backend.Backend
	current  uint64
}

func New() *ServerPool {
	return &ServerPool{
		backends: make([]*backend.Backend, 0),
	}
}

func (s *ServerPool) AddBackend(b *backend.Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) GetNextPeer() *backend.Backend {
	next := s.NextIndex()
	l := len(s.backends) + next

	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func (s *ServerPool) MarkBackendStatus(backendUrl *url.URL, alive bool) {
	for _, b := range s.backends {
		if b.URL.String() == backendUrl.String() {
			b.SetAlive(alive)
			break
		}
	}
}

func (s *ServerPool) GetBackends() []*backend.Backend {
	return s.backends
}
