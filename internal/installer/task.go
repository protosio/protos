package installer

import (
	"github.com/pkg/errors"
	"protos/internal/core"
)

// DownloadTask downloads and installer and conforms to the task interface
type DownloadTask struct {
	b       core.Task
	Inst    Installer
	AppID   string
	Version string
}

// Name returns the name of the task type
func (t *DownloadTask) Name() string {
	return "Download application"
}

// Run starts the async task
func (t *DownloadTask) Run(tskID string, p core.Progress) error {
	// SetKillable is left from the previous code
	// tsk.SetKillable()

	log.WithField("proc", tskID).Debugf("Running download installer task [%s] based on installer %s:%s", tskID, t.Inst.ID, t.Version)
	t.b.AddApp(t.AppID)
	t.b.Save()

	err := t.Inst.Download(*t)
	if err != nil {
		return errors.Wrapf(err, "Could not download installer %s:%s", t.Inst.ID, t.Version)
	}

	return nil
}
