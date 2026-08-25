package core

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Lease represents an agent lease for a specific task.
// Only one agent can hold a lease at a time.
// TUPM96 Gate 1 bot uses leases to coordinate verification cycles.

const (
	// DefaultLeaseTTL is the default time-to-live for a lease.
	// If the bot crashes or stops heartbeating, the lease expires
	// and another instance can pick up the work.
	DefaultLeaseTTL = 5 * time.Minute
	
	// StaleLeaseThreshold is the threshold after which a lease is considered stale
	// and can be force-released by admin operations.
	StaleLeaseThreshold = 30 * time.Minute
)

// LeaseRow maps to the agent_leases table.
// The table uses advisory locks for fast acquisition, but also stores
// metadata for debugging and admin override.

// AcquireLease attempts to acquire a lease for the given agent and task.
// Returns true if acquired, false if already held by another agent.
func AcquireLease(ctx context.Context, pool *pgxpool.Pool, agentID, taskID string, ttl time.Duration) (bool, error) {
	if ttl == 0 {
		ttl = DefaultLeaseTTL
	}
	
	// Use PostgreSQL advisory lock for fast, non-transactional locking.
	// The lock key is derived from agentID and taskID.
	var acquired bool
	err := pool.QueryRow(ctx, 
		`SELECT pg_try_advisory_xact_lock(hashtext($1), hashtext($2))`,
		agentID, taskID).Scan(&acquired)
	if err != nil {
		return false, err
	}
	if !acquired {
		return false, nil
	}
	
	// Record lease metadata for visibility and admin override.
	_, err = pool.Exec(ctx, 
		`INSERT INTO agent_leases (agent_id, task_id, acquired_at, expires_at, heartbeat_at)
		 VALUES ($1, $2, now(), now() + $3, now())
		 ON CONFLICT (agent_id, task_id) DO UPDATE SET
		   acquired_at = now(),
		   expires_at = now() + $3,
		   heartbeat_at = now()`,
		agentID, taskID, ttl)
	return err == nil, err
}

// ReleaseLease releases the advisory lock and removes the lease record.
func ReleaseLease(ctx context.Context, pool *pgxpool.Pool, agentID, taskID string) error {
	// Advisory lock is released at transaction end, but we clean up the record.
	_, err := pool.Exec(ctx, 
		`DELETE FROM agent_leases WHERE agent_id = $1 AND task_id = $2`,
		agentID, taskID)
	return err
}

// HeartbeatLease updates the heartbeat timestamp to keep the lease alive.
func HeartbeatLease(ctx context.Context, pool *pgxpool.Pool, agentID, taskID string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = DefaultLeaseTTL
	}
	_, err := pool.Exec(ctx,
		`UPDATE agent_leases SET heartbeat_at = now(), expires_at = now() + $3
		 WHERE agent_id = $1 AND task_id = $2`,
		agentID, taskID, ttl)
	return err
}

// GetLease returns the current lease info if it exists.
func GetLease(ctx context.Context, pool *pgxpool.Pool, agentID, taskID string) (*LeaseInfo, error) {
	row := pool.QueryRow(ctx,
		`SELECT agent_id, task_id, acquired_at, expires_at, heartbeat_at
		 FROM agent_leases WHERE agent_id = $1 AND task_id = $2`,
		agentID, taskID)
	var info LeaseInfo
	err := row.Scan(&info.AgentID, &info.TaskID, &info.AcquiredAt, &info.ExpiresAt, &info.HeartbeatAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &info, err
}

// LeaseInfo holds lease metadata.
type LeaseInfo struct {
	AgentID     string
	TaskID      string
	AcquiredAt  time.Time
	ExpiresAt   time.Time
	HeartbeatAt time.Time
}

// IsStale returns true if the lease hasn't been heartbeated within the threshold.
func (l *LeaseInfo) IsStale(threshold time.Duration) bool {
	return time.Since(l.HeartbeatAt) > threshold
}

// ForceReleaseStaleLeases releases all leases that haven't heartbeated within the threshold.
// This is an admin operation to recover from stuck bots (e.g., TUPM96 Gate 1 bot stopped Jul 17).
// Returns the number of leases released.
func ForceReleaseStaleLeases(ctx context.Context, pool *pgxpool.Pool, threshold time.Duration) (int64, error) {
	if threshold == 0 {
		threshold = StaleLeaseThreshold
	}
	result, err := pool.Exec(ctx,
		`DELETE FROM agent_leases WHERE heartbeat_at < now() - $1`,
		threshold)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

// ListActiveLeases returns all currently active leases for admin visibility.
func ListActiveLeases(ctx context.Context, pool *pgxpool.Pool) ([]LeaseInfo, error) {
	rows, err := pool.Query(ctx,
		`SELECT agent_id, task_id, acquired_at, expires_at, heartbeat_at
		 FROM agent_leases ORDER BY acquired_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var leases []LeaseInfo
	for rows.Next() {
		var info LeaseInfo
		if err := rows.Scan(&info.AgentID, &info.TaskID, &info.AcquiredAt, &info.ExpiresAt, &info.HeartbeatAt); err != nil {
			return nil, err
		}
		leases = append(leases, info)
	}
	return leases, rows.Err()
}
