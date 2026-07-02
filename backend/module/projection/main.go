package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
)

const (
	NatsChainSubject   = "account:chain"
	NatsStreamSubject  = "account:stream"
)

type Config struct {
	WriteDBURL string
	ReadDBURL  string
	NatsURL    string
}

type EventRecord struct {
	BlockHeight int64           `json:"block_height"`
	EventIndex  int             `json:"event_index"`
	EventType   string          `json:"event_type"`
	Payload     json.RawMessage `json:"payload"`
}

type RefPointer struct {
	BlockHeight int64  `json:"block_height"`
	EventIndex  int    `json:"event_index"`
	Ref         string `json:"ref"`
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.WriteDBURL, "write-db-url", os.Getenv("WRITE_DB_URL"), "Write DB URL")
	flag.StringVar(&cfg.ReadDBURL, "read-db-url", os.Getenv("READ_DB_URL"), "Read DB URL")
	flag.StringVar(&cfg.NatsURL, "nats-url", os.Getenv("NATS_URL"), "NATS URL")
	flag.Parse()

	if cfg.WriteDBURL == "" {
		cfg.WriteDBURL = "postgres://ingestion_writer:sovereign_write_pwd@db-write:5432/sovereign_write"
	}
	if cfg.ReadDBURL == "" {
		cfg.ReadDBURL = "postgres://api_reader:sovereign_read_pwd@db-read:5432/sovereign_read"
	}
	if cfg.NatsURL == "" {
		cfg.NatsURL = nats.DefaultURL
	}

	log.Printf("Starting Projection service...")
	log.Printf("Write DB URL: %s", cfg.WriteDBURL)
	log.Printf("Read DB URL: %s", cfg.ReadDBURL)
	log.Printf("NATS URL: %s", cfg.NatsURL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to Write DB (read-only for pointer resolution)
	writeDB, err := pgxpool.New(ctx, cfg.WriteDBURL)
	if err != nil {
		log.Fatalf("failed to connect to Write DB: %v", err)
	}
	defer writeDB.Close()

	// Connect to Read DB
	readDB, err := pgxpool.New(ctx, cfg.ReadDBURL)
	if err != nil {
		log.Fatalf("failed to connect to Read DB: %v", err)
	}
	defer readDB.Close()

	nkeyOpt, err := getNatsNkeyOption("projection")
	if err != nil {
		log.Fatalf("failed to configure NATS NKey: %v", err)
	}

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nkeyOpt,
	)
	if err != nil {
		log.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	var js nats.JetStreamContext
	for {
		js, err = nc.JetStream()
		if err == nil {
			break
		}
		log.Printf("Waiting for JetStream context: %v. Retrying in 2 seconds...", err)
		time.Sleep(2 * time.Second)
	}

	// Create output STREAM stream if not exists
	for {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:      "STREAM",
			Subjects:  []string{NatsStreamSubject},
			Retention: nats.LimitsPolicy,
			MaxAge:    365 * 24 * time.Hour,
			Replicas:  3,
		})
		if err == nil || strings.Contains(err.Error(), "already exists") {
			break
		}
		log.Printf("Waiting to AddStream STREAM: %v. Retrying in 2 seconds...", err)
		time.Sleep(2 * time.Second)
	}

	// Subscribe to account:chain
	var sub *nats.Subscription
	for {
		sub, err = js.Subscribe(NatsChainSubject, func(m *nats.Msg) {
			var pointer RefPointer
			var record EventRecord

			// Parse the incoming message
			err := json.Unmarshal(m.Data, &pointer)
			if err == nil && pointer.Ref == "db" {
				// Resolve pointer from Write DB
				log.Printf("Resolving large event reference pointer for block %d, index %d", pointer.BlockHeight, pointer.EventIndex)
				err = writeDB.QueryRow(ctx, "SELECT block_height, event_index, event_type, payload FROM events WHERE block_height = $1 AND event_index = $2",
					pointer.BlockHeight, pointer.EventIndex).Scan(&record.BlockHeight, &record.EventIndex, &record.EventType, &record.Payload)
				if err != nil {
					log.Printf("Error resolving pointer from Write DB: %v", err)
					m.Nak()
					return
				}
			} else {
				// Normal record
				if err := json.Unmarshal(m.Data, &record); err != nil {
					log.Printf("Error unmarshaling normal EventRecord: %v", err)
					m.Term() // Bad message format, terminate it
					return
				}
			}

			// Project to Read DB
			err = projectEvent(ctx, readDB, record)
			if err != nil {
				log.Printf("Error projecting event %s at height %d: %v", record.EventType, record.BlockHeight, err)
				m.Nak() // Retry processing
				return
			}

			// Publish transformed/enriched event to account:stream
			enrichedJSON, _ := json.Marshal(record)
			_, err = js.Publish(NatsStreamSubject, enrichedJSON)
			if err != nil {
				log.Printf("Error publishing enriched event to stream: %v", err)
			}

			m.Ack()
		}, nats.ManualAck())

		if err == nil {
			break
		}
		log.Printf("Waiting to subscribe to JetStream %s: %v. Retrying in 2 seconds...", NatsChainSubject, err)
		time.Sleep(2 * time.Second)
	}
	defer sub.Unsubscribe()

	log.Printf("Projection service is running. Subscribed to %s.", NatsChainSubject)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutting down Projection service...")
}

func projectEvent(ctx context.Context, readDB *pgxpool.Pool, ev EventRecord) error {
	log.Printf("Projecting event: Type=%s, Block=%d, Index=%d", ev.EventType, ev.BlockHeight, ev.EventIndex)

	// Map event types to projections
	switch ev.EventType {
	case "MsgBridgeIn":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		receiver := attrs["receiver"]
		amtStr := attrs["amount"]
		nonceStr := attrs["nonce"]

		nonce, _ := strconv.ParseInt(nonceStr, 16, 64)
		amount := parseAmount(amtStr)

		// 1. Update bridge_pending
		_, err := readDB.Exec(ctx,
			"INSERT INTO bridge_pending (nonce, token_address, amount, recipient, status) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (nonce) DO UPDATE SET status = EXCLUDED.status",
			nonce, "usov", amount, receiver, "executed",
		)
		if err != nil {
			return fmt.Errorf("failed to update bridge_pending: %w", err)
		}

		// 2. Update bridge_volume
		if err := updateBridgeVolume(ctx, readDB, "usov", "sovereign", amount, 0); err != nil {
			return fmt.Errorf("failed to update bridge_volume: %w", err)
		}

		// 3. Write to bridge_events hypertable
		_, err = readDB.Exec(ctx,
			`INSERT INTO bridge_events (block_height, event_index, direction, asset, amount)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (block_height, event_index) DO NOTHING`,
			ev.BlockHeight, ev.EventIndex, "lock", "usov", amount,
		)
		if err != nil {
			return fmt.Errorf("failed to insert bridge_events: %w", err)
		}

	case "MsgBridgeOut":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		bscRecipient := attrs["bsc_recipient"]
		amtStr := attrs["amount"]

		amount := parseAmount(amtStr)

		// Surrogated pending nonce (since block_height is unique per block, combined with event_index)
		nonce := ev.BlockHeight*1000 + int64(ev.EventIndex)

		// 1. Update bridge_pending
		_, err := readDB.Exec(ctx,
			"INSERT INTO bridge_pending (nonce, token_address, amount, recipient, status) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (nonce) DO UPDATE SET status = EXCLUDED.status",
			nonce, "usov", amount, bscRecipient, "pending",
		)
		if err != nil {
			return fmt.Errorf("failed to update bridge_pending: %w", err)
		}

		// 2. Update bridge_volume
		if err := updateBridgeVolume(ctx, readDB, "usov", "bsc", 0, amount); err != nil {
			return fmt.Errorf("failed to update bridge_volume: %w", err)
		}

		// 3. Write to bridge_events hypertable
		_, err = readDB.Exec(ctx,
			`INSERT INTO bridge_events (block_height, event_index, direction, asset, amount)
			 VALUES ($1, $2, $3, $4, $5)
			 ON CONFLICT (block_height, event_index) DO NOTHING`,
			ev.BlockHeight, ev.EventIndex, "release", "usov", amount,
		)
		if err != nil {
			return fmt.Errorf("failed to insert bridge_events: %w", err)
		}

	case "milestone_achieved", "milestone_expired", "milestone_stale_blocked":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		milestoneID := attrs["milestone_id"]
		status := "achieved"
		if ev.EventType == "milestone_expired" {
			status = "expired"
		} else if ev.EventType == "milestone_stale_blocked" {
			status = "stale"
		}

		_, err := readDB.Exec(ctx,
			"INSERT INTO milestone_status (milestone_id, status, block_height) VALUES ($1, $2, $3) ON CONFLICT (milestone_id) DO UPDATE SET status = EXCLUDED.status, block_height = EXCLUDED.block_height",
			milestoneID, status, ev.BlockHeight,
		)
		return err

	case "settlement_executed":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		witnessID := attrs["witness_id"]
		destination := attrs["destination"]

		_, err := readDB.Exec(ctx,
			"INSERT INTO settlements (settlement_id, proof, status, block_height, signatures) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (settlement_id) DO UPDATE SET status = EXCLUDED.status",
			witnessID, []byte(destination), "executed", ev.BlockHeight, []string{},
		)
		return err

	case "validator_uptime":
		type ValidatorStatus struct {
			Address string `json:"address"`
			Signed  bool   `json:"signed"`
		}
		type ValidatorUptimePayload struct {
			Proposer   string            `json:"proposer"`
			Validators []ValidatorStatus `json:"validators"`
		}

		var payload ValidatorUptimePayload
		if err := json.Unmarshal(ev.Payload, &payload); err != nil {
			return err
		}

		// Write to block_stats hypertable
		_, err := readDB.Exec(ctx,
			`INSERT INTO block_stats (block_height, block_time_ms, tx_count, avg_fee_uatom)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (block_height) DO NOTHING`,
			ev.BlockHeight, 6000, len(payload.Validators), 150,
		)
		if err != nil {
			return err
		}

		for _, val := range payload.Validators {
			missed := 0
			if !val.Signed {
				missed = 1
			}

			_, err = readDB.Exec(ctx,
				`INSERT INTO validator_uptime (validator_address, total_blocks, missed_blocks, uptime_percentage)
				 VALUES ($1, 1, $2, $3)
				 ON CONFLICT (validator_address)
				 DO UPDATE SET
				 	total_blocks = validator_uptime.total_blocks + 1,
				 	missed_blocks = validator_uptime.missed_blocks + EXCLUDED.missed_blocks,
				 	uptime_percentage = (CAST(validator_uptime.total_blocks + 1 - (validator_uptime.missed_blocks + EXCLUDED.missed_blocks) AS DOUBLE PRECISION) / (validator_uptime.total_blocks + 1)) * 100.0`,
				val.Address, missed, float64(100-missed*100),
			)
			if err != nil {
				return err
			}

			// Write to validator_signatures hypertable
			_, err = readDB.Exec(ctx,
				`INSERT INTO validator_signatures (block_height, validator_address, signed)
				 VALUES ($1, $2, $3)
				 ON CONFLICT (block_height, validator_address) DO NOTHING`,
				ev.BlockHeight, val.Address, val.Signed,
			)
			if err != nil {
				return err
			}
		}

	case "oracle_commit":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		operator := attrs["operator"]

		_, err := readDB.Exec(ctx,
			`INSERT INTO oracle_participation (oracle_address, total_requests, successful_reveals, participation_rate)
			 VALUES ($1, 1, 0, 0.0)
			 ON CONFLICT (oracle_address)
			 DO UPDATE SET
			 	total_requests = oracle_participation.total_requests + 1,
			 	participation_rate = (CAST(oracle_participation.successful_reveals AS DOUBLE PRECISION) / (oracle_participation.total_requests + 1)) * 100.0`,
			operator,
		)
		return err

	case "oracle_reveal":
		var attrs map[string]string
		if err := json.Unmarshal(ev.Payload, &attrs); err != nil {
			return err
		}
		operator := attrs["operator"]

		_, err := readDB.Exec(ctx,
			`INSERT INTO oracle_participation (oracle_address, total_requests, successful_reveals, participation_rate)
			 VALUES ($1, 0, 1, 100.0)
			 ON CONFLICT (oracle_address)
			 DO UPDATE SET
			 	successful_reveals = oracle_participation.successful_reveals + 1,
			 	participation_rate = (CAST(oracle_participation.successful_reveals + 1 AS DOUBLE PRECISION) / COALESCE(NULLIF(oracle_participation.total_requests, 0), 1)) * 100.0`,
			operator,
		)
		if err != nil {
			return err
		}

		assetID := attrs["feed_id"]
		if assetID == "" {
			assetID = "BTC-USD"
		}
		priceVal := parseAmount(attrs["value"])
		if priceVal == 0 {
			priceVal = 1000.0
		}

		// Write to oracle_submissions hypertable
		_, err = readDB.Exec(ctx,
			`INSERT INTO oracle_submissions (block_height, asset_id, price, validator)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (block_height, asset_id, validator) DO NOTHING`,
			ev.BlockHeight, assetID, priceVal, operator,
		)
		return err
	}

	return nil
}

func updateBridgeVolume(ctx context.Context, readDB *pgxpool.Pool, token string, chain string, minted float64, burned float64) error {
	now := time.Now().UTC()
	hourBucket := now.Truncate(time.Hour)
	dayBucket := now.Truncate(24 * time.Hour)
	epochBucket := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

	buckets := []struct {
		Timeframe string
		Time      time.Time
	}{
		{"hourly", hourBucket},
		{"daily", dayBucket},
		{"all", epochBucket},
	}

	for _, b := range buckets {
		_, err := readDB.Exec(ctx,
			`INSERT INTO bridge_volume (token_address, chain_id, timeframe, bucket_time, total_minted, total_burned, transaction_count)
			 VALUES ($1, $2, $3, $4, $5, $6, 1)
			 ON CONFLICT (token_address, chain_id, timeframe, bucket_time)
			 DO UPDATE SET
			 	total_minted = bridge_volume.total_minted + EXCLUDED.total_minted,
			 	total_burned = bridge_volume.total_burned + EXCLUDED.total_burned,
			 	transaction_count = bridge_volume.transaction_count + 1`,
			token, chain, b.Timeframe, b.Time, minted, burned,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// parseAmount extracts all digits from the string and parses them as float64
func parseAmount(amtStr string) float64 {
	re := regexp.MustCompile(`[0-9]+`)
	match := re.FindString(amtStr)
	if match == "" {
		return 0
	}
	val, _ := strconv.ParseFloat(match, 64)
	return val
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
		if role == "ingestion" {
			seed = "SUAFFNTD6H6ST7VGTZDXYQDC5BPNGYRTEFY4TZM32TJEMBTFN5TJO4WNXU"
		} else if role == "projection" {
			seed = "SUAINVHHXAR4PZTQC4VEME4P3HB2CQ3QNQY4WK3YNULE2IJZLNOLNDGBUE"
		} else if role == "stream" {
			seed = "SUAO6IIZLMQHQYVKKHJIEXIC5T6XNKM2PUVF4EGZW23UALD7WTFFE7R2LQ"
		}
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
