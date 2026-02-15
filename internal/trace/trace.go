package trace

import (
	"sync"
	"time"
)

type Event struct {
	Timestamp time.Time `json:"timestamp"`
	TraceID   string    `json:"traceId"`
	RequestID string    `json:"requestId"`
	Message   string    `json:"message"`
}

type Store struct {
	mu     sync.RWMutex
	events []Event
	max    int
}

func NewStore(max int) *Store {
	if max <= 0 {
		max = 2000
	}
	return &Store{events: make([]Event, 0, max), max: max}
}

func (s *Store) Add(e Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.events) == s.max {
		s.events = s.events[1:]
	}
	s.events = append(s.events, e)
}

func (s *Store) List(limit int) []Event {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.events) {
		limit = len(s.events)
	}
	out := make([]Event, 0, limit)
	for i := len(s.events) - 1; i >= len(s.events)-limit; i-- {
		out = append(out, s.events[i])
	}
	return out
}
