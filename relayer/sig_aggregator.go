package relayer

import (
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sovereign-l1/chain/x/bridge"
)

type VoteMsg struct {
	NonceHex        string `json:"nonce_hex"`
	RelayerAddress  string `json:"relayer_address"`
	Signature       []byte `json:"signature"`
}

type SigAggregator struct {
	db              *RelayerDB
	bus             EventBus
	quorumThreshold int
	timeoutSeconds  int
	maxRetries      int
	retryCounters   map[string]int // nonceHex -> retry count
	stuckAlerts     map[string]bool
	bscChainID      *big.Int
	lockBoxAddress  string
}

func NewSigAggregator(db *RelayerDB, bus EventBus, quorum, timeout, maxRetries int, bscChainID *big.Int, lockBoxAddress string) *SigAggregator {
	return &SigAggregator{
		db:              db,
		bus:             bus,
		quorumThreshold: quorum,
		timeoutSeconds:  timeout,
		maxRetries:      maxRetries,
		retryCounters:   make(map[string]int),
		stuckAlerts:     make(map[string]bool),
		bscChainID:      bscChainID,
		lockBoxAddress:  lockBoxAddress,
	}
}

// IngestVote registers a signature from a peer relayer.
func (a *SigAggregator) IngestVote(vote VoteMsg) (bool, error) {
	// Cryptographically verify the signature
	var messageHash []byte

	lockEvent, err := a.db.GetLockEvent(vote.NonceHex)
	if err != nil {
		return false, err
	}

	if lockEvent != nil {
		// Cosmos message hash calculation
		amountCoins := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(int64(lockEvent.Amount))))
		messageHash = bridge.ComputeBridgeMessageHash(lockEvent.CosmosRecipient, amountCoins, lockEvent.Nonce)
	} else {
		burnEvent, err := a.db.GetBurnEvent(vote.NonceHex)
		if err != nil {
			return false, err
		}
		if burnEvent != nil {
			// BSC LockBox message hash calculation (domain-bound)
			bscRecipient := common.HexToAddress(burnEvent.BscRecipient)
			amountBig := new(big.Int).SetUint64(burnEvent.Amount)
			nonceBig := new(big.Int).SetBytes(burnEvent.Nonce)
			lockBoxAddr := common.HexToAddress(a.lockBoxAddress)

			packed := append(common.LeftPadBytes(a.bscChainID.Bytes(), 32), lockBoxAddr.Bytes()...)
			packed = append(packed, bscRecipient.Bytes()...)
			packed = append(packed, common.LeftPadBytes(amountBig.Bytes(), 32)...)
			packed = append(packed, common.LeftPadBytes(nonceBig.Bytes(), 32)...)
			innerHash := crypto.Keccak256(packed)
			
			prefix := []byte("\x19Ethereum Signed Message:\n32")
			messageHash = crypto.Keccak256(append(prefix, innerHash...))
		} else {
			return false, fmt.Errorf("unknown transaction nonce: %s", vote.NonceHex)
		}
	}

	// Verify signature
	ok, err := verifySignature(vote.RelayerAddress, messageHash, vote.Signature)
	if err != nil || !ok {
		return false, fmt.Errorf("invalid signature from relayer %s: %v", vote.RelayerAddress, err)
	}

	// Deduplicate and save vote
	count, err := a.db.AddVote(vote.NonceHex, vote.RelayerAddress, vote.Signature)
	if err != nil {
		return false, err
	}

	if count >= a.quorumThreshold {
		_ = a.db.SetNonceState(vote.NonceHex, "ready")
		return true, nil // Quorum met!
	}

	return false, nil
}

func verifySignature(address string, messageHash []byte, signature []byte) (bool, error) {
	if len(signature) != 65 {
		return false, fmt.Errorf("invalid signature length: %d", len(signature))
	}
	
	sigCopy := make([]byte, 65)
	copy(sigCopy, signature)
	if sigCopy[64] >= 27 {
		sigCopy[64] -= 27
	}
	
	pubKeyBytes, err := crypto.Ecrecover(messageHash, sigCopy)
	if err != nil {
		return false, fmt.Errorf("failed to ecrecover: %w", err)
	}
	
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	
	compressed := crypto.CompressPubkey(pubKey)
	cosmosPubKey := &secp256k1.PubKey{Key: compressed}
	derivedAddress := sdk.AccAddress(cosmosPubKey.Address()).String()
	
	return derivedAddress == address, nil
}

// HandleTimeout processes a tick checking if transactions have timed out waiting for signatures.
func (a *SigAggregator) HandleTimeout(nonceHex string) {
	state, _ := a.db.GetNonceState(nonceHex)
	if state == "ready" || state == "submitted" || state == "stuck" {
		return
	}

	a.retryCounters[nonceHex]++
	if a.retryCounters[nonceHex] > a.maxRetries {
		_ = a.db.SetNonceState(nonceHex, "stuck")
		a.stuckAlerts[nonceHex] = true
		
		// Publish stuck alert event to NATS
		alert := map[string]string{
			"nonce": nonceHex,
			"error": "quorum timeout: exceeded max retries",
		}
		bz, _ := json.Marshal(alert)
		_ = a.bus.Publish("bridge.stuck", bz)
	} else {
		// Re-publish the timeout ping to retry signature aggregation
		retryMsg := map[string]interface{}{
			"nonce": nonceHex,
			"retry": a.retryCounters[nonceHex],
		}
		bz, _ := json.Marshal(retryMsg)
		_ = a.bus.Publish(fmt.Sprintf("bridge.sig.retry.%s", nonceHex), bz)
	}
}
