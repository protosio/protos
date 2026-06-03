//go:build darwin

package wireguard

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	cf "github.com/tmc/apple/corefoundation"
)

const (
	cfStringEncodingUTF8 = 0x08000100
	globalDNSKey         = "State:/Network/Global/DNS"
	serviceDNSKeyPattern = "State:/Network/Service/.*/DNS"
)

type (
	cfPropertyListRef   = cf.CFPropertyListRef
	scDynamicStoreRef   uintptr
	systemConfigBoolean = bool
)

type systemConfigurationStore struct {
	ref scDynamicStoreRef
}

var (
	systemConfigurationOnce sync.Once
	systemConfigurationErr  error

	scDynamicStoreCreate      func(cf.CFAllocatorRef, cf.CFStringRef, uintptr, unsafe.Pointer) scDynamicStoreRef
	scDynamicStoreCopyValue   func(scDynamicStoreRef, cf.CFStringRef) cf.CFPropertyListRef
	scDynamicStoreCopyKeyList func(scDynamicStoreRef, cf.CFStringRef) cf.CFArrayRef
	scDynamicStoreSetValue    func(scDynamicStoreRef, cf.CFStringRef, cf.CFPropertyListRef) systemConfigBoolean
	scDynamicStoreRemoveValue func(scDynamicStoreRef, cf.CFStringRef) systemConfigBoolean
	scDynamicStoreNotifyValue func(scDynamicStoreRef, cf.CFStringRef) systemConfigBoolean
)

func newSystemConfigurationStore() (*systemConfigurationStore, error) {
	if err := ensureSystemConfiguration(); err != nil {
		return nil, err
	}

	name := cfString("Protos")
	defer cfRelease(name)
	ref := scDynamicStoreCreate(0, cf.CFStringRef(name), 0, nil)
	if ref == 0 {
		return nil, fmt.Errorf("create SystemConfiguration dynamic store")
	}
	return &systemConfigurationStore{ref: ref}, nil
}

func (s *systemConfigurationStore) CopyGlobalDNS() (cf.CFPropertyListRef, error) {
	return s.CopyValue(globalDNSKey)
}

func (s *systemConfigurationStore) CopyValue(key string) (cf.CFPropertyListRef, error) {
	if s == nil || s.ref == 0 {
		return 0, fmt.Errorf("SystemConfiguration dynamic store is closed")
	}
	keyRef := cfString(key)
	if keyRef == 0 {
		return 0, fmt.Errorf("create SystemConfiguration key %q", key)
	}
	defer cfRelease(keyRef)
	return scDynamicStoreCopyValue(s.ref, cf.CFStringRef(keyRef)), nil
}

func (s *systemConfigurationStore) CopyDNSKeys() ([]string, error) {
	if s == nil || s.ref == 0 {
		return nil, fmt.Errorf("SystemConfiguration dynamic store is closed")
	}
	pattern := cfString(serviceDNSKeyPattern)
	if pattern == 0 {
		return nil, fmt.Errorf("create SystemConfiguration key pattern")
	}
	defer cfRelease(pattern)

	keysRef := scDynamicStoreCopyKeyList(s.ref, cf.CFStringRef(pattern))
	if keysRef == 0 {
		return nil, nil
	}
	defer cfRelease(keysRef)

	keys := make([]string, 0, cf.CFArrayGetCount(keysRef))
	for i := 0; i < cf.CFArrayGetCount(keysRef); i++ {
		value := cf.CFArrayGetValueAtIndex(keysRef, i)
		if value == nil || cf.CFGetTypeID(value) != cf.CFStringGetTypeID() {
			continue
		}
		key, err := cfStringToString(cf.CFStringRef(uintptr(value)))
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (s *systemConfigurationStore) SetGlobalDNS(value cf.CFPropertyListRef) error {
	return s.SetValue(globalDNSKey, value)
}

func (s *systemConfigurationStore) SetValue(key string, value cf.CFPropertyListRef) error {
	if s == nil || s.ref == 0 {
		return fmt.Errorf("SystemConfiguration dynamic store is closed")
	}
	if value == 0 {
		return fmt.Errorf("global DNS value is empty")
	}
	keyRef := cfString(key)
	if keyRef == 0 {
		return fmt.Errorf("create SystemConfiguration key %q", key)
	}
	defer cfRelease(keyRef)
	if !scDynamicStoreSetValue(s.ref, cf.CFStringRef(keyRef), value) {
		return fmt.Errorf("set macOS DNS resolver %s", key)
	}
	scDynamicStoreNotifyValue(s.ref, cf.CFStringRef(keyRef))
	return nil
}

func (s *systemConfigurationStore) RemoveValue(key string) error {
	if s == nil || s.ref == 0 {
		return fmt.Errorf("SystemConfiguration dynamic store is closed")
	}
	keyRef := cfString(key)
	if keyRef == 0 {
		return fmt.Errorf("create SystemConfiguration key %q", key)
	}
	defer cfRelease(keyRef)
	scDynamicStoreRemoveValue(s.ref, cf.CFStringRef(keyRef))
	scDynamicStoreNotifyValue(s.ref, cf.CFStringRef(keyRef))
	return nil
}

func (s *systemConfigurationStore) RemoveGlobalDNS() error {
	return s.RemoveValue(globalDNSKey)
}

func (s *systemConfigurationStore) Close() {
	if s == nil {
		return
	}
	if s.ref != 0 {
		cf.CFRelease(cfType(uintptr(s.ref)))
		s.ref = 0
	}
}

func ensureSystemConfiguration() error {
	systemConfigurationOnce.Do(func() {
		handle, err := purego.Dlopen("/System/Library/Frameworks/SystemConfiguration.framework/SystemConfiguration", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			systemConfigurationErr = fmt.Errorf("load SystemConfiguration framework: %w", err)
			return
		}
		for name, target := range map[string]any{
			"SCDynamicStoreCreate":      &scDynamicStoreCreate,
			"SCDynamicStoreCopyValue":   &scDynamicStoreCopyValue,
			"SCDynamicStoreCopyKeyList": &scDynamicStoreCopyKeyList,
			"SCDynamicStoreSetValue":    &scDynamicStoreSetValue,
			"SCDynamicStoreRemoveValue": &scDynamicStoreRemoveValue,
			"SCDynamicStoreNotifyValue": &scDynamicStoreNotifyValue,
		} {
			if err := registerSystemConfigurationFunc(target, handle, name); err != nil {
				systemConfigurationErr = err
				return
			}
		}
	})
	return systemConfigurationErr
}

func registerSystemConfigurationFunc(target any, handle uintptr, name string) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("register SystemConfiguration symbol %s: %v", name, recovered)
		}
	}()
	purego.RegisterLibFunc(target, handle, name)
	return nil
}

func newProtosGlobalDNS(server string, port int) (cf.CFDictionaryRef, error) {
	serverRef := cfString(server)
	if serverRef == 0 {
		return 0, fmt.Errorf("create DNS server string")
	}
	defer cfRelease(serverRef)

	addresses := []cf.CFStringRef{cf.CFStringRef(serverRef)}
	addressArray := cf.CFArrayCreate(0, unsafe.Pointer(&addresses[0]), len(addresses), &cf.KCFTypeArrayCallBacks)
	if addressArray == 0 {
		return 0, fmt.Errorf("create DNS server address array")
	}
	defer cfRelease(addressArray)

	portNumber := cfNumber(port)
	if portNumber == 0 {
		return 0, fmt.Errorf("create DNS server port")
	}
	defer cfRelease(portNumber)

	searchOrder := cfNumber(1)
	if searchOrder == 0 {
		return 0, fmt.Errorf("create DNS search order")
	}
	defer cfRelease(searchOrder)

	markerValue := cfString(protosDNSMarkerValue)
	if markerValue == 0 {
		return 0, fmt.Errorf("create DNS marker")
	}
	defer cfRelease(markerValue)

	keys := []cf.CFStringRef{
		cf.CFStringRef(cfString(globalDNSServerAddresses)),
		cf.CFStringRef(cfString(globalDNSServerPort)),
		cf.CFStringRef(cfString(globalDNSSearchOrder)),
		cf.CFStringRef(cfString(protosDNSMarkerKey)),
	}
	for _, key := range keys {
		defer cfRelease(key)
	}
	values := []cf.CFTypeRef{
		cfType(uintptr(addressArray)),
		cfType(uintptr(portNumber)),
		cfType(uintptr(searchOrder)),
		cfType(uintptr(markerValue)),
	}
	dict := cf.CFDictionaryCreate(
		0,
		unsafe.Pointer(&keys[0]),
		unsafe.Pointer(&values[0]),
		len(keys),
		&cf.KCFTypeDictionaryKeyCallBacks,
		&cf.KCFTypeDictionaryValueCallBacks,
	)
	if dict == 0 {
		return 0, fmt.Errorf("create DNS dictionary")
	}
	return dict, nil
}

func isProtosGlobalDNS(value cf.CFPropertyListRef) bool {
	if value == 0 {
		return false
	}
	if cf.CFGetTypeID(cfType(uintptr(value))) != cf.CFDictionaryGetTypeID() {
		return false
	}
	key := cfString(protosDNSMarkerKey)
	defer cfRelease(key)
	expected := cfString(protosDNSMarkerValue)
	defer cfRelease(expected)

	var actual unsafe.Pointer
	if !cf.CFDictionaryGetValueIfPresent(cf.CFDictionaryRef(value), unsafe.Pointer(key), unsafe.Pointer(&actual)) || actual == nil {
		return false
	}
	return cf.CFEqual(actual, cfType(uintptr(expected)))
}

func cfString(value string) cf.CFStringRef {
	return cf.CFStringCreateWithCString(0, value, cfStringEncodingUTF8)
}

func cfStringToString(value cf.CFStringRef) (string, error) {
	if value == 0 {
		return "", fmt.Errorf("CoreFoundation string is empty")
	}
	length := cf.CFStringGetLength(value)
	bufferSize := cf.CFStringGetMaximumSizeForEncoding(length, cfStringEncodingUTF8) + 1
	if bufferSize <= 1 {
		return "", nil
	}
	buffer := make([]byte, bufferSize)
	if !cf.CFStringGetCString(value, &buffer[0], bufferSize, cfStringEncodingUTF8) {
		return "", fmt.Errorf("convert CoreFoundation string")
	}
	n := 0
	for n < len(buffer) && buffer[n] != 0 {
		n++
	}
	return string(buffer[:n]), nil
}

func cfNumber(value int) cf.CFNumberRef {
	v := int32(value)
	return cf.CFNumberCreate(0, cf.KCFNumberIntType, unsafe.Pointer(&v))
}

func cfRelease(value any) {
	switch v := value.(type) {
	case cf.CFTypeRef:
		if v != nil {
			cf.CFRelease(v)
		}
	case cf.CFStringRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFArrayRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFDictionaryRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFNumberRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFPropertyListRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFDataRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	case cf.CFErrorRef:
		if v != 0 {
			cf.CFRelease(cfType(uintptr(v)))
		}
	}
}

func cfType(value uintptr) cf.CFTypeRef {
	return unsafe.Pointer(value)
}

func cfPropertyListToXML(value cf.CFPropertyListRef) ([]byte, error) {
	var cfErr cf.CFErrorRef
	data := cf.CFPropertyListCreateData(0, value, cf.KCFPropertyListXMLFormat_v1_0, 0, &cfErr)
	if cfErr != 0 {
		defer cfRelease(cfErr)
	}
	if data == 0 {
		return nil, fmt.Errorf("create property list data")
	}
	defer cfRelease(data)

	length := cf.CFDataGetLength(data)
	ptr := cf.CFDataGetBytePtr(data)
	if ptr == nil && length > 0 {
		return nil, fmt.Errorf("property list data is empty")
	}
	return append([]byte(nil), unsafe.Slice(ptr, length)...), nil
}

func cfPropertyListFromXML(data []byte) (cf.CFPropertyListRef, error) {
	if len(data) == 0 {
		return 0, fmt.Errorf("property list data is empty")
	}
	cfData := cf.CFDataCreate(0, data, len(data))
	if cfData == 0 {
		return 0, fmt.Errorf("create property list data")
	}
	defer cfRelease(cfData)

	var format cf.CFPropertyListFormat
	var cfErr cf.CFErrorRef
	value := cf.CFPropertyListCreateWithData(0, cfData, 0, &format, &cfErr)
	if cfErr != 0 {
		defer cfRelease(cfErr)
	}
	if value == 0 {
		return 0, fmt.Errorf("parse property list data")
	}
	return value, nil
}
