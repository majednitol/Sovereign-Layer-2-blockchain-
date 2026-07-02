package relayer

import (
	"encoding/json"
	"fmt"
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
}

func NewSigAggregator(db *RelayerDB, bus EventBus, quorum, timeout, maxRetries int) *SigAggregator {
	return &SigAggregator{
		db:              db,
		bus:             bus,
		quorumThreshold: quorum,
		timeoutSeconds:  timeout,
		maxRetries:      maxRetries,
		retryCounters:   make(map[string]int),
		stuckAlerts:     make(map[string]bool),
	}
}

// IngestVote registers a signature from a peer relayer.
func (a *SigAggregator) IngestVote(vote VoteMsg) (bool, error) {
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
