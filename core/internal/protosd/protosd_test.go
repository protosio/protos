package protosd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
)

func TestParseCapabilitiesExplicitList(t *testing.T) {
	caps, err := ParseCapabilities("api,network,provisioner")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || !caps.Network || !caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesCanDisableDefaults(t *testing.T) {
	caps, err := ParseCapabilities("default,no-network,no-provisioner,no-app-runtime")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || caps.Network || caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesNoneResets(t *testing.T) {
	caps, err := ParseCapabilities("all,none,api")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || caps.Network || caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesReadsEnvWhenUnset(t *testing.T) {
	t.Setenv("PROTOS_CAPABILITIES", "api,provisioner")
	caps, err := ParseCapabilities("")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || !caps.Provision || caps.Network || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesRejectsUnknown(t *testing.T) {
	if _, err := ParseCapabilities("api,unknown"); err == nil {
		t.Fatal("expected unknown capability to fail")
	}
}

func TestActiveMembershipPeerIDsExcludeStoppedInstances(t *testing.T) {
	runningKey := testPublicKey(t, "running")
	stoppedKey := testPublicKey(t, "stopped")
	deviceKey := testPublicKey(t, "device")
	runningPeerID := testPeerID(t, runningKey)
	stoppedPeerID := testPeerID(t, stoppedKey)
	devicePeerID := testPeerID(t, deviceKey)

	peerIDs := activeMembershipPeerIDs([]provisioners.InstanceInfo{
		{Name: "running", PublicKey: runningKey, DesiredStatus: provisioners.ServerStateRunning},
		{Name: "stopped", PublicKey: stoppedKey, DesiredStatus: provisioners.ServerStateStopped},
	}, []user.UserDevice{
		{ID: "device-id", Name: "laptop", PublicKey: deviceKey},
	})

	if _, found := peerIDs[runningPeerID]; !found {
		t.Fatalf("running peer %s missing from active set %#v", runningPeerID, peerIDs)
	}
	if _, found := peerIDs[devicePeerID]; !found {
		t.Fatalf("device peer %s missing from active set %#v", devicePeerID, peerIDs)
	}
	if _, found := peerIDs[stoppedPeerID]; found {
		t.Fatalf("stopped peer %s included in active set %#v", stoppedPeerID, peerIDs)
	}
	if len(peerIDs) != 2 {
		t.Fatalf("active peer IDs = %#v, want running instance and device only", peerIDs)
	}
}

func testPublicKey(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	key, err := pcrypto.GetLocalKey(dir)
	if err != nil {
		t.Fatal(err)
	}
	return key.PublicString()
}

func testPeerID(t *testing.T, publicKey string) string {
	t.Helper()
	peerID, err := db.PeerIDFromPublicKeyString(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return peerID
}
