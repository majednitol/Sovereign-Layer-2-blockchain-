package relayer

import (
	"sort"
	"time"
)

type Submitter struct {
	db          *RelayerDB
	operatorAdd string
	relayers    []string // sorted list of relayer addresses
	delayFactor time.Duration
}

func NewSubmitter(db *RelayerDB, operatorAdd string, relayersList []string, delayFactor time.Duration) *Submitter {
	// Sort relayers deterministically
	sorted := make([]string, len(relayersList))
	copy(sorted, relayersList)
	sort.Strings(sorted)

	return &Submitter{
		db:          db,
		operatorAdd: operatorAdd,
		relayers:    sorted,
		delayFactor: delayFactor,
	}
}

func (s *Submitter) GetMyIndex() int {
	for i, rel := range s.relayers {
		if rel == s.operatorAdd {
			return i
		}
	}
	return -1
}

// CheckIfIShouldSubmit determines if this relayer node is designated to submit,
// or if it should promote to submitter based on elapsed block time/delay.
func (s *Submitter) CheckIfIShouldSubmit(blockHeight uint64, nonceHex string, firstSeen time.Time) (bool, time.Duration) {
	myIndex := s.GetMyIndex()
	if myIndex == -1 {
		return false, 0
	}

	totalRelayers := len(s.relayers)
	if totalRelayers == 0 {
		return false, 0
	}

	// Submitter index determined by height modulo total relayers
	designatedIndex := int(blockHeight % uint64(totalRelayers))

	// Determine slot offset distance
	slotDiff := (myIndex - designatedIndex + totalRelayers) % totalRelayers

	// Slot delay time required before this relayer promotes
	requiredDelay := time.Duration(slotDiff) * s.delayFactor
	elapsed := time.Since(firstSeen)

	if elapsed >= requiredDelay {
		// Verify if it has already been submitted to prevent duplication
		state, _ := s.db.GetNonceState(nonceHex)
		if state == "submitted" {
			return false, 0
		}
		return true, 0
	}

	return false, requiredDelay - elapsed
}

func (s *Submitter) MarkSubmitted(nonceHex string) error {
	return s.db.SetNonceState(nonceHex, "submitted")
}
