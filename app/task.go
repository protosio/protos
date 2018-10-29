package app

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/installer"
	"github.com/protosio/protos/task"
)

// CreateAppTask is an async task for creating an app
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
	log.WithField("proc", pt.ID).Debugf("Running app creation task [%s] based on installer %s:%s", t.InstallerID, t.InstallerVersion, t.AppName)
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
	pt.Progress.Percentage = 10
	pt.Progress.State = "Created application"
	pt.Update()

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", pt.ID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		tsk := inst.DownloadAsync(t.InstallerVersion)
		app.AddTask(tsk.ID)
		err := tsk.Wait()
		if err != nil {
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	} else {
		log.WithField("proc", pt.ID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
		pt.Progress.Percentage = 50
		pt.Progress.State = "Docker image found locally"
		pt.Update()
	}

	err = app.createContainer()
	if err != nil {
		app.Remove()
		return errors.Wrapf(err, "Could not create application %s", t.AppName)
	}
	pt.Progress.Percentage = 70
	pt.Progress.State = "Created Docker container"
	pt.Update()

	if t.StartOnCreation {
		err = app.Start()
		if err != nil {
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	}

	return nil
}
