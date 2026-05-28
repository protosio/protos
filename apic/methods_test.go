package apic

import "testing"

func TestIsPublicExitIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "5.161.52.86", want: true},
		{name: "public IPv6", ip: "2606:4700:4700::1111", want: true},
		{name: "private IPv4", ip: "10.0.0.1", want: false},
		{name: "carrier NAT IPv4", ip: "100.64.0.1", want: false},
		{name: "documentation IPv4", ip: "203.0.113.10", want: false},
		{name: "documentation IPv6", ip: "2001:db8::1", want: false},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "invalid", ip: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPublicExitIP(tt.ip); got != tt.want {
				t.Fatalf("isPublicExitIP(%q) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
}
