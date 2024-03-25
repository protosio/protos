package app

import (
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"

	"github.com/protosio/protos/internal/runtime"

	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("app")

// Defines structure for config parameters
// specific to each application
const (
	// app states
	statusRunning    = "running"
	statusStopped    = "stopped"
	statusCreating   = "creating"
	statusFailed     = "failed"
	statusUnknown    = "unknown"
	statusDeleted    = "deleted"
	statusWillDelete = "willdelete"
)

// Config the application config
type Config struct {
	Description string
	Image       string
	Ports       map[string]string
	Data        string
}

// App represents the application state
type App struct {
	access *sync.Mutex
	mgr    *Manager

	// Public members
	Name          string `json:"name"`
	ID            string `json:"id"`
	InstallerRef  string `json:"installer-ref"`
	InstanceName  string `json:"instance-id"`
	DesiredStatus string `json:"desired-status"`
	IP            net.IP `json:"ip"`
	Persistence   bool   `json:"persistence"`
}

//
// Utilities
//

// createSandbox create the underlying container
func (app *App) createSandbox() (runtime.RuntimeSandbox, error) {
	log.Infof("Creating sandbox for app '%s'[%s] at '%s'", app.Name, app.ID, app.IP.String())
	cnt, err := app.mgr.runtime.NewSandbox(app.Name, app.ID, app.InstallerRef, app.Persistence)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for app '%s': %w", app.ID, err)
	}
	return cnt, nil
}

func (app *App) getOrcreateSandbox() (runtime.RuntimeSandbox, error) {
	cnt, err := app.mgr.runtime.GetSandbox(app.ID)
	if err != nil {
		if errors.Is(err, runtime.ErrSandboxNotFound) {
			cnt, err := app.createSandbox()
			if err != nil {
				return nil, err
			}
			return cnt, nil
		}
		return nil, fmt.Errorf("failed to retrieve container for app '%s': %w", app.ID, err)
	}
	return cnt, nil
}

//
// Methods for application instance
//

// GetID returns the id of the application
func (app *App) GetID() string {
	return app.ID
}

// GetName returns the id of the application
func (app *App) GetName() string {
	return app.Name
}

// GetStatus returns the status of an application
func (app *App) GetStatus() string {
	cnt, err := app.mgr.runtime.GetSandbox(app.ID)
	if err != nil {
		if !errors.Is(err, runtime.ErrSandboxNotFound) {
			log.Warnf("Failed to retrieve app (%s) sandbox: %s", app.ID, err.Error())
		}
		return statusStopped
	}

	return cnt.GetStatus()
}

// GetVersion returns the version of an application
func (app *App) GetVersion() string {
	return app.InstallerRef
}

// Start starts an application
func (app *App) Start() error {
	log.Infof("Starting application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.getOrcreateSandbox()
	if err != nil {
		return fmt.Errorf("failed to start application '%s': %w", app.ID, err)
	}

	err = cnt.Start(app.IP)
	if err != nil {
		return fmt.Errorf("failed to start application '%s': %w", app.ID, err)
	}
	return nil
}

// Stop stops an application
func (app *App) Stop() error {
	log.Infof("Stopping application '%s'[%s]", app.Name, app.ID)

	cnt, err := app.mgr.runtime.GetSandbox(app.ID)
	if err != nil {
		if !util.IsErrorType(err, runtime.ErrContainerNotFound) {
			return err
		}
		log.Warnf("Application '%s'(%s) has no sandbox to stop", app.Name, app.ID)
		return nil
	}

	err = cnt.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop application '%s'(%s): %w", app.Name, app.ID, err)
	}

	return nil
}

// GetIP returns the ip address of the app
func (app *App) GetIP() net.IP {
	return app.IP
}
