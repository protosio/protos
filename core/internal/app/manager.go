package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/runtime"

	"github.com/pkg/errors"
)

const (
	appDS       = "app"
	TypeProtosd = "protosd"
)

// Manager keeps track of all the apps
type Manager struct {
	ptype         string
	db            *db.DB
	runtime       runtime.RuntimePlatform
	imageResolver ImageResolver
	notifyMu      sync.Mutex
}

type ImageResolver interface {
	ResolveImage(ctx context.Context, imageRef string) error
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(ptype string, runtime runtime.RuntimePlatform, db *db.DB) *Manager {

	manager := &Manager{ptype: ptype, db: db, runtime: runtime}

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

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installer string, name string, instanceName string, persistence bool, installerParams map[string]string) (*App, error) {

	var app *App
	if name == "" || installer == "" || instanceName == "" {
		return app, fmt.Errorf("application name, installer ID, installer version or instance ID cannot be empty")
	}

	key, err := pcrypto.CreateManager(am.db).GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate application key: %w", err)
	}

	appID, err := db.NewUUIDv7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate application id: %w", err)
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

	err = db.Insert(am.db, createAppInsertMapper(*app))
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	log.Debug("Created application ", name, "[", appID, "]")
	return app, nil
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
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOne(am.db, createAppQueryMapper([]sq.Predicate{db.UUIDEq(appModel.ID, id)}))
	if err == nil {
		return am.bind(app), nil
	}

	app, nameErr := db.SelectOne(am.db, createAppQueryMapper([]sq.Predicate{appModel.NAME.EqString(id)}))
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

// Refresh checks the db for new apps and deploys them if they belong to the current instance
func (am *Manager) Notify() {
	am.notifyMu.Lock()
	defer am.notifyMu.Unlock()

	log.Debug("Syncing apps")
	dbapps, err := db.SelectMultiple(am.db, createAppQueryMapper(nil))
	if err != nil {
		log.Errorf("Failed to get apps from db: %s", err.Error())
		return
	}

	appsMap := map[string]App{}
	for _, app := range dbapps {
		appsMap[app.ID] = app
		if !am.shouldReconcile(app) {
			continue
		}

		app = am.bind(app)
		log.Infof("App '%s' desired status: '%s'", app.Name, app.DesiredStatus)
		if app.DesiredStatus == statusRunning {
			err := app.Start()
			if err != nil {
				log.Errorf("Failed to start app '%s': '%s'", app.Name, err.Error())
				continue
			}
		} else if app.DesiredStatus == statusStopped {
			if app.GetStatus() != statusStopped {
				err := app.Stop()
				if err != nil {
					log.Errorf("Failed to stop app '%s': '%s'", app.Name, err.Error())
					continue
				}
			}
		}
		log.Infof("App '%s' actual status: '%s'", app.Name, app.GetStatus())
	}

	allSandboxes, err := am.runtime.GetAllSandboxes()
	if err != nil {
		log.Errorf("Failed to get all sandboxes: %s", err.Error())
		return
	}
	for id, sandbox := range allSandboxes {
		if _, found := appsMap[id]; !found {
			log.Infof("App '%s' not found. Stopping and removing existing sandbox", id)
			err = sandbox.Stop()
			if err != nil {
				log.Errorf("Failed to remove sandbox for app '%s': %s", id, err.Error())
				continue
			}
			err = sandbox.Remove()
			if err != nil {
				log.Errorf("Failed to remove sandbox for app '%s': %s", id, err.Error())
				continue
			}
		}
	}
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

// Start sets the desired status of the app to stopped, which triggers the stopping of the app on the hosting instance
func (am *Manager) Start(name string) error {
	app, err := am.Get(name)
	if err != nil {
		return err
	}

	app.DesiredStatus = statusRunning
	err = db.Update(am.db, createAppUpdateMapper(app))
	if err != nil {
		return fmt.Errorf("failed to set desired application status to '%s'(%s): %w", statusRunning, app.Name, err)
	}

	return nil
}

// Stop sets the desired status of the app to stopped, which triggers the stopping of the app on the hosting instance
func (am *Manager) Stop(name string) error {
	app, err := am.Get(name)
	if err != nil {
		return err
	}

	app.DesiredStatus = statusStopped
	err = db.Update(am.db, createAppUpdateMapper(app))
	if err != nil {
		return fmt.Errorf("failed to set desired application status to '%s'(%s): %w", statusStopped, app.Name, err)
	}
	return nil
}

// Remove removes an application based on the provided id
func (am *Manager) Remove(id string) error {
	app, err := am.Get(id)
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application %s", id)
	}

	if app.DesiredStatus != statusStopped {
		return fmt.Errorf("application '%s' should be stopped before being removed", id)
	}

	err = db.Delete(am.db, createAppDeleteByNameQuery(app.ID))
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application %s", id)
	}

	return nil
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
