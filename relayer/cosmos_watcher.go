package relayer

import (
	"encoding/json"
	"fmt"
)

type BurnEvent struct {
	Sender       string `json:"sender"`
	BscRecipient string `json:"bsc_recipient"`
	Amount       uint64 `json:"amount"`
	Nonce        []byte `json:"nonce"`
	BlockHeight  uint64 `json:"block_height"`
}

type CosmosWatcher struct {
	db           *RelayerDB
	bus          EventBus
	pendingBurns []BurnEvent
}

func NewCosmosWatcher(db *RelayerDB, bus EventBus) *CosmosWatcher {
	return &CosmosWatcher{
		db:           db,
		bus:          bus,
		pendingBurns: make([]BurnEvent, 0),
	}
}

// IngestBurnEvent processes a raw MsgBridgeOut event scanned from a block.
func (w *CosmosWatcher) IngestBurnEvent(event BurnEvent) error {
	nonceHex := fmt.Sprintf("%x", event.Nonce)

	// Check if already processed
	state, err := w.db.GetNonceState(nonceHex)
	if err == nil && (state == "observed" || state == "burned") {
		return nil
	}

	// 1. Save burn event and mark state as observed in database
	if err := w.db.SaveBurnEvent(event); err != nil {
		return err
	}
	_ = w.db.SetNonceState(nonceHex, "observed")
	_ = w.db.SaveCheckpoint("cosmos_last_seen_height", event.BlockHeight)

	// 2. Publish to NATS bus
	bz, err := json.Marshal(event)
	if err != nil {
		return err
	}
	subject := fmt.Sprintf("bridge.cosmos.burnout.%s", nonceHex)
	err = w.bus.Publish(subject, bz)
	if err != nil {
		// Log the error but do not fail; the background processor will retry later.
		fmt.Printf("[CosmosWatcher] Failed to publish burn event %s: %v. Retrying in background.\n", nonceHex, err)
		return nil
	}

	// 3. Mark state as burned upon successful publication
	_ = w.db.SetNonceState(nonceHex, "burned")
	return nil
}

// ProcessPendingBurns queries the database for all burns in 'observed' state and attempts to publish them.
func (w *CosmosWatcher) ProcessPendingBurns() error {
	pending, err := w.db.GetPendingBurns()
	if err != nil {
		return err
	}

	for _, event := range pending {
		bz, err := json.Marshal(event)
		if err != nil {
			continue
		}
		nonceHex := fmt.Sprintf("%x", event.Nonce)
		subject := fmt.Sprintf("bridge.cosmos.burnout.%s", nonceHex)
		err = w.bus.Publish(subject, bz)
		if err != nil {
			return fmt.Errorf("failed to publish pending burn %s: %w", nonceHex, err)
		}
		_ = w.db.SetNonceState(nonceHex, "burned")
	}
	return nil
}
