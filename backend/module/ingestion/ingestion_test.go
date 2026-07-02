package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDecodeBase64(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"cmVjZWl2ZXI=", "receiver"},
		{"c292MTIzNDU=", "sov12345"},
		{"not_base64!", "not_base64!"}, // fallback
	}

	for _, tc := range tests {
		got := decodeBase64(tc.input)
		if got != tc.expected {
			t.Errorf("decodeBase64(%q) = %q; expected %q", tc.input, got, tc.expected)
		}
	}
}

func TestEventPayloadThreshold(t *testing.T) {
	// 1. Event smaller than 750KB
	smallEvent := EventRecord{
		BlockHeight: 10,
		EventIndex:  1,
		EventType:   "MsgBridgeIn",
		Payload:     json.RawMessage(`{"receiver":"sov123","amount":"500usov"}`),
	}

	smallPayload, err := json.Marshal(smallEvent)
	if err != nil {
		t.Fatalf("failed to marshal small event: %v", err)
	}

	if len(smallPayload) > PayloadThreshold {
		t.Errorf("expected small event size to be under threshold, got %d", len(smallPayload))
	}

	// 2. Event larger than 750KB
	largeData := strings.Repeat("a", PayloadThreshold+100)
	largeEvent := EventRecord{
		BlockHeight: 11,
		EventIndex:  2,
		EventType:   "MsgBridgeIn",
		Payload:     json.RawMessage(`{"receiver":"sov123","data":"` + largeData + `"}`),
	}

	largePayloadBytes, err := json.Marshal(largeEvent)
	if err != nil {
		t.Fatalf("failed to marshal large event: %v", err)
	}

	if len(largePayloadBytes) <= PayloadThreshold {
		t.Errorf("expected large event size to be over threshold, got %d", len(largePayloadBytes))
	}

	// Verify that serializing the RefPointer yields a tiny payload under the threshold
	if len(largePayloadBytes) > PayloadThreshold {
		ref := RefPointer{
			BlockHeight: largeEvent.BlockHeight,
			EventIndex:  largeEvent.EventIndex,
			Ref:         "db",
		}
		refBytes, err := json.Marshal(ref)
		if err != nil {
			t.Fatalf("failed to marshal ref pointer: %v", err)
		}

		if len(refBytes) > PayloadThreshold {
			t.Errorf("expected ref pointer payload to be tiny, got %d", len(refBytes))
		}

		var parsedRef RefPointer
		if err := json.Unmarshal(refBytes, &parsedRef); err != nil {
			t.Fatalf("failed to unmarshal ref pointer: %v", err)
		}

		if parsedRef.BlockHeight != 11 || parsedRef.EventIndex != 2 || parsedRef.Ref != "db" {
			t.Errorf("ref pointer values mismatch, got %+v", parsedRef)
		}
	}
}
