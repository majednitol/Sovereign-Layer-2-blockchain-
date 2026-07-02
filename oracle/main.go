package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sovereign-l1/chain/x/oracle"
)

var (
	roundsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oracle_rounds_total",
			Help: "Total number of oracle rounds processed.",
		},
		[]string{"feed_id"},
	)
	skippedRoundsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oracle_skipped_rounds_total",
			Help: "Total number of oracle rounds skipped due to outage.",
		},
		[]string{"feed_id"},
	)
	priceValue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "oracle_price_value",
			Help: "Latest successfully fetched price value.",
		},
		[]string{"feed_id"},
	)
	broadcastErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "oracle_broadcast_errors_total",
			Help: "Total number of transaction broadcast errors.",
		},
		[]string{"feed_id", "stage"},
	)
)

func init() {
	prometheus.MustRegister(roundsTotal)
	prometheus.MustRegister(skippedRoundsTotal)
	prometheus.MustRegister(priceValue)
	prometheus.MustRegister(broadcastErrorsTotal)
}

func retryWithBackoff(ctx context.Context, feedID, stage string, operation func() error) error {
	baseDelay := 100 * time.Millisecond
	maxDelay := 2 * time.Second
	factor := 2.0

	delay := baseDelay
	for {
		err := operation()
		if err == nil {
			return nil
		}

		broadcastErrorsTotal.WithLabelValues(feedID, stage).Inc()
		fmt.Printf("[Oracle] [%s] [%s] Broadcast failed: %v. Retrying in %v...\n", feedID, stage, err, delay)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * factor)
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func runFeedWorker(ctx context.Context, feedID string, sources []PriceSource, client *ChainClient, hsm KeyManager, maxRounds int, wg *sync.WaitGroup) {
	defer wg.Done()

	fetcher := NewFetcher(sources)
	operator := "cosmosvaloper1x..."
	roundID := uint64(1)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		roundsTotal.WithLabelValues(feedID).Inc()

		// 1. Fetch Price with consensus
		fmt.Printf("[Oracle] [%s] Starting round %d, fetching price...\n", feedID, roundID)
		value, err := fetcher.FetchPrice(ctx)
		if err != nil {
			// BSC Outage / RPC Outage skipping logic
			skippedRoundsTotal.WithLabelValues(feedID).Inc()
			fmt.Printf("[Oracle] [%s] Skipping round %d due to outage / fetching failure: %v\n", feedID, roundID, err)
			time.Sleep(5 * time.Second)
			roundID++
			if maxRounds > 0 && int(roundID) > maxRounds {
				return
			}
			continue
		}

		priceValue.WithLabelValues(feedID).Set(float64(value))
		nonce := fmt.Sprintf("nonce_%d_%d", roundID, rand.Intn(100000))
		hash := oracle.ComputeCommitHash(operator, feedID, roundID, value, nonce)

		// 2. Commit Stage
		err = retryWithBackoff(ctx, feedID, "commit", func() error {
			return client.BroadcastCommit(ctx, operator, feedID, roundID, hash)
		})
		if err != nil {
			fmt.Printf("[Oracle] [%s] Failed to commit round %d: %v\n", feedID, roundID, err)
			return
		}

		// 3. Wait for reveal window (simulated delay)
		time.Sleep(500 * time.Millisecond)

		// 4. Reveal Stage
		err = retryWithBackoff(ctx, feedID, "reveal", func() error {
			return client.BroadcastReveal(ctx, operator, feedID, roundID, value, nonce)
		})
		if err != nil {
			fmt.Printf("[Oracle] [%s] Failed to reveal round %d: %v\n", feedID, roundID, err)
			return
		}

		fmt.Printf("[Oracle] [%s] Finished round %d, price: %d\n", feedID, roundID, value)

		roundID++
		if maxRounds > 0 && int(roundID) > maxRounds {
			return
		}

		time.Sleep(2 * time.Second)
	}
}

func main() {
	fmt.Println("--------------------------------------------------")
	fmt.Println("Sovereign L1 Oracle Aggregator Daemon")
	fmt.Println("--------------------------------------------------")

	// Start Prometheus server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		fmt.Println("[Prometheus] Serving metrics on :9200/metrics")
		if err := http.ListenAndServe(":9200", nil); err != nil {
			fmt.Printf("[Prometheus] Server failed: %v\n", err)
		}
	}()

	feeds := map[string][]PriceSource{
		"BTC_USD": {
			{Name: "primary-btc", URL: "http://localhost:8080/price/btc"},
			{Name: "fallback-btc", URL: "http://localhost:8081/price/btc"},
		},
		"ETH_USD": {
			{Name: "primary-eth", URL: "http://localhost:8082/price/eth"},
			{Name: "fallback-eth", URL: "http://localhost:8083/price/eth"},
		},
	}

	hsm, err := NewHSMKeyManager(os.Getenv("HSM_CONFIG"), []byte("oracle_key"))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize HSM key manager: %v", err))
	}

	client := NewChainClient("localhost:9090", hsm)

	maxRounds := 0
	if maxRoundsStr := os.Getenv("MAX_ROUNDS"); maxRoundsStr != "" {
		if val, err := strconv.Atoi(maxRoundsStr); err == nil {
			maxRounds = val
			fmt.Printf("[Oracle] Running in test mode, limit to %d rounds\n", maxRounds)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	for feedID, sources := range feeds {
		wg.Add(1)
		go runFeedWorker(ctx, feedID, sources, client, hsm, maxRounds, &wg)
	}

	wg.Wait()
	fmt.Println("[PASS] Oracle Aggregator executed successfully.")
}
