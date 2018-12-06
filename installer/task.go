package installer

import (
	"github.com/pkg/errors"
	"github.com/protosio/protos/task"
)

// DownloadTask downloads and installer and conforms to the task interface
type DownloadTask struct {
	b       task.Task
	Inst    Installer
	AppID   string
	Version string
}

// Name returns the name of the task type
func (t *DownloadTask) Name() string {
	return "Download application"
}

// SetBase embedds the task base details
func (t *DownloadTask) SetBase(tsk task.Task) {
	tsk.SetKillable()
	t.b = tsk
}

// Run starts the async task
func (t *DownloadTask) Run() error {
	tskID := t.b.GetID()
	log.WithField("proc", tskID).Debugf("Running download installer task [%s] based on installer %s:%s", tskID, t.Inst.ID, t.Version)
	t.b.AddApp(t.AppID)
	t.b.Save()

	err := t.Inst.Download(*t)
	if err != nil {
		return errors.Wrapf(err, "Could not download installer %s:%s", t.Inst.ID, t.Version)
	}

	return nil
}
