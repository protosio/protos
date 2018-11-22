package installer

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/task"
)

// DownloadTask downloads and installer and conforms to the task interface
type DownloadTask struct {
	*task.Base
	Inst    Installer
	Version string
}

// Name returns the name of the task type
func (t *DownloadTask) Name() string {
	return "Download application"
}

// SetBase embedds the task base details
func (t *DownloadTask) SetBase(base *task.Base) {
	t.Base = base
}

// Run starts the async task
func (t *DownloadTask) Run() {
	log.WithField("proc", t.ID).Debugf("Running download installer task [%s] based on installer %s:%s", t.ID, t.Inst.ID, t.Version)
	t.Status = task.INPROGRESS
	t.Save()

	err := t.Inst.Download(t)
	if err != nil {
		t.Finish(errors.Wrapf(err, "Could not download installer %s:%s", t.Inst.ID, t.Version))
	}

	t.Finish(nil)
}
