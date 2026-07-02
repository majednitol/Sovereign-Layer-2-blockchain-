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

func NewHSMKeyManager(configPath string, keyID []byte) (KeyManager, error) {
	if configPath == "" {
		fmt.Println("[HSM] Config path is empty, falling back to Mock HSM Key Manager")
		return NewMockHSMKeyManager()
	}

	bz, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("[HSM] Failed to read config file from %s: %v. Falling back to Mock HSM Key Manager.\n", configPath, err)
		return NewMockHSMKeyManager()
	}

	var config crypto11.Config
	err = json.Unmarshal(bz, &config)
	if err != nil {
		fmt.Printf("[HSM] Failed to parse config JSON: %v. Falling back to Mock HSM Key Manager.\n", err)
		return NewMockHSMKeyManager()
	}

	ctx, err := crypto11.Configure(&config)
	if err != nil {
		fmt.Printf("[HSM] Failed to configure PKCS#11: %v. Falling back to Mock HSM Key Manager.\n", err)
		return NewMockHSMKeyManager()
	}

	signer, err := ctx.FindKeyPair(keyID, nil)
	if err != nil || signer == nil {
		fmt.Printf("[HSM] Key pair not found or error: %v. Falling back to Mock HSM Key Manager.\n", err)
		return NewMockHSMKeyManager()
	}

	pubBytes, ok := signer.Public().(ed25519.PublicKey)
	if !ok {
		return nil, fmt.Errorf("HSM key is not an Ed25519 key")
	}

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
