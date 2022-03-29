package backup

import (
	"fmt"
	"log"

	"github.com/protosio/protos/internal/db"
)

const (
	backupDS = "backup"
)

type Backup struct {
	Name     string
	App      string
	Provider string
}

type BackupProvider struct {
	Name string
	Type string
}

type BackupManager struct {
	db db.DB
}

func CreateManager(db db.DB) *BackupManager {

	err := db.InitDataset(backupDS, nil)
	if err != nil {
		log.Fatal("Failed to initialize backup dataset: ", err)
	}

	return &BackupManager{db: db}
}

func (b *BackupManager) GetProviders() (map[string]BackupProvider, error) {
	backupProviders := map[string]BackupProvider{}
	return backupProviders, nil
}

func (b *BackupManager) GetProviderInfo(name string) (BackupProvider, error) {
	return BackupProvider{}, nil
}

func (b *BackupManager) GetBackupInfo(name string) (Backup, error) {
	return Backup{}, nil
}

func (b *BackupManager) GetBackups() (map[string]Backup, error) {
	backups := map[string]Backup{}
	err := b.db.GetMap(backupDS, &backups)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve backups: %w", err)
	}

	return backups, nil
}

func (b *BackupManager) CreateBackup(name string, provider string) error {

	providers, err := b.GetProviders()
	if err != nil {
		return fmt.Errorf("could not create backup: %w", err)
	}

	if _, found := providers[provider]; !found {
		return fmt.Errorf("could not create backup: backup provider '%s' does not exist", provider)
	}
	backup := &Backup{Name: name, Provider: provider}
	err = b.db.InsertInMap(backupDS, name, backup)
	if err != nil {
		return fmt.Errorf("could not create backup: %w", err)
	}
	return nil
}

func (b *BackupManager) RemoveBackup(name string, provider string) error {
	return nil
}
