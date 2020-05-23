package core

import (
	"golang.org/x/crypto/ssh"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type SSHManager interface {
	GenerateKey() (Key, error)
	GetKeyByPub(pubKey string) (Key, error)
	NewKeyFromSeed(seed []byte) (Key, error)
	NewAuthFromKeyFile(keyPath string) (ssh.AuthMethod, error)
}

type Key interface {
	Public() []byte
	PublicWG() wgtypes.Key
	Seed() []byte
	EncodePrivateKeytoPEM() string
	SSHAuth() ssh.AuthMethod
	AuthorizedKey() string
	Save()
}
