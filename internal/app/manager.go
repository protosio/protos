package app

import (
	"fmt"
	"sync"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/runtime"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

const (
	appDS       = "app"
	TypeProtosc = "protosc"
	TypeProtosd = "protosd"
)

// Manager keeps track of all the apps
type Manager struct {
	ptype   string
	db      *db.DB
	runtime runtime.RuntimePlatform
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(ptype string, runtime runtime.RuntimePlatform, db *db.DB) *Manager {

	manager := &Manager{ptype: ptype, db: db, runtime: runtime}

	return manager
}

//
// Client methods
//

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installer string, name string, instanceName string, persistence bool, installerParams map[string]string) (*App, error) {

	var app *App
	if name == "" || instanceName == "" {
		return app, fmt.Errorf("application name, installer ID, installer version or instance ID cannot be empty")
	}

	// FIXME: here a key needs to be generated for the app, so that we can also derive the IP from it
	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installer)
	app = &App{
		access: &sync.Mutex{},
		mgr:    am,

		Name:          name,
		ID:            guid.String(),
		InstallerRef:  installer,
		InstanceID:    instanceName,
		DesiredStatus: statusStopped,
		Persistence:   persistence,
	}

	err := db.Insert(am.db, createAppInsertMapper(*app))
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	log.Debug("Created application ", name, "[", guid.String(), "]")
	return app, nil
}

//
// Instance methods
//

// GetByID returns an application based on its id
func (am *Manager) GetByID(id string) (App, error) {
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOne(am.db, createAppQueryMapper([]sq.Predicate{appModel.ID.EqString(id)}))
	if err != nil {
		return app, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	return App{}, errors.Wrapf(err, "Could not find application '%s'", id)
}

// Get returns a copy of an application based on its name
func (am *Manager) Get(id string) (App, error) {
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOne(am.db, createAppQueryMapper([]sq.Predicate{appModel.ID.EqString(id)}))
	if err != nil {
		return app, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	return App{}, fmt.Errorf("could not find application '%s'", id)
}

// GetAll returns a copy of all the applications
func (am *Manager) GetAll() ([]App, error) {
	apps, err := db.SelectMultiple(am.db, createAppQueryMapper(nil))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return apps, nil
}

// GetAll returns a copy of all the applications
func (am *Manager) GetByIntance(instance string) ([]App, error) {
	appModel := sq.New[db.APP]("")
	apps, err := db.SelectMultiple(am.db, createAppQueryMapper([]sq.Predicate{appModel.INSTANCE_ID.EqString(instance)}))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return apps, nil
}

// Refresh checks the db for new apps and deploys them if they belong to the current instance
func (am *Manager) Notify() {
	if am.ptype == TypeProtosc {
		return
	}

	log.Debug("Syncing apps")
	// TODO: fix the query
	dbapps, err := db.SelectMultiple(am.db, createAppQueryMapper([]sq.Predicate{sq.New[db.APP]("").INSTANCE_ID.EqString("n/a")}))
	if err != nil {
		log.Errorf("Failed to get apps from db: %s", err.Error())
		return
	}

	appsMap := map[string]App{}
	for _, app := range dbapps {
		appsMap[app.ID] = app

		app.mgr = am
		app.access = &sync.Mutex{}
		log.Infof("App '%s' desired status: '%s'", app.Name, app.DesiredStatus)
		if app.DesiredStatus == statusRunning {
			if app.GetStatus() != statusRunning {
				err := app.Start()
				if err != nil {
					log.Errorf("Failed to start app '%s': '%s'", app.Name, err.Error())
					continue
				}
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

	err = db.Delete(am.db, createAppDeleteByNameQuery(id))
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
