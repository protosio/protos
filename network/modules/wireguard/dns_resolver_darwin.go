//go:build darwin

package wireguard

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	resolverPath             = "/etc/resolver"
	globalDNSBackupPath      = "/var/run/protos-exit-dns-global.plist"
	globalDNSBackupAbsent    = "protos:absent\n"
	protosDNSMarkerKey       = "ProtosManaged"
	protosDNSMarkerValue     = "exit-route-v1"
	globalDNSServerAddresses = "ServerAddresses"
	globalDNSServerPort      = "ServerPort"
	globalDNSSearchOrder     = "SearchOrder"
)

type globalDNSBackup struct {
	Version int                    `json:"version"`
	Entries []globalDNSBackupEntry `json:"entries"`
}

type globalDNSBackupEntry struct {
	Key    string `json:"key"`
	Absent bool   `json:"absent,omitempty"`
	XML    string `json:"xml,omitempty"`
}

type DNSManager struct {
	resolverPath        string
	globalDNSBackupPath string
	store               *systemConfigurationStore
}

func (m *DNSManager) AddDomainServer(domain string, server net.IP, port int) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}
	if server == nil {
		return fmt.Errorf("server cannot be empty")
	}

	if err := os.MkdirAll(m.resolverDir(), 0755); err != nil {
		return fmt.Errorf("could not create DNS resolver directory '%s': %w", m.resolverDir(), err)
	}
	resolverFile := path.Join(m.resolverDir(), domain)

	dnsData := fmt.Sprintf("domain %s\nport %d\nnameserver %s\n", domain, port, server.String())
	if err := os.WriteFile(resolverFile, []byte(dnsData), 0644); err != nil {
		return fmt.Errorf("could not add DNS server for domains '%s': %w", domain, err)
	}

	return nil
}

func (m *DNSManager) DelDomainServer(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	resolverFile := path.Join(m.resolverDir(), domain)
	if err := os.Remove(resolverFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete dns resolver file '%s': %w", resolverFile, err)
	}

	return nil
}

func (m *DNSManager) SetGlobalServer(server net.IP, port int) error {
	if server == nil {
		return fmt.Errorf("server cannot be empty")
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("DNS server port must be between 1 and 65535")
	}

	store, err := m.dynamicStore()
	if err != nil {
		return err
	}

	current, err := store.CopyGlobalDNS()
	if err != nil {
		return err
	}
	if current != 0 {
		defer cfRelease(current)
	}

	if !isProtosGlobalDNS(current) && !m.hasGlobalDNSBackup() {
		keys, err := m.globalDNSKeys(store)
		if err != nil {
			return err
		}
		if err := m.writeGlobalDNSBackup(store, keys); err != nil {
			return err
		}
	}

	desired, err := newProtosGlobalDNS(server.String(), port)
	if err != nil {
		return err
	}
	defer cfRelease(desired)

	keys, err := m.globalDNSKeys(store)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := store.SetValue(key, cfPropertyListRef(desired)); err != nil {
			return err
		}
	}
	return nil
}

func (m *DNSManager) DelGlobalServer() error {
	store, err := m.dynamicStore()
	if err != nil {
		if !m.hasGlobalDNSBackup() {
			return nil
		}
		return err
	}

	current, err := store.CopyGlobalDNS()
	if err != nil {
		return err
	}
	if current != 0 {
		defer cfRelease(current)
	}

	if !m.hasGlobalDNSBackup() && !isProtosGlobalDNS(current) {
		return nil
	}

	if err := m.restoreGlobalDNSBackup(store); err != nil {
		return err
	}
	if err := os.Remove(m.globalDNSBackupFile()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove DNS backup: %w", err)
	}
	return nil
}

func (m *DNSManager) globalDNSKeys(store *systemConfigurationStore) ([]string, error) {
	serviceKeys, err := store.CopyDNSKeys()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{globalDNSKey: {}}
	keys := []string{globalDNSKey}
	for _, key := range serviceKeys {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	return keys, nil
}

func (m *DNSManager) HasGlobalServerBackup() bool {
	return m.hasGlobalDNSBackup()
}

func (m *DNSManager) Close() error {
	if m.store == nil {
		return nil
	}
	m.store.Close()
	m.store = nil
	return nil
}

func (m *DNSManager) dynamicStore() (*systemConfigurationStore, error) {
	if m.store != nil {
		return m.store, nil
	}
	store, err := newSystemConfigurationStore()
	if err != nil {
		return nil, err
	}
	m.store = store
	return store, nil
}

func (m *DNSManager) resolverDir() string {
	if m.resolverPath != "" {
		return m.resolverPath
	}
	return resolverPath
}

func (m *DNSManager) globalDNSBackupFile() string {
	if m.globalDNSBackupPath != "" {
		return m.globalDNSBackupPath
	}
	return globalDNSBackupPath
}

func (m *DNSManager) hasGlobalDNSBackup() bool {
	_, err := os.Stat(m.globalDNSBackupFile())
	return err == nil
}

func (m *DNSManager) writeGlobalDNSBackup(store *systemConfigurationStore, keys []string) error {
	backupFile := m.globalDNSBackupFile()
	if err := os.MkdirAll(filepath.Dir(backupFile), 0755); err != nil {
		return fmt.Errorf("create DNS backup directory: %w", err)
	}
	backup := globalDNSBackup{
		Version: 1,
		Entries: make([]globalDNSBackupEntry, 0, len(keys)),
	}
	for _, key := range keys {
		current, err := store.CopyValue(key)
		if err != nil {
			return err
		}
		entry := globalDNSBackupEntry{Key: key, Absent: current == 0}
		if current != 0 {
			data, err := cfPropertyListToXML(current)
			cfRelease(current)
			if err != nil {
				return fmt.Errorf("serialize DNS backup for %s: %w", key, err)
			}
			entry.XML = string(data)
		}
		backup.Entries = append(backup.Entries, entry)
	}
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize DNS backup: %w", err)
	}
	if err := os.WriteFile(backupFile, data, 0600); err != nil {
		return fmt.Errorf("write DNS backup: %w", err)
	}
	return nil
}

func (m *DNSManager) restoreGlobalDNSBackup(store *systemConfigurationStore) error {
	backupFile := m.globalDNSBackupFile()
	data, err := os.ReadFile(backupFile)
	if errors.Is(err, os.ErrNotExist) {
		return store.RemoveGlobalDNS()
	}
	if err != nil {
		return fmt.Errorf("read DNS backup: %w", err)
	}
	if strings.TrimSpace(string(data)) == strings.TrimSpace(globalDNSBackupAbsent) {
		return store.RemoveGlobalDNS()
	}
	var backup globalDNSBackup
	if err := json.Unmarshal(data, &backup); err == nil && backup.Version == 1 {
		for _, entry := range backup.Entries {
			if entry.Key == "" {
				continue
			}
			if entry.Absent {
				if err := store.RemoveValue(entry.Key); err != nil {
					return err
				}
				continue
			}
			restored, err := cfPropertyListFromXML([]byte(entry.XML))
			if err != nil {
				return fmt.Errorf("parse DNS backup for %s: %w", entry.Key, err)
			}
			if err := store.SetValue(entry.Key, restored); err != nil {
				cfRelease(restored)
				return err
			}
			cfRelease(restored)
		}
		return nil
	}

	restored, err := cfPropertyListFromXML(data)
	if err != nil {
		return fmt.Errorf("parse DNS backup: %w", err)
	}
	defer cfRelease(restored)
	return store.SetGlobalDNS(restored)
}

func NewDNSManager() (*DNSManager, error) {
	return &DNSManager{
		resolverPath:        resolverPath,
		globalDNSBackupPath: globalDNSBackupPath,
	}, nil
}
