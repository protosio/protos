package app

import (
	"fmt"
	"net"
	"sync"

	"github.com/protosio/protos/internal/capability"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/meta"
	"github.com/protosio/protos/internal/resource"
	"github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/task"

	"github.com/pkg/errors"
	"github.com/rs/xid"
)

const (
	appDS       = "app"
	TypeProtosc = "protosc"
	TypeProtosd = "protosd"
)

// WSPublisher returns a channel that can be used to publish WS messages to the frontend
type WSPublisher interface {
	GetWSPublishChannel() chan interface{}
}

type appStore interface {
	GetInstaller(id string) (*installer.Installer, error)
}

// // dnsResource is only used locally to retrieve the Name of a DNS record
// type dnsResource interface {
// 	GetName() string
// 	GetValue() string
// 	Update(value resource.ResourceValue)
// 	Sanitize() resource.ResourceValue
// }

// Manager keeps track of all the apps
type Manager struct {
	ptype   string
	store   appStore
	rm      *resource.Manager
	tm      *task.Manager
	m       *meta.Meta
	db      db.DB
	cm      *capability.Manager
	runtime runtime.RuntimePlatform
}

//
// Public methods
//

// CreateManager returns a Manager, which implements the *AppManager interface
func CreateManager(ptype string, rm *resource.Manager, tm *task.Manager, runtime runtime.RuntimePlatform, db db.DB, meta *meta.Meta, appStore appStore, cm *capability.Manager) *Manager {

	if rm == nil || tm == nil || db == nil || meta == nil || appStore == nil || cm == nil {
		log.Panic("Failed to create app manager: none of the inputs can be nil")
	}

	log.Debug("Retrieving applications from DB")

	manager := &Manager{ptype: ptype, rm: rm, tm: tm, db: db, m: meta, runtime: runtime, store: appStore, cm: cm}

	err := db.InitDataset(appDS, manager)
	if err != nil {
		log.Fatal("Failed to initialize app dataset: ", err)
	}

	dbapps := map[string]App{}
	err = db.GetMap(appDS, &dbapps)
	if err != nil {
		log.Fatal("Could not retrieve applications from database: ", err)
	}

	return manager
}

// methods to satisfy local interfaces

func (am *Manager) getResourceManager() *resource.Manager {
	return am.rm
}

func (am *Manager) getCapabilityManager() *capability.Manager {
	return am.cm
}

//
// Client methods
//

// Create takes an image and creates an application, without starting it
func (am *Manager) Create(installer *installer.Installer, name string, instanceName string, instanceNetwork string, persistence bool, installerParams map[string]string) (*App, error) {

	var app *App
	if name == "" || instanceName == "" {
		return app, fmt.Errorf("application name, installer ID, installer version or instance ID cannot be empty")
	}

	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not create application '%s'", name)
	}

	for _, app := range apps {
		if app.Name == name {
			return nil, fmt.Errorf("could not create application '%s': another application exists with the same name", name)
		}
	}

	appIP, err := allocateIP(apps, instanceNetwork)
	if err != nil {
		return nil, fmt.Errorf("could not create application '%s': %w", name, err)
	}

	guid := xid.New()
	log.Debugf("Creating application %s(%s), based on installer %s", guid.String(), name, installer.Name)
	app = &App{
		access: &sync.Mutex{},
		mgr:    am,

		Name:          name,
		ID:            guid.String(),
		InstallerRef:  installer.Name,
		Version:       installer.Version,
		InstanceName:  instanceName,
		Tasks:         []string{},
		IP:            appIP,
		DesiredStatus: statusStopped,
		Persistence:   persistence,
	}

	err = validateInstallerParams(installerParams, installer.GetParams())
	if err != nil {
		return app, err
	}
	app.InstallerParams = installerParams
	app.Capabilities = createCapabilities(am.cm, installer.GetCapabilities())
	err = am.saveApp(app)
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
func (am *Manager) GetByID(id string) (*App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve application '%s'", id)
	}

	for _, app := range apps {
		if app.ID == id {
			app.mgr = am
			app.access = &sync.Mutex{}
			return &app, nil
		}
	}

	return nil, errors.Wrapf(err, "Could not find application '%s'", id)
}

// Get returns a copy of an application based on its name
func (am *Manager) Get(name string) (*App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve application '%s'", name)
	}

	for _, app := range apps {
		if app.Name == name {
			app.mgr = am
			app.access = &sync.Mutex{}
			return &app, nil
		}
	}

	return nil, fmt.Errorf("could not find application '%s'", name)
}

// GetAll returns a copy of all the applications
func (am *Manager) GetAll() (map[string]App, error) {
	apps := map[string]App{}
	err := am.db.GetMap(appDS, &apps)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not retrieve applications")
	}

	return apps, nil
}

// Refresh checks the db for new apps and deploys them if they belong to the current instance
func (am *Manager) Refresh() error {
	if am.ptype == TypeProtosc {
		return nil
	}

	log.Debug("Syncing apps")
	dbapps := map[string]App{}
	err := am.db.GetMap(appDS, &dbapps)
	if err != nil {
		return fmt.Errorf("could not retrieve applications from database: %w", err)
	}

	for _, app := range dbapps {
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
		if _, found := dbapps[id]; !found {
			log.Infof("App '%s' not found. Stopping and removing existing sandbox", id)
			err = sandbox.Stop()
			if err != nil {
				log.Errorf("Failed to remove sandbox for app '%s': %w", id, err)
				continue
			}
			err = sandbox.Remove()
			if err != nil {
				log.Errorf("Failed to remove sandbox for app '%s': %w", id, err)
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

	app.SetDesiredStatus(statusRunning)
	return nil
}

// Stop sets the desired status of the app to stopped, which triggers the stopping of the app on the hosting instance
func (am *Manager) Stop(name string) error {
	app, err := am.Get(name)
	if err != nil {
		return err
	}

	app.SetDesiredStatus(statusStopped)
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

	err = am.db.RemoveFromMap(appDS, app.ID)
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

func (am *Manager) saveApp(app *App) error {
	err := am.db.InsertInMap(appDS, app.ID, *app)
	if err != nil {
		return errors.Wrap(err, "Could not save app to database")
	}
	return nil
}

func allocateIP(apps map[string]App, networkStr string) (net.IP, error) {

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
