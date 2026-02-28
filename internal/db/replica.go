// Package db provides database interfaces and implementations for RAD Gateway.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"

	_ "github.com/lib/pq"
)

// ReplicaRouter routes queries between primary and replica databases
type ReplicaRouter struct {
	primary  *sql.DB
	replicas []*sql.DB
	counter  uint32 // For round-robin replica selection
}

// NewReplicaRouter creates a new replica router
func NewReplicaRouter(primaryDSN string, replicaDSNs []string) (*ReplicaRouter, error) {
	// Connect to primary
	primary, err := sql.Open("postgres", primaryDSN)
	if err != nil {
		return nil, fmt.Errorf("connect to primary: %w", err)
	}

	// Configure primary pool
	primary.SetMaxOpenConns(10)
	primary.SetMaxIdleConns(3)

	// Connect to replicas
	replicas := make([]*sql.DB, 0, len(replicaDSNs))
	for _, dsn := range replicaDSNs {
		if dsn == "" {
			continue
		}
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("connect to replica: %w", err)
		}
		db.SetMaxOpenConns(10)
		db.SetMaxIdleConns(3)
		replicas = append(replicas, db)
	}

	return &ReplicaRouter{
		primary:  primary,
		replicas: replicas,
	}, nil
}

// GetWriter returns the primary database for writes
func (r *ReplicaRouter) GetWriter() *sql.DB {
	return r.primary
}

// GetReader returns a replica database for reads (round-robin)
func (r *ReplicaRouter) GetReader() *sql.DB {
	if len(r.replicas) == 0 {
		return r.primary // Fallback to primary if no replicas
	}

	idx := atomic.AddUint32(&r.counter, 1) % uint32(len(r.replicas))
	return r.replicas[idx]
}

// Close closes all database connections
func (r *ReplicaRouter) Close() error {
	r.primary.Close()
	for _, replica := range r.replicas {
		replica.Close()
	}
	return nil
}

// Health checks all connections
func (r *ReplicaRouter) Health(ctx context.Context) map[string]error {
	results := make(map[string]error)

	// Check primary
	results["primary"] = r.primary.PingContext(ctx)

	// Check replicas
	for i, replica := range r.replicas {
		results[fmt.Sprintf("replica-%d", i)] = replica.PingContext(ctx)
	}

	return results
}
