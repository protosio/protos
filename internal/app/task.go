package app

import (
	"protos/internal/core"

	"github.com/pkg/errors"
)

type taskParent interface {
	createAppForTask(installerID string, installerVersion string, name string, installerParams map[string]string, installerMetadata core.InstallerMetadata, taskID string) (app, error)
	Remove(appID string) error
	getTaskManager() core.TaskManager
	getAppStore() appStore
}

type app interface {
	Start() error
	Stop() error
	AddTask(id string)
	GetID() string
	SetStatus(status string)
	StartAsync() core.Task
	createContainer() (core.PlatformRuntimeUnit, error)
}

// CreateAppTask creates an app and implements the task interface
type CreateAppTask struct {
	am                taskParent
	InstallerID       string
	InstallerVersion  string
	AppName           string
	InstallerMetadata *core.InstallerMetadata
	InstallerParams   map[string]string
	StartOnCreation   bool
}

// Name returns the task type name
func (t *CreateAppTask) Name() string {
	return "Create application"
}

// Run starts the async task
func (t CreateAppTask) Run(tskID string, p core.Progress) error {
	log.WithField("proc", tskID).Debugf("Running app creation task [%s] based on installer %s:%s", tskID, t.InstallerID, t.InstallerVersion)

	if t.am == nil {
		log.Panic("Failed to run CreateAppTask: application manager is nil")
	}

	var inst core.Installer
	var metadata core.InstallerMetadata
	var err error

	if t.InstallerMetadata == nil {
		// normal app creation, using the app store
		inst, err = t.am.getAppStore().GetInstaller(t.InstallerID)
		if err != nil {
			return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
		}

		metadata, err = inst.GetMetadata(t.InstallerVersion)
		if err != nil {
			return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
		}
	} else {
		// app creation using local container (dev purposes)
		log.Info("Creating application using local installer (DEV)")
		metadata = *t.InstallerMetadata
		inst = t.am.getAppStore().CreateTemporaryInstaller(t.InstallerID, map[string]core.InstallerMetadata{t.InstallerVersion: *t.InstallerMetadata})
	}

	var app app
	app, err = t.am.createAppForTask(t.InstallerID, t.InstallerVersion, t.AppName, t.InstallerParams, metadata, tskID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}
	app.AddTask(tskID)
	p.SetPercentage(10)
	p.SetState("Created application")

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", tskID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		tsk := inst.DownloadAsync(t.am.getTaskManager(), t.InstallerVersion, app.GetID())
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
		}
	} else {
		log.WithField("proc", tskID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
		p.SetPercentage(50)
		p.SetState("Docker image found locally")
	}

	_, err = app.createContainer()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrapf(err, "Could not create application '%s'", t.AppName)
	}
	p.SetPercentage(70)
	p.SetState("Created Docker container")

	if t.StartOnCreation {
		tsk := app.StartAsync()
		app.AddTask(tsk.GetID())
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

// Name returns the task type name
func (t *StartAppTask) Name() string {
	return "Start application"
}

// Run starts the async task
func (t *StartAppTask) Run(tskID string, p core.Progress) error {
	p.SetPercentage(50)
	t.app.AddTask(tskID)
	return t.app.Start()
}

// StopAppTask stops an app and implements the task interface
type StopAppTask struct {
	app app
}

// Name returns the task type name
func (t *StopAppTask) Name() string {
	return "Stop application"
}

// Run starts the async task
func (t *StopAppTask) Run(tskID string, p core.Progress) error {
	p.SetPercentage(50)
	t.app.AddTask(tskID)
	return t.app.Stop()
}

// RemoveAppTask removes an application and implements the task interface
type RemoveAppTask struct {
	am    taskParent
	appID string
}

// Name returns the task type name
func (t *RemoveAppTask) Name() string {
	return "Remove application"
}

// Run starts the async task
func (t *RemoveAppTask) Run(tskID string, p core.Progress) error {
	if t.am == nil {
		log.Panic("Failed to run RemoveAppTask: application manager is nil")
	}

	p.SetState("Deleting application")
	p.SetPercentage(50)
	return t.am.Remove(t.appID)
}
