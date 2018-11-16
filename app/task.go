package app

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"
)

// CreateAppTask creates an app and implements the task interface
type CreateAppTask struct {
	InstallerID      string
	InstallerVersion string
	AppName          string
	InstallerParams  map[string]string
	StartOnCreation  bool
}

// Name returns the task type name
func (t CreateAppTask) Name() string {
	return "Create application"
}

// Run starts the async task
func (t CreateAppTask) Run(pt *task.Task) error {
	log.WithField("proc", pt.ID).Debugf("Running app creation task [%s] based on installer %s:%s", pt.ID, t.InstallerID, t.InstallerVersion)
	pt.Status = task.INPROGRESS
	pt.Update()

	inst, err := installer.StoreGetID(t.InstallerID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}

	metadata, err := inst.ReadVersion(t.InstallerVersion)
	if err != nil {
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}

	app, err := Create(t.InstallerID, t.InstallerVersion, t.AppName, t.InstallerParams, metadata, pt.ID)
	if err != nil {
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}
	defer app.Save()
	pt.Progress.Percentage = 10
	pt.Progress.State = "Created application"
	pt.Apps = append(pt.Apps, app.ID)
	pt.Update()

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", pt.ID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		tsk := inst.DownloadAsync(t.InstallerVersion)
		app.AddTask(tsk.ID)
		err := tsk.Wait()
		if err != nil {
			app.Status = statusFailed
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	} else {
		log.WithField("proc", pt.ID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
		pt.Progress.Percentage = 50
		pt.Progress.State = "Docker image found locally"
		pt.Update()
	}

	_, err = app.createContainer()
	if err != nil {
		app.Status = statusFailed
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}
	pt.Progress.Percentage = 70
	pt.Progress.State = "Created Docker container"
	pt.Update()

	if t.StartOnCreation {
		tsk := app.StartAsync()
		app.AddTask(tsk.ID)
		err := tsk.Wait()
		if err != nil {
			app.Status = statusFailed
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	}
	app.Status = statusRunning
	return nil
}

//
// Set app state tasks
//

// StartAppTask starts an app and implements the task interface
type StartAppTask struct {
	app App
}

// Name returns the task type name
func (t StartAppTask) Name() string {
	return "Start application"
}

// Run starts the async task
func (t StartAppTask) Run(pt *task.Task) error {
	pt.Status = task.INPROGRESS
	pt.Progress.Percentage = 50
	pt.Apps = append(pt.Apps, t.app.ID)
	t.app.AddTask(pt.ID)
	pt.Update()
	return t.app.Start()
}

// StopAppTask stops an app and implements the task interface
type StopAppTask struct {
	app App
}

// Name returns the task type name
func (t StopAppTask) Name() string {
	return "Stop application"
}

// Run starts the async task
func (t StopAppTask) Run(pt *task.Task) error {
	pt.Status = task.INPROGRESS
	pt.Progress.Percentage = 50
	pt.Apps = append(pt.Apps, t.app.ID)
	t.app.AddTask(pt.ID)
	pt.Update()
	return t.app.Stop()
}
