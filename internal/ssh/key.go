package ssh

import (
	"crypto/ed25519"
	"encoding/pem"

	"github.com/mikesmitty/edkey"
	"golang.org/x/crypto/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// Key is an SSH key
type Key struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
}

func (k Key) Public() []byte {
	return k.public
}

func (k Key) PublicWG() wgtypes.Key {
	var wgkey wgtypes.Key
	copy(wgkey[:], k.Seed())
	return wgkey.PublicKey()
}

func (k Key) Seed() []byte {
	return k.private[:32]
}

func (k Key) EncodePrivateKeytoPEM() string {
	// Get ASN.1 DER format
	privDER := edkey.MarshalED25519PrivateKey(k.private)

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
	signer, _ := ssh.NewSignerFromKey(k.private)
	return ssh.PublicKeys(signer)
}

// AuthorizedKey return the public key in a format that can be written directly to the ~/.ssh/authorized_keys file
func (k Key) AuthorizedKey() string {
	publicKey, _ := ssh.NewPublicKey(k.public)
	return string(ssh.MarshalAuthorizedKey(publicKey))
}
