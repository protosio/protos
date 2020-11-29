package app

import (
	"github.com/protosio/protos/internal/installer"
	"github.com/protosio/protos/internal/platform"
	"github.com/protosio/protos/internal/task"

	"github.com/pkg/errors"
)

type taskParent interface {
	createAppForTask(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata installer.InstallerMetadata, taskID string) (app, error)
	Remove(appID string) error
	getTaskManager() *task.Manager
	getAppStore() appStore
}

type app interface {
	Start() error
	Stop() error
	AddTask(id string)
	GetID() string
	SetStatus(status string)
	StartAsync() *task.Base
	createSandbox() (platform.PlatformRuntimeUnit, error)
}

// CreateAppTask creates an app and implements the task interface
type CreateAppTask struct {
	am               taskParent
	InstallerID      string
	InstallerVersion string
	AppName          string
	InstallerParams  map[string]string
	StartOnCreation  bool
}

// Run starts the async task
func (t CreateAppTask) Run(parent *task.Base, tskID string, p task.Progrs) error {
	log.WithField("proc", tskID).Debugf("Running app creation task '%s' based on installer '%s:%s'", tskID, t.InstallerID, t.InstallerVersion)

	if t.InstallerID == "" || t.AppName == "" || t.am == nil {
		return errors.Errorf("Failed to run CreateAppTask for app '%s' because one of the required task fields are missing", t.AppName)
	}

	var inst *installer.Installer
	var version string
	var metadata installer.InstallerMetadata
	var err error

	// normal app creation, using the app store
	inst, err = t.am.getAppStore().GetInstaller(t.InstallerID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}

	if t.InstallerVersion == "" {
		version = inst.GetLastVersion()
		log.Infof("Creating application using latest version (%s) of installer '%s'", version, t.InstallerID)
	} else {
		version = t.InstallerVersion
	}

	metadata, err = inst.GetMetadata(version)
	if err != nil {
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}

	var app app
	app, err = t.am.createAppForTask(t.InstallerID, version, t.AppName, t.InstallerParams, metadata, tskID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}
	app.AddTask(tskID)
	p.SetPercentage(10)
	p.SetState("Created application")

	available, err := inst.IsPlatformImageAvailable(version)
	if err != nil {
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}
	if available != true {
		log.WithField("proc", tskID).Debugf("Container image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, version)
		tsk := inst.DownloadAsync(version, app.GetID())
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
		}
	} else {
		log.WithField("proc", tskID).Debugf("Container image for installer %s(%s) found locally", t.InstallerID, version)
		p.SetPercentage(50)
		p.SetState("Container image found locally")
	}

	_, err = app.createSandbox()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}
	p.SetPercentage(70)
	p.SetState("Created container")

	if t.StartOnCreation {
		tsk := app.StartAsync()
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
		}
	}
	app.SetStatus(statusRunning)
	return nil
}

//
// Set app state tasks
//

// StartAppTask starts an app and implements the task interface
type StartAppTask struct {
	app app
}

// Run starts the async task
func (t *StartAppTask) Run(parent *task.Base, tskID string, p task.Progrs) error {
	log.WithField("proc", tskID).Infof("Running start app task '%s'", tskID)
	p.SetPercentage(50)
	t.app.AddTask(tskID)
	return t.app.Start()
}

// StopAppTask stops an app and implements the task interface
type StopAppTask struct {
	app app
}

// Run starts the async task
func (t *StopAppTask) Run(parent *task.Base, tskID string, p task.Progrs) error {
	log.WithField("proc", tskID).Infof("Running stop app task '%s'", tskID)
	p.SetPercentage(50)
	t.app.AddTask(tskID)
	return t.app.Stop()
}

// RemoveAppTask removes an application and implements the task interface
type RemoveAppTask struct {
	am    taskParent
	appID string
}

// Run starts the async task
func (t *RemoveAppTask) Run(parent *task.Base, tskID string, p task.Progrs) error {
	if t.am == nil {
		log.Panic("Failed to run RemoveAppTask: application manager is nil")
	}

	p.SetState("Deleting application")
	p.SetPercentage(50)
	return t.am.Remove(t.appID)
}
