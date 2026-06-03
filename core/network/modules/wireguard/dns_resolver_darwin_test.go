//go:build darwin

package wireguard

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestDNSManagerDomainResolverUsesConfiguredDirectory(t *testing.T) {
	manager := &DNSManager{resolverPath: t.TempDir()}

	if err := manager.AddDomainServer("protos.internal", net.ParseIP("127.0.0.1"), 10053); err != nil {
		t.Fatalf("AddDomainServer: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(manager.resolverPath, "protos.internal"))
	if err != nil {
		t.Fatalf("read resolver: %v", err)
	}
	want := "domain protos.internal\nport 10053\nnameserver 127.0.0.1\n"
	if string(data) != want {
		t.Fatalf("resolver data = %q, want %q", string(data), want)
	}

	if err := manager.DelDomainServer("protos.internal"); err != nil {
		t.Fatalf("DelDomainServer: %v", err)
	}
	if _, err := os.Stat(filepath.Join(manager.resolverPath, "protos.internal")); !os.IsNotExist(err) {
		t.Fatalf("resolver still exists after delete: %v", err)
	}
}

func TestProtosGlobalDNSDictionaryRoundTrip(t *testing.T) {
	dict, err := newProtosGlobalDNS("127.0.0.1", 10053)
	if err != nil {
		t.Fatalf("newProtosGlobalDNS: %v", err)
	}
	defer cfRelease(dict)

	if !isProtosGlobalDNS(cfPropertyListRef(dict)) {
		t.Fatalf("global DNS dictionary is missing Protos marker")
	}

	data, err := cfPropertyListToXML(cfPropertyListRef(dict))
	if err != nil {
		t.Fatalf("cfPropertyListToXML: %v", err)
	}
	restored, err := cfPropertyListFromXML(data)
	if err != nil {
		t.Fatalf("cfPropertyListFromXML: %v", err)
	}
	defer cfRelease(restored)

	if !isProtosGlobalDNS(restored) {
		t.Fatalf("restored global DNS dictionary is missing Protos marker")
	}
}

func TestSystemConfigurationStoreCanReadGlobalDNS(t *testing.T) {
	store, err := newSystemConfigurationStore()
	if err != nil {
		t.Fatalf("newSystemConfigurationStore: %v", err)
	}
	defer store.Close()

	value, err := store.CopyGlobalDNS()
	if err != nil {
		t.Fatalf("CopyGlobalDNS: %v", err)
	}
	if value != 0 {
		cfRelease(value)
	}
}
