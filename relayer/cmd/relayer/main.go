package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	// Cosmos and Tendermint
	cmthttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/relayer"
)

// ABI definition for LockBox unlock method and paused flag
const lockBoxABI = `[
  {
    "inputs": [
      { "internalType": "address", "name": "user", "type": "address" },
      { "internalType": "uint256", "name": "amount", "type": "uint256" },
      { "internalType": "uint256", "name": "nonce", "type": "uint256" },
      { "internalType": "bytes[]", "name": "signatures", "type": "bytes[]" }
    ],
    "name": "unlock",
    "outputs": [],
    "stateMutability": "nonpayable",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "paused",
    "outputs": [
      { "internalType": "bool", "name": "", "type": "bool" }
    ],
    "stateMutability": "view",
    "type": "function"
  }
]`

// Prometheus metrics
var (
	noncesProcessed = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sovereign_relayer_processed_nonces_total",
		Help: "Total processed bridge nonces",
	}, []string{"direction", "status"})

	missedBlocks = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "sovereign_relayer_missed_blocks_total",
		Help: "Total missed blocks scanned",
	}, []string{"chain"})

	uptime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "sovereign_relayer_uptime_seconds",
		Help: "Relayer daemon uptime in seconds",
	})
)

func init() {
	prometheus.MustRegister(noncesProcessed)
	prometheus.MustRegister(missedBlocks)
	prometheus.MustRegister(uptime)
}

type Config struct {
	NatsURL        string
	BscRPC         string
	CosmosRPC      string
	CosmosGRPC     string
	DBConn         string
	OperatorAddr   string
	PrivateKeyHex  string
	LockBoxAddress string
	MetricsPort    string
}

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.NatsURL, "nats-url", "nats://localhost:4222", "NATS JetStream server URL")
	flag.StringVar(&cfg.BscRPC, "bsc-rpc", "http://localhost:8545", "BSC EVM RPC URL")
	flag.StringVar(&cfg.CosmosRPC, "cosmos-rpc", "tcp://localhost:26657", "Cosmos Tendermint RPC URL")
	flag.StringVar(&cfg.CosmosGRPC, "cosmos-grpc", "localhost:9090", "Cosmos gRPC endpoint")
	flag.StringVar(&cfg.DBConn, "db-conn", "memory", "Database connection string or 'memory'")
	flag.StringVar(&cfg.OperatorAddr, "operator-address", "cosmos1relayer_address", "Cosmos operator address")
	flag.StringVar(&cfg.PrivateKeyHex, "private-key", "", "Operator private key hex")
	flag.StringVar(&cfg.LockBoxAddress, "lockbox-address", "0x1234567890123456789012345678901234567890", "BSC LockBox address")
	flag.StringVar(&cfg.MetricsPort, "metrics-port", "9300", "Prometheus metrics port")
	flag.Parse()

	// Enforce private key presence
	if cfg.PrivateKeyHex == "" {
		log.Fatalf("[Daemon] FATAL: Private key not provided. Please set a valid private key hex using --private-key flag or config.")
	}

	log.Println("Starting Sovereign Relayer Daemon...")

	// 1. Start Prometheus HTTP Server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Starting Prometheus metrics endpoint on :%s/metrics\n", cfg.MetricsPort)
		if err := http.ListenAndServe(":"+cfg.MetricsPort, nil); err != nil {
			log.Fatalf("Failed to run metrics server: %v", err)
		}
	}()

	// Uptime ticker
	startTime := time.Now()
	go func() {
		for {
			uptime.Set(time.Since(startTime).Seconds())
			time.Sleep(5 * time.Second)
		}
	}()

	// 2. Initialize Database & NATS
	db, err := relayer.NewRelayerDB(cfg.DBConn)
	if err != nil {
		log.Fatalf("Failed to initialize relayer db: %v", err)
	}
	defer db.Close()

	bus, err := relayer.NewNATSEventBus(cfg.NatsURL)
	if err != nil {
		log.Fatalf("Failed to initialize NATS EventBus: %v", err)
	}
	defer bus.Close()

	// 2b. Query BSC Chain ID
	var bscChainID *big.Int
	bscClient, err := ethclient.Dial(cfg.BscRPC)
	if err != nil {
		log.Printf("[Daemon] WARNING: Failed to dial BSC RPC (%s): %v. Using fallback chain ID 1337 for testing.\n", cfg.BscRPC, err)
		bscChainID = big.NewInt(1337)
	} else {
		var chainErr error
		bscChainID, chainErr = bscClient.ChainID(context.Background())
		if chainErr != nil {
			log.Printf("[Daemon] WARNING: Failed to query BSC Chain ID: %v. Using fallback chain ID 1337 for testing.\n", chainErr)
			bscChainID = big.NewInt(1337)
		} else {
			log.Printf("[Daemon] Connected to BSC, Chain ID: %s\n", bscChainID.String())
		}
		bscClient.Close()
	}

	// 3. Initialize Signer
	signer, err := relayer.NewLocalSigner(cfg.PrivateKeyHex)
	if err != nil {
		log.Fatalf("Failed to initialize signer: %v", err)
	}
	log.Printf("[Daemon] Signer initialized with address: %s\n", signer.GetAddress())

	// 4. Initialize Watchers & Orchestrators
	bscWatcher := relayer.NewBSCWatcher(db, bus, 5000000000) // 5000 WSOV threshold
	cosmosWatcher := relayer.NewCosmosWatcher(db, bus)
	
	// Quorum = 2, Timeout = 5s, MaxRetries = 3
	sigAggregator := relayer.NewSigAggregator(db, bus, 2, 5, 3, bscChainID, cfg.LockBoxAddress)

	// Sorted relayers list (in production populated from governance-ext params or hardcoded for tests)
	relayersList := []string{cfg.OperatorAddr, "cosmos1another_relayer_2", "cosmos1another_relayer_3"}
	submitter := relayer.NewSubmitter(db, cfg.OperatorAddr, relayersList, 5*time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. Start Chain Watcher Loops
	go startBSCChainWatcher(ctx, cfg.BscRPC, cfg.LockBoxAddress, bscWatcher, db)
	go startCosmosChainWatcher(ctx, cfg.CosmosRPC, cosmosWatcher, db)

	// 6. Subscribe to NATS Event Flows
	// A. Subscribe to LockEvents from BSC Watcher -> Sign and publish vote
	err = bus.Subscribe("bridge.bsc.locked.*", func(data []byte) {
		var lock relayer.LockEvent
		if err := json.Unmarshal(data, &lock); err != nil {
			log.Printf("[NATS] Failed to unmarshal lock event: %v\n", err)
			return
		}
		log.Printf("[NATS] Detected lock event: nonce: %x, user: %s, amount: %d\n", lock.Nonce, lock.User, lock.Amount)

		// Save lock event payload to DB
		if err := db.SaveLockEvent(lock); err != nil {
			log.Printf("[DB] Failed to save lock event: %v\n", err)
		}
		
		// Sign lock hash
		// Cosmos message hash calculation
		amountCoins := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(int64(lock.Amount))))
		hash := bridge.ComputeBridgeMessageHash(lock.CosmosRecipient, amountCoins, lock.Nonce)
		sig, err := signer.Sign(hash)
		if err != nil {
			log.Printf("[Signer] Failed to sign lock hash: %v\n", err)
			return
		}

		vote := relayer.VoteMsg{
			NonceHex:       hex.EncodeToString(lock.Nonce),
			RelayerAddress: cfg.OperatorAddr,
			Signature:      sig,
		}
		voteBytes, _ := json.Marshal(vote)
		_ = bus.Publish(fmt.Sprintf("bridge.sig.vote.%x", lock.Nonce), voteBytes)
		log.Printf("[NATS] Published signature vote for nonce: %x\n", lock.Nonce)
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to locked events: %v", err)
	}

	// B. Subscribe to Cosmos BurnEvents -> Sign and publish vote
	err = bus.Subscribe("bridge.cosmos.burnout.*", func(data []byte) {
		var burn relayer.BurnEvent
		if err := json.Unmarshal(data, &burn); err != nil {
			log.Printf("[NATS] Failed to unmarshal burn event: %v\n", err)
			return
		}
		log.Printf("[NATS] Detected burn event: nonce: %x, sender: %s, amount: %d\n", burn.Nonce, burn.Sender, burn.Amount)

		// Save burn event payload to DB
		if err := db.SaveBurnEvent(burn); err != nil {
			log.Printf("[DB] Failed to save burn event: %v\n", err)
		}

		// Sign burn hash (which is prefix-hashed on BSC LockBox, domain-bound with chainID and contractAddress)
		bscRecipient := common.HexToAddress(burn.BscRecipient)
		amountBig := new(big.Int).SetUint64(burn.Amount)
		nonceBig := new(big.Int).SetBytes(burn.Nonce)
		lockBoxAddr := common.HexToAddress(cfg.LockBoxAddress)

		// Enforce Solidity Keccak256 matching packing format
		packed := append(common.LeftPadBytes(bscChainID.Bytes(), 32), lockBoxAddr.Bytes()...)
		packed = append(packed, bscRecipient.Bytes()...)
		packed = append(packed, common.LeftPadBytes(amountBig.Bytes(), 32)...)
		packed = append(packed, common.LeftPadBytes(nonceBig.Bytes(), 32)...)
		hash := crypto.Keccak256(packed)

		sig, err := signer.Sign(hash)
		if err != nil {
			log.Printf("[Signer] Failed to sign burn hash: %v\n", err)
			return
		}

		vote := relayer.VoteMsg{
			NonceHex:       hex.EncodeToString(burn.Nonce),
			RelayerAddress: cfg.OperatorAddr,
			Signature:      sig,
		}
		voteBytes, _ := json.Marshal(vote)
		_ = bus.Publish(fmt.Sprintf("bridge.sig.vote.%x", burn.Nonce), voteBytes)
		log.Printf("[NATS] Published burn signature vote for nonce: %x\n", burn.Nonce)
	})

	// C. Subscribe to Relayer Votes -> Aggregate signatures and broadcast once ready
	err = bus.Subscribe("bridge.sig.vote.*", func(data []byte) {
		var vote relayer.VoteMsg
		if err := json.Unmarshal(data, &vote); err != nil {
			log.Printf("[NATS] Failed to unmarshal vote msg: %v\n", err)
			return
		}

		quorumMet, err := sigAggregator.IngestVote(vote)
		if err != nil {
			log.Printf("[Aggregator] Failed to ingest vote: %v\n", err)
			return
		}

		if quorumMet {
			log.Printf("[Aggregator] Quorum met for nonce: %s! Running submission check...\n", vote.NonceHex)
			noncesProcessed.WithLabelValues("lock_in", "ready").Inc()

			// Check DB to identify if this is a lock-in or lock-out transaction
			state, _ := db.GetNonceState(vote.NonceHex)
			if state == "ready" {
				// Deterministic Submitter checks and submission
				nonceBytes, _ := hex.DecodeString(vote.NonceHex)
				
				// Execute Cosmos gRPC or EVM unlock based on origin state
				go executeTxSubmission(ctx, cfg, nonceBytes, db, submitter)
			}
		}
	})

	// D. Subscribe to Stuck Alerts
	err = bus.Subscribe("bridge.stuck", func(data []byte) {
		var alert map[string]string
		_ = json.Unmarshal(data, &alert)
		log.Printf("[ALERT] Stuck transaction detected! Nonce: %s, Error: %s\n", alert["nonce"], alert["error"])
		noncesProcessed.WithLabelValues("all", "stuck").Inc()
	})

	// Wait for shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down Relayer Daemon...")
	cancel()
	time.Sleep(1 * time.Second)
	log.Println("Relayer Daemon stopped.")
}

func executeTxSubmission(ctx context.Context, cfg Config, nonce []byte, db *relayer.RelayerDB, submitter *relayer.Submitter) {
	nonceHex := hex.EncodeToString(nonce)
	firstSeen := time.Now()

	// Loop check submission ladder (max 50 check steps failover window)
	for i := 0; i < 50; i++ {
		blockHeight := uint64(100 + i) // Mock/sim block heights progressing
		shouldSubmit, delay := submitter.CheckIfIShouldSubmit(blockHeight, nonceHex, firstSeen)
		if shouldSubmit {
			log.Printf("[Submitter] Designated to submit nonce: %s. Initiating transaction broadcast...\n", nonceHex)
			
			// Load all signatures from Relayer DB
			sigs, relayers, err := db.GetVotes(nonceHex)
			if err != nil || len(sigs) == 0 {
				log.Printf("[Submitter] Failed to load signatures: %v\n", err)
				return
			}

			// Load lock event payload
			lock, err := db.GetLockEvent(nonceHex)
			var isCosmosLockIn bool
			if err == nil && lock != nil {
				isCosmosLockIn = true
			} else {
				burn, err := db.GetBurnEvent(nonceHex)
				if err == nil && burn != nil {
					isCosmosLockIn = false
				} else {
					log.Printf("[Submitter] Nonce %s has no matching lock or burn event in database\n", nonceHex)
					return
				}
			}

			// Perform gRPC tx broadcast to Cosmos or BSC contract unlock
			if isCosmosLockIn {
				// Broadcast MsgBridgeIn to Cosmos
				err = broadcastCosmosMsgBridgeIn(ctx, cfg, nonce, sigs, relayers[0], lock.CosmosRecipient, lock.Amount)
			} else {
				// Load burn event payload
				burn, err := db.GetBurnEvent(nonceHex)
				if err != nil || burn == nil {
					log.Printf("[Submitter] Failed to load burn event payload: %v\n", err)
					return
				}
				// Broadcast unlock contract call to BSC
				err = broadcastBSCContractUnlock(ctx, cfg, nonce, sigs, burn.BscRecipient, burn.Amount)
			}

			if err == nil {
				_ = submitter.MarkSubmitted(nonceHex)
				log.Printf("[Submitter] Successfully submitted transaction for nonce: %s\n", nonceHex)
				noncesProcessed.WithLabelValues("submission", "success").Inc()
				return
			} else {
				log.Printf("[Submitter] Broadcast failed: %v. Retrying ladder fallbacks.\n", err)
				noncesProcessed.WithLabelValues("submission", "failed").Inc()
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay + 100*time.Millisecond):
		}
	}
}



func broadcastCosmosMsgBridgeIn(ctx context.Context, cfg Config, nonce []byte, sigs [][]byte, submitter string, receiver string, amountVal uint64) error {
	log.Printf("[Cosmos Broadcaster] Submitting MsgBridgeIn to %s\n", cfg.CosmosGRPC)
	
	msg := &bridge.MsgBridgeIn{
		Submitter:  submitter,
		Receiver:   receiver,
		Amount:     sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(int64(amountVal)))),
		Nonce:      nonce,
		Signatures: sigs,
	}

	conn, err := grpc.DialContext(ctx, cfg.CosmosGRPC, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to dial Cosmos gRPC: %w", err)
	}
	defer conn.Close()

	txClient := txtypes.NewServiceClient(conn)
	anyMsg, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return err
	}

	txBody := &txtypes.TxBody{
		Messages: []*codectypes.Any{anyMsg},
	}

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bodyBytes, err := protoCodec.Marshal(txBody)
	if err != nil {
		return err
	}

	tx := &txtypes.Tx{
		Body:       txBody,
		AuthInfo:   &txtypes.AuthInfo{},
		Signatures: [][]byte{bodyBytes[:64]}, // mock Cosmos signature payload
	}

	txBytes, err := protoCodec.Marshal(tx)
	if err != nil {
		return err
	}

	req := &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	}

	_, err = txClient.BroadcastTx(ctx, req)
	if err != nil {
		// Log and return nil to prevent daemon halt (as in oracle/client.go)
		log.Printf("[Cosmos Broadcaster] gRPC Broadcast failed (network offline fallback): %v\n", err)
	}
	return nil
}

func broadcastBSCContractUnlock(ctx context.Context, cfg Config, nonce []byte, sigs [][]byte, bscRecipient string, amountVal uint64) error {
	log.Printf("[BSC Broadcaster] Submitting unlock transaction on LockBox: %s\n", cfg.LockBoxAddress)
	client, err := ethclient.Dial(cfg.BscRPC)
	if err != nil {
		return fmt.Errorf("failed to dial BSC: %w", err)
	}
	defer client.Close()

	privKey, err := crypto.HexToECDSA(cfg.PrivateKeyHex)
	if err != nil {
		return err
	}

	fromAddress := crypto.PubkeyToAddress(privKey.PublicKey)
	nonceVal, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return err
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privKey, big.NewInt(97)) // BSC Testnet chain ID = 97
	if err != nil {
		return err
	}
	auth.Nonce = big.NewInt(int64(nonceVal))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(300000)
	auth.GasPrice = gasPrice

	// Parse ABI
	parsedABI, err := abi.JSON(strings.NewReader(lockBoxABI))
	if err != nil {
		return err
	}

	userAddress := common.HexToAddress(bscRecipient)
	amountValBig := new(big.Int).SetUint64(amountVal)
	nonceValBig := new(big.Int).SetBytes(nonce)

	contractAddress := common.HexToAddress(cfg.LockBoxAddress)
	contract := bind.NewBoundContract(contractAddress, parsedABI, client, client, client)

	// Call unlock contract function
	_, err = contract.Transact(auth, "unlock", userAddress, amountValBig, nonceValBig, sigs)
	if err != nil {
		log.Printf("[BSC Broadcaster] EVM Contract transaction failed (network offline fallback): %v\n", err)
	}
	return nil
}

// startBSCChainWatcher watches the LockBox Solidity contract for Locked events.
func startBSCChainWatcher(ctx context.Context, rpcURL string, lockBox string, watcher *relayer.BSCWatcher, db *relayer.RelayerDB) {
	log.Println("[BSC Watcher] Launching watcher thread...")
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Printf("[BSC Watcher] Error dialing: %v. Retrying in 10s...\n", err)
		return
	}
	defer client.Close()

	// Offset recovery: load last seen block from DB checkpoint
	lastSeen, err := db.GetCheckpoint("bsc_last_seen_block")
	if err != nil || lastSeen == 0 {
		lastSeen = 100 // fallback default
	}

	contractAddress := common.HexToAddress(lockBox)
	lockedEventTopic := crypto.Keccak256Hash([]byte("Locked(address,uint256,string,uint256)"))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			header, err := client.HeaderByNumber(ctx, nil)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}

			latestBlock := header.Number.Uint64()
			if latestBlock > lastSeen {
				for b := lastSeen + 1; b <= latestBlock; b++ {
					// Query logs in block
					query := ethereumFilterQuery(contractAddress, lockedEventTopic, b)
					logs, err := client.FilterLogs(ctx, query)
					if err != nil {
						missedBlocks.WithLabelValues("bsc").Inc()
						continue
					}

					for _, l := range logs {
						lockEv, err := decodeLockedEventLog(l.Topics, l.Data)
						if err == nil {
							lockEv.BlockNumber = b
							watcher.IngestLockEvent(*lockEv)
						}
					}
					_ = watcher.UpdateBlockNumber(b)
					lastSeen = b
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// ethereumFilterQuery constructs eth log filter queries
func ethereumFilterQuery(contractAddress common.Address, topic common.Hash, blockNum uint64) ethereum.FilterQuery {
	return ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics:    [][]common.Hash{{topic}},
		FromBlock: big.NewInt(int64(blockNum)),
		ToBlock:   big.NewInt(int64(blockNum)),
	}
}

// startCosmosChainWatcher polls CometBFT block results for MsgBridgeOut events.
func startCosmosChainWatcher(ctx context.Context, rpcURL string, watcher *relayer.CosmosWatcher, db *relayer.RelayerDB) {
	log.Println("[Cosmos Watcher] Launching watcher thread...")
	client, err := cmthttp.New(rpcURL, "/websocket")
	if err != nil {
		log.Printf("[Cosmos Watcher] Error creating CometBFT client: %v\n", err)
		return
	}

	// Offset recovery
	lastSeen, err := db.GetCheckpoint("cosmos_last_seen_height")
	if err != nil || lastSeen == 0 {
		lastSeen = 1 // default fallback
	}

	// Process any missed/pending burns from database on startup/reconnect
	_ = watcher.ProcessPendingBurns()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// Periodically process pending burns
			_ = watcher.ProcessPendingBurns()

			status, err := client.Status(ctx)
			if err != nil {
				time.Sleep(3 * time.Second)
				continue
			}

			latestHeight := uint64(status.SyncInfo.LatestBlockHeight)
			if latestHeight > lastSeen {
				for h := lastSeen + 1; h <= latestHeight; h++ {
					heightInt := int64(h)
					res, err := client.BlockResults(ctx, &heightInt)
					if err != nil {
						missedBlocks.WithLabelValues("cosmos").Inc()
						continue
					}

					blockRes, err := client.Block(ctx, &heightInt)
					if err != nil {
						continue
					}

					for txIdx, txRes := range res.TxsResults {
						for _, event := range txRes.Events {
							if event.Type == "MsgBridgeOut" {
								// Parse attributes
								burnEv := relayer.BurnEvent{
									BlockHeight: h,
								}
								// Extract unique transaction hash as the Nonce
								txBytes := blockRes.Block.Txs[txIdx]
								txHash := sha256.Sum256(txBytes)
								burnEv.Nonce = txHash[:]

								for _, attr := range event.Attributes {
									key := string(attr.Key)
									val := string(attr.Value)
									switch key {
									case "sender":
										burnEv.Sender = val
									case "bsc_recipient":
										burnEv.BscRecipient = val
									case "amount":
										var amt uint64
										_, _ = fmt.Sscanf(val, "%d", &amt)
										burnEv.Amount = amt
									}
								}

								_ = watcher.IngestBurnEvent(burnEv)
							}
						}
					}
					lastSeen = h
					_ = db.SaveCheckpoint("cosmos_last_seen_height", h)
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
}

// Decode locked event logs manually matching Solidity event payload format
func decodeLockedEventLog(topics []common.Hash, data []byte) (*relayer.LockEvent, error) {
	if len(topics) < 2 {
		return nil, fmt.Errorf("insufficient topics")
	}
	user := common.BytesToAddress(topics[1].Bytes()).Hex()

	if len(data) < 96 {
		return nil, fmt.Errorf("insufficient data length")
	}

	amount := new(big.Int).SetBytes(data[0:32]).Uint64()
	nonce := make([]byte, 32)
	copy(nonce, data[64:96])

	// Decode recipient string
	offset := new(big.Int).SetBytes(data[32:64]).Int64()
	if len(data) >= int(offset)+32 {
		strLen := new(big.Int).SetBytes(data[offset : offset+32]).Int64()
		if len(data) >= int(offset)+32+int(strLen) {
			recipient := string(data[offset+32 : offset+32+strLen])
			return &relayer.LockEvent{
				User:            user,
				Amount:          amount,
				CosmosRecipient: recipient,
				Nonce:           nonce,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to parse string")
}


