package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"Communicate/internal/store/db"
)

const (
	connectionTimeout = 20 * time.Second
	queryTimeout      = 10 * time.Second
	selectRuns        = 5
	reconnectRuns     = 3
	jitterRuns        = 100
	writeLatencyRuns  = 20
)

func main() {
	database, openDuration, err := openDatabase()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	fmt.Printf("database ping ok in %s\n", openDuration)

	// ---------------------------------------------------------------------------
	// Baseline
	// ---------------------------------------------------------------------------
	if err := runRepeatedSelectCheck(database); err != nil {
		log.Fatalf("failed repeated SELECT 1 test: %v", err)
	}

	if err := runReconnectCheck(); err != nil {
		log.Fatalf("failed reconnect test: %v", err)
	}

	// ---------------------------------------------------------------------------
	// Jitter
	// ---------------------------------------------------------------------------
	if err := runJitterCheck(database); err != nil {
		log.Fatalf("failed jitter test: %v", err)
	}

	// ---------------------------------------------------------------------------
	// Replication lag
	// ---------------------------------------------------------------------------
	if err := runReplicationLagCheck(database); err != nil {
		log.Fatalf("failed replication lag test: %v", err)
	}

	// ---------------------------------------------------------------------------
	// Write latency
	// ---------------------------------------------------------------------------
	if err := runWriteLatencyCheck(database); err != nil {
		log.Fatalf("failed write latency test: %v", err)
	}

	// ---------------------------------------------------------------------------
	// Server info
	// ---------------------------------------------------------------------------
	if err := runServerInfo(database); err != nil {
		log.Fatalf("failed server info: %v", err)
	}

	fmt.Println("\nall checks passed")
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func openDatabase() (*db.Database, time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	openStartedAt := time.Now()
	database, err := db.Open(ctx)
	if err != nil {
		return nil, 0, err
	}

	return database, time.Since(openStartedAt), nil
}

func runRepeatedSelectCheck(database *db.Database) error {
	fmt.Printf("\n─── Repeated SELECT 1 (%d runs) ───\n", selectRuns)

	var totalDuration time.Duration

	for i := 1; i <= selectRuns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)

		queryStartedAt := time.Now()
		var result int
		err := database.Pool().QueryRow(ctx, `SELECT 1`).Scan(&result)
		queryDuration := time.Since(queryStartedAt)
		cancel()
		if err != nil {
			return err
		}

		totalDuration += queryDuration
		fmt.Printf("  run %d: result=%d time=%s\n", i, result, queryDuration)
	}

	fmt.Printf("  average: %s\n", totalDuration/time.Duration(selectRuns))
	return nil
}

func runReconnectCheck() error {
	fmt.Printf("\n─── Reconnect checks (%d runs) ───\n", reconnectRuns)

	var totalDuration time.Duration
	var successCount int
	var failureCount int

	for i := 1; i <= reconnectRuns; i++ {
		database, openDuration, err := openDatabase()
		if err != nil {
			failureCount++
			fmt.Printf("  run %d: failed=%v\n", i, err)
			continue
		}

		totalDuration += openDuration
		successCount++
		fmt.Printf("  run %d: handshake=%s\n", i, openDuration)
		database.Close()
	}

	if successCount > 0 {
		fmt.Printf("  average handshake: %s\n", totalDuration/time.Duration(successCount))
	}
	fmt.Printf("  summary: success=%d failed=%d timeout=%s\n", successCount, failureCount, connectionTimeout)

	if successCount == 0 {
		return fmt.Errorf("all reconnect checks failed")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Jitter – query consistency over many iterations
// ---------------------------------------------------------------------------

func runJitterCheck(database *db.Database) error {
	fmt.Printf("\n─── Jitter test (%d SELECT 1 runs) ───\n", jitterRuns)

	var durations []float64
	var totalDuration time.Duration
	var minDuration, maxDuration time.Duration
	minDuration = 1<<63 - 1 // max duration

	for i := 1; i <= jitterRuns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)

		startedAt := time.Now()
		var result int
		err := database.Pool().QueryRow(ctx, `SELECT 1`).Scan(&result)
		d := time.Since(startedAt)
		cancel()
		if err != nil {
			return fmt.Errorf("run %d: %w", i, err)
		}

		durations = append(durations, float64(d))
		totalDuration += d
		if d < minDuration {
			minDuration = d
		}
		if d > maxDuration {
			maxDuration = d
		}
	}

	sort.Float64s(durations)

	mean := totalDuration.Seconds() / float64(jitterRuns)
	median := durations[len(durations)/2]
	p95 := durations[int(float64(len(durations))*0.95)]
	p99 := durations[int(float64(len(durations))*0.99)]

	var varianceSum float64
	for _, d := range durations {
		diff := d/1e9 - mean // both in seconds for stability
		varianceSum += diff * diff
	}
	stddev := math.Sqrt(varianceSum/float64(jitterRuns)) * 1e9 // back to nanoseconds
	jitter := maxDuration - minDuration

	fmt.Printf("  samples   : %d\n", jitterRuns)
	fmt.Printf("  min       : %s\n", minDuration)
	fmt.Printf("  max       : %s\n", maxDuration)
	fmt.Printf("  mean      : %s\n", time.Duration(mean*1e9))
	fmt.Printf("  median    : %s\n", time.Duration(median))
	fmt.Printf("  P95       : %s\n", time.Duration(p95))
	fmt.Printf("  P99       : %s\n", time.Duration(p99))
	fmt.Printf("  stddev    : %s\n", time.Duration(stddev))
	fmt.Printf("  jitter    : %s (max-min spread)\n", jitter)

	return nil
}

// ---------------------------------------------------------------------------
// Replication lag
// ---------------------------------------------------------------------------

func runReplicationLagCheck(database *db.Database) error {
	fmt.Printf("\n─── Replication lag ───\n")

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	// Check if this server is a replica
	var inRecovery bool
	if err := database.Pool().QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&inRecovery); err != nil {
		return fmt.Errorf("pg_is_in_recovery: %w", err)
	}

	if inRecovery {
		fmt.Printf("  role: REPLICA\n")

		// WAL lag – bytes behind primary
		var receiveLSN, replayLSN string
		if err := database.Pool().QueryRow(ctx,
			`SELECT pg_last_wal_receive_lsn()::text, pg_last_wal_replay_lsn()::text`,
		).Scan(&receiveLSN, &replayLSN); err != nil {
			return fmt.Errorf("wal lsn: %w", err)
		}

		var lagBytes int64
		if err := database.Pool().QueryRow(ctx,
			`SELECT COALESCE(pg_wal_lsn_diff(pg_last_wal_receive_lsn(), pg_last_wal_replay_lsn()), 0)`,
		).Scan(&lagBytes); err != nil {
			return fmt.Errorf("wal diff: %w", err)
		}

		fmt.Printf("  receive LSN  : %s\n", receiveLSN)
		fmt.Printf("  replay LSN   : %s\n", replayLSN)
		fmt.Printf("  lag bytes    : %d\n", lagBytes)
	} else {
		fmt.Printf("  role: PRIMARY\n")

		// Show connected replicas
		rows, err := database.Pool().Query(ctx, `
			SELECT
				COALESCE(client_addr::text, 'local'),
				COALESCE(state, 'unknown'),
				COALESCE(pg_wal_lsn_diff(sent_lsn, write_lsn), 0) AS write_lag_bytes,
				COALESCE(pg_wal_lsn_diff(sent_lsn, flush_lsn), 0) AS flush_lag_bytes,
				COALESCE(pg_wal_lsn_diff(sent_lsn, replay_lsn), 0) AS replay_lag_bytes
			FROM pg_stat_replication
		`)
		if err != nil {
			return fmt.Errorf("pg_stat_replication: %w", err)
		}
		defer rows.Close()

		replicaCount := 0
		for rows.Next() {
			var addr, state string
			var writeLag, flushLag, replayLag int64
			if err := rows.Scan(&addr, &state, &writeLag, &flushLag, &replayLag); err != nil {
				return err
			}
			fmt.Printf("  replica %d: addr=%s state=%s write_lag_bytes=%d flush_lag_bytes=%d replay_lag_bytes=%d\n",
				replicaCount+1, addr, state, writeLag, flushLag, replayLag)
			replicaCount++
		}
		if replicaCount == 0 {
			fmt.Printf("  replicas: none connected\n")
		}
	}

	// Clock skew — compare server NOW() to local time
	var serverNow time.Time
	if err := database.Pool().QueryRow(ctx, `SELECT NOW()`).Scan(&serverNow); err != nil {
		return fmt.Errorf("server now: %w", err)
	}
	clockSkew := time.Since(serverNow)
	fmt.Printf("  clock skew  : %s (local - server)\n", clockSkew)

	return nil
}

// ---------------------------------------------------------------------------
// Write latency – temp table insert, select, drop
// ---------------------------------------------------------------------------

func runWriteLatencyCheck(database *db.Database) error {
	fmt.Printf("\n─── Write latency (%d runs) ───\n", writeLatencyRuns)

	var totalWrite, totalRead time.Duration

	for i := 1; i <= writeLatencyRuns; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)

		// CREATE TEMP TABLE
		startedAt := time.Now()
		_, err := database.Pool().Exec(ctx, `
			CREATE TEMP TABLE IF NOT EXISTS _dbcheck_write_test (
				id SERIAL PRIMARY KEY,
				payload TEXT NOT NULL DEFAULT md5(random()::text),
				created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
			)
		`)
		createDuration := time.Since(startedAt)
		if err != nil {
			cancel()
			return fmt.Errorf("create temp table: %w", err)
		}

		// INSERT
		startedAt = time.Now()
		_, err = database.Pool().Exec(ctx, `
			INSERT INTO _dbcheck_write_test (payload) VALUES (md5(random()::text))
		`)
		insertDuration := time.Since(startedAt)
		if err != nil {
			cancel()
			return fmt.Errorf("insert: %w", err)
		}

		// SELECT the row back
		startedAt = time.Now()
		var id int
		var payload string
		err = database.Pool().QueryRow(ctx, `SELECT id, payload FROM _dbcheck_write_test LIMIT 1`).Scan(&id, &payload)
		selectDuration := time.Since(startedAt)
		if err != nil {
			cancel()
			return fmt.Errorf("select back: %w", err)
		}

		// DROP
		startedAt = time.Now()
		_, err = database.Pool().Exec(ctx, `DROP TABLE IF EXISTS _dbcheck_write_test`)
		dropDuration := time.Since(startedAt)
		cancel()
		if err != nil {
			return fmt.Errorf("drop temp table: %w", err)
		}

		totalWrite += createDuration + insertDuration + dropDuration
		totalRead += selectDuration

		if i <= 5 || i == writeLatencyRuns {
			fmt.Printf("  run %d: create=%s insert=%s select=%s drop=%s\n",
				i, createDuration, insertDuration, selectDuration, dropDuration)
		} else if i == 6 {
			fmt.Printf("  ...\n")
		}
	}

	fmt.Printf("  average write (create+insert+drop): %s\n", totalWrite/time.Duration(writeLatencyRuns))
	fmt.Printf("  average read  (select back)        : %s\n", totalRead/time.Duration(writeLatencyRuns))

	return nil
}

// ---------------------------------------------------------------------------
// Server info
// ---------------------------------------------------------------------------

func runServerInfo(database *db.Database) error {
	fmt.Printf("\n─── Server info ───\n")

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	// PostgreSQL version
	var version string
	if err := database.Pool().QueryRow(ctx, `SELECT version()`).Scan(&version); err != nil {
		return fmt.Errorf("version: %w", err)
	}
	fmt.Printf("  version     : %s\n", version)

	// Uptime
	var uptime string
	if err := database.Pool().QueryRow(ctx,
		`SELECT COALESCE(pg_postmaster_start_time()::text, 'unknown')`,
	).Scan(&uptime); err != nil {
		return fmt.Errorf("uptime: %w", err)
	}
	fmt.Printf("  started at  : %s\n", uptime)

	// Connection count
	var totalConns, activeConns int
	if err := database.Pool().QueryRow(ctx,
		`SELECT count(*), count(*) FILTER (WHERE state = 'active') FROM pg_stat_activity`,
	).Scan(&totalConns, &activeConns); err != nil {
		return fmt.Errorf("connections: %w", err)
	}
	fmt.Printf("  connections : %d total, %d active\n", totalConns, activeConns)

	return nil
}
