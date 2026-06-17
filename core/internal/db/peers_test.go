package db

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"github.com/bokwoon95/sq"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/protosio/protos/internal/config"
)

func TestGetActiveRuntimePeerIDsExcludesInactiveMachines(t *testing.T) {
	store := openPeerTestDB(t)
	activeMachineKey := testPeerPublicKey(t)
	stoppedMachineKey := testPeerPublicKey(t)
	deletingMachineKey := testPeerPublicKey(t)
	deviceKey := testPeerPublicKey(t)

	activeMachine, activeMachineMetadata := testMachinePeerInsertMappers("active", "running", activeMachineKey)
	stoppedMachine, stoppedMachineMetadata := testMachinePeerInsertMappers("stopped", "stopped", stoppedMachineKey)
	deletingMachine, deletingMachineMetadata := testMachinePeerInsertMappers("deleting", "deleting", deletingMachineKey)
	if err := Insert(
		store,
		CreatePeerInsertMapper(activeMachineKey),
		CreatePeerInsertMapper(stoppedMachineKey),
		CreatePeerInsertMapper(deletingMachineKey),
		CreatePeerInsertMapper(deviceKey),
		activeMachine,
		activeMachineMetadata,
		stoppedMachine,
		stoppedMachineMetadata,
		deletingMachine,
		deletingMachineMetadata,
	); err != nil {
		t.Fatal(err)
	}

	got, err := GetActiveRuntimePeerIDs(store)
	if err != nil {
		t.Fatal(err)
	}
	activePeerID := mustPeerIDFromPublicKeyString(t, activeMachineKey)
	stoppedPeerID := mustPeerIDFromPublicKeyString(t, stoppedMachineKey)
	deletingPeerID := mustPeerIDFromPublicKeyString(t, deletingMachineKey)
	devicePeerID := mustPeerIDFromPublicKeyString(t, deviceKey)
	if _, found := got[activePeerID]; !found {
		t.Fatalf("active machine peer %s missing from %#v", activePeerID, got)
	}
	if _, found := got[devicePeerID]; !found {
		t.Fatalf("user device peer %s missing from %#v", devicePeerID, got)
	}
	if _, found := got[stoppedPeerID]; found {
		t.Fatalf("stopped machine peer %s remained in %#v", stoppedPeerID, got)
	}
	if _, found := got[deletingPeerID]; found {
		t.Fatalf("deleting machine peer %s remained in %#v", deletingPeerID, got)
	}
}

func openPeerTestDB(t *testing.T) *DB {
	t.Helper()
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	privateKey, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	store, err := Open(workDir, "protos_peer_test", testSwarmionRawSigner{privateKey: privateKey, publicKey: publicKey})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}
	return store
}

func testMachinePeerInsertMappers(name string, desiredStatus string, publicKey string) (InsertMapper, InsertMapper) {
	id := MustNewUUIDv7()
	machineMapper := func() sq.InsertQuery {
		m := sq.New[MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(m.ID, MustUUIDBytes(id))
			col.SetString(m.NAME, name)
			col.SetString(m.KIND, "test")
			col.SetString(m.DESIRED_STATUS, desiredStatus)
			col.SetInt(m.REPLICATION_PRIORITY, 10)
		}
		return sq.InsertInto(m).ColumnValues(mapper)
	}
	metadataMapper := func() sq.InsertQuery {
		cmm := sq.New[CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(cmm.ID, MustUUIDBytes(id))
			col.SetString(cmm.CLOUD_ID, "test-provider")
			col.SetString(cmm.PROVIDER_RESOURCE_ID, "provider-"+name)
			col.SetString(cmm.PUBLIC_IP, "192.0.2.10")
			col.SetString(cmm.LOCATION, "test")
			col.SetString(cmm.ARCHITECTURE, "arm64")
			col.SetString(cmm.PUBLIC_KEY, publicKey)
		}
		return sq.InsertInto(cmm).ColumnValues(mapper)
	}
	return machineMapper, metadataMapper
}

func testPeerPublicKey(t *testing.T) string {
	t.Helper()
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(publicKey)
}

func mustPeerIDFromPublicKeyString(t *testing.T, publicKey string) string {
	t.Helper()
	peerID, err := PeerIDFromPublicKeyString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return peerID
}
