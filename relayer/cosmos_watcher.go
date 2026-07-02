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
	w.pendingBurns = append(w.pendingBurns, event)
	_ = w.db.SaveCheckpoint("cosmos_last_seen_height", event.BlockHeight)

	bz, err := json.Marshal(event)
	if err != nil {
		return err
	}
	nonceHex := fmt.Sprintf("%x", event.Nonce)
	subject := fmt.Sprintf("bridge.cosmos.burnout.%s", nonceHex)
	err = w.bus.Publish(subject, bz)
	if err != nil {
		return err
	}
	_ = w.db.SetNonceState(nonceHex, "burned")
	return nil
}
