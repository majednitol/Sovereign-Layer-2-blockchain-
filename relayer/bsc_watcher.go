package relayer

import (
	"encoding/json"
	"fmt"
)

type LockEvent struct {
	User            string `json:"user"`
	Amount          uint64 `json:"amount"`
	CosmosRecipient string `json:"cosmos_recipient"`
	Nonce           []byte `json:"nonce"`
	BlockNumber     uint64 `json:"block_number"`
}

type EventBus interface {
	Publish(subject string, data []byte) error
	Subscribe(subject string, handler func(msg []byte)) error
}

type BSCWatcher struct {
	db                     *RelayerDB
	bus                    EventBus
	largeTransferThreshold uint64
	currentBscBlock        uint64
	pendingLocks           []LockEvent
}

func NewBSCWatcher(db *RelayerDB, bus EventBus, threshold uint64) *BSCWatcher {
	return &BSCWatcher{
		db:                     db,
		bus:                    bus,
		largeTransferThreshold: threshold,
		pendingLocks:           make([]LockEvent, 0),
	}
}

// IngestLockEvent adds a new locked event discovered from the LockBox contract logs.
func (w *BSCWatcher) IngestLockEvent(event LockEvent) {
	w.pendingLocks = append(w.pendingLocks, event)
}

// UpdateBlockNumber simulates progressing BSC blocks and checks confirmation depths.
func (w *BSCWatcher) UpdateBlockNumber(blockNum uint64) error {
	w.currentBscBlock = blockNum
	_ = w.db.SaveCheckpoint("bsc_last_seen_block", blockNum)

	var remaining []LockEvent
	for _, lock := range w.pendingLocks {
		requiredConfirmations := uint64(15)
		if lock.Amount >= w.largeTransferThreshold {
			requiredConfirmations = 50
		}

		if w.currentBscBlock >= lock.BlockNumber+requiredConfirmations {
			// Confirmed! Publish to NATS JetStream
			bz, err := json.Marshal(lock)
			if err != nil {
				return err
			}
			nonceHex := fmt.Sprintf("%x", lock.Nonce)
			subject := fmt.Sprintf("bridge.bsc.locked.%s", nonceHex)
			err = w.bus.Publish(subject, bz)
			if err != nil {
				return err
			}
			_ = w.db.SetNonceState(nonceHex, "confirmed")
		} else {
			remaining = append(remaining, lock)
		}
	}
	w.pendingLocks = remaining
	return nil
}
