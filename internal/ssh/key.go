package ssh

import (
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/base64"
	"encoding/pem"

	"github.com/mikesmitty/edkey"
	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Key is an SSH key
type Key struct {
	parent *Manager `noms:"-"`

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
	var wgkey wgtypes.Key
	copy(wgkey[:], k.Seed())
	return wgkey.PublicKey()
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

// Save persists key to database
func (k Key) Save() {
	err := k.parent.db.InsertInMap(sshDS, k.PublicWG().String(), k)
	if err != nil {
		log.Panicf("Failed to save resource to db: %s", err.Error())
	}
}
