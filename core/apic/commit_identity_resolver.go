package apic

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/user"
)

type commitIdentityResolver struct {
	services *Services

	mu          sync.RWMutex
	loaded      bool
	byPublicKey map[string]string
}

func newCommitIdentityResolver(services *Services) *commitIdentityResolver {
	resolver := &commitIdentityResolver{
		services:    services,
		byPublicKey: map[string]string{},
	}
	resolver.refresh()
	if services != nil && services.DB != nil {
		services.DB.RegisterTableChangeCallback("users", resolver)
		services.DB.RegisterTableChangeCallback("user_devices_metadata", resolver)
	}
	return resolver
}

func (r *commitIdentityResolver) Notify() {
	r.refresh()
}

func (r *commitIdentityResolver) displayCommitter(commit db.Commit) string {
	signerPublicKey := strings.TrimSpace(commit.SignerPublicKey)
	if signerPublicKey == "" {
		signerPublicKey = db.ExtractCommitSignerPublicKey(commit.Message)
	}
	if signerPublicKey != "" {
		if label, ok := r.resolvePublicKey(signerPublicKey); ok {
			return label
		}
		return fmt.Sprintf("unknown device (%s)", shortPublicKey(signerPublicKey))
	}
	if isSyntheticCommitter(commit.Committer) {
		return "system"
	}
	return commit.Committer
}

func (r *commitIdentityResolver) resolvePublicKey(publicKey string) (string, bool) {
	if r == nil {
		return "", false
	}
	r.ensureLoaded()
	keys := publicKeyLookupKeys(publicKey)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, key := range keys {
		if label, found := r.byPublicKey[key]; found {
			return label, true
		}
	}
	return "", false
}

func (r *commitIdentityResolver) ensureLoaded() {
	if r == nil {
		return
	}
	r.mu.RLock()
	loaded := r.loaded
	r.mu.RUnlock()
	if loaded {
		return
	}
	r.refresh()
}

func (r *commitIdentityResolver) refresh() {
	if r == nil || r.services == nil || r.services.Manager == nil {
		return
	}
	identities, err := r.services.Manager.GetDeviceIdentities()
	if err != nil {
		log.Debugf("failed to refresh commit identity cache: %s", err.Error())
		return
	}
	byPublicKey := make(map[string]string, len(identities)*2)
	for _, identity := range identities {
		if identity.UserDisabled {
			continue
		}
		label := deviceIdentityLabel(identity)
		if label == "" {
			continue
		}
		for _, key := range publicKeyLookupKeys(identity.PublicKey) {
			byPublicKey[key] = label
		}
	}
	r.mu.Lock()
	r.byPublicKey = byPublicKey
	r.loaded = true
	r.mu.Unlock()
}

func deviceIdentityLabel(identity user.DeviceIdentity) string {
	username := strings.TrimSpace(identity.Username)
	if username == "" {
		username = strings.TrimSpace(identity.UserName)
	}
	if username == "" {
		username = strings.TrimSpace(identity.UserID)
	}
	if username == "" {
		return ""
	}
	deviceName := strings.TrimSpace(identity.DeviceName)
	if deviceName == "" {
		return username
	}
	return fmt.Sprintf("%s (%s)", username, deviceName)
}

func publicKeyLookupKeys(publicKey string) []string {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return nil
	}

	keys := make([]string, 0, 3)
	seen := map[string]struct{}{}
	addPublicKeyLookupKey(&keys, seen, publicKey)

	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return keys
	}
	if pub, err := libp2pcrypto.UnmarshalPublicKey(publicKeyBytes); err == nil {
		addLibp2pPublicKeyAliases(&keys, seen, pub)
	}
	if pub, err := libp2pcrypto.UnmarshalEd25519PublicKey(publicKeyBytes); err == nil {
		addLibp2pPublicKeyAliases(&keys, seen, pub)
	}

	return keys
}

func addLibp2pPublicKeyAliases(keys *[]string, seen map[string]struct{}, pub libp2pcrypto.PubKey) {
	if pub == nil {
		return
	}
	if raw, err := pub.Raw(); err == nil {
		addPublicKeyLookupKey(keys, seen, base64.StdEncoding.EncodeToString(raw))
	}
	if marshaled, err := libp2pcrypto.MarshalPublicKey(pub); err == nil {
		addPublicKeyLookupKey(keys, seen, base64.StdEncoding.EncodeToString(marshaled))
	}
}

func addPublicKeyLookupKey(keys *[]string, seen map[string]struct{}, key string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	if _, found := seen[key]; found {
		return
	}
	seen[key] = struct{}{}
	*keys = append(*keys, key)
}

func isSyntheticCommitter(committer string) bool {
	switch strings.TrimSpace(committer) {
	case "swarmion-checkpoint", "swarmion-sync":
		return true
	default:
		return false
	}
}

func shortPublicKey(publicKey string) string {
	publicKey = strings.TrimSpace(publicKey)
	if len(publicKey) <= 12 {
		return publicKey
	}
	return publicKey[:8] + "..." + publicKey[len(publicKey)-4:]
}
