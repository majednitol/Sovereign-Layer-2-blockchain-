package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

func runFeedWorker(ctx context.Context, operator string, feedID string, sources []PriceSource, client *ChainClient, hsm KeyManager, maxRounds int, wg *sync.WaitGroup) {
	defer wg.Done()

	fetcher := NewFetcher(sources)
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

		// M2 FIX: Use crypto/rand for nonce generation (256-bit entropy)
		nonceBytes := make([]byte, 32)
		if _, err := rand.Read(nonceBytes); err != nil {
			fmt.Printf("[Oracle] [%s] FATAL: crypto/rand failed: %v\n", feedID, err)
			return
		}
		nonce := hex.EncodeToString(nonceBytes)
		hash := oracle.ComputeCommitHash(operator, feedID, roundID, value, nonce)

		// Get block height before commit
		commitHeight, err := client.GetLatestBlockHeight(ctx)
		if err != nil {
			commitHeight = 0 // Fallback
		}

		// 2. Commit Stage
		err = retryWithBackoff(ctx, feedID, "commit", func() error {
			return client.BroadcastCommit(ctx, operator, feedID, roundID, hash)
		})
		if err != nil {
			fmt.Printf("[Oracle] [%s] Failed to commit round %d: %v\n", feedID, roundID, err)
			return
		}

		// 3. Wait for reveal window: Wait for block height to increment (at least +1 block)
		if commitHeight > 0 {
			fmt.Printf("[Oracle] [%s] Waiting for new block after commit at height %d...\n", feedID, commitHeight)
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				currentHeight, err := client.GetLatestBlockHeight(ctx)
				if err == nil && currentHeight > commitHeight {
					fmt.Printf("[Oracle] [%s] Block height progressed to %d. Commencing reveal.\n", feedID, currentHeight)
					break
				}
				time.Sleep(100 * time.Millisecond) // Lightweight polling
			}
		} else {
			// Fallback sleep if height query failed
			time.Sleep(1 * time.Second)
		}

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

		// Wait for next block before starting next round to prevent transaction spamming in the same block
		if commitHeight > 0 {
			revealHeight, err := client.GetLatestBlockHeight(ctx)
			if err == nil {
				fmt.Printf("[Oracle] [%s] Waiting for next block after reveal at height %d...\n", feedID, revealHeight)
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					currentHeight, err := client.GetLatestBlockHeight(ctx)
					if err == nil && currentHeight > revealHeight {
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
		} else {
			time.Sleep(2 * time.Second)
		}
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

	// 1. Load config file
	configPath := os.Getenv("ORACLE_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to read config file at %s: %v", configPath, err))
	}

	var cfgData struct {
		Feeds map[string][]PriceSource `json:"feeds"`
	}
	if err := json.Unmarshal(configFile, &cfgData); err != nil {
		panic(fmt.Sprintf("Failed to parse config file: %v", err))
	}

	// 2. Initialize HSM and Client
	hsm, err := NewHSMKeyManager(os.Getenv("HSM_CONFIG"), []byte("oracle_key"))
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize HSM key manager: %v", err))
	}

	grpcEndpoint := os.Getenv("GRPC_ENDPOINT")
	if grpcEndpoint == "" {
		grpcEndpoint = "localhost:9090"
	}

	chainID := os.Getenv("CHAIN_ID")
	if chainID == "" {
		chainID = "sovereign-1"
	}

	client := NewChainClient(grpcEndpoint, hsm, chainID)

	// 3. Resolve operator address dynamically — FATAL on failure.
	// Running with a placeholder address on mainnet would silently
	// submit oracle reports that validators will reject, wasting gas
	// and masking a critical configuration error.
	operator, err := client.GetOperatorAddress()
	if err != nil {
		log.Fatalf("[Oracle] FATAL: Failed to derive operator address from HSM: %v. "+
			"Cannot run oracle daemon without a valid operator identity. "+
			"Ensure HSM_CONFIG is set and the oracle key is provisioned.", err)
	}
	fmt.Printf("[Oracle] Derived operator address dynamically: %s\n", operator)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize account sequence from chain before starting workers
	accAddr := sdk.AccAddress(sdk.ValAddress(operator).Bytes()).String()
	if err := client.initSequence(ctx, accAddr); err != nil {
		log.Printf("[Oracle] WARNING: Failed to initialize account sequence: %v. Will retry on first tx.\n", err)
	}

	// 4. Query active feeds from the chain (with fallback to config feeds)
	var activeFeeds []string
	chainFeeds, err := client.GetActiveFeeds(ctx)
	if err != nil || len(chainFeeds) == 0 {
		fmt.Printf("[Oracle] [WARNING] Failed to query active feeds from chain: %v. Falling back to all feeds configured in config.json.\n", err)
		for feedID := range cfgData.Feeds {
			activeFeeds = append(activeFeeds, feedID)
		}
	} else {
		fmt.Printf("[Oracle] Successfully queried active feeds from chain: %v\n", chainFeeds)
		// Filter/only run feeds configured in config.json that are active on-chain
		for _, f := range chainFeeds {
			if _, exists := cfgData.Feeds[f]; exists {
				activeFeeds = append(activeFeeds, f)
			}
		}
		// If intersection is empty, fallback to config feeds
		if len(activeFeeds) == 0 {
			fmt.Printf("[Oracle] No active feeds intersected with config.json, running all feeds in config.json.\n")
			for feedID := range cfgData.Feeds {
				activeFeeds = append(activeFeeds, feedID)
			}
		}
	}

	maxRounds := 0
	if maxRoundsStr := os.Getenv("MAX_ROUNDS"); maxRoundsStr != "" {
		if val, err := strconv.Atoi(maxRoundsStr); err == nil {
			maxRounds = val
			fmt.Printf("[Oracle] Running in test mode, limit to %d rounds\n", maxRounds)
		}
	}

	var wg sync.WaitGroup
	for _, feedID := range activeFeeds {
		sources := cfgData.Feeds[feedID]
		wg.Add(1)
		go runFeedWorker(ctx, operator, feedID, sources, client, hsm, maxRounds, &wg)
	}

	wg.Wait()
	fmt.Println("[PASS] Oracle Aggregator executed successfully.")
}
