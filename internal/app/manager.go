package app

import (
	"fmt"
	"net"
	"sync"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/meta"
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
	m       *meta.Meta
	db      *db.DB
	cm      *capability.Manager
	runtime runtime.RuntimePlatform
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(ptype string, runtime runtime.RuntimePlatform, db *db.DB, meta *meta.Meta, cm *capability.Manager) *Manager {

	manager := &Manager{ptype: ptype, db: db, m: meta, runtime: runtime, cm: cm}

	return manager
}

//
// Client methods
//

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installer string, name string, instanceName string, instanceNetwork string, persistence bool, installerParams map[string]string) (*App, error) {

	var app *App
	if name == "" || instanceName == "" {
		return app, fmt.Errorf("application name, installer ID, installer version or instance ID cannot be empty")
	}

	apps, err := db.SelectMultiple(am.db, createInstanceQueryMapper(sq.New[db.APP](""), nil))
	if err != nil {
		return nil, fmt.Errorf("could not create application '%s': %w", name, err)
	}

	appIP, err := allocateIP(apps, instanceNetwork)
	if err != nil {
		return nil, fmt.Errorf("could not create application '%s': %w", name, err)
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installer)
	app = &App{
		access: &sync.Mutex{},
		mgr:    am,

		Name:          name,
		ID:            guid.String(),
		InstallerRef:  installer,
		InstanceName:  instanceName,
		IP:            appIP,
		DesiredStatus: statusStopped,
		Persistence:   persistence,
	}

	err = db.Insert(am.db, createAppInsertMapper(*app))
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
	app, err := db.SelectOne(am.db, createInstanceQueryMapper(appModel, []sq.Predicate{appModel.ID.EqString(id)}))
	if err != nil {
		return app, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	return App{}, errors.Wrapf(err, "Could not find application '%s'", id)
}

// Get returns a copy of an application based on its name
func (am *Manager) Get(name string) (App, error) {
	appModel := sq.New[db.APP]("")
	app, err := db.SelectOne(am.db, createInstanceQueryMapper(appModel, []sq.Predicate{appModel.ID.EqString(name)}))
	if err != nil {
		return app, fmt.Errorf("failed to retrieve instance: %w", err)
	}

	return App{}, fmt.Errorf("could not find application '%s'", name)
}

// GetAll returns a copy of all the applications
func (am *Manager) GetAll() ([]App, error) {
	apps, err := db.SelectMultiple(am.db, createInstanceQueryMapper(sq.New[db.APP](""), nil))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return apps, nil
}

// GetAll returns a copy of all the applications
func (am *Manager) GetByIntance(instance string) ([]App, error) {
	appModel := sq.New[db.APP]("")
	apps, err := db.SelectMultiple(am.db, createInstanceQueryMapper(appModel, []sq.Predicate{appModel.INSTANCE_NAME.EqString(instance)}))
	if err != nil {
		return nil, fmt.Errorf("could not get all applications: %w", err)
	}

	return apps, nil
}

// Refresh checks the db for new apps and deploys them if they belong to the current instance
func (am *Manager) Refresh() error {
	if am.ptype == TypeProtosc {
		return nil
	}

	log.Debug("Syncing apps")
	dbapps, err := db.SelectMultiple(am.db, createInstanceQueryMapper(sq.New[db.APP](""), nil))
	if err != nil {
		return fmt.Errorf("failure during application refresh: %w", err)
	}

	appsMap := map[string]App{}
	for _, app := range dbapps {
		appsMap[app.ID] = app
		if app.InstanceName == am.m.GetInstanceName() {
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
	}

	allSandboxes, err := am.runtime.GetAllSandboxes()
	if err != nil {
		return fmt.Errorf("failure during application refresh: %w", err)
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

	return nil
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
		return fmt.Errorf("failed to set desired application status to '%s'(%s): %v", statusRunning, app.Name, err)
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
		return fmt.Errorf("failed to set desired application status to '%s'(%s): %v", statusStopped, app.Name, err)
	}
	return nil
}

// Remove removes an application based on the provided id
func (am *Manager) Remove(name string) error {
	app, err := am.Get(name)
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application %s", name)
	}

	if app.DesiredStatus != statusStopped {
		return fmt.Errorf("application '%s' should be stopped before being removed", name)
	}

	err = db.Delete(am.db, createAppDeleteByNameQuery(name))
	if err != nil {
		return errors.Wrapf(err, "Failed to remove application %s", name)
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

func allocateIP(apps []App, networkStr string) (net.IP, error) {

	_, network, err := net.ParseCIDR(networkStr)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	usedIPs := map[string]bool{}
	for _, app := range apps {
		ip := app.GetIP()
		if ip != nil && network.Contains(ip) {
			usedIPs[ip.String()] = true
		}
	}

	allIPs := []net.IP{}
	for ip := network.IP.Mask(network.Mask); network.Contains(ip); incIP(ip) {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		allIPs = append(allIPs, newIP)
	}

	// starting from the 4th position in the slice to avoid allocating the network IP, WG and bridge interface IPs
	for _, ip := range allIPs[3 : len(allIPs)-1] {
		if _, found := usedIPs[ip.String()]; !found {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("failed to allocate IP. No IP's left")
}

// From https://play.golang.org/p/m8TNTtygK0
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
