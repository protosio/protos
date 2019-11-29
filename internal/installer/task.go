package installer

import (
	"github.com/protosio/protos/internal/core"

	"github.com/pkg/errors"
)

// DownloadTask downloads and installer and conforms to the task interface
type DownloadTask struct {
	b       core.Task
	Inst    Installer
	AppID   string
	Version string
}

// Run starts the async task
func (t *DownloadTask) Run(parent core.Task, tskID string, p core.Progress) error {
	t.b = parent
	t.b.SetKillable()

	log.WithField("proc", tskID).Debugf("Running download installer task [%s] for installer '%s' version '%s'", tskID, t.Inst.ID, t.Version)
	t.b.AddApp(t.AppID)
	t.b.Save()

	err := t.Inst.Download(*t)
	if err != nil {
		return errors.Wrapf(err, "Could not download installer id '%s' version '%s'", t.Inst.ID, t.Version)
	}

	return nil
}
