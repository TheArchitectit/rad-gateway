package db

import (
	"errors"
	"testing"
	"time"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()
	if mc == nil {
		t.Fatal("NewMetricsCollector() returned nil")
	}

	// Verify initial stats are zero
	stats := mc.GetStats()
	if stats.QueryCount != 0 {
		t.Errorf("QueryCount = %d, want 0", stats.QueryCount)
	}
	if stats.QueryErrors != 0 {
		t.Errorf("QueryErrors = %d, want 0", stats.QueryErrors)
	}
}

func TestMetricsCollector_RecordQuery(t *testing.T) {
	mc := NewMetricsCollector()

	// Record successful query
	mc.RecordQuery("SELECT", 100*time.Millisecond, nil)

	stats := mc.GetStats()
	if stats.QueryCount != 1 {
		t.Errorf("QueryCount = %d, want 1", stats.QueryCount)
	}
	if stats.QueryErrors != 0 {
		t.Errorf("QueryErrors = %d, want 0", stats.QueryErrors)
	}
	if stats.AvgLatencyMs != 100.0 {
		t.Errorf("AvgLatencyMs = %f, want 100.0", stats.AvgLatencyMs)
	}
}

func TestMetricsCollector_RecordQueryError(t *testing.T) {
	mc := NewMetricsCollector()

	// Record failed query
	mc.RecordQuery("INSERT", 50*time.Millisecond, errors.New("connection failed"))

	stats := mc.GetStats()
	if stats.QueryCount != 1 {
		t.Errorf("QueryCount = %d, want 1", stats.QueryCount)
	}
	if stats.QueryErrors != 1 {
		t.Errorf("QueryErrors = %d, want 1", stats.QueryErrors)
	}
}

func TestMetricsCollector_QueryTypeStats(t *testing.T) {
	mc := NewMetricsCollector()

	// Record different query types
	mc.RecordQuery("SELECT", 100*time.Millisecond, nil)
	mc.RecordQuery("SELECT", 200*time.Millisecond, nil)
	mc.RecordQuery("INSERT", 50*time.Millisecond, nil)

	stats := mc.GetStats()

	// Check SELECT stats
	selectStats, ok := stats.QueryTypes["SELECT"]
	if !ok {
		t.Fatal("SELECT stats not found")
	}
	if selectStats.Count != 2 {
		t.Errorf("SELECT Count = %d, want 2", selectStats.Count)
	}

	// Check INSERT stats
	insertStats, ok := stats.QueryTypes["INSERT"]
	if !ok {
		t.Fatal("INSERT stats not found")
	}
	if insertStats.Count != 1 {
		t.Errorf("INSERT Count = %d, want 1", insertStats.Count)
	}
}

func TestMetricsCollector_AvgLatency(t *testing.T) {
	mc := NewMetricsCollector()

	// Should be 0 initially
	if mc.getAvgLatency() != 0 {
		t.Errorf("avg latency = %f, want 0", mc.getAvgLatency())
	}

	// Record queries
	mc.RecordQuery("SELECT", 100*time.Millisecond, nil)
	mc.RecordQuery("SELECT", 200*time.Millisecond, nil)

	// Average should be 150ms
	if mc.getAvgLatency() != 150.0 {
		t.Errorf("avg latency = %f, want 150.0", mc.getAvgLatency())
	}
}

func TestMetricsCollector_HealthCheck(t *testing.T) {
	tests := []struct {
		name      string
		queries   int
		errors    int
		wantHealthy bool
	}{
		{
			name:      "healthy - no errors",
			queries:   100,
			errors:    0,
			wantHealthy: true,
		},
		{
			name:      "healthy - low error rate",
			queries:   100,
			errors:    4, // 4% error rate
			wantHealthy: true,
		},
		{
			name:      "unhealthy - high error rate",
			queries:   100,
			errors:    10, // 10% error rate
			wantHealthy: false,
		},
		{
			name:      "healthy - no queries",
			queries:   0,
			errors:    0,
			wantHealthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMetricsCollector()

			// Record queries
			for i := 0; i < tt.queries-tt.errors; i++ {
				mc.RecordQuery("SELECT", 10*time.Millisecond, nil)
			}
			for i := 0; i < tt.errors; i++ {
				mc.RecordQuery("SELECT", 10*time.Millisecond, errors.New("error"))
			}

			health := mc.HealthCheck()
			if health.Healthy != tt.wantHealthy {
				t.Errorf("Healthy = %v, want %v", health.Healthy, tt.wantHealthy)
			}
		})
	}
}

func TestMetricsCollector_HealthCheck_ErrorRate(t *testing.T) {
	mc := NewMetricsCollector()

	// Record 10 queries with 2 errors (20% error rate)
	for i := 0; i < 8; i++ {
		mc.RecordQuery("SELECT", 10*time.Millisecond, nil)
	}
	for i := 0; i < 2; i++ {
		mc.RecordQuery("SELECT", 10*time.Millisecond, errors.New("error"))
	}

	health := mc.HealthCheck()
	if health.ErrorRate != 0.2 {
		t.Errorf("ErrorRate = %f, want 0.2", health.ErrorRate)
	}
	if health.QueryCount != 10 {
		t.Errorf("QueryCount = %d, want 10", health.QueryCount)
	}
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	mc := NewMetricsCollector()

	// Record queries concurrently
	done := make(chan bool, 100)
	for i := 0; i < 100; i++ {
		go func() {
			mc.RecordQuery("SELECT", 10*time.Millisecond, nil)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	stats := mc.GetStats()
	if stats.QueryCount != 100 {
		t.Errorf("QueryCount = %d, want 100", stats.QueryCount)
	}
}
