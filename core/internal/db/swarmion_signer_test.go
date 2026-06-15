package db

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/martinlindhe/base36"
	"github.com/nustiueudinastea/swarmion/runtime/identity"
)

func TestSwarmionSignerUsesLibp2pPublicKeyEncoding(t *testing.T) {
	privateKey, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	key := testSwarmionRawSigner{privateKey: privateKey, publicKey: publicKey}
	signer, err := newSwarmionSigner(key, privateKey.GetPublic())
	if err != nil {
		t.Fatalf("create swarmion signer: %v", err)
	}

	peerID, err := identity.PeerIDFromPublicKey(signer.PublicKey())
	if err != nil {
		t.Fatalf("decode swarmion public key: %v", err)
	}
	if peerID != key.GetID() {
		t.Fatalf("swarmion public key peer id = %s, want %s", peerID, key.GetID())
	}

	signature, err := signer.Sign("checkpoint-event-root")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	var verifier identity.Identity
	if err := verifier.Verify("checkpoint-event-root", signature, signer.PublicKey()); err != nil {
		t.Fatalf("verify libp2p-encoded public key: %v", err)
	}
	if err := signer.Verify("checkpoint-event-root", signature, signer.PublicKey()); err != nil {
		t.Fatalf("verify through compatibility wrapper: %v", err)
	}
}

type testSwarmionRawSigner struct {
	privateKey libp2pcrypto.PrivKey
	publicKey  libp2pcrypto.PubKey
}

func (s testSwarmionRawSigner) Sign(payload string) (string, error) {
	signature, err := s.privateKey.Sign([]byte(payload))
	if err != nil {
		return "", err
	}
	return base36.EncodeBytes(signature), nil
}

func (s testSwarmionRawSigner) Verify(payload string, signature string, publicKey string) error {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return err
	}
	pub, err := libp2pcrypto.UnmarshalEd25519PublicKey(publicKeyBytes)
	if err != nil {
		return err
	}
	ok, err := pub.Verify([]byte(payload), base36.DecodeToBytes(signature))
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func (s testSwarmionRawSigner) PublicKey() string {
	raw, err := s.publicKey.Raw()
	if err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func (s testSwarmionRawSigner) GetID() string {
	peerID, err := peer.IDFromPrivateKey(s.privateKey)
	if err != nil {
		panic(err)
	}
	return peerID.String()
}

func (s testSwarmionRawSigner) Private() []byte {
	raw, err := s.privateKey.Raw()
	if err != nil {
		panic(err)
	}
	return raw
}
