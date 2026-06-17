package db

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/bokwoon95/sq"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	runtimeInactiveMachineStatusDeleting = "deleting"
	runtimeInactiveMachineStatusStopped  = "stopped"
)

type machinePeerStatus struct {
	publicKey     string
	desiredStatus string
}

func CreatePeerInsertMapper(publicKey string) InsertMapper {
	return func() sq.InsertQuery {
		p := sq.New[PEER]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(p.ID, MustUUIDBytes(MustNewUUIDv7()))
			col.SetString(p.PUBLIC_KEY, publicKey)
		}
		return sq.InsertInto(p).ColumnValues(mapper)
	}
}

func CreatePeerDeleteMapper(publicKey string) DeleteMapper {
	return func() sq.DeleteQuery {
		p := sq.New[PEER]("")
		return sq.DeleteFrom(p).Where(p.PUBLIC_KEY.EqString(publicKey))
	}
}

func GetPeerIDs(database *DB) (map[string]struct{}, error) {
	publicKeys, err := SelectMultiple(database, createPeerQueryAllMapper())
	if err != nil {
		return nil, err
	}

	peers := make(map[string]struct{}, len(publicKeys))
	for _, publicKey := range publicKeys {
		peerID, err := PeerIDFromPublicKeyString(publicKey)
		if err != nil {
			return nil, fmt.Errorf("derive peer id from peer public key: %w", err)
		}
		peers[peerID] = struct{}{}
	}
	return peers, nil
}

func GetActiveRuntimePeerIDs(database *DB) (map[string]struct{}, error) {
	peers, err := GetPeerIDs(database)
	if err != nil {
		return nil, err
	}

	machines, err := SelectMultiple(database, createMachinePeerStatusQueryAllMapper())
	if err != nil {
		return nil, err
	}
	for _, machine := range machines {
		if !isInactiveRuntimeMachineStatus(machine.desiredStatus) {
			continue
		}
		if strings.TrimSpace(machine.publicKey) == "" {
			continue
		}
		peerID, err := PeerIDFromPublicKeyString(machine.publicKey)
		if err != nil {
			return nil, fmt.Errorf("derive peer id from inactive machine public key: %w", err)
		}
		delete(peers, peerID)
	}
	return peers, nil
}

func createPeerQueryAllMapper() QueryMapper[string] {
	p := sq.New[PEER]("")
	query := sq.From(p)

	return func() (sq.SelectQuery, func(row *sq.Row) string) {
		mapper := func(row *sq.Row) string {
			return row.StringField(p.PUBLIC_KEY)
		}
		return query, mapper
	}
}

func createMachinePeerStatusQueryAllMapper() QueryMapper[machinePeerStatus] {
	m := sq.New[MACHINE]("")
	cmm := sq.New[CLOUD_MACHINE_METADATA]("")
	query := sq.From(m).Join(cmm, cmm.ID.Eq(m.ID))

	return func() (sq.SelectQuery, func(row *sq.Row) machinePeerStatus) {
		mapper := func(row *sq.Row) machinePeerStatus {
			return machinePeerStatus{
				publicKey:     row.StringField(cmm.PUBLIC_KEY),
				desiredStatus: row.StringField(m.DESIRED_STATUS),
			}
		}
		return query, mapper
	}
}

func isInactiveRuntimeMachineStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case runtimeInactiveMachineStatusDeleting, runtimeInactiveMachineStatusStopped:
		return true
	default:
		return false
	}
}

func PeerIDFromPublicKeyString(publicKey string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return "", fmt.Errorf("invalid ed25519 public key length: got %d bytes, want %d", len(publicKeyBytes), ed25519.PublicKeySize)
	}
	pubKey, err := libp2pcrypto.UnmarshalEd25519PublicKey(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("decode libp2p public key: %w", err)
	}
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("derive peer id from public key: %w", err)
	}
	return peerID.String(), nil
}
