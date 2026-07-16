package main

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ThalesGroup/crypto11"
)

type KeyManager interface {
	Sign(payload []byte) ([]byte, error)
	GetPublicKey() []byte
}

// MockHSMKeyManager is for LOCAL DEVELOPMENT AND TESTING ONLY.
// It generates a random ephemeral Ed25519 keypair in memory.
// On mainnet, NewHSMKeyManager will refuse to return this.
type MockHSMKeyManager struct {
	privKey ed25519.PrivateKey
	pubKey  ed25519.PublicKey
}

func NewMockHSMKeyManager() (*MockHSMKeyManager, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &MockHSMKeyManager{
		privKey: priv,
		pubKey:  pub,
	}, nil
}

func (m *MockHSMKeyManager) Sign(payload []byte) ([]byte, error) {
	return ed25519.Sign(m.privKey, payload), nil
}

func (m *MockHSMKeyManager) GetPublicKey() []byte {
	return m.pubKey
}

type HSMKeyManager struct {
	ctx    *crypto11.Context
	signer crypto.Signer
	pubKey []byte
}

// NewHSMKeyManager initializes key management for the oracle daemon.
//
// PRODUCTION MODE (ALLOW_MOCK_HSM unset or "false"):
//   - Requires HSM_CONFIG to be set to a valid PKCS#11 config
//   - Returns an error (never falls back to mock) on any HSM failure
//
// DEVELOPMENT MODE (ALLOW_MOCK_HSM="true"):
//   - Falls back to MockHSMKeyManager if HSM is unavailable
//   - Prints warnings on each fallback
func NewHSMKeyManager(configPath string, keyID []byte) (KeyManager, error) {
	allowMock := os.Getenv("ALLOW_MOCK_HSM") == "true"

	if configPath == "" {
		if allowMock {
			fmt.Println("[HSM] WARNING: Config path is empty, using Mock HSM Key Manager (ALLOW_MOCK_HSM=true)")
			return NewMockHSMKeyManager()
		}
		return nil, fmt.Errorf("HSM_CONFIG is not set. " +
			"Set HSM_CONFIG to a valid PKCS#11 configuration file path. " +
			"For development only, set ALLOW_MOCK_HSM=true to use an ephemeral key")
	}

	bz, err := os.ReadFile(configPath)
	if err != nil {
		if allowMock {
			fmt.Printf("[HSM] WARNING: Failed to read config file from %s: %v. Using Mock HSM (ALLOW_MOCK_HSM=true).\n", configPath, err)
			return NewMockHSMKeyManager()
		}
		return nil, fmt.Errorf("failed to read HSM config file from %s: %w", configPath, err)
	}

	var config crypto11.Config
	err = json.Unmarshal(bz, &config)
	if err != nil {
		if allowMock {
			fmt.Printf("[HSM] WARNING: Failed to parse config JSON: %v. Using Mock HSM (ALLOW_MOCK_HSM=true).\n", err)
			return NewMockHSMKeyManager()
		}
		return nil, fmt.Errorf("failed to parse HSM config JSON: %w", err)
	}

	ctx, err := crypto11.Configure(&config)
	if err != nil {
		if allowMock {
			fmt.Printf("[HSM] WARNING: Failed to configure PKCS#11: %v. Using Mock HSM (ALLOW_MOCK_HSM=true).\n", err)
			return NewMockHSMKeyManager()
		}
		return nil, fmt.Errorf("failed to configure PKCS#11: %w. "+
			"Ensure the PKCS#11 library is installed and the HSM device is connected", err)
	}

	signer, err := ctx.FindKeyPair(keyID, nil)
	if err != nil || signer == nil {
		if allowMock {
			fmt.Printf("[HSM] WARNING: Key pair not found: %v. Using Mock HSM (ALLOW_MOCK_HSM=true).\n", err)
			return NewMockHSMKeyManager()
		}
		return nil, fmt.Errorf("HSM key pair not found for keyID %x: %w. "+
			"Ensure the oracle key is provisioned on the HSM device", keyID, err)
	}

	pubBytes, ok := signer.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("HSM key is not an Ed25519 key — oracle requires Ed25519 for Cosmos SDK compatibility")
	}

	fmt.Println("[HSM] Successfully initialized PKCS#11 hardware key manager")
	return &HSMKeyManager{
		ctx:    ctx,
		signer: signer,
		pubKey: pubBytes,
	}, nil
}

func (h *HSMKeyManager) Sign(payload []byte) ([]byte, error) {
	return h.signer.Sign(rand.Reader, payload, crypto.Hash(0))
}

func (h *HSMKeyManager) GetPublicKey() []byte {
	return h.pubKey
}
