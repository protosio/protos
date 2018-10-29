package installer

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/task"
)

// DownloadTask downloads and installer and conforms to the task interface
type DownloadTask struct {
	Inst    Installer
	Version string
}

// Name returns the name of the task type
func (t DownloadTask) Name() string {
	return "Download application"
}

// Run starts the async task
func (t DownloadTask) Run(pt *task.Task) error {
	log.WithField("proc", pt.ID).Debugf("Running download installer task [%s] based on installer %s:%s", pt.ID, t.Inst.ID, t.Version)
	pt.Name = "Download installer"
	pt.Status = task.INPROGRESS
	pt.Update()

	err := t.Inst.Download(pt, t.Version)
	if err != nil {
		return errors.Wrapf(err, "Could not download installer %s:%s", t.Inst.ID, t.Version)
	}

	return nil
}
