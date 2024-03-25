package pcrypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/bokwoon95/sq"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/db"
	"golang.org/x/crypto/ssh"
)

const (
	sshDS = "ssh"
)

// Manager keeps track of all the keys
type Manager struct {
	db *db.DB
}

// GenerateKey generates a SSH key pair
func (sm *Manager) GenerateKey() (*Key, error) {
	key := &Key{}
	var err error
	key.Pub, key.Priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return key, errors.Wrap(err, "Failed to generate SSH key")
	}
	return key, nil
}

// GetKeyByPub returns a key that has the provided pubkey (base64 encoded)
func (sm *Manager) GetKeyByPub(pubKey string) (Key, error) {
	keyModel := sq.New[db.SSH_KEY]("")
	key, err := db.SelectOne(sm.db, createKeyQueryMapper(keyModel, []sq.Predicate{keyModel.PUBLIC.EqString(pubKey)}))
	if err != nil {
		return key, fmt.Errorf("failed to retrieve key: %w", err)
	}

	return key, nil
}

// NewAuthFromKeyFile takes a file path and returns an ssh authentication
func (sm *Manager) NewAuthFromKeyFile(keyPath string) (ssh.AuthMethod, error) {

	privKey, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// NewKeyFromSeed takes an ed25519 key seed and return a Key
func (sm *Manager) NewKeyFromSeed(seedStr string) (*Key, error) {
	seed, err := base64.StdEncoding.DecodeString(seedStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode seed: %w", err)
	}
	key := &Key{}
	if len(seed) != 32 {
		return key, errors.Errorf("Can't create key from seed. Seed has incorrect length: %d bytes", len(seed))
	}
	key.Priv = ed25519.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, key.Priv[32:])
	key.Pub = publicKey
	return key, nil
}

// CreateManager returns a Manager, which implements the core.ProviderManager interface
func CreateManager(db *db.DB) *Manager {
	return &Manager{db: db}
}
