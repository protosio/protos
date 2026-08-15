package protosd

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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

func TestRejectFreshInitializationForRepository(t *testing.T) {
	permanentErr := errors.New("permanent bootstrap admission failure")
	tests := []struct {
		name      string
		readiness db.RepositoryReadiness
		wantText  string
		wantCause error
	}{
		{
			name:      "existing bootstrap pending",
			readiness: db.RepositoryReadiness{ExistingRepository: true, BootstrapPending: true},
			wantText:  "awaiting bootstrap recovery",
		},
		{
			name:      "existing permanent bootstrap error",
			readiness: db.RepositoryReadiness{ExistingRepository: true, BootstrapError: permanentErr},
			wantText:  "recovery failed",
			wantCause: permanentErr,
		},
		{
			name:      "existing repository worker stopped",
			readiness: db.RepositoryReadiness{ExistingRepository: true},
			wantText:  "not ready",
		},
		{name: "genuinely fresh repository"},
		{
			name:      "recovered existing repository",
			readiness: db.RepositoryReadiness{Initialized: true, ExistingRepository: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := rejectFreshInitializationForRepository(test.readiness)
			if test.wantText == "" {
				if err != nil {
					t.Fatalf("guard error = %v, want admission", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantText) {
				t.Fatalf("guard error = %v, want text %q", err, test.wantText)
			}
			if test.wantCause != nil && !errors.Is(err, test.wantCause) {
				t.Fatalf("guard error = %v, want cause %v", err, test.wantCause)
			}
		})
	}
}

func TestStopNodeComponentsOrdersTaskRunnerBeforeSwarmionAndOtherServices(t *testing.T) {
	var calls []string
	stopper := func(name string) func() error {
		return func() error {
			calls = append(calls, name)
			return nil
		}
	}

	stopNodeComponents(map[string]func() error{
		"p2p-server":  stopper("p2p-server"),
		"p2p-scope":   stopper("p2p-scope"),
		"p2p-host":    stopper("p2p-host"),
		"task-runner": stopper("task-runner"),
		"api":         stopper("api"),
		"network":     stopper("network"),
	}, func() {
		calls = append(calls, "prepare-swarmion")
	})

	want := []string{"task-runner", "api", "network", "p2p-server", "prepare-swarmion", "p2p-scope", "p2p-host"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("shutdown order = %v, want %v", calls, want)
	}
}

func TestNodeStopInvokesEachStopperOnce(t *testing.T) {
	var calls []string
	node := &Node{stoppers: map[string]func() error{
		"task-runner": func() error {
			calls = append(calls, "task-runner")
			return nil
		},
		"api": func() error {
			calls = append(calls, "api")
			return nil
		},
	}}

	node.Stop()
	node.Stop()

	want := []string{"task-runner", "api"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("stopper calls = %v, want %v", calls, want)
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

func TestDBNotifierDoesNotPreClearOrFinalizeRemovedSwarmionPeers(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source path")
	}
	productionFile := filepath.Join(filepath.Dir(testFile), "protosd.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), productionFile, nil, 0)
	if err != nil {
		t.Fatalf("parse protosd.go: %v", err)
	}
	forbidden := map[string]struct{}{
		"ClearPeerCaches":               {},
		"EvictPeer":                     {},
		"ReconcileRemovedSwarmionPeers": {},
		"BeginReplicationPeerDrain":     {},
		"WatchReplicationPeerDrain":     {},
		"WaitReplicationPeerDrainReady": {},
		"FinalizeReplicationPeerDrain":  {},
		"StartReplicationPeerDrain":     {},
	}
	foundNotify := false
	ast.Inspect(parsed, func(node ast.Node) bool {
		declaration, ok := node.(*ast.FuncDecl)
		if !ok || declaration.Name.Name != "Notify" || declaration.Recv == nil || len(declaration.Recv.List) != 1 {
			return true
		}
		star, ok := declaration.Recv.List[0].Type.(*ast.StarExpr)
		if !ok {
			return true
		}
		receiver, ok := star.X.(*ast.Ident)
		if !ok || receiver.Name != "DBNotifier" {
			return true
		}
		foundNotify = true
		ast.Inspect(declaration.Body, func(bodyNode ast.Node) bool {
			selector, ok := bodyNode.(*ast.SelectorExpr)
			if ok {
				if _, forbiddenCall := forbidden[selector.Sel.Name]; forbiddenCall {
					t.Errorf("DBNotifier.Notify must not perform peer drain/cache completion: found %s", selector.Sel.Name)
				}
			}
			return true
		})
		return false
	})
	if !foundNotify {
		t.Fatal("DBNotifier.Notify declaration not found")
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
