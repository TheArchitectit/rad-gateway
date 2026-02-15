package usage

import (
	"sync"
	"time"

	"radgateway/internal/models"
)

type Record struct {
	Timestamp      time.Time    `json:"timestamp"`
	RequestID      string       `json:"requestId"`
	TraceID        string       `json:"traceId"`
	APIKeyName     string       `json:"apiKeyName"`
	IncomingAPI    string       `json:"incomingApiType"`
	IncomingModel  string       `json:"incomingModel"`
	SelectedModel  string       `json:"selectedModel"`
	Provider       string       `json:"provider"`
	ResponseStatus string       `json:"responseStatus"`
	DurationMs     int64        `json:"durationMs"`
	Usage          models.Usage `json:"usage"`
}

type Sink interface {
	Add(r Record)
	List(limit int) []Record
}

type InMemory struct {
	mu      sync.RWMutex
	records []Record
	max     int
}

func NewInMemory(max int) *InMemory {
	if max <= 0 {
		max = 1000
	}
	return &InMemory{max: max, records: make([]Record, 0, max)}
}

func (s *InMemory) Add(r Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.records) == s.max {
		s.records = s.records[1:]
	}
	s.records = append(s.records, r)
}

func (s *InMemory) List(limit int) []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.records) {
		limit = len(s.records)
	}
	out := make([]Record, 0, limit)
	for i := len(s.records) - 1; i >= len(s.records)-limit; i-- {
		out = append(out, s.records[i])
	}
	return out
}
