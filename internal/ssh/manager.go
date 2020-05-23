package ssh

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/core"
)

const (
	sshDS = "ssh"
)

// Manager keeps track of all the keys
type Manager struct {
	db core.DB
}

// GenerateKey generates a SSH key pair
func (sm *Manager) GenerateKey() (core.Key, error) {
	key := Key{}
	var err error
	key.public, key.private, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return key, errors.Wrap(err, "Failed to generate SSH key")
	}
	return key, nil
}

// GetKeyByPub returns a key that has the provided pubkey (base64 encoded)
func (sm *Manager) GetKeyByPub(pubKey string) (core.Key, error) {
	var keys []Key
	err := sm.db.GetSet(sshDS, &keys)
	if err != nil {
		return Key{}, err
	}

	for _, k := range keys {
		if k.PublicWG().String() == pubKey {
			k.parent = sm
			return k, nil
		}
	}
	return Key{}, fmt.Errorf("Could not find key with pubkey '%s'", pubKey)
}

// CreateManager returns a Manager, which implements the core.ProviderManager interface
func CreateManager(db core.DB) *Manager {
	if db == nil {
		log.Panic("Failed to create resource manager: none of the inputs can be nil")
	}
	return &Manager{db: db}
}
