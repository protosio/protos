package app

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"
)

// CreateAppTask creates an app and implements the task interface
type CreateAppTask struct {
	*task.Base
	InstallerID      string
	InstallerVersion string
	AppName          string
	InstallerParams  map[string]string
	StartOnCreation  bool
}

// Name returns the task type name
func (t *CreateAppTask) Name() string {
	return "Create application"
}

// SetBase embedds the task base details
func (t *CreateAppTask) SetBase(base *task.Base) {
	t.Base = base
}

// Run starts the async task
func (t CreateAppTask) Run() {
	log.WithField("proc", t.ID).Debugf("Running app creation task [%s] based on installer %s:%s", t.ID, t.InstallerID, t.InstallerVersion)
	t.Status = task.INPROGRESS
	t.Save()

	inst, err := installer.StoreGetID(t.InstallerID)
	if err != nil {
		t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
	}

	metadata, err := inst.ReadVersion(t.InstallerVersion)
	if err != nil {
		t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
	}

	app, err := Create(t.InstallerID, t.InstallerVersion, t.AppName, t.InstallerParams, metadata, t.ID)
	if err != nil {
		t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
	}
	add(app)
	t.Progress.Percentage = 10
	t.Progress.State = "Created application"
	t.Apps = append(t.Apps, app.ID)
	t.Save()

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", t.ID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		tsk := inst.DownloadAsync(t.InstallerVersion, app.ID)
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
		}
	} else {
		log.WithField("proc", t.ID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
		t.Progress.Percentage = 50
		t.Progress.State = "Docker image found locally"
		t.Save()
	}

	_, err = app.createContainer()
	if err != nil {
		app.SetStatus(statusFailed)
		t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
	}
	t.Progress.Percentage = 70
	t.Progress.State = "Created Docker container"
	t.Save()

	if t.StartOnCreation {
		tsk := app.StartAsync()
		app.AddTask(tsk.GetID())
		err := tsk.Wait()
		if err != nil {
			app.SetStatus(statusFailed)
			t.Finish(errors.Wrapf(err, "Could not create application %s", t.AppName))
		}
	}
	app.SetStatus(statusRunning)
	t.Finish(nil)
}

//
// Set app state tasks
//

// StartAppTask starts an app and implements the task interface
type StartAppTask struct {
	*task.Base
	app *App
}

// Name returns the task type name
func (t *StartAppTask) Name() string {
	return "Start application"
}

// SetBase embedds the task base details
func (t *StartAppTask) SetBase(base *task.Base) {
	t.Base = base
}

// Run starts the async task
func (t *StartAppTask) Run() {
	t.Status = task.INPROGRESS
	t.Progress.Percentage = 50
	t.Apps = append(t.Apps, t.app.ID)
	t.app.AddTask(t.ID)
	t.Save()
	t.Finish(t.app.Start())
}

// StopAppTask stops an app and implements the task interface
type StopAppTask struct {
	*task.Base
	app *App
}

// Name returns the task type name
func (t *StopAppTask) Name() string {
	return "Stop application"
}

// SetBase embedds the task base details
func (t *StopAppTask) SetBase(base *task.Base) {
	t.Base = base
}

// Run starts the async task
func (t *StopAppTask) Run() {
	t.Status = task.INPROGRESS
	t.Progress.Percentage = 50
	t.Apps = append(t.Apps, t.app.ID)
	t.app.AddTask(t.ID)
	t.Save()
	t.Finish(t.app.Stop())
}

// RemoveAppTask removes an application and implements the task interface
type RemoveAppTask struct {
	*task.Base
	app *App
}

// Name returns the task type name
func (t *RemoveAppTask) Name() string {
	return "Stop application"
}

// SetBase embedds the task base details
func (t *RemoveAppTask) SetBase(base *task.Base) {
	t.Base = base
}

// Run starts the async task
func (t *RemoveAppTask) Run() {
	t.Status = task.INPROGRESS
	t.Progress.Percentage = 30
	t.Save()
	t.Finish(t.app.Remove())
}
