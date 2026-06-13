package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"Communicate/internal/store/db"
)

const (
	connectionTimeout = 20 * time.Second
	queryTimeout      = 10 * time.Second
	selectRuns        = 5
	reconnectRuns     = 3
)

func main() {
	database, openDuration, err := openDatabase()
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	fmt.Printf("database ping ok in %s\n", openDuration)

	if err := runRepeatedSelectCheck(database); err != nil {
		log.Fatalf("failed repeated SELECT 1 test: %v", err)
	}

	if err := runReconnectCheck(); err != nil {
		log.Fatalf("failed reconnect test: %v", err)
	}
}

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
	fmt.Printf("running %d repeated SELECT 1 queries\n", selectRuns)

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
		fmt.Printf("SELECT 1 run %d: result=%d time=%s\n", i, result, queryDuration)
	}

	fmt.Printf("SELECT 1 average time: %s\n", totalDuration/time.Duration(selectRuns))
	return nil
}

func runReconnectCheck() error {
	fmt.Printf("running %d reconnect checks\n", reconnectRuns)

	var totalDuration time.Duration
	var successCount int
	var failureCount int

	for i := 1; i <= reconnectRuns; i++ {
		database, openDuration, err := openDatabase()
		if err != nil {
			failureCount++
			fmt.Printf("reconnect run %d: failed=%v\n", i, err)
			continue
		}

		totalDuration += openDuration
		successCount++
		fmt.Printf("reconnect run %d: handshake=%s\n", i, openDuration)
		database.Close()
	}

	if successCount > 0 {
		fmt.Printf("average reconnect handshake: %s\n", totalDuration/time.Duration(successCount))
	}
	fmt.Printf("reconnect summary: success=%d failed=%d timeout=%s\n", successCount, failureCount, connectionTimeout)

	if successCount == 0 {
		return fmt.Errorf("all reconnect checks failed")
	}

	return nil
}
