package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCollector(t *testing.T) {
	c := NewCollector()
	if c == nil {
		t.Fatal("NewCollector() returned nil")
	}
}

func TestCollector_RecordHTTPRequest(t *testing.T) {
	c := NewCollector()

	c.RecordHTTPRequest(100*time.Millisecond, 200)
	c.RecordHTTPRequest(200*time.Millisecond, 500)

	output := c.PrometheusFormat()

	if !strings.Contains(output, "radgateway_http_requests_total 2") {
		t.Error("Expected request count of 2")
	}
	if !strings.Contains(output, "radgateway_http_request_errors_total 1") {
		t.Error("Expected error count of 1")
	}
}

func TestCollector_RecordDBQuery(t *testing.T) {
	c := NewCollector()

	c.RecordDBQuery(50*time.Millisecond, nil)
	c.RecordDBQuery(100*time.Millisecond, nil)
	c.RecordDBQuery(150*time.Millisecond, nil)

	output := c.PrometheusFormat()

	if !strings.Contains(output, "radgateway_db_queries_total 3") {
		t.Error("Expected query count of 3")
	}
}

func TestCollector_RecordProviderRequest(t *testing.T) {
	c := NewCollector()

	c.RecordProviderRequest("openai", 100*time.Millisecond, nil)
	c.RecordProviderRequest("openai", 200*time.Millisecond, nil)
	c.RecordProviderRequest("anthropic", 150*time.Millisecond, nil)

	output := c.PrometheusFormat()

	if !strings.Contains(output, `radgateway_provider_requests_total{provider="openai"} 2`) {
		t.Error("Expected 2 OpenAI requests")
	}
	if !strings.Contains(output, `radgateway_provider_requests_total{provider="anthropic"} 1`) {
		t.Error("Expected 1 Anthropic request")
	}
}

func TestCollector_PrometheusFormat(t *testing.T) {
	c := NewCollector()

	// Record some metrics
	c.RecordHTTPRequest(100*time.Millisecond, 200)
	c.RecordDBQuery(50*time.Millisecond, nil)
	c.RecordA2ATask(true)

	output := c.PrometheusFormat()

	// Verify format contains expected metrics
	expectedMetrics := []string{
		"radgateway_http_requests_total",
		"radgateway_db_queries_total",
		"radgateway_a2a_tasks_created_total",
		"radgateway_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(output, metric) {
			t.Errorf("Expected output to contain %s", metric)
		}
	}
}

func TestCollector_Handler(t *testing.T) {
	c := NewCollector()

	// Record some metrics
	c.RecordHTTPRequest(100*time.Millisecond, 200)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()

	handler := c.Handler()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("Expected text/plain content type, got %s", contentType)
	}

	if !strings.Contains(rr.Body.String(), "radgateway_http_requests_total") {
		t.Error("Expected metrics in response")
	}
}
