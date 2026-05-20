package app

import (
	"net"
	"testing"

	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	appruntime "github.com/protosio/protos/internal/runtime"
)

func TestNotifyDoesNotRemoveSandboxForAppOutsideLocalScope(t *testing.T) {
	store := newTestAppDB(t)
	insertTestApp(t, store, "local-app", "local-app", "local-node", statusStopped)
	insertTestApp(t, store, "remote-app", "remote-app", "remote-instance", statusRunning)

	remoteSandbox := &fakeRuntimeSandbox{id: "remote-app", status: statusRunning}
	runtime := &fakeRuntimePlatform{
		sandboxes: map[string]*fakeRuntimeSandbox{
			"remote-app": remoteSandbox,
		},
	}
	manager := CreateManager("local-node", runtime, store)

	manager.Notify()

	if remoteSandbox.removed {
		t.Fatal("remote app sandbox was removed even though its declarative row still exists")
	}
	if runtime.newSandboxCalls != 0 {
		t.Fatalf("expected out-of-scope app not to be started locally, got %d starts", runtime.newSandboxCalls)
	}
}

func TestGetReturnsAppWithInstanceID(t *testing.T) {
	store := newTestAppDB(t)
	insertTestApp(t, store, "app-id", "app", "vm-id", statusStopped)
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store)

	got, err := manager.Get("app-id")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "app-id" {
		t.Fatalf("ID = %q, want app-id", got.ID)
	}
	if got.InstanceID != "vm-id" {
		t.Fatalf("InstanceID = %q, want vm-id", got.InstanceID)
	}
}

func newTestAppDB(t *testing.T) *db.DB {
	t.Helper()
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.P2PPort = previousP2PPort
	})

	workDir := t.TempDir()
	key, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}
	store, err := db.Open(workDir, "protos_test", key)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}
	return store
}

func insertTestApp(t *testing.T, store *db.DB, id string, name string, instanceID string, desiredStatus string) {
	t.Helper()
	if err := db.Insert(store, createAppInsertMapper(App{
		ID:            id,
		Name:          name,
		InstallerRef:  "installer",
		InstanceID:    instanceID,
		DesiredStatus: desiredStatus,
		Persistence:   false,
	})); err != nil {
		t.Fatalf("insert app %s: %v", id, err)
	}
}

type fakeRuntimePlatform struct {
	sandboxes       map[string]*fakeRuntimeSandbox
	newSandboxCalls int
}

func (f *fakeRuntimePlatform) Init() error {
	return nil
}

func (f *fakeRuntimePlatform) GetSandbox(id string) (appruntime.RuntimeSandbox, error) {
	if f.sandboxes != nil {
		if sandbox, found := f.sandboxes[id]; found {
			return sandbox, nil
		}
	}
	return nil, appruntime.ErrSandboxNotFound
}

func (f *fakeRuntimePlatform) GetAllSandboxes() (map[string]appruntime.RuntimeSandbox, error) {
	result := map[string]appruntime.RuntimeSandbox{}
	for id, sandbox := range f.sandboxes {
		result[id] = sandbox
	}
	return result, nil
}

func (f *fakeRuntimePlatform) GetImage(string) (appruntime.PlatformImage, error) {
	return nil, nil
}

func (f *fakeRuntimePlatform) ImageExistsLocally(string) (bool, error) {
	return true, nil
}

func (f *fakeRuntimePlatform) GetAllImages() (map[string]appruntime.PlatformImage, error) {
	return nil, nil
}

func (f *fakeRuntimePlatform) PullImage(string) error {
	return nil
}

func (f *fakeRuntimePlatform) RemoveImage(string) error {
	return nil
}

func (f *fakeRuntimePlatform) NewSandbox(name string, appID string, imageID string, persistence bool) (appruntime.RuntimeSandbox, error) {
	f.newSandboxCalls++
	sandbox := &fakeRuntimeSandbox{id: appID, status: statusStopped}
	if f.sandboxes == nil {
		f.sandboxes = map[string]*fakeRuntimeSandbox{}
	}
	f.sandboxes[appID] = sandbox
	return sandbox, nil
}

func (f *fakeRuntimePlatform) GetHWStats() (appruntime.HardwareStats, error) {
	return appruntime.HardwareStats{}, nil
}

type fakeRuntimeSandbox struct {
	id      string
	status  string
	removed bool
}

func (f *fakeRuntimeSandbox) Start(net.IP) error {
	f.status = statusRunning
	return nil
}

func (f *fakeRuntimeSandbox) Stop() error {
	f.status = statusStopped
	return nil
}

func (f *fakeRuntimeSandbox) Update() error {
	return nil
}

func (f *fakeRuntimeSandbox) Remove() error {
	f.removed = true
	return nil
}

func (f *fakeRuntimeSandbox) GetID() string {
	return f.id
}

func (f *fakeRuntimeSandbox) GetStatus() string {
	return f.status
}

func (f *fakeRuntimeSandbox) GetLogs() ([]byte, error) {
	return nil, nil
}

func (f *fakeRuntimeSandbox) GetExitCode() int {
	return 0
}
