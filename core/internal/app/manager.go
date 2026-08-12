package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/tasks"

	"github.com/pkg/errors"
)

const (
	appDS       = "app"
	TypeProtosd = "protosd"

	AppReconcileTaskStream = "apps.runtime.reconcile"

	taskSubjectAppRuntime = "app_runtime"
)

// Manager keeps track of all the apps
type Manager struct {
	ptype         string
	db            *db.DB
	runtime       runtime.RuntimePlatform
	imageResolver ImageResolver
	tasks         *tasks.Manager
	notifyMu      sync.Mutex
	reconcileSig  string
}

type reconcileRuntimeTaskPayload struct {
	PeerID string `json:"peer_id"`
}

type reconcileRuntimeTaskResult struct {
	PeerID     string `json:"peer_id"`
	Reconciled int    `json:"reconciled"`
	Removed    int    `json:"removed"`
}

type ImageResolver interface {
	ResolveImage(ctx context.Context, imageRef string, progress func(int, string, any) error) error
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(ptype string, runtime runtime.RuntimePlatform, db *db.DB, taskManager *tasks.Manager) *Manager {

	manager := &Manager{ptype: ptype, db: db, runtime: runtime, tasks: taskManager}
	if taskManager != nil {
		if err := tasks.RegisterIfAbsent(taskManager, tasks.Stream[reconcileRuntimeTaskPayload, reconcileRuntimeTaskResult]{
			Name: AppReconcileTaskStream,
			Run:  manager.runReconcileRuntimeTask,
		}); err != nil {
			log.Errorf("failed to register app runtime task stream: %s", err.Error())
		}
	}

	return manager
}

func (am *Manager) SetImageResolver(resolver ImageResolver) {
	if am == nil {
		return
	}
	am.imageResolver = resolver
}

func (am *Manager) bind(app App) App {
	app.mgr = am
	if app.access == nil {
		app.access = &sync.Mutex{}
	}
	return app
}

func (am *Manager) bindAll(apps []App) []App {
	for i := range apps {
		apps[i] = am.bind(apps[i])
	}
	return apps
}

//
// Client methods
//

// Create takes an image and creates an application, without starting it.
func (am *Manager) Create(ctx context.Context, installer string, name string, instanceName string, persistence bool, installerParams map[string]string) (*App, error) {
	app, _, err := am.CreateWithConfirmation(ctx, installer, name, instanceName, persistence, installerParams)
	return app, err
}

// CreateWithConfirmation creates an application and returns the strongest
// ordinary-write boundary observed before returning. A locally accepted write
// remains successful when another-peer availability is not presently
// provable; callers must use the returned receipt to continue observation
// instead of replaying the create.
func (am *Manager) CreateWithConfirmation(ctx context.Context, installer string, name string, instanceName string, persistence bool, installerParams map[string]string) (*App, db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var app *App
	if name == "" || installer == "" || instanceName == "" {
		return app, db.PublishedWriteConfirmation{}, fmt.Errorf("application name, installer ID, installer version or instance ID cannot be empty")
	}

	key, err := pcrypto.CreateManager(am.db).GenerateKey()
	if err != nil {
		return nil, db.PublishedWriteConfirmation{}, fmt.Errorf("failed to generate application key: %w", err)
	}

	appID, err := db.NewUUIDv7()
	if err != nil {
		return nil, db.PublishedWriteConfirmation{}, fmt.Errorf("failed to generate application id: %w", err)
	}
	log.Debugf("Creating application %s(%s), based on installer %s", appID, name, installer)
	app = &App{
		access: &sync.Mutex{},
		mgr:    am,

		Name:          name,
		ID:            appID,
		InstallerRef:  installer,
		InstanceID:    instanceName,
		DesiredStatus: statusStopped,
		IP:            appIPFromPublicKey(key.PublicString()),
		PublicKey:     key.PublicString(),
		Persistence:   persistence,
	}

	confirmation, err := db.InsertWithAvailabilityContext(ctx, am.db, createAppInsertMapper(*app))
	if err != nil {
		return nil, confirmation, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	log.Debug("Created application ", name, "[", appID, "]")
	return app, confirmation, nil
}

//
// Instance methods
//

// GetByID returns an application based on its id
func (am *Manager) GetByID(id string) (App, error) {
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOne(am.db, createAppQueryMapper([]sq.Predicate{db.UUIDEq(appModel.ID, id)}))
	if err != nil {
		return app, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	return am.bind(app), nil
}

// Get returns a copy of an application based on its name or id
func (am *Manager) Get(id string) (App, error) {
	return am.GetContext(context.Background(), id)
}

func (am *Manager) GetContext(ctx context.Context, id string) (App, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOneContext(ctx, am.db, createAppQueryMapper([]sq.Predicate{db.UUIDEq(appModel.ID, id)}))
	if err == nil {
		return am.bind(app), nil
	}

	app, nameErr := db.SelectOneContext(ctx, am.db, createAppQueryMapper([]sq.Predicate{appModel.NAME.EqString(id)}))
	if nameErr != nil {
		return app, fmt.Errorf("failed to retrieve app by id or name %q: id lookup: %w; name lookup: %w", id, err, nameErr)
	}

	return am.bind(app), nil
}

// GetAll returns a copy of all the applications
func (am *Manager) GetAll() ([]App, error) {
	apps, err := db.SelectMultiple(am.db, createAppQueryMapper(nil))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return am.bindAll(apps), nil
}

// GetAll returns a copy of all the applications
func (am *Manager) GetByIntance(instance string) ([]App, error) {
	appModel := sq.New[db.APP]("")
	apps, err := db.SelectMultiple(am.db, createAppQueryMapper([]sq.Predicate{appModel.INSTANCE_ID.EqString(instance)}))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return am.bindAll(apps), nil
}

// Notify schedules app runtime reconciliation. Runtime operations can perform
// image resolution, pulls, and sandbox lifecycle work, so the database notifier
// must not run them inline.
func (am *Manager) Notify() {
	if am == nil {
		return
	}
	am.notifyMu.Lock()
	defer am.notifyMu.Unlock()

	signature, err := am.reconcileSignature()
	if err != nil {
		log.Errorf("failed to inspect app runtime reconciliation state: %s", err.Error())
		return
	}
	if signature == "" || signature == am.reconcileSig {
		return
	}

	_, inserted, err := am.enqueueReconcileTask()
	if err != nil {
		log.Errorf("failed to queue app runtime reconciliation: %s", err.Error())
		return
	}

	// Only record the signature once a fresh task was actually queued. When the
	// enqueue is deduplicated against a reconcile that is already pending or
	// running, that in-flight task may have already observed an older desired
	// state, so advancing the signature here would silently drop the new desired
	// state (the "stopped (running)" hang). Leaving the signature untouched on a
	// dedupe lets a later Notify - including the periodic safety-net notifier -
	// re-evaluate and queue a follow-up once the in-flight reconcile finishes.
	if inserted {
		am.reconcileSig = signature
	}
}

func (am *Manager) clearReconcileSignature() {
	if am == nil {
		return
	}
	am.notifyMu.Lock()
	defer am.notifyMu.Unlock()
	am.reconcileSig = ""
}

func (am *Manager) reconcileSignature() (string, error) {
	if am == nil || am.db == nil {
		return "", nil
	}
	dbapps, err := db.SelectMultiple(am.db, createAppQueryMapper(nil))
	if err != nil {
		return "", fmt.Errorf("failed to get apps from db: %w", err)
	}

	parts := make([]string, 0, len(dbapps))
	for _, application := range dbapps {
		if !am.shouldReconcile(application) {
			continue
		}
		application = am.bind(application)
		parts = append(parts, strings.Join([]string{
			"app",
			application.ID,
			application.Name,
			application.InstallerRef,
			application.InstanceID,
			application.DesiredStatus,
			application.PublicKey,
		}, "\x00"))
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n"), nil
}

// enqueueReconcileTask enqueues a runtime reconcile task, deduplicated against
// any reconcile that is already pending or running. It returns whether a fresh
// task was actually inserted so callers can decide whether to advance the
// reconcile signature.
func (am *Manager) enqueueReconcileTask() (tasks.Record, bool, error) {
	if am == nil || am.tasks == nil {
		return tasks.Record{}, false, fmt.Errorf("app runtime task manager is not configured")
	}
	peerID := strings.TrimSpace(am.ptype)
	if peerID == "" {
		peerID = "local"
	}
	return tasks.EnqueueUnique(am.tasks, tasks.EnqueueUniqueOptions[reconcileRuntimeTaskPayload]{
		EnqueueOptions: tasks.EnqueueOptions[reconcileRuntimeTaskPayload]{
			Stream:      AppReconcileTaskStream,
			SubjectType: taskSubjectAppRuntime,
			SubjectID:   peerID,
			Title:       "Reconcile app runtime",
			Message:     "queued",
			Payload:     reconcileRuntimeTaskPayload{PeerID: peerID},
			MaxAttempts: 1,
		},
	})
}

func (am *Manager) runReconcileRuntimeTask(ctx context.Context, task *tasks.RunContext[reconcileRuntimeTaskPayload]) (reconcileRuntimeTaskResult, error) {
	payload := task.Payload()
	if err := task.Update(5, "loading app state", map[string]string{"peer_id": payload.PeerID}); err != nil {
		return reconcileRuntimeTaskResult{}, err
	}
	result, err := am.reconcileRuntime(ctx, func(progress int, message string, details any) error {
		return task.Progress(progress, message, details)
	})
	if err != nil {
		am.clearReconcileSignature()
		return reconcileRuntimeTaskResult{}, err
	}
	if err := task.Update(95, "app runtime reconciled", result); err != nil {
		am.clearReconcileSignature()
		return reconcileRuntimeTaskResult{}, err
	}
	return result, nil
}

// reconcileRuntime checks the db for new apps and deploys them if they belong to the current instance.
func (am *Manager) reconcileRuntime(ctx context.Context, progress func(int, string, any) error) (reconcileRuntimeTaskResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	if am == nil {
		return reconcileRuntimeTaskResult{}, nil
	}
	result := reconcileRuntimeTaskResult{PeerID: strings.TrimSpace(am.ptype)}
	if am.runtime == nil {
		_ = progress(10, "app runtime not configured", map[string]string{"peer_id": result.PeerID})
		return result, nil
	}

	log.Debug("Syncing apps")
	dbapps, err := db.SelectMultiple(am.db, createAppQueryMapper(nil))
	if err != nil {
		return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to get apps from db: %w", err)
	}

	appsMap := map[string]App{}
	for _, app := range dbapps {
		if err := ctx.Err(); err != nil {
			return reconcileRuntimeTaskResult{}, err
		}
		appsMap[app.ID] = app
		if !am.shouldReconcile(app) {
			continue
		}

		app = am.bind(app)
		log.Infof("App '%s' desired status: '%s'", app.Name, app.DesiredStatus)
		switch app.DesiredStatus {
		case statusRunning:
			_ = progress(20, "starting app", map[string]string{"app_id": app.ID, "app": app.Name})
			err := app.StartWithProgress(progress)
			if err != nil {
				return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to start app '%s': %w", app.Name, err)
			}
		case statusStopped:
			if app.GetStatus() != statusStopped {
				_ = progress(40, "stopping app", map[string]string{"app_id": app.ID, "app": app.Name})
				err := app.Stop()
				if err != nil {
					return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to stop app '%s': %w", app.Name, err)
				}
			}
		}
		result.Reconciled++
		log.Infof("App '%s' actual status: '%s'", app.Name, app.GetStatus())
	}

	allSandboxes, err := am.runtime.GetAllSandboxes()
	if err != nil {
		return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to get all sandboxes: %w", err)
	}
	for id, sandbox := range allSandboxes {
		if err := ctx.Err(); err != nil {
			return reconcileRuntimeTaskResult{}, err
		}
		if _, found := appsMap[id]; !found {
			log.Infof("App '%s' not found. Stopping and removing existing sandbox", id)
			_ = progress(70, "removing orphan sandbox", map[string]string{"app_id": id})
			err = sandbox.Stop()
			if err != nil {
				return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to stop sandbox for app '%s': %w", id, err)
			}
			err = sandbox.Remove()
			if err != nil {
				return reconcileRuntimeTaskResult{}, fmt.Errorf("failed to remove sandbox for app '%s': %w", id, err)
			}
			result.Removed++
		}
	}
	return result, nil
}

func (am *Manager) shouldReconcile(app App) bool {
	instanceID := strings.TrimSpace(app.InstanceID)
	if instanceID == "n/a" {
		return true
	}
	if instanceID == "" || strings.TrimSpace(am.ptype) == "" {
		return false
	}
	if _, err := db.UUIDBytes(instanceID); err != nil {
		log.Debugf("Skipping app '%s': assigned instance id %q is not a UUID", app.Name, instanceID)
		return false
	}

	publicKey, err := db.SelectOne(am.db, createAppInstancePublicKeyQueryMapper(instanceID))
	if err != nil {
		log.Debugf("Skipping app '%s': assigned instance %q could not be resolved: %s", app.Name, instanceID, err.Error())
		return false
	}
	peerID, err := db.PeerIDFromPublicKeyString(publicKey)
	if err != nil {
		log.Debugf("Skipping app '%s': assigned instance %q has invalid public key: %s", app.Name, instanceID, err.Error())
		return false
	}
	return peerID == am.ptype
}

// Start sets the desired status of the app to running, which triggers the app reconciler on the hosting instance.
func (am *Manager) Start(ctx context.Context, name string) error {
	_, err := am.StartWithConfirmation(ctx, name)
	return err
}

// StartWithConfirmation records the desired running state and exposes its
// exact availability boundary without confusing a pending second-peer proof
// with a failed mutation.
func (am *Manager) StartWithConfirmation(ctx context.Context, name string) (db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	app, err := am.GetContext(ctx, name)
	if err != nil {
		return db.PublishedWriteConfirmation{}, err
	}

	app.DesiredStatus = statusRunning
	confirmation, err := db.UpdateWithAvailabilityContext(ctx, am.db, createAppUpdateMapper(app))
	if err != nil {
		return confirmation, fmt.Errorf("failed to set desired application status to '%s'(%s): %w", statusRunning, app.Name, err)
	}

	return confirmation, nil
}

// Stop sets the desired status of the app to stopped, which triggers the stopping of the app on the hosting instance
func (am *Manager) Stop(ctx context.Context, name string) error {
	_, err := am.StopWithConfirmation(ctx, name)
	return err
}

// StopWithConfirmation records the desired stopped state and returns the
// accepted write's current other-peer availability stage.
func (am *Manager) StopWithConfirmation(ctx context.Context, name string) (db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	app, err := am.GetContext(ctx, name)
	if err != nil {
		return db.PublishedWriteConfirmation{}, err
	}

	app.DesiredStatus = statusStopped
	confirmation, err := db.UpdateWithAvailabilityContext(ctx, am.db, createAppUpdateMapper(app))
	if err != nil {
		return confirmation, fmt.Errorf("failed to set desired application status to '%s'(%s): %w", statusStopped, app.Name, err)
	}
	return confirmation, nil
}

// Remove removes an application based on the provided id
func (am *Manager) Remove(ctx context.Context, id string) error {
	_, err := am.RemoveWithConfirmation(ctx, id)
	return err
}

// RemoveWithConfirmation removes the declarative app row and returns its
// accepted write boundary. Runtime cleanup remains reconciler-owned.
func (am *Manager) RemoveWithConfirmation(ctx context.Context, id string) (db.PublishedWriteConfirmation, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	app, err := am.GetContext(ctx, id)
	if err != nil {
		return db.PublishedWriteConfirmation{}, errors.Wrapf(err, "Failed to remove application %s", id)
	}

	if app.DesiredStatus != statusStopped {
		return db.PublishedWriteConfirmation{}, fmt.Errorf("application '%s' should be stopped before being removed", id)
	}

	confirmation, err := db.DeleteWithAvailabilityContext(ctx, am.db, createAppDeleteByNameQuery(app.ID))
	if err != nil {
		return confirmation, errors.Wrapf(err, "Failed to remove application %s", id)
	}

	return confirmation, nil
}

// GetLogs retrieves the logs for a specific app
func (am *Manager) GetLogs(name string) ([]byte, error) {
	app, err := am.Get(name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for application '%s': %w", name, err)
	}

	cnt, err := am.runtime.GetSandbox(app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for application '%s': %w", name, err)
	}

	logs, err := cnt.GetLogs()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for application '%s': %w", name, err)
	}

	return logs, nil
}

// GetLogs retrieves the logs for a specific app
func (am *Manager) GetStatus(name string) (string, error) {
	app, err := am.Get(name)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for application '%s': %w", name, err)
	}

	return app.GetStatus(), nil
}
