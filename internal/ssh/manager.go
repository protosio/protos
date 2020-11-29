package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"io/ioutil"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/db"
	"golang.org/x/crypto/ssh"
)

const (
	sshDS = "ssh"
)

// Manager keeps track of all the keys
type Manager struct {
	db db.DB
}

// GenerateKey generates a SSH key pair
func (sm *Manager) GenerateKey() (*Key, error) {
	key := &Key{parent: sm}
	var err error
	key.Pub, key.Priv, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return key, errors.Wrap(err, "Failed to generate SSH key")
	}
	return key, nil
}

// GetKeyByPub returns a key that has the provided pubkey (base64 encoded)
func (sm *Manager) GetKeyByPub(pubKey string) (*Key, error) {
	var keys map[string]*Key
	err := sm.db.GetMap(sshDS, &keys)
	if err != nil {
		return nil, err
	}

	for _, k := range keys {
		if k.PublicWG().String() == pubKey {
			k.parent = sm
			return k, nil
		}
	}
	return nil, fmt.Errorf("Could not find key with pubkey '%s'", pubKey)
}

// NewAuthFromKeyFile takes a file path and returns an ssh authentication
func (sm *Manager) NewAuthFromKeyFile(keyPath string) (ssh.AuthMethod, error) {

	privKey, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// NewKeyFromSeed takes an ed25519 key seed and return a Key
func (sm *Manager) NewKeyFromSeed(seed []byte) (*Key, error) {
	key := &Key{}
	if len(seed) != 32 {
		return key, errors.Errorf("Can't create key from seed. Seed has incorrect length: %d bytes", len(seed))
	}
	key.Priv = ed25519.NewKeyFromSeed(seed)
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey, key.Priv[32:])
	key.Pub = publicKey
	key.parent = sm
	return key, nil
}

// CreateManager returns a Manager, which implements the core.ProviderManager interface
func CreateManager(db db.DB) *Manager {
	if db == nil {
		log.Panic("Failed to create resource manager: none of the inputs can be nil")
	}

	err := db.InitMap(sshDS, false)
	if err != nil {
		log.Fatal("Failed to initialize ssh dataset: ", err)
	}

	return &Manager{db: db}
}
