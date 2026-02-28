package db

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewReplicaRouter(t *testing.T) {
	// Skip if no PostgreSQL
	primaryDSN := os.Getenv("TEST_PRIMARY_DSN")
	if primaryDSN == "" {
		t.Skip("TEST_PRIMARY_DSN not set")
	}

	replicaDSN := os.Getenv("TEST_REPLICA_DSN")
	var replicaDSNs []string
	if replicaDSN != "" {
		replicaDSNs = []string{replicaDSN}
	}

	router, err := NewReplicaRouter(primaryDSN, replicaDSNs)
	if err != nil {
		t.Fatalf("NewReplicaRouter failed: %v", err)
	}
	defer router.Close()

	// Verify primary is set
	if router.GetWriter() == nil {
		t.Error("GetWriter() returned nil")
	}

	// Verify reader is set
	if router.GetReader() == nil {
		t.Error("GetReader() returned nil")
	}
}

func TestReplicaRouter_GetReader(t *testing.T) {
	primaryDSN := os.Getenv("TEST_PRIMARY_DSN")
	if primaryDSN == "" {
		t.Skip("TEST_PRIMARY_DSN not set")
	}

	replicaDSN := os.Getenv("TEST_REPLICA_DSN")
	var replicaDSNs []string
	if replicaDSN != "" {
		replicaDSNs = []string{replicaDSN}
	}

	router, err := NewReplicaRouter(primaryDSN, replicaDSNs)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer router.Close()

	// Test round-robin
	reader1 := router.GetReader()
	if reader1 == nil {
		t.Fatal("GetReader() returned nil")
	}

	// Should return same connection if no replicas
	reader2 := router.GetReader()
	if reader2 == nil {
		t.Error("GetReader() returned nil on second call")
	}
}

func TestReplicaRouter_Health(t *testing.T) {
	primaryDSN := os.Getenv("TEST_PRIMARY_DSN")
	if primaryDSN == "" {
		t.Skip("TEST_PRIMARY_DSN not set")
	}

	replicaDSN := os.Getenv("TEST_REPLICA_DSN")
	var replicaDSNs []string
	if replicaDSN != "" {
		replicaDSNs = []string{replicaDSN}
	}

	router, err := NewReplicaRouter(primaryDSN, replicaDSNs)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer router.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := router.Health(ctx)

	// Verify primary is healthy
	if results["primary"] != nil {
		t.Errorf("primary health check failed: %v", results["primary"])
	}
}

func TestReplicaRouter_NoReplicas(t *testing.T) {
	primaryDSN := os.Getenv("TEST_PRIMARY_DSN")
	if primaryDSN == "" {
		t.Skip("TEST_PRIMARY_DSN not set")
	}

	// Create router with no replicas
	router, err := NewReplicaRouter(primaryDSN, nil)
	if err != nil {
		t.Skipf("Skipping: %v", err)
	}
	defer router.Close()

	// Reader should fallback to primary
	reader := router.GetReader()
	if reader == nil {
		t.Fatal("GetReader() returned nil")
	}

	// Should be same as writer
	writer := router.GetWriter()
	if reader != writer {
		t.Error("GetReader() should return primary when no replicas")
	}
}
