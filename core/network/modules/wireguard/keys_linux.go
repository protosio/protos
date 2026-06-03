//go:build linux

package wireguard

import "golang.zx2c4.com/wireguard/wgctrl/wgtypes"

func publicEd25519ToWireGuard(publicKey string) (wgtypes.Key, error) {
	keyBytes, err := publicEd25519ToWireGuardBytes(publicKey)
	if err != nil {
		return wgtypes.Key{}, err
	}

	var key wgtypes.Key
	copy(key[:], keyBytes[:])
	return key, nil
}

func privateWireGuardKey(privateKey string) (wgtypes.Key, error) {
	keyBytes, err := privateWireGuardKeyBytes(privateKey)
	if err != nil {
		return wgtypes.Key{}, err
	}

	var key wgtypes.Key
	copy(key[:], keyBytes[:])
	return key, nil
}
