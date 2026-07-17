package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	advisoryLockHeld = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "sovereign_backend_ingestion_advisory_lock_held",
			Help: "Gauge indicating if the PostgreSQL session advisory lock is held (1 = held, 0 = not held)",
		},
	)
)

func init() {
	prometheus.MustRegister(advisoryLockHeld)
}


const (
	AdvisoryLockID     = 41892305
	PayloadThreshold   = 750 * 1024 // 750 KB
	NatsSubject        = "account:chain"
	ReconcileInterval  = 1 * time.Second
	BackfillInterval   = 5 * time.Second
)

type Config struct {
	WriteDBURL     string
	NatsURL        string
	CometBFTURL    string
	PollIntervalMS int
}

// EventRecord represents the event structure saved in Write DB and published to NATS
type EventRecord struct {
	BlockHeight int64           `json:"block_height"`
	EventIndex  int             `json:"event_index"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

// RefPointer represents the pointer sent to NATS for large events
type RefPointer struct {
	BlockHeight int64  `json:"block_height"`
	EventIndex  int    `json:"event_index"`
	Ref         string `json:"ref"` // Always "db"
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.WriteDBURL, "write-db-url", os.Getenv("WRITE_DB_URL"), "Write DB URL")
	flag.StringVar(&cfg.NatsURL, "nats-url", os.Getenv("NATS_URL"), "NATS URL")
	flag.StringVar(&cfg.CometBFTURL, "cometbft-url", os.Getenv("COMETBFT_RPC_URL"), "CometBFT RPC URL")
	flag.IntVar(&cfg.PollIntervalMS, "poll-interval-ms", 500, "Block polling interval in ms")
	flag.Parse()

	if cfg.WriteDBURL == "" {
		cfg.WriteDBURL = "postgres://ingestion_writer:sovereign_write_pwd@db-write:5432/sovereign_write"
	}
	if cfg.NatsURL == "" {
		cfg.NatsURL = nats.DefaultURL
	}
	if cfg.CometBFTURL == "" {
		cfg.CometBFTURL = "http://chain-node:26657"
	}

	log.Printf("Starting Ingestion service...")
	log.Printf("Write DB URL: %s", cfg.WriteDBURL)
	log.Printf("NATS URL: %s", cfg.NatsURL)
	log.Printf("CometBFT URL: %s", cfg.CometBFTURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Write DB
	db, err := pgxpool.New(ctx, cfg.WriteDBURL)
	if err != nil {
		log.Fatalf("failed to connect to Write DB: %v", err)
	}
	defer db.Close()

	// Start Prometheus metrics server on port 9091
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		log.Println("Serving ingestion metrics on :9091/metrics")
		if err := http.ListenAndServe(":9091", mux); err != nil {
			log.Printf("Prometheus metrics server failed: %v", err)
		}
	}()

	// 1. Acquire PostgreSQL session advisory lock to ensure singleton behavior
	var locked bool
	for attempt := 1; attempt <= 10; attempt++ {
		err = db.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", AdvisoryLockID).Scan(&locked)
		if err != nil {
			log.Printf("[Attempt %d/10] failed to query advisory lock: %v", attempt, err)
		} else if locked {
			log.Printf("Successfully acquired advisory lock %d on attempt %d. Running as singleton.", AdvisoryLockID, attempt)
			advisoryLockHeld.Set(1)
			break
		} else {
			log.Printf("[Attempt %d/10] advisory lock %d is currently held by another instance. Retrying in 1s...", attempt, AdvisoryLockID)
		}

		if attempt < 10 {
			select {
			case <-ctx.Done():
				log.Fatalf("Context cancelled during lock acquisition: %v", ctx.Err())
			case <-time.After(1 * time.Second):
			}
		}
	}

	if !locked {
		advisoryLockHeld.Set(0)
		log.Fatalf("could not acquire advisory lock %d after 10 attempts. Another instance of ingestion is already running.", AdvisoryLockID)
	}

	defer func() {
		// Attempt to unlock on exit
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		_, _ = db.Exec(cleanupCtx, "SELECT pg_advisory_unlock($1)", AdvisoryLockID)
		advisoryLockHeld.Set(0)
		log.Printf("Released advisory lock.")
	}()

	nkeyOpt, err := getNatsNkeyOption("ingestion")
	if err != nil {
		log.Fatalf("failed to configure NATS NKey: %v", err)
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nkeyOpt,
		nats.DisconnectErrHandler(func(c *nats.Conn, err error) {
			log.Printf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Printf("NATS reconnected: %s", c.ConnectedUrl())
		}),
	)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Initialize JetStream
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("failed to initialize NATS JetStream: %v", err)
	}

	// Create/Update the EVENTS stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "EVENTS",
		Subjects:  []string{NatsSubject},
		Retention: nats.LimitsPolicy,
		MaxAge:    365 * 24 * time.Hour,
		Replicas:  3,
	})
	if err != nil {
		log.Printf("Note: AddStream EVENTS returned error (it might already exist): %v", err)
	}

	// Start NATS back-fill worker
	go runBackfillWorker(ctx, db, js)

	// Channel to signal graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block polling loop
	go func() {
		ticker := time.NewTicker(time.Duration(cfg.PollIntervalMS) * time.Millisecond)
		defer ticker.Stop()

		var lastProcessedHeight int64

		// 2. Query MAX(block_height) from DB for startup reconciliation
		err = db.QueryRow(ctx, "SELECT COALESCE(MAX(block_height), 0) FROM events").Scan(&lastProcessedHeight)
		if err != nil {
			log.Printf("Error querying max block height from db: %v", err)
		}
		log.Printf("Reconciliation startup: last processed block height = %d", lastProcessedHeight)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				latestHeight, err := fetchLatestBlockHeight(cfg.CometBFTURL)
				if err != nil {
					log.Printf("Error fetching latest block height: %v", err)
					continue
				}

				if latestHeight > lastProcessedHeight {
					for h := lastProcessedHeight + 1; h <= latestHeight; h++ {
						log.Printf("Ingesting block height %d...", h)
						events, err := fetchBlockEvents(cfg.CometBFTURL, h)
						if err != nil {
							log.Printf("Error fetching block events at height %d: %v", h, err)
							break // Retry this height on next tick
						}

						// Save to DB and publish
						for _, ev := range events {
							err = saveAndPublishEvent(ctx, db, js, ev)
							if err != nil {
								log.Printf("Error saving/publishing event at height %d: %v", h, err)
								// Note: we continue, but the backfill worker will retry publishing if save succeeded.
							}
						}

						lastProcessedHeight = h
					}
				}
			}
		}
	}()

	<-sigChan
	log.Printf("Shutting down Ingestion service...")
}

// runBackfillWorker periodically queries the db for unpublished events and attempts to publish them to JetStream.
func runBackfillWorker(ctx context.Context, db *pgxpool.Pool, js nats.JetStreamContext) {
	ticker := time.NewTicker(BackfillInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rows, err := db.Query(ctx, "SELECT block_height, event_index, event_type, payload FROM events WHERE nats_published = false ORDER BY block_height ASC, event_index ASC LIMIT 100")
			if err != nil {
				log.Printf("Backfill query error: %v", err)
				continue
			}

			var records []EventRecord
			for rows.Next() {
				var rec EventRecord
				if err := rows.Scan(&rec.BlockHeight, &rec.EventIndex, &rec.EventType, &rec.Payload); err != nil {
					log.Printf("Backfill scan error: %v", err)
					continue
				}
				records = append(records, rec)
			}
			rows.Close()

			if len(records) > 0 {
				log.Printf("Backfill worker found %d unpublished events. Backfilling...", len(records))
			}

			for _, rec := range records {
				err = publishToNats(js, rec)
				if err == nil {
					_, err = db.Exec(ctx, "UPDATE events SET nats_published = true WHERE block_height = $1 AND event_index = $2", rec.BlockHeight, rec.EventIndex)
					if err != nil {
						log.Printf("Failed to update nats_published in backfill: %v", err)
					}
				} else {
					log.Printf("Backfill publish failed for block %d event %d: %v", rec.BlockHeight, rec.EventIndex, err)
					break // Stop and try again later if NATS is offline
				}
			}
		}
	}
}

func saveAndPublishEvent(ctx context.Context, db *pgxpool.Pool, js nats.JetStreamContext, ev EventRecord) error {
	// 1. Save event to Write DB first with nats_published = false
	_, err := db.Exec(ctx,
		`INSERT INTO events (block_height, event_index, event_type, payload, nats_published)
		 SELECT $1, $2, $3, $4, false
		 WHERE NOT EXISTS (
		     SELECT 1 FROM events
		     WHERE block_height = $1 AND event_index = $2
		 )`,
		ev.BlockHeight, ev.EventIndex, ev.EventType, ev.Payload,
	)
	if err != nil {
		return fmt.Errorf("failed to save event to Write DB: %w", err)
	}

	// 2. Publish to NATS JetStream
	err = publishToNats(js, ev)
	if err != nil {
		log.Printf("NATS publish failed for height %d index %d: %v (will retry in backfill)", ev.BlockHeight, ev.EventIndex, err)
		return nil // Return nil because it's saved in DB and will be backfilled
	}

	// 3. Mark as published in DB
	_, err = db.Exec(ctx,
		"UPDATE events SET nats_published = true WHERE block_height = $1 AND event_index = $2",
		ev.BlockHeight, ev.EventIndex,
	)
	if err != nil {
		log.Printf("Failed to set nats_published = true for height %d index %d: %v", ev.BlockHeight, ev.EventIndex, err)
	}

	return nil
}

func publishToNats(js nats.JetStreamContext, ev EventRecord) error {
	var payloadBytes []byte
	var err error

	payloadBytes, err = json.Marshal(ev)
	if err != nil {
		return err
	}

	// 750KB threshold check
	if len(payloadBytes) > PayloadThreshold {
		log.Printf("Event size %d exceeds threshold 750KB. Publishing reference pointer.", len(payloadBytes))
		ref := RefPointer{
			BlockHeight: ev.BlockHeight,
			EventIndex:  ev.EventIndex,
			Ref:         "db",
		}
		payloadBytes, err = json.Marshal(ref)
		if err != nil {
			return err
		}
	}

	_, err = js.Publish(NatsSubject, payloadBytes)
	return err
}

func fetchLatestBlockHeight(cometBFTURL string) (int64, error) {
	resp, err := http.Get(fmt.Sprintf("%s/status", cometBFTURL))
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

func fetchBlockEvents(cometBFTURL string, height int64) ([]EventRecord, error) {
	var records []EventRecord
	eventIdx := 0

	// 1. Fetch Block (for validator block proposer & commit signatures)
	resp, err := http.Get(fmt.Sprintf("%s/block?height=%d", cometBFTURL, height))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block info: %w", err)
	}
	defer resp.Body.Close()

	var blockResp struct {
		Result struct {
			Block struct {
				Header struct {
					Height           string `json:"height"`
					ProposerAddress  string `json:"proposer_address"`
				} `json:"header"`
				LastCommit struct {
					Signatures []struct {
						ValidatorAddress string `json:"validator_address"`
						BlockIDFlag      int    `json:"block_id_flag"`
					} `json:"signatures"`
				} `json:"last_commit"`
			} `json:"block"`
		} `json:"result"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &blockResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	// Construct and add validator uptime block event
	type ValidatorStatus struct {
		Address string `json:"address"`
		Signed  bool   `json:"signed"`
	}
	type ValidatorUptimeEventPayload struct {
		Proposer   string            `json:"proposer"`
		Validators []ValidatorStatus `json:"validators"`
	}

	uptimePayload := ValidatorUptimeEventPayload{
		Proposer: blockResp.Result.Block.Header.ProposerAddress,
	}

	for _, sig := range blockResp.Result.Block.LastCommit.Signatures {
		if sig.ValidatorAddress != "" {
			uptimePayload.Validators = append(uptimePayload.Validators, ValidatorStatus{
				Address: sig.ValidatorAddress,
				// Flag 2 is BlockIDFlagCommit (signed), Flag 3 is BlockIDFlagAbsent (missed)
				Signed:  sig.BlockIDFlag == 2,
			})
		}
	}

	payloadJSON, _ := json.Marshal(uptimePayload)
	records = append(records, EventRecord{
		BlockHeight: height,
		EventIndex:  eventIdx,
		EventType:   "validator_uptime",
		Payload:     payloadJSON,
	})
	eventIdx++

	// 2. Fetch Block Results (for actual transaction execution events)
	respResults, err := http.Get(fmt.Sprintf("%s/block_results?height=%d", cometBFTURL, height))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block results: %w", err)
	}
	defer respResults.Body.Close()

	var resultsResp struct {
		Result struct {
			TxsResults []struct {
				Code   int `json:"code"`
				Events []struct {
					Type       string `json:"type"`
					Attributes []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
					} `json:"attributes"`
				} `json:"events"`
			} `json:"txs_results"`
			BeginBlockEvents []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"begin_block_events"`
			EndBlockEvents []struct {
				Type       string `json:"type"`
				Attributes []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"attributes"`
			} `json:"end_block_events"`
		} `json:"result"`
	}

	bodyResults, err := io.ReadAll(respResults.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(bodyResults, &resultsResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal block results: %w", err)
	}

	// Helper to extract events
	parseCometEvents := func(events []struct {
		Type       string `json:"type"`
		Attributes []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"attributes"`
	}) {
		for _, e := range events {
			// Interested in specific events: MsgBridgeIn, MsgBridgeOut, milestone_*, settlement_executed, oracle commits/reveals
			switch e.Type {
			case "MsgBridgeIn", "MsgBridgeOut", "milestone_achieved", "milestone_expired", "milestone_stale_blocked", "settlement_executed", "message":
				attrs := make(map[string]string)
				for _, attr := range e.Attributes {
					k := decodeBase64(attr.Key)
					v := decodeBase64(attr.Value)
					attrs[k] = v
				}

				// If it's a "message" event, we check if it is oracle commit or reveal
				if e.Type == "message" {
					action := attrs["action"]
					if action == "/oracle.MsgCommitOracleHash" || action == "MsgCommitOracleHash" {
						e.Type = "oracle_commit"
					} else if action == "/oracle.MsgRevealOracleReport" || action == "MsgRevealOracleReport" {
						e.Type = "oracle_reveal"
					} else {
						// Not an event we care about
						continue
					}
				}

				payloadJSON, _ := json.Marshal(attrs)
				records = append(records, EventRecord{
					BlockHeight: height,
					EventIndex:  eventIdx,
					EventType:   e.Type,
					Payload:     payloadJSON,
				})
				eventIdx++
			}
		}
	}

	// Parse begin block events
	parseCometEvents(resultsResp.Result.BeginBlockEvents)

	// Parse tx events
	for _, txRes := range resultsResp.Result.TxsResults {
		if txRes.Code == 0 { // Only ingest successful transactions
			parseCometEvents(txRes.Events)
		}
	}

	// Parse end block events
	parseCometEvents(resultsResp.Result.EndBlockEvents)

	return records, nil
}

func decodeBase64(s string) string {
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return s
	}
	return string(dec)
}

func getNatsNkeyOption(role string) (nats.Option, error) {
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")
	
	var seed string

	if vaultAddr != "" && vaultToken != "" {
		url := fmt.Sprintf("%s/v1/secret/data/sovereign/nats", vaultAddr)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Vault-Token", vaultToken)
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode == 200 {
			var result struct {
				Data struct {
					Data map[string]interface{} `json:"data"`
				} `json:"data"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
				if keyVal, ok := result.Data.Data[role+"_nkey"]; ok {
					seed = fmt.Sprintf("%v", keyVal)
				}
			}
		}
	}
	
	if seed == "" {
		envName := strings.ToUpper(role) + "_NKEY_SEED"
		seed = os.Getenv(envName)
	}
	
	if seed == "" {
		return nil, fmt.Errorf("NKey seed not found for role %s", role)
	}

	kp, err := nkeys.FromSeed([]byte(seed))
	if err != nil {
		return nil, fmt.Errorf("failed to parse NKey seed: %w", err)
	}
	pubKey, err := kp.PublicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key from NKey seed: %w", err)
	}
	opt := nats.Nkey(pubKey, func(nonce []byte) ([]byte, error) {
		return kp.Sign(nonce)
	})
	return opt, nil
}
