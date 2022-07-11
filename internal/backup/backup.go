package backup

import (
	"fmt"
	"log"

	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/db"
)

const (
	backupDS                      = "backup"
	backupStatusRequested         = "requested"
	backupStatusInProgress        = "inProgress"
	backupStatusCompleted         = "completed"
	backupStatusRequestedDeletion = "requestedDeletion"
	backupStatusDeleted           = "deleted"
)

type Backup struct {
	Name     string
	App      string
	Provider string
	Status   string
}

type BackupProvider struct {
	Name  string
	Cloud string
	Type  string
}

type BackupManager struct {
	db           db.DB
	cloudManager *cloud.Manager
	appManager   *app.Manager
}

func CreateManager(db db.DB, cloudManager *cloud.Manager, appManager *app.Manager) *BackupManager {

	err := db.InitDataset(backupDS, nil)
	if err != nil {
		log.Fatal("Failed to initialize backup dataset: ", err)
	}

	return &BackupManager{db: db, cloudManager: cloudManager, appManager: appManager}
}

func (b *BackupManager) GetProviders() (map[string]BackupProvider, error) {
	backupProviders := map[string]BackupProvider{}
	cloudProviders, err := b.cloudManager.GetProviders()
	if err != nil {
		return backupProviders, fmt.Errorf("could not retrieve backup providers: %w", err)
	}

	for _, cloud := range cloudProviders {
		backupProviders[cloud.NameStr()] = BackupProvider{Name: cloud.NameStr(), Cloud: cloud.TypeStr(), Type: "S3"}
	}

	return backupProviders, nil
}

func (b *BackupManager) GetProviderInfo(name string) (BackupProvider, error) {
	cloudProvider, err := b.cloudManager.GetProvider(name)
	if err != nil {
		return BackupProvider{}, fmt.Errorf("could not retrieve backup provider '%s': %w", name, err)
	}
	return BackupProvider{Name: cloudProvider.NameStr(), Cloud: cloudProvider.TypeStr(), Type: "S3"}, nil
}

func (b *BackupManager) GetBackupInfo(name string) (Backup, error) {
	return Backup{}, nil
}

func (b *BackupManager) GetBackups() (map[string]Backup, error) {
	backups := map[string]Backup{}
	err := b.db.GetMap(backupDS, &backups)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve backups from db: %w", err)
	}

	return backups, nil
}

func (b *BackupManager) CreateBackup(name string, app string, provider string) error {

	providers, err := b.GetProviders()
	if err != nil {
		return fmt.Errorf("could not create backup: %w", err)
	}
	if _, found := providers[provider]; !found {
		return fmt.Errorf("could not create backup: backup provider '%s' does not exist", provider)
	}

	_, err = b.appManager.Get(app)
	if err != nil {
		return fmt.Errorf("could not create backup: failed to retrieve app '%s': %w", app, err)
	}

	backup := Backup{Name: name, App: app, Provider: provider, Status: backupStatusRequested}
	err = b.db.InsertInMap(backupDS, name, backup)
	if err != nil {
		return fmt.Errorf("could not create backup: %w", err)
	}
	return nil
}

func (b *BackupManager) RemoveBackup(name string) error {
	backups := map[string]Backup{}
	err := b.db.GetMap(backupDS, &backups)
	if err != nil {
		return fmt.Errorf("could not mark backup '%s' for removal: %w", name, err)
	}

	for _, backup := range backups {
		if backup.Name == name {
			backup.Status = backupStatusRequestedDeletion
			err = b.db.InsertInMap(backupDS, name, backup)
			if err != nil {
				return fmt.Errorf("could not mark backup '%s' for removal: %w", name, err)
			}
			return nil
		}
	}

	return fmt.Errorf("could not remove backup: could not find backup '%s'", name)
}
