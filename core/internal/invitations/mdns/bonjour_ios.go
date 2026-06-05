//go:build ios

package mdns

/*
#include <stdlib.h>

extern char* ProtosNativeBonjourStart(const char *instance, const char *service, const char *domain, int port, const char *txt_json);
extern char* ProtosNativeBonjourBrowse(const char *service, const char *domain, int timeout_ms, char **out_json);
extern void ProtosNativeBonjourStop(void);
*/
import "C"

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"unsafe"

	"github.com/protosio/protos/internal/invitations"
)

type nativeAdvertisement struct{}

type nativeBrowseEntry struct {
	HostName string   `json:"hostName"`
	Port     int      `json:"port"`
	IPs      []string `json:"ips"`
	Text     []string `json:"text"`
}

func startBonjourAdvertisement(instanceName string, service string, domain string, port int, txt []string) (bonjourAdvertisement, error) {
	txtJSON, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}
	cInstance := C.CString(instanceName)
	cService := C.CString(service)
	cDomain := C.CString(domain)
	cTXT := C.CString(string(txtJSON))
	defer C.free(unsafe.Pointer(cInstance))
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cDomain))
	defer C.free(unsafe.Pointer(cTXT))

	if cErr := C.ProtosNativeBonjourStart(cInstance, cService, cDomain, C.int(port), cTXT); cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return nil, fmt.Errorf("%s", C.GoString(cErr))
	}
	return nativeAdvertisement{}, nil
}

func (nativeAdvertisement) Shutdown() {
	C.ProtosNativeBonjourStop()
}

func browseBonjour(ctx context.Context, timeout time.Duration) ([]invitations.NearbyInvite, error) {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	type result struct {
		items []invitations.NearbyInvite
		err   error
	}
	done := make(chan result, 1)
	go func() {
		items, err := browseBonjourNative(timeout)
		done <- result{items: items, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-done:
		if result.err != nil {
			return nil, result.err
		}
		return filterNearby(result.items), nil
	}
}

func browseBonjourNative(timeout time.Duration) ([]invitations.NearbyInvite, error) {
	cService := C.CString(ServiceTCP)
	cDomain := C.CString(ServiceDomain)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cDomain))

	var raw *C.char
	if cErr := C.ProtosNativeBonjourBrowse(cService, cDomain, C.int(timeout.Milliseconds()), &raw); cErr != nil {
		defer C.free(unsafe.Pointer(cErr))
		return nil, fmt.Errorf("%s", C.GoString(cErr))
	}
	if raw == nil {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(raw))

	var entries []nativeBrowseEntry
	if err := json.Unmarshal([]byte(C.GoString(raw)), &entries); err != nil {
		return nil, fmt.Errorf("decode native Bonjour browse result: %w", err)
	}
	items := make([]invitations.NearbyInvite, 0, len(entries))
	for _, entry := range entries {
		item, ok := parseTXTEntry(entry.Text, entry.HostName, entry.Port, entry.IPs)
		if ok {
			items = append(items, item)
		}
	}
	return items, nil
}
