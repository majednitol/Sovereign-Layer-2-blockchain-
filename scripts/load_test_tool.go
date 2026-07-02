package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Sovereign Load Test Simulator
func main() {
	fmt.Println("==================================================")
	fmt.Println("    Sovereign Load Test and TPS Benchmarker       ")
	fmt.Println("==================================================")

	targetTPS := 500
	duration := 5 * time.Second

	var wg sync.Mutex
	txCount := 0
	startTime := time.Now()

	fmt.Printf("Starting load test. Target: %d TPS, Duration: %v...\n", targetTPS, duration)

	stopChan := time.After(duration)
	ticker := time.NewTicker(time.Second / time.Duration(targetTPS))
	defer ticker.Stop()

	// Simulate concurrent client transaction submission workers
	for {
		select {
		case <-stopChan:
			goto Done
		case <-ticker.C:
			// Send transaction (Simulated)
			go func() {
				// Simulate RPC round-trip delay
				time.Sleep(time.Duration(10+rand.Intn(40)) * time.Millisecond)
				wg.Lock()
				txCount++
				wg.Unlock()
			}()
		}
	}

Done:
	elapsed := time.Since(startTime)
	actualTPS := float64(txCount) / elapsed.Seconds()
	fmt.Printf("Load test completed in %v.\n", elapsed)
	fmt.Printf("Total transactions sent: %d\n", txCount)
	fmt.Printf("Measured throughput: %.2f TPS\n", actualTPS)
	fmt.Println("==================================================")
}
