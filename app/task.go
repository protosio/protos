package app

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"
)

// CreateAppTask creates an app and implements the task interface
type CreateAppTask struct {
	b                task.Task
	InstallerID      string
	InstallerVersion string
	AppName          string
	InstallerMedata  *installer.Metadata
	InstallerParams  map[string]string
	StartOnCreation  bool
}

// Name returns the task type name
func (t *CreateAppTask) Name() string {
	return "Create application"
}

// SetBase embedds the task base details
func (t *CreateAppTask) SetBase(tsk task.Task) {
	t.b = tsk
}

// Run starts the async task
func (t CreateAppTask) Run() error {
	tskID := t.b.GetID()
	log.WithField("proc", tskID).Debugf("Running app creation task [%s] based on installer %s:%s", tskID, t.InstallerID, t.InstallerVersion)

	var inst installer.Installer
	var metadata installer.Metadata
	var err error

	// normal app creation, using the app store
	if t.InstallerMedata == nil {
		inst, err = installer.StoreGetID(t.InstallerID)
		if err != nil {
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}

		metadata, err = inst.ReadVersion(t.InstallerVersion)
		if err != nil {
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
		// app creation using local container (dev purposes)
	} else {
		log.Info("Creating application using local installer (DEV)")
		metadata = *t.InstallerMedata
		inst = installer.Installer{
			ID:       t.InstallerID,
			Versions: map[string]*installer.Metadata{t.InstallerVersion: t.InstallerMedata},
		}
	}

	app, err := Create(t.InstallerID, t.InstallerVersion, t.AppName, t.InstallerParams, metadata, tskID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}
	add(app)
	t.b.SetPercentage(10)
	t.b.SetState("Created application")
	t.b.AddApp(app.ID)
	t.b.Save()

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", tskID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		tsk := inst.DownloadAsync(t.InstallerVersion, app.ID)
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	} else {
		log.WithField("proc", tskID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
		t.b.SetPercentage(50)
		t.b.SetState("Docker image found locally")
		t.b.Save()
	}

	_, err = app.createContainer()
	if err != nil {
		app.SetStatus(statusFailed)
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}
	t.b.SetPercentage(70)
	t.b.SetState("Created Docker container")
	t.b.Save()

	if t.StartOnCreation {
		tsk := app.StartAsync()
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
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
	b   task.Task
	app *App
}

// Name returns the task type name
func (t *StartAppTask) Name() string {
	return "Start application"
}

// SetBase embedds the task base details
func (t *StartAppTask) SetBase(tsk task.Task) {
	t.b = tsk
}

// Run starts the async task
func (t *StartAppTask) Run() error {
	t.b.SetStatus(task.INPROGRESS)
	t.b.SetPercentage(50)
	t.b.AddApp(t.app.ID)
	t.app.AddTask(t.b.GetID())
	t.b.Save()
	return t.app.Start()
}

// StopAppTask stops an app and implements the task interface
type StopAppTask struct {
	b   task.Task
	app *App
}

// Name returns the task type name
func (t *StopAppTask) Name() string {
	return "Stop application"
}

// SetBase embedds the task base details
func (t *StopAppTask) SetBase(tsk task.Task) {
	t.b = tsk
}

// Run starts the async task
func (t *StopAppTask) Run() error {
	t.b.SetStatus(task.INPROGRESS)
	t.b.SetPercentage(50)
	t.b.AddApp(t.app.ID)
	t.app.AddTask(t.b.GetID())
	t.b.Save()
	return t.app.Stop()
}

// RemoveAppTask removes an application and implements the task interface
type RemoveAppTask struct {
	b   task.Task
	app *App
}

// Name returns the task type name
func (t *RemoveAppTask) Name() string {
	return "Remove application"
}

// SetBase embedds the task base details
func (t *RemoveAppTask) SetBase(tsk task.Task) {
	t.b = tsk
}

// Run starts the async task
func (t *RemoveAppTask) Run() error {
	t.b.SetStatus(task.INPROGRESS)
	t.b.SetPercentage(50)
	t.b.Save()
	return t.app.Remove()
}
