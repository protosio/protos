package app

import (
	"context"
	"net"
	"testing"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	appruntime "github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/tasks"
)

func TestNotifyDoesNotRemoveSandboxForAppOutsideLocalScope(t *testing.T) {
	store := newTestAppDB(t)
	localInstanceID, localPeerID := insertTestMachine(t, store, "local-instance")
	remoteInstanceID, _ := insertTestMachine(t, store, "remote-instance")
	localAppID := db.MustNewUUIDv7()
	remoteAppID := db.MustNewUUIDv7()
	insertTestApp(t, store, localAppID, "local-app", localInstanceID, statusStopped)
	insertTestApp(t, store, remoteAppID, "remote-app", remoteInstanceID, statusRunning)

	remoteSandbox := &fakeRuntimeSandbox{id: remoteAppID, status: statusRunning}
	runtime := &fakeRuntimePlatform{
		sandboxes: map[string]*fakeRuntimeSandbox{
			remoteAppID: remoteSandbox,
		},
	}
	manager := CreateManager(localPeerID, runtime, store, tasks.NewManager(store))

	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}

	if remoteSandbox.removed {
		t.Fatal("remote app sandbox was removed even though its declarative row still exists")
	}
	if runtime.newSandboxCalls != 0 {
		t.Fatalf("expected out-of-scope app not to be started locally, got %d starts", runtime.newSandboxCalls)
	}
}

func TestNotifyNoopsWhenRuntimeUnavailable(t *testing.T) {
	store := newTestAppDB(t)
	taskManager := tasks.NewManager(store)
	manager := CreateManager("local-node", nil, store, taskManager)

	manager.Notify()
	if err := taskManager.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	records, _, err := taskManager.List(tasks.ListOptions{Stream: AppReconcileTaskStream, MaxResults: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Fatalf("task count = %d, want 0", len(records))
	}
}

func TestGetReturnsAppWithInstanceID(t *testing.T) {
	store := newTestAppDB(t)
	appID := db.MustNewUUIDv7()
	insertTestApp(t, store, appID, "app", "vm-id", statusStopped)
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store, tasks.NewManager(store))

	got, err := manager.Get(appID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != appID {
		t.Fatalf("ID = %q, want %s", got.ID, appID)
	}
	if got.InstanceID != "vm-id" {
		t.Fatalf("InstanceID = %q, want vm-id", got.InstanceID)
	}

	got, err = manager.Get("app")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != appID {
		t.Fatalf("name lookup ID = %q, want %s", got.ID, appID)
	}
}

func TestGetStatusForHydratedAppUsesManagerRuntime(t *testing.T) {
	store := newTestAppDB(t)
	insertTestApp(t, store, db.MustNewUUIDv7(), "app", "vm-id", statusStopped)
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store, tasks.NewManager(store))

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
	manager := CreateManager("local-node", &fakeRuntimePlatform{}, store, tasks.NewManager(store))

	created, err := manager.Create(context.Background(), "docker.io/library/busybox:latest", "app", "vm-id", false, nil)
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
	localInstanceID, localPeerID := insertTestMachine(t, store, "local-instance")
	runtime := &fakeRuntimePlatform{imageExists: false}
	manager := CreateManager(localPeerID, runtime, store, tasks.NewManager(store))

	created, err := manager.Create(context.Background(), "docker.io/library/busybox:latest", "app", localInstanceID, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), created.Name); err != nil {
		t.Fatal(err)
	}
	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.pullImageCalls != 1 {
		t.Fatalf("pullImageCalls = %d, want 1", runtime.pullImageCalls)
	}
	if runtime.newSandboxCalls != 1 {
		t.Fatalf("newSandboxCalls = %d, want 1", runtime.newSandboxCalls)
	}
}

func TestReconcileDoesNotConsumeDesiredStateWrittenDuringTask(t *testing.T) {
	store := newTestAppDB(t)
	localInstanceID, localPeerID := insertTestMachine(t, store, "local-instance")
	runtime := &fakeRuntimePlatform{imageExists: true}
	manager := CreateManager(localPeerID, runtime, store, tasks.NewManager(store))

	created, err := manager.Create(context.Background(), "docker.io/library/busybox:latest", "app", localInstanceID, false, nil)
	if err != nil {
		t.Fatal(err)
	}

	updatedDuringReconcile := false
	runtime.getAllSandboxesHook = func() {
		if updatedDuringReconcile {
			return
		}
		updatedDuringReconcile = true
		if err := manager.Start(context.Background(), created.Name); err != nil {
			t.Fatalf("start app during reconcile: %v", err)
		}
	}

	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.newSandboxCalls != 0 {
		t.Fatalf("newSandboxCalls after stopped reconcile = %d, want 0", runtime.newSandboxCalls)
	}

	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.newSandboxCalls != 1 {
		t.Fatalf("newSandboxCalls after follow-up reconcile = %d, want 1", runtime.newSandboxCalls)
	}
}

func TestStartResolvesMissingImageFromPeersBeforePull(t *testing.T) {
	store := newTestAppDB(t)
	localInstanceID, localPeerID := insertTestMachine(t, store, "local-instance")
	runtime := &fakeRuntimePlatform{imageExists: false}
	manager := CreateManager(localPeerID, runtime, store, tasks.NewManager(store))
	manager.SetImageResolver(fakeImageResolver{
		resolve: func(context.Context, string) error {
			runtime.imageExists = true
			return nil
		},
	})

	created, err := manager.Create(context.Background(), "docker.io/library/busybox:latest", "app", localInstanceID, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background(), created.Name); err != nil {
		t.Fatal(err)
	}
	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}
	if runtime.pullImageCalls != 0 {
		t.Fatalf("pullImageCalls = %d, want 0", runtime.pullImageCalls)
	}
	if runtime.newSandboxCalls != 1 {
		t.Fatalf("newSandboxCalls = %d, want 1", runtime.newSandboxCalls)
	}
}

func TestNotifyRepairsAlreadyRunningLocalApp(t *testing.T) {
	store := newTestAppDB(t)
	localInstanceID, localPeerID := insertTestMachine(t, store, "local-instance")
	key, err := pcrypto.CreateManager(store).GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	app := App{
		ID:            db.MustNewUUIDv7(),
		Name:          "app",
		InstallerRef:  "installer",
		InstanceID:    localInstanceID,
		DesiredStatus: statusRunning,
		IP:            appIPFromPublicKey(key.PublicString()),
		PublicKey:     key.PublicString(),
	}
	if err := db.Insert(store, createAppInsertMapper(app)); err != nil {
		t.Fatal(err)
	}

	sandbox := &fakeRuntimeSandbox{id: app.ID, status: statusRunning}
	runtime := &fakeRuntimePlatform{
		sandboxes: map[string]*fakeRuntimeSandbox{app.ID: sandbox},
	}
	manager := CreateManager(localPeerID, runtime, store, tasks.NewManager(store))

	manager.Notify()
	if err := manager.tasks.RunPending(context.Background()); err != nil {
		t.Fatal(err)
	}

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

func insertTestMachine(t *testing.T, store *db.DB, name string) (string, string) {
	t.Helper()
	key, err := pcrypto.CreateManager(store).GenerateKey()
	if err != nil {
		t.Fatalf("generate machine key: %v", err)
	}
	peerID, err := db.PeerIDFromPublicKeyString(key.PublicString())
	if err != nil {
		t.Fatalf("derive peer id: %v", err)
	}
	instanceID := db.MustNewUUIDv7()
	machineInsert := func() sq.InsertQuery {
		m := sq.New[db.MACHINE]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(m.ID, db.MustUUIDBytes(instanceID))
			col.SetString(m.NAME, name)
			col.SetString(m.KIND, "cloud_vm")
			col.SetString(m.DESIRED_STATUS, "running")
			col.SetInt(m.REPLICATION_PRIORITY, 100)
		}
		return sq.InsertInto(m).ColumnValues(mapper)
	}
	metadataInsert := func() sq.InsertQuery {
		cmm := sq.New[db.CLOUD_MACHINE_METADATA]("")
		mapper := func(col *sq.Column) {
			col.SetBytes(cmm.ID, db.MustUUIDBytes(instanceID))
			col.SetString(cmm.CLOUD_ID, "test-provider")
			col.SetString(cmm.PROVIDER_RESOURCE_ID, name)
			col.SetString(cmm.PUBLIC_IP, "127.0.0.1")
			col.SetString(cmm.LOCATION, "test")
			col.SetString(cmm.ARCHITECTURE, "amd64")
			col.SetString(cmm.PUBLIC_KEY, key.PublicString())
		}
		return sq.InsertInto(cmm).ColumnValues(mapper)
	}
	if err := db.Insert(store, machineInsert, metadataInsert); err != nil {
		t.Fatalf("insert machine %s: %v", name, err)
	}
	return instanceID, peerID
}

type fakeRuntimePlatform struct {
	sandboxes           map[string]*fakeRuntimeSandbox
	newSandboxCalls     int
	imageExists         bool
	pullImageCalls      int
	getAllSandboxesHook func()
}

type fakeImageResolver struct {
	resolve func(context.Context, string) error
}

func (f fakeImageResolver) ResolveImage(ctx context.Context, imageRef string, _ func(int, string, any) error) error {
	return f.resolve(ctx, imageRef)
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
	if f.getAllSandboxesHook != nil {
		f.getAllSandboxesHook()
	}
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
