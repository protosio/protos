package app

import (
	"fmt"

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
		return fmt.Errorf("Could not create application %s: %s", t.AppName, err.Error())
	}

	app, err := Create(t.InstallerID, t.InstallerVersion, t.AppName, t.InstallerParams, metadata)
	if err != nil {
		return fmt.Errorf("Could not create application %s: %s", t.AppName, err.Error())
	}
	defer app.Remove()
	pt.Progress.Percentage = 10
	pt.Progress.StatusMessage = "Created application"
	pt.Update()

	if inst.IsPlatformImageAvailable(t.InstallerVersion) != true {
		log.WithField("proc", pt.ID).Debugf("Docker image %s for installer %s(%s) is not available locally. Downloading...", metadata.PlatformID, t.InstallerID, t.InstallerVersion)
		err = inst.Download(pt, t.InstallerVersion)
		if err != nil {
			return errors.Wrapf(err, "Could not create application %s", t.AppName)
		}
	} else {
		log.WithField("proc", pt.ID).Debugf("Docker image for installer %s(%s) found locally", t.InstallerID, t.InstallerVersion)
	}

	return nil
}
