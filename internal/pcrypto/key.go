package pcrypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"

	"filippo.io/edwards25519"
	"github.com/bokwoon95/sq"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/martinlindhe/base36"
	"github.com/mikesmitty/edkey"
	"github.com/protosio/protos/internal/db"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	privateKeyFileName = "protos.key"
)

func createKeyQueryMapper(s db.SSH_KEY, predicates []sq.Predicate) func() (sq.Table, func(row *sq.Row) Key, []sq.Predicate) {
	return func() (sq.Table, func(row *sq.Row) Key, []sq.Predicate) {
		mapper := func(row *sq.Row) Key {
			priv, err := base64.StdEncoding.DecodeString(row.StringField(s.PRIVATE))
			if err != nil {
				log.Errorf("failed to decode private key: %v", err)
			}
			pub, err := base64.StdEncoding.DecodeString(row.StringField(s.PUBLIC))
			if err != nil {
				log.Errorf("failed to decode public key: %v", err)
			}
			return Key{
				Priv: priv,
				Pub:  pub,
			}
		}
		return s, mapper, predicates
	}
}

// Key is an SSH key
type Key struct {
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
}

func (k Key) Public() []byte {
	return k.Pub
}

func (k Key) PublicString() string {
	return base64.StdEncoding.EncodeToString(k.Pub)
}

func (k Key) Private() []byte {
	return k.Priv
}

func (k Key) PrivateWG() wgtypes.Key {
	var wgkey wgtypes.Key

	h := sha512.New()
	h.Write(k.Seed())
	out := h.Sum(nil)

	// Magic from the high priests of Crypto (libsodium)
	out[0] &= 248
	out[31] &= 127
	out[31] |= 64
	copy(wgkey[:], out[:curve25519.ScalarSize])

	return wgkey

}

func (k Key) PublicWG() wgtypes.Key {
	return k.PrivateWG().PublicKey()
}

func (k Key) Seed() []byte {
	return k.Priv[:32]
}

func (k Key) EncodePrivateKeytoPEM() string {
	// Get ASN.1 DER format
	privDER := edkey.MarshalED25519PrivateKey(k.Priv)

	// pem.Block
	privBlock := pem.Block{
		Type:    "OPENSSH PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return string(privatePEM)
}

// SSHAuth returns an ssh.AuthMethod that can be used to configure an ssh client
func (k Key) SSHAuth() ssh.AuthMethod {
	signer, _ := ssh.NewSignerFromKey(k.Priv)
	return ssh.PublicKeys(signer)
}

// AuthorizedKey return the public key in a format that can be written directly to the ~/.ssh/authorized_keys file
func (k Key) AuthorizedKey() string {
	publicKey, _ := ssh.NewPublicKey(k.Pub)
	return string(ssh.MarshalAuthorizedKey(publicKey))
}

func (k Key) Sign(commit string) (string, error) {
	prvKey, err := crypto.UnmarshalEd25519PrivateKey(k.Private())
	if err != nil {
		return "", err
	}

	sig, err := prvKey.Sign([]byte(commit))
	if err != nil {
		return "", fmt.Errorf("failed to create signature: %w", err)
	}

	return base36.EncodeBytes(sig), nil
}

func (k Key) Verify(commit string, signature string, publicKey string) error {
	// Decode the base64-encoded public key string to bytes
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Unmarshal the public key bytes into a public key object
	pubKey, err := crypto.UnmarshalPublicKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	// Decode the base64-encoded signature string to bytes
	signatureBytes := base36.DecodeToBytes(signature)

	// Verify the signature using the public key
	verified, err := pubKey.Verify([]byte(commit), signatureBytes)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !verified {
		return fmt.Errorf("verification failed for public key %s commit %s signature %s ", publicKey, commit, signature)
	}

	return nil
}

func (k Key) PublicKey() string {
	return k.PublicString()
}

func (k Key) GetID() string {
	prvKey, err := crypto.UnmarshalEd25519PrivateKey(k.Private())
	if err != nil {
		panic(err)
	}

	peerID, err := peer.IDFromPrivateKey(prvKey)
	if err != nil {
		panic(err)
	}

	return peerID.String()
}

//
// Module functions
//

func GetLocalKey(workdir string) (*Key, error) {
	key := &Key{}
	keyFilePath := workdir + "/" + privateKeyFileName

	// Check if the key file exists
	if _, err := os.Stat(keyFilePath); err == nil {
		// Key file exists, read the file
		keyData, err := os.ReadFile(keyFilePath)
		if err != nil {
			return nil, err
		}

		// Decode PEM block
		block, _ := pem.Decode(keyData)
		if block == nil || block.Type != "PRIVATE KEY" {
			return nil, err
		}

		// Convert PEM block to ed25519.PrivateKey
		key.Priv = ed25519.PrivateKey(block.Bytes)
		key.Pub = key.Priv.Public().(ed25519.PublicKey)
	} else {
		// Key file does not exist, generate a new key
		publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		key.Priv = privateKey
		key.Pub = publicKey

		// Convert privateKey to PEM block
		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKey,
		}
		pemData := pem.EncodeToMemory(block)

		// Write the PEM block to file
		err = os.WriteFile(keyFilePath, pemData, 0600)
		if err != nil {
			return nil, err
		}
	}

	return key, nil
}

func ConvertPublicEd25519ToCurve25519(ed25519Key string) (wgtypes.Key, error) {

	pubKey, err := base64.StdEncoding.DecodeString(ed25519Key)
	if err != nil {
		return wgtypes.Key{}, fmt.Errorf("failed to decode base64 public key: %w", err)
	}

	var pubkey wgtypes.Key
	edPoint, err := new(edwards25519.Point).SetBytes(pubKey)
	if err != nil {
		return pubkey, fmt.Errorf("failed to convert public Ed25519 key to WG public key: %w", err)
	}

	copy(pubkey[:], edPoint.BytesMontgomery())
	return pubkey, nil
}
