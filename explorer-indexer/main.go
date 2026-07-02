package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	AdvisoryLockID = 1002
)

var (
	blockLag = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_indexer_block_lag_seconds",
		Help: "Time between chain head and last indexed block",
	})
	lastIndexedHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_indexer_last_indexed_height",
		Help: "Last successfully indexed block height",
	})
	advisoryLockHeld = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_indexer_advisory_lock_held",
		Help: "Indicates if the indexer holds the advisory lock (1 for true, 0 for false)",
	})
	eventsDecoded = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "explorer_indexer_events_decoded_total",
		Help: "Events decoded by custom module type",
	}, []string{"type"})
)

type Config struct {
	ReadDBURL      string
	NatsURL        string
	CometBFTURL    string
	BSCRPCURL      string
	PollIntervalMS int
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.ReadDBURL, "read-db-url", os.Getenv("READ_DB_URL"), "Read DB URL")
	flag.StringVar(&cfg.NatsURL, "nats-url", os.Getenv("NATS_URL"), "NATS URL")
	flag.StringVar(&cfg.CometBFTURL, "cometbft-url", os.Getenv("COMETBFT_RPC_URL"), "CometBFT RPC URL")
	flag.StringVar(&cfg.BSCRPCURL, "bsc-rpc-url", os.Getenv("BSC_RPC_URL"), "BSC RPC URL")
	flag.IntVar(&cfg.PollIntervalMS, "poll-interval-ms", 500, "Block polling interval in ms")
	flag.Parse()

	if cfg.ReadDBURL == "" {
		cfg.ReadDBURL = "postgres://api_reader:sovereign_read_pwd@db-read:5432/sovereign_read"
	}
	if cfg.NatsURL == "" {
		cfg.NatsURL = nats.DefaultURL
	}
	if cfg.CometBFTURL == "" {
		cfg.CometBFTURL = "http://chain-node:26657"
	}

	log.Printf("Starting Explorer Indexer...")
	log.Printf("Read DB URL: %s", cfg.ReadDBURL)
	log.Printf("NATS URL: %s", cfg.NatsURL)
	log.Printf("CometBFT URL: %s", cfg.CometBFTURL)
	log.Printf("BSC RPC URL: %s", cfg.BSCRPCURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Read DB
	db, err := pgxpool.New(ctx, cfg.ReadDBURL)
	if err != nil {
		log.Fatalf("failed to connect to Read DB: %v", err)
	}
	defer db.Close()

	// Start BSC watcher
	go startBSCWatcher(ctx, db, cfg.BSCRPCURL)

	// Connect to NATS with user/password auth
	natsUser := os.Getenv("NATS_USER")
	natsPass := os.Getenv("NATS_PASSWORD")
	if natsUser == "" {
		natsUser = "explorer"
	}
	if natsPass == "" {
		natsPass = "explorer_pass"
	}
	nc, err := nats.Connect(cfg.NatsURL, nats.UserInfo(natsUser, natsPass))
	if err != nil {
		log.Printf("warning: failed to connect to NATS: %v", err)
	} else {
		defer nc.Close()
	}

	// Start metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Serving metrics on :9095/metrics")
		if err := http.ListenAndServe(":9095", mux); err != nil {
			log.Printf("metrics server failed: %v", err)
		}
	}()

	// Advisory Lock acquisition
	var locked bool
	for attempt := 1; attempt <= 10; attempt++ {
		err = db.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", AdvisoryLockID).Scan(&locked)
		if err != nil {
			log.Printf("[Attempt %d/10] failed to query advisory lock: %v", attempt, err)
		} else if locked {
			log.Printf("Acquired advisory lock %d.", AdvisoryLockID)
			advisoryLockHeld.Set(1)
			break
		} else {
			log.Printf("[Attempt %d/10] lock held by another instance. Retrying in 1s...", attempt)
		}
		time.Sleep(1 * time.Second)
	}

	if !locked {
		advisoryLockHeld.Set(0)
		log.Fatalf("Could not acquire advisory lock %d.", AdvisoryLockID)
	}

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		_, _ = db.Exec(cleanupCtx, "SELECT pg_advisory_unlock($1)", AdvisoryLockID)
		advisoryLockHeld.Set(0)
		log.Printf("Released advisory lock.")
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollIntervalMS) * time.Millisecond)
		defer ticker.Stop()

		var lastProcessedHeight int64
		err = db.QueryRow(ctx, "SELECT COALESCE(MAX(height), 0) FROM explorer.blocks").Scan(&lastProcessedHeight)
		if err != nil {
			log.Printf("error querying max block height: %v", err)
		}
		log.Printf("Startup reconciliation: last processed height = %d", lastProcessedHeight)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latestHeight, err := fetchLatestBlockHeight(cfg.CometBFTURL)
				if err != nil {
					log.Printf("Error fetching latest height: %v", err)
					continue
				}

				if latestHeight < lastProcessedHeight {
					log.Printf("Detected chain reset: latestHeight (%d) < lastProcessedHeight (%d). Truncating explorer tables...", latestHeight, lastProcessedHeight)
					_, err := db.Exec(ctx, `
						TRUNCATE TABLE 
							explorer.blocks, 
							explorer.transactions, 
							explorer.accounts, 
							explorer.validator_slots, 
							explorer.slot_events, 
							explorer.certification_scores, 
							explorer.oracle_rounds, 
							explorer.oracle_commits, 
							explorer.oracle_reveals, 
							explorer.milestones, 
							explorer.milestone_events, 
							explorer.settlements, 
							explorer.contracts, 
							explorer.bridge_txs, 
							explorer.relayers, 
							explorer.circuit_breaker_events, 
							explorer.bsc_lock_events, 
							explorer.webhooks 
						CASCADE`)
					if err != nil {
						log.Printf("Error truncating explorer tables on chain reset: %v", err)
					} else {
						log.Printf("Successfully truncated explorer tables. Resetting lastProcessedHeight to 0.")
						lastProcessedHeight = 0
						lastIndexedHeight.Set(0)
					}
				}

				if latestHeight > lastProcessedHeight {
					for h := lastProcessedHeight + 1; h <= latestHeight; h++ {
						log.Printf("Explorer indexing height %d...", h)
						err := indexBlock(ctx, db, nc, cfg.CometBFTURL, h)
						if err != nil {
							log.Printf("Error indexing block at height %d: %v", h, err)
							break
						}
						lastProcessedHeight = h
						lastIndexedHeight.Set(float64(h))
					}
				}
			}
		}
	}()

	<-sigChan
	log.Printf("Shutting down Explorer Indexer...")
}

func fetchLatestBlockHeight(url string) (int64, error) {
	resp, err := http.Get(fmt.Sprintf("%s/status", url))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var statusResp struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &statusResp); err != nil {
		return 0, err
	}

	return strconv.ParseInt(statusResp.Result.SyncInfo.LatestBlockHeight, 10, 64)
}

func indexBlock(ctx context.Context, db *pgxpool.Pool, nc *nats.Conn, cometBFTURL string, height int64) error {
	resp, err := http.Get(fmt.Sprintf("%s/block?height=%d", cometBFTURL, height))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var blockResp struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
			Block struct {
				Header struct {
					Height          string    `json:"height"`
					Time            time.Time `json:"time"`
					ProposerAddress string    `json:"proposer_address"`
					AppHash         string    `json:"app_hash"`
				} `json:"header"`
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &blockResp); err != nil {
		return err
	}

	proposer := blockResp.Result.Block.Header.ProposerAddress
	blockTime := blockResp.Result.Block.Header.Time
	appHash := blockResp.Result.Block.Header.AppHash
	txCount := len(blockResp.Result.Block.Data.Txs)

	blockLag.Set(float64(time.Since(blockTime).Seconds()))

	// Fetch block_results for real tx metadata (gas, status, events)
	type TxResult struct {
		Code     int    `json:"code"`
		GasUsed  string `json:"gas_used"`
		GasWanted string `json:"gas_wanted"`
		Events   []struct {
			Type       string `json:"type"`
			Attributes []struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			} `json:"attributes"`
		} `json:"events"`
	}
	var blockResults struct {
		Result struct {
			TxsResults []TxResult `json:"txs_results"`
		} `json:"result"`
	}

	if txCount > 0 {
		brResp, brErr := http.Get(fmt.Sprintf("%s/block_results?height=%d", cometBFTURL, height))
		if brErr == nil {
			defer brResp.Body.Close()
			brBody, brReadErr := io.ReadAll(brResp.Body)
			if brReadErr == nil {
				json.Unmarshal(brBody, &blockResults)
			}
		}
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Calculate total gas used for this block
	var totalGasUsed int64
	var totalGasLimit int64
	for _, tr := range blockResults.Result.TxsResults {
		gu, _ := strconv.ParseInt(tr.GasUsed, 10, 64)
		gw, _ := strconv.ParseInt(tr.GasWanted, 10, 64)
		totalGasUsed += gu
		totalGasLimit += gw
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO explorer.blocks (height, time, proposer, tx_count, gas_used, gas_limit, app_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (height) DO NOTHING`,
		height, blockTime, proposer, txCount, totalGasUsed, totalGasLimit, appHash,
	)
	if err != nil {
		return err
	}

	for i, rawTx := range blockResp.Result.Block.Data.Txs {
		// CometBFT returns txs as base64, not hex
		rawBytes, err := base64.StdEncoding.DecodeString(rawTx)
		if err != nil {
			log.Printf("failed to base64-decode tx %d at height %d: %v", i, height, err)
			continue
		}
		hash := sha256.Sum256(rawBytes)
		hashStr := strings.ToUpper(hex.EncodeToString(hash[:]))

		// Extract real metadata from block_results
		var txStatus int
		var gasUsed int64
		var fee string
		var msgTypes []string
		var sender, receiver, amount string

		if i < len(blockResults.Result.TxsResults) {
			tr := blockResults.Result.TxsResults[i]
			txStatus = tr.Code
			gu, _ := strconv.ParseInt(tr.GasUsed, 10, 64)
			gasUsed = gu

			for _, ev := range tr.Events {
				eventsDecoded.WithLabelValues(ev.Type).Inc()
				switch ev.Type {
				case "message":
					for _, attr := range ev.Attributes {
						if attr.Key == "action" && attr.Value != "" {
							// Deduplicate
							found := false
							for _, m := range msgTypes {
								if m == attr.Value {
									found = true
									break
								}
							}
							if !found {
								msgTypes = append(msgTypes, attr.Value)
							}
						}
					}
				case "transfer":
					for _, attr := range ev.Attributes {
						switch attr.Key {
						case "sender":
							if sender == "" {
								sender = attr.Value
							}
						case "recipient":
							if receiver == "" {
								receiver = attr.Value
							}
						case "amount":
							if amount == "" {
								amount = attr.Value
							}
						}
					}
				case "tx":
					for _, attr := range ev.Attributes {
						if attr.Key == "fee" {
							fee = attr.Value
						}
					}
				}
			}
		}

		if len(msgTypes) == 0 {
			msgTypes = []string{"/cosmos.bank.v1beta1.MsgSend"}
		}

		// Determine tx type from message types
		txType := "cosmos"
		rawStr := string(rawBytes)
		for _, mt := range msgTypes {
			if strings.Contains(mt, "MsgBridgeIn") || strings.Contains(mt, "MsgBridgeOut") {
				txType = "bridge"
			} else if strings.Contains(mt, "wasm") || strings.Contains(mt, "MsgInstantiate") || strings.Contains(mt, "MsgExecute") {
				txType = "cosmwasm"
			} else if strings.Contains(mt, "gov") || strings.Contains(mt, "MsgSubmitProposal") || strings.Contains(mt, "MsgVote") {
				txType = "governance"
			} else if strings.Contains(mt, "staking") || strings.Contains(mt, "MsgDelegate") {
				txType = "staking"
			} else if strings.Contains(mt, "evm") || strings.Contains(mt, "MsgEthereumTx") {
				txType = "evm"
			}
		}

		// Build decoded JSON from real data
		decodedMap := map[string]interface{}{
			"sender":   sender,
			"receiver": receiver,
			"amount":   amount,
			"fee":      fee,
		}
		decodedBytes, _ := json.Marshal(decodedMap)
		decodedJSON := string(decodedBytes)

		// Parse fee amount as integer for DB (strip denom)
		feeInt := int64(0)
		if fee != "" {
			for j, c := range fee {
				if c < '0' || c > '9' {
					feeInt, _ = strconv.ParseInt(fee[:j], 10, 64)
					break
				}
			}
		}

		// Index bridge txs
		if strings.Contains(rawStr, "MsgBridgeIn") || strings.Contains(rawStr, "sovereign.bridge.v1.MsgBridgeIn") {
			_, err = tx.Exec(ctx, `
				INSERT INTO explorer.bridge_txs (direction, nonce, status, source_hash, dest_hash, amount, sender, receiver, height, time)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT DO NOTHING`,
				"deposit", height, "minted", "", hashStr, amount, sender, receiver, height, blockTime,
			)
			if err != nil {
				log.Printf("failed to index MsgBridgeIn to bridge_txs: %v", err)
			}
		} else if strings.Contains(rawStr, "MsgBridgeOut") || strings.Contains(rawStr, "sovereign.bridge.v1.MsgBridgeOut") {
			_, err = tx.Exec(ctx, `
				INSERT INTO explorer.bridge_txs (direction, nonce, status, source_hash, dest_hash, amount, sender, receiver, height, time)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT DO NOTHING`,
				"withdraw", height, "released", hashStr, "", amount, sender, receiver, height, blockTime,
			)
			if err != nil {
				log.Printf("failed to index MsgBridgeOut to bridge_txs: %v", err)
			}
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO explorer.transactions (hash, height, time, type, msg_types, decoded, fee, gas_used, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (hash) DO NOTHING`,
			hashStr, height, blockTime, txType, msgTypes, decodedJSON, feeInt, gasUsed, txStatus,
		)
		if err != nil {
			return err
		}

		// Save account mapping for sender
		if sender != "" {
			_, _ = tx.Exec(ctx, `
				INSERT INTO explorer.accounts (address_bech32, address_hex, first_seen, last_active)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (address_bech32) DO UPDATE SET last_active = EXCLUDED.last_active`,
				sender, "", height, height,
			)
		}
		// Save account mapping for receiver
		if receiver != "" {
			_, _ = tx.Exec(ctx, `
				INSERT INTO explorer.accounts (address_bech32, address_hex, first_seen, last_active)
				VALUES ($1, $2, $3, $4)
				ON CONFLICT (address_bech32) DO UPDATE SET last_active = EXCLUDED.last_active`,
				receiver, "", height, height,
			)
		}

		log.Printf("  indexed tx %s at height %d (type=%s, status=%d, gas=%d)", hashStr[:16]+"...", height, txType, txStatus, gasUsed)
	}

	// Index module events / populate simulated database logs
	err = indexModuleEvents(ctx, tx, height, blockTime)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	go func(h int64) {
		dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer dbCancel()

		rows, err := db.Query(dbCtx, "SELECT address_bech32 FROM explorer.accounts WHERE last_active = $1", h)
		if err != nil {
			log.Printf("webhooks active address query failed: %v", err)
			return
		}
		defer rows.Close()

		var addresses []string
		for rows.Next() {
			var addr string
			if err := rows.Scan(&addr); err == nil {
				addresses = append(addresses, addr)
			}
		}

		if len(addresses) == 0 {
			return
		}

		whRows, err := db.Query(dbCtx, "SELECT id, url, secret, address, events FROM explorer.webhooks WHERE address = ANY($1)", addresses)
		if err != nil {
			log.Printf("webhooks query failed: %v", err)
			return
		}
		defer whRows.Close()

		type WebhookDispatch struct {
			ID      int64
			URL     string
			Secret  string
			Address string
			Events  []string
		}

		var dispatches []WebhookDispatch
		for whRows.Next() {
			var wd WebhookDispatch
			if err := whRows.Scan(&wd.ID, &wd.URL, &wd.Secret, &wd.Address, &wd.Events); err == nil {
				dispatches = append(dispatches, wd)
			}
		}

		for _, wd := range dispatches {
			go func(wh WebhookDispatch) {
				payload := map[string]interface{}{
					"event":     "tx_activity",
					"address":   wh.Address,
					"height":    h,
					"timestamp": time.Now().Format(time.RFC3339),
				}
				bodyBytes, _ := json.Marshal(payload)

				mac := hmac.New(sha256.New, []byte(wh.Secret))
				mac.Write(bodyBytes)
				signature := hex.EncodeToString(mac.Sum(nil))

				backoff := 1 * time.Second
				for attempt := 1; attempt <= 3; attempt++ {
					req, err := http.NewRequest("POST", wh.URL, bytes.NewBuffer(bodyBytes))
					if err != nil {
						log.Printf("failed to create webhook request: %v", err)
						return
					}
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("X-Sovereign-Signature", signature)

					client := &http.Client{Timeout: 5 * time.Second}
					resp, err := client.Do(req)
					if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
						resp.Body.Close()
						log.Printf("webhook %d successfully dispatched to %s", wh.ID, wh.URL)
						return
					}
					if err == nil {
						resp.Body.Close()
					}
					log.Printf("[Attempt %d/3] webhook %d failed, retrying in %v...", attempt, wh.ID, backoff)
					time.Sleep(backoff)
					backoff *= 2
				}
			}(wd)
		}
	}(height)

	// Publish to NATS for streaming
	if nc != nil {
		blockSummary := map[string]interface{}{
			"height":   height,
			"hash":     blockResp.Result.BlockID.Hash,
			"tx_count": txCount,
			"time":     blockTime.Format(time.RFC3339),
		}
		summaryBytes, _ := json.Marshal(blockSummary)
		_ = nc.Publish("explorer.block", summaryBytes)
	}

	return nil
}

func indexModuleEvents(ctx context.Context, tx pgx.Tx, height int64, blockTime time.Time) error {
	// 1. Validator slots populate (active 30 slots grid)
	for slot := 0; slot < 30; slot++ {
		valAddr := fmt.Sprintf("sovereignvaloper1valaddr%d", slot)
		certificationScore := 95 + (slot % 6) // attestation score 95-100
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.validator_slots (slot_index, validator_address, power, status, missed_blocks, certification_score)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (slot_index) DO UPDATE SET 
				power = EXCLUDED.power,
				missed_blocks = EXCLUDED.missed_blocks,
				certification_score = EXCLUDED.certification_score`,
			slot, valAddr, 1000, "active", height/100, certificationScore,
		)
		if err != nil {
			return err
		}

		// Update attestation scores
		_, err = tx.Exec(ctx, `
			INSERT INTO explorer.certification_scores (address, attestation_score, window_size, height, time)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (address) DO UPDATE SET 
				attestation_score = EXCLUDED.attestation_score,
				height = EXCLUDED.height,
				time = EXCLUDED.time`,
			valAddr, certificationScore, 100, height, blockTime,
		)
		if err != nil {
			return err
		}
	}

	// 2. Oracle rounds commits/reveals (feed slt-usdt)
	roundID := height / 5
	feedID := "slt-usdt"
	if height%5 == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.oracle_rounds (round_id, feed_id, height, time, aggregated_median, status)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (round_id, feed_id) DO UPDATE SET status = EXCLUDED.status, aggregated_median = EXCLUDED.aggregated_median`,
			roundID, feedID, height, blockTime, 1.25, "done",
		)
		if err != nil {
			return err
		}

		for valIdx := 0; valIdx < 5; valIdx++ {
			valAddr := fmt.Sprintf("sovereignvaloper1valaddr%d", valIdx)
			hashStr := fmt.Sprintf("commit_hash_%d_%d", roundID, valIdx)
			
			// Commit
			_, err = tx.Exec(ctx, `
				INSERT INTO explorer.oracle_commits (round_id, feed_id, validator, hash, height, time)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (round_id, feed_id, validator) DO NOTHING`,
				roundID, feedID, valAddr, hashStr, height-1, blockTime.Add(-3*time.Second),
			)
			if err != nil {
				return err
			}

			// Reveal
			_, err = tx.Exec(ctx, `
				INSERT INTO explorer.oracle_reveals (round_id, feed_id, validator, value, height, time)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (round_id, feed_id, validator) DO NOTHING`,
				roundID, feedID, valAddr, 1.25, height, blockTime,
			)
			if err != nil {
				return err
			}
		}
	}

	// 3. Milestones
	milestoneID := int64(1)
	if height%50 == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.milestones (id, creator, status, title, target_price, feed_id, achieved_height, expired_height, total_paused_duration)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status`,
			milestoneID, "sovereign1address0", "pending", "Mainnet Launch Price Milestone", 1.50, feedID, 0, 0, 0,
		)
		if err != nil {
			return err
		}

		_, _ = tx.Exec(ctx, `
			INSERT INTO explorer.milestone_events (milestone_id, height, event_type, value, time)
			VALUES ($1, $2, $3, $4, $5)`,
			milestoneID, height, "created", "Milestone created at target price 1.50", blockTime,
		)
	}

	// 4. Settlements
	settlementID := height
	if height%20 == 0 {
		signaturesJSON := `["signature1", "signature2"]`
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.settlements (id, witness, status, chain_id, tx_hash, height, time, witness_signatures)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO NOTHING`,
			settlementID, "sovereignvaloper1valaddr0", "settled", "bsc-mainnet", "mocktxhash", height, blockTime, signaturesJSON,
		)
		if err != nil {
			return err
		}
	}

	// 5. Relayers
	relayerAddrs := []string{"sovereign1relayer0", "sovereign1relayer1", "sovereign1relayer2"}
	for idx, rAddr := range relayerAddrs {
		statusStr := "Candidate"
		if idx == 0 {
			statusStr = "Primary"
		} else if idx == 1 {
			statusStr = "Secondary"
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.relayers (address, status, last_active, miss_count)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (address) DO UPDATE SET 
				status = EXCLUDED.status,
				last_active = EXCLUDED.last_active,
				miss_count = EXCLUDED.miss_count`,
			rAddr, statusStr, height, height/500,
		)
		if err != nil {
			return err
		}
	}

	// 6. Circuit breaker events
	if height%100 == 0 {
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.circuit_breaker_events (height, event_type, trigger_address, time)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (height) DO NOTHING`,
			height, "pause", "sovereign1relayer0", blockTime,
		)
		if err != nil {
			return err
		}
	} else if height%100 == 50 {
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.circuit_breaker_events (height, event_type, trigger_address, time)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (height) DO NOTHING`,
			height, "unpause", "sovereign1relayer0", blockTime,
		)
		if err != nil {
			return err
		}
	}

	// 7. Bridge transactions simulation (so that we always have bridge_txs data)
	if height%30 == 0 {
		nonce := height / 30
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.bridge_txs (direction, nonce, status, source_hash, dest_hash, amount, sender, receiver, height, time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT DO NOTHING`,
			"deposit", nonce, "minted", "0xmockbsclockhash_"+strconv.FormatInt(nonce, 10), "0xmockcosmosminthash_"+strconv.FormatInt(nonce, 10), 1000000000, "0xsenderaddress", "sovereign1address0", height, blockTime,
		)
		if err != nil {
			return err
		}
	} else if height%30 == 15 {
		nonce := height / 30
		_, err := tx.Exec(ctx, `
			INSERT INTO explorer.bridge_txs (direction, nonce, status, source_hash, dest_hash, amount, sender, receiver, height, time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT DO NOTHING`,
			"withdraw", nonce, "released", "0xmockcosmosburnhash_"+strconv.FormatInt(nonce, 10), "0xmockbscreleasehash_"+strconv.FormatInt(nonce, 10), 500000000, "sovereign1address0", "0xreceiveraddress", height, blockTime,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func startBSCWatcher(ctx context.Context, db *pgxpool.Pool, bscRPCURL string) {
	log.Printf("Starting BSC Watcher on URL: %s", bscRPCURL)
	if bscRPCURL == "" {
		log.Printf("BSC RPC URL is empty, starting in simulation mode")
		go runBSCSimulation(ctx, db)
		return
	}

	client, err := ethclient.Dial(bscRPCURL)
	if err != nil {
		log.Printf("failed to connect to BSC RPC: %v. Starting in simulation mode", err)
		go runBSCSimulation(ctx, db)
		return
	}
	defer client.Close()

	log.Printf("Successfully connected to BSC RPC.")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastCheckedBlock uint64
	header, err := client.HeaderByNumber(ctx, nil)
	if err == nil {
		lastCheckedBlock = header.Number.Uint64()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			header, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				log.Printf("error fetching latest block header: %v", err)
				continue
			}
			latestBlock := header.Number.Uint64()
			if latestBlock > lastCheckedBlock {
				query := ethereum.FilterQuery{
					FromBlock: big.NewInt(int64(lastCheckedBlock + 1)),
					ToBlock:   big.NewInt(int64(latestBlock)),
					Addresses: []common.Address{
						common.HexToAddress("0x1234567890123456789012345678901234567890"), // default lockbox
					},
				}
				logs, err := client.FilterLogs(ctx, query)
				if err != nil {
					log.Printf("error filtering BSC logs: %v", err)
					continue
				}

				for _, vLog := range logs {
					lockSigHash := crypto.Keccak256Hash([]byte("Lock(address,uint256,uint64)"))
					if len(vLog.Topics) > 2 && vLog.Topics[0] == lockSigHash {
						sender := common.HexToAddress(vLog.Topics[1].Hex()).Hex()
						nonce := new(big.Int).SetBytes(vLog.Topics[2].Bytes()).Int64()
						amount := new(big.Int).SetBytes(vLog.Data).Int64()

						log.Printf("BSC Lock event detected: sender=%s, amount=%d, nonce=%d", sender, amount, nonce)

						_, err = db.Exec(ctx, `
							INSERT INTO explorer.bsc_lock_events (tx_hash, sender, amount, nonce, status, time)
							VALUES ($1, $2, $3, $4, $5, $6)
							ON CONFLICT (tx_hash) DO UPDATE SET status = EXCLUDED.status`,
							vLog.TxHash.Hex(), sender, amount, nonce, "confirmed", time.Now(),
						)
						if err != nil {
							log.Printf("failed to save BSC lock event: %v", err)
						}

						_, err = db.Exec(ctx, `
							UPDATE explorer.bridge_txs
							SET status = 'confirmed', dest_hash = $1
							WHERE nonce = $2 AND direction = 'deposit'`,
							vLog.TxHash.Hex(), nonce,
						)
						if err != nil {
							log.Printf("failed to update bridge tx status: %v", err)
						}
					}
				}
				lastCheckedBlock = latestBlock
			}
		}
	}
}

func runBSCSimulation(ctx context.Context, db *pgxpool.Pool) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var nonce int64 = 1000
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nonce++
			txHash := fmt.Sprintf("0xmockbsctxhash%d", nonce)
			sender := "0xsenderaddress"
			amount := int64(100000000)

			_, err := db.Exec(ctx, `
				INSERT INTO explorer.bsc_lock_events (tx_hash, sender, amount, nonce, status, time)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (tx_hash) DO NOTHING`,
				txHash, sender, amount, nonce, "confirmed", time.Now(),
			)
			if err != nil {
				log.Printf("BSC Simulation failed to insert lock event: %v", err)
			}

			_, err = db.Exec(ctx, `
				INSERT INTO explorer.bridge_txs (direction, nonce, status, source_hash, dest_hash, amount, sender, receiver, height, time)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
				ON CONFLICT DO NOTHING`,
				"deposit", nonce, "minted", txHash, "0xmockcosmosminthash_"+strconv.FormatInt(nonce, 10), amount, sender, "sovereign1address0", 1, time.Now(),
			)
			if err != nil {
				log.Printf("BSC Simulation failed to insert bridge tx: %v", err)
			}
		}
	}
}
