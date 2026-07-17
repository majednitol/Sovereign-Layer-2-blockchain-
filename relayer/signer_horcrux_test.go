package relayer

import (
	"crypto/rand"
	"testing"
)

// TestShamirSecretSharing verifies that a secret split into 3 shares can be reconstructed
// by any 2 shares, but not by 1 share.
func TestShamirSecretSharing(t *testing.T) {
	// 1. Generate a mock 32-byte secret
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		t.Fatalf("failed to generate random secret: %v", err)
	}

	// 2. Split secret into 3 shares with threshold 2
	shares := make(map[byte][]byte)
	shares[1] = make([]byte, 32)
	shares[2] = make([]byte, 32)
	shares[3] = make([]byte, 32)

	for i := 0; i < 32; i++ {
		s := secret[i]
		// Choose a random non-zero coefficient
		var a byte = 0
		for a == 0 {
			buf := make([]byte, 1)
			_, _ = rand.Read(buf)
			a = buf[0]
		}

		// f(x) = s ^ (a * x)
		// Share 1 (x=1): f(1) = s ^ a
		shares[1][i] = s ^ gf256Mul(a, 1)
		// Share 2 (x=2): f(2) = s ^ (a * 2)
		shares[2][i] = s ^ gf256Mul(a, 2)
		// Share 3 (x=3): f(3) = s ^ (a * 3)
		shares[3][i] = s ^ gf256Mul(a, 3)
	}

	// 3. Test reconstruction with all combinations of 2 shares
	combos := []struct {
		name   string
		subset map[byte][]byte
	}{
		{"1 and 2", map[byte][]byte{1: shares[1], 2: shares[2]}},
		{"1 and 3", map[byte][]byte{1: shares[1], 3: shares[3]}},
		{"2 and 3", map[byte][]byte{2: shares[2], 3: shares[3]}},
		{"1, 2 and 3", map[byte][]byte{1: shares[1], 2: shares[2], 3: shares[3]}},
	}

	for _, tc := range combos {
		t.Run(tc.name, func(t *testing.T) {
			reconstructed, err := ReconstructSecret(tc.subset)
			if err != nil {
				t.Fatalf("reconstruction failed: %v", err)
			}
			for i := 0; i < 32; i++ {
				if reconstructed[i] != secret[i] {
					t.Fatalf("byte mismatch at index %d: expected %d, got %d", i, secret[i], reconstructed[i])
				}
			}
		})
	}

	// 4. Test reconstruction fails or is incorrect with 1 share
	t.Run("insufficient shares (1)", func(t *testing.T) {
		singleShare := map[byte][]byte{1: shares[1]}
		_, err := ReconstructSecret(singleShare)
		if err == nil {
			t.Fatal("expected error when reconstructing with only 1 share")
		}
	})
}
