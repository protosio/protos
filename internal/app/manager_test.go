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

	got, err = manager.Get("app")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "app-id" {
		t.Fatalf("name lookup ID = %q, want app-id", got.ID)
	}
}

func TestGetStatusForHydratedAppUsesManagerRuntime(t *testing.T) {
	store := newTestAppDB(t)
	insertTestApp(t, store, "app-id", "app", "vm-id", statusStopped)
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store)

	status, err := manager.GetStatus("app")
	if err != nil {
		t.Fatal(err)
	}
	if status != statusStopped {
		t.Fatalf("status = %q, want %q", status, statusStopped)
	}
}

func TestCreateAssignsAppPublicKeyAndOverlayIP(t *testing.T) {
	store := newTestAppDB(t)
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store)

	created, err := manager.Create("docker.io/library/busybox:latest", "app", "vm-id", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created.PublicKey == "" {
		t.Fatal("PublicKey is empty")
	}
	if created.IP == nil {
		t.Fatal("IP is nil")
	}

	got, err := manager.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PublicKey != created.PublicKey {
		t.Fatalf("PublicKey = %q, want %q", got.PublicKey, created.PublicKey)
	}
	if !got.IP.Equal(created.IP) {
		t.Fatalf("IP = %s, want %s", got.IP, created.IP)
	}
}

func TestStartPullsMissingImage(t *testing.T) {
	store := newTestAppDB(t)
	runtime := &fakeRuntimePlatform{imageExists: false}
	manager := CreateManager("local-node", runtime, store)

	created, err := manager.Create("docker.io/library/busybox:latest", "app", "local-node", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(created.Name); err != nil {
		t.Fatal(err)
	}
	manager.Notify()
	if runtime.pullImageCalls != 1 {
		t.Fatalf("pullImageCalls = %d, want 1", runtime.pullImageCalls)
	}
	if runtime.newSandboxCalls != 1 {
		t.Fatalf("newSandboxCalls = %d, want 1", runtime.newSandboxCalls)
	}
}

func TestNotifyRepairsAlreadyRunningLocalApp(t *testing.T) {
	store := newTestAppDB(t)
	key, err := pcrypto.CreateManager(store).GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	app := App{
		ID:            "app-id",
		Name:          "app",
		InstallerRef:  "installer",
		InstanceID:    "local-node",
		DesiredStatus: statusRunning,
		IP:            appIPFromPublicKey(key.PublicString()),
		PublicKey:     key.PublicString(),
	}
	if err := db.Insert(store, createAppInsertMapper(app)); err != nil {
		t.Fatal(err)
	}

	sandbox := &fakeRuntimeSandbox{id: "app-id", status: statusRunning}
	runtime := &fakeRuntimePlatform{
		sandboxes: map[string]*fakeRuntimeSandbox{"app-id": sandbox},
	}
	manager := CreateManager("local-node", runtime, store)

	manager.Notify()

	if runtime.newSandboxCalls != 0 {
		t.Fatalf("newSandboxCalls = %d, want 0", runtime.newSandboxCalls)
	}
	if sandbox.startCalls != 1 {
		t.Fatalf("startCalls = %d, want 1", sandbox.startCalls)
	}
	if sandbox.status != statusRunning {
		t.Fatalf("status = %q, want %q", sandbox.status, statusRunning)
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
	imageExists     bool
	pullImageCalls  int
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
	return f.imageExists, nil
}

func (f *fakeRuntimePlatform) GetAllImages() (map[string]appruntime.PlatformImage, error) {
	return nil, nil
}

func (f *fakeRuntimePlatform) PullImage(string) error {
	f.pullImageCalls++
	f.imageExists = true
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
	id         string
	status     string
	removed    bool
	startCalls int
}

func (f *fakeRuntimeSandbox) Start(net.IP) error {
	f.startCalls++
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
