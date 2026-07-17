//go:build !dev

package relayer

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// CosignRequest is the payload sent to a cosigner peer.
type CosignRequest struct {
	Hash string `json:"hash"`
}

// CosignResponse is the response returned by a cosigner peer containing a Shamir key share.
type CosignResponse struct {
	ShareID   int    `json:"share_id"`
	ShareData string `json:"share_data"`
}

// GF(256) tables using primitive polynomial 0x11d
var (
	gf256Exp [512]byte
	gf256Log [256]byte
)

func init() {
	var x byte = 1
	for i := 0; i < 255; i++ {
		gf256Exp[i] = x
		gf256Log[x] = byte(i)
		if x&0x80 != 0 {
			x = (x << 1) ^ 0x1d
		} else {
			x <<= 1
		}
	}
	for i := 255; i < 512; i++ {
		gf256Exp[i] = gf256Exp[i-255]
	}
}

func gf256Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return gf256Exp[uint16(gf256Log[a])+uint16(gf256Log[b])]
}

func gf256Div(a, b byte) byte {
	if a == 0 {
		return 0
	}
	if b == 0 {
		panic("division by zero in GF(256)")
	}
	diff := int(gf256Log[a]) - int(gf256Log[b])
	if diff < 0 {
		diff += 255
	}
	return gf256Exp[diff]
}

// ReconstructSecret reconstructs the secret using Lagrange interpolation over GF(256)
func ReconstructSecret(shares map[byte][]byte) ([]byte, error) {
	if len(shares) < 2 {
		return nil, fmt.Errorf("insufficient shares for reconstruction: got %d, need at least 2", len(shares))
	}

	var secretLen int
	for _, data := range shares {
		secretLen = len(data)
		break
	}

	secret := make([]byte, secretLen)

	for byteIdx := 0; byteIdx < secretLen; byteIdx++ {
		var sum byte = 0
		for xi, yi := range shares {
			var num byte = 1
			var denom byte = 1
			for xj := range shares {
				if xi == xj {
					continue
				}
				num = gf256Mul(num, xj)
				denom = gf256Mul(denom, xj^xi)
			}
			coeff := gf256Div(num, denom)
			sum ^= gf256Mul(coeff, yi[byteIdx])
		}
		secret[byteIdx] = sum
	}

	return secret, nil
}

// Sign requests Shamir shares from the cosigner peers, reconstructs the private key,
// generates the ECDSA signature, and safely wipes the sensitive memory.
func (s *HorcruxSignerClient) Sign(hash []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("invalid hash length: expected 32 bytes, got %d", len(hash))
	}

	hashHex := hex.EncodeToString(hash)
	reqBody, err := json.Marshal(CosignRequest{Hash: hashHex})
	if err != nil {
		return nil, err
	}

	// Channel to receive shares from concurrent workers
	type shareResult struct {
		id   byte
		data []byte
		err  error
	}
	shareChan := make(chan shareResult, len(s.peers))

	// Setup context with timeout to cancel outstanding requests once quorum is met
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Get auth token from environment for production security
	authToken := os.Getenv("HORCRUX_COSIGNER_TOKEN")

	// Dispatch requests concurrently
	for _, peer := range s.peers {
		go func(peerAddr string) {
			req, err := http.NewRequestWithContext(ctx, "POST", peerAddr+"/cosign", bytes.NewBuffer(reqBody))
			if err != nil {
				shareChan <- shareResult{err: err}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if authToken != "" {
				req.Header.Set("Authorization", "Bearer "+authToken)
			}

			httpClient := &http.Client{
				Timeout: 2 * time.Second,
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				shareChan <- shareResult{err: err}
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				shareChan <- shareResult{err: fmt.Errorf("peer returned status %d", resp.StatusCode)}
				return
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				shareChan <- shareResult{err: err}
				return
			}

			var cosignResp CosignResponse
			if err := json.Unmarshal(body, &cosignResp); err != nil {
				shareChan <- shareResult{err: err}
				return
			}

			shareBytes, err := hex.DecodeString(cosignResp.ShareData)
			if err != nil || len(shareBytes) != 32 {
				shareChan <- shareResult{err: fmt.Errorf("invalid share length or format")}
				return
			}

			shareChan <- shareResult{id: byte(cosignResp.ShareID), data: shareBytes}
		}(peer)
	}

	shares := make(map[byte][]byte)
	var lastErr error

	// Collect shares until we meet quorum (2) or all peers respond
	for i := 0; i < len(s.peers); i++ {
		select {
		case res := <-shareChan:
			if res.err != nil {
				lastErr = res.err
				continue
			}
			shares[res.id] = res.data
			if len(shares) >= 2 {
				// Quorum met! Cancel outstanding HTTP requests immediately
				cancel()
				break
			}
		case <-ctx.Done():
			break
		}
		if len(shares) >= 2 {
			break
		}
	}

	if len(shares) < 2 {
		if lastErr != nil {
			return nil, fmt.Errorf("threshold signing failed: collected %d shares (need 2). Last error: %w", len(shares), lastErr)
		}
		return nil, fmt.Errorf("threshold signing failed: collected %d shares, quorum requires 2", len(shares))
	}

	// Reconstruct private key bytes
	privKeyBytes, err := ReconstructSecret(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct private key from shares: %w", err)
	}

	// Convert bytes to ecdsa PrivateKey
	privKey, err := crypto.ToECDSA(privKeyBytes)
	if err != nil {
		// Zero the key bytes on error
		for i := range privKeyBytes {
			privKeyBytes[i] = 0
		}
		return nil, fmt.Errorf("invalid reconstructed private key: %w", err)
	}

	// Generate Ethereum-style signature
	sig, err := crypto.Sign(hash, privKey)

	// Securely zero out the reconstructed private key and shares
	for i := range privKeyBytes {
		privKeyBytes[i] = 0
	}
	if privKey.D != nil {
		privKey.D.SetInt64(0)
	}
	for k := range shares {
		for i := range shares[k] {
			shares[k][i] = 0
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to sign with reconstructed key: %w", err)
	}

	// Ethereum V adjustment: add 27 to recovery ID
	sig[64] += 27
	return sig, nil
}
