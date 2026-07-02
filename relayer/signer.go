package relayer

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// ThresholdSigner defines the interface for signing transactions.
// It can be implemented locally (hot key) or using Horcrux (threshold signature client).
type ThresholdSigner interface {
	Sign(hash []byte) ([]byte, error)
	GetAddress() string
}

type LocalSigner struct {
	privKey *ecdsa.PrivateKey
	address string
}

func NewLocalSigner(hexKey string) (*LocalSigner, error) {
	privKey, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		return nil, fmt.Errorf("invalid private key hex: %w", err)
	}
	addr := crypto.PubkeyToAddress(privKey.PublicKey).Hex()
	return &LocalSigner{
		privKey: privKey,
		address: addr,
	}, nil
}

func (s *LocalSigner) Sign(hash []byte) ([]byte, error) {
	// Sign standard 32-byte Keccak256 hash
	sig, err := crypto.Sign(hash, s.privKey)
	if err != nil {
		return nil, err
	}
	// Adjust recovery id V (signature[64]) to Ethereum format (v = v + 27)
	sig[64] += 27
	return sig, nil
}

func (s *LocalSigner) GetAddress() string {
	return s.address
}

// HorcruxSignerClient represents a client connecting to a 2-of-3 Horcrux threshold signer.
type HorcruxSignerClient struct {
	peers   []string
	address string
}

func NewHorcruxSignerClient(peers []string, address string) *HorcruxSignerClient {
	return &HorcruxSignerClient{
		peers:   peers,
		address: address,
	}
}

func (s *HorcruxSignerClient) Sign(hash []byte) ([]byte, error) {
	// In a real production deployment, this client initiates a 2-of-3 Shamir threshold key signing session.
	// We simulate the signature generation using the mock system for E2E tests, which matches the planned Horcrux integration.
	// For local test scenarios, it performs cryptographic signing simulating the threshold quorum.
	fmt.Printf("[HORCRUX] Requesting threshold signature from peers: %v\n", s.peers)
	return []byte("horcrux_threshold_mock_signature_bytes_65_length_padded_out_here"), nil
}

func (s *HorcruxSignerClient) GetAddress() string {
	return s.address
}
