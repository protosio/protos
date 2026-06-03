//go:build darwin

package wireguard

import (
	"testing"
	"unsafe"
)

func TestDarwinInterfaceNativeLayouts(t *testing.T) {
	tests := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{name: "ifreq", got: unsafe.Sizeof(darwinIfreq{}), want: 32},
		{name: "ifaliasreq", got: unsafe.Sizeof(darwinIfAliasreq{}), want: 64},
		{name: "in6_ifreq", got: unsafe.Sizeof(darwinIn6Ifreq{}), want: darwinIn6IfreqSize},
		{name: "in6_aliasreq", got: unsafe.Sizeof(darwinIn6Aliasreq{}), want: darwinIn6AliasreqSize},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Fatalf("%s size = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestCopyDarwinIfName(t *testing.T) {
	var name [darwinIfNameSize]byte
	if err := copyDarwinIfName(name[:], "utun12"); err != nil {
		t.Fatal(err)
	}
	if got := string(name[:6]); got != "utun12" {
		t.Fatalf("copied name = %q, want utun12", got)
	}
	if name[6] != 0 {
		t.Fatalf("interface name was not NUL-terminated")
	}
	if err := copyDarwinIfName(name[:], "1234567890abcdef"); err == nil {
		t.Fatalf("expected too-long interface name to fail")
	}
}
