package wireguard

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"net/netip"

	"filippo.io/edwards25519"
)

const wireGuardKeyLen = 32

func ipv6AddressFromPublicKeyBase64(publicKey string) (netip.Addr, error) {
	pubKey, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed to decode base64 public key: %w", err)
	}
	if len(pubKey) != ed25519.PublicKeySize {
		return netip.Addr{}, fmt.Errorf("invalid public key size")
	}

	var buf [ed25519.PublicKeySize]byte
	copy(buf[:], pubKey)
	for idx := range buf {
		buf[idx] = ^buf[idx]
	}

	var addr [16]byte
	temp := make([]byte, 0, 32)
	done := false
	ones := byte(0)
	bits := byte(0)
	nBits := 0
	for idx := 0; idx < 8*len(buf); idx++ {
		bit := (buf[idx/8] & (0x80 >> byte(idx%8))) >> byte(7-(idx%8))
		if !done && bit != 0 {
			ones++
			continue
		}
		if !done && bit == 0 {
			done = true
			continue
		}
		bits = (bits << 1) | bit
		nBits++
		if nBits == 8 {
			nBits = 0
			temp = append(temp, bits)
		}
	}

	prefix := [...]byte{0x02}
	copy(addr[:], prefix[:])
	addr[len(prefix)] = ones
	copy(addr[len(prefix)+1:], temp)

	return netip.AddrFrom16(addr), nil
}

func publicEd25519ToWireGuardBytes(publicKey string) ([wireGuardKeyLen]byte, error) {
	pubKey, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return [wireGuardKeyLen]byte{}, fmt.Errorf("failed to decode base64 public key: %w", err)
	}

	var wgKey [wireGuardKeyLen]byte
	edPoint, err := new(edwards25519.Point).SetBytes(pubKey)
	if err != nil {
		return wgKey, fmt.Errorf("failed to convert public Ed25519 key to WG public key: %w", err)
	}

	copy(wgKey[:], edPoint.BytesMontgomery())
	return wgKey, nil
}

func privateWireGuardKeyBytes(privateKey string) ([wireGuardKeyLen]byte, error) {
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return [wireGuardKeyLen]byte{}, fmt.Errorf("failed to decode WireGuard private key: %w", err)
	}
	if len(decodedPrivateKey) != wireGuardKeyLen {
		return [wireGuardKeyLen]byte{}, fmt.Errorf("invalid WireGuard private key size")
	}

	var wgKey [wireGuardKeyLen]byte
	copy(wgKey[:], decodedPrivateKey)
	return wgKey, nil
}
