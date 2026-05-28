package network

import "testing"

func TestNormalizeDNSServer(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "", want: ""},
		{name: "ipv4 default port", input: "8.8.8.8", want: "8.8.8.8:53"},
		{name: "ipv4 explicit port", input: "1.1.1.1:5353", want: "1.1.1.1:5353"},
		{name: "ipv6 default port", input: "2001:4860:4860::8888", want: "[2001:4860:4860::8888]:53"},
		{name: "ipv6 explicit port", input: "[2001:4860:4860::8888]:5353", want: "[2001:4860:4860::8888]:5353"},
		{name: "hostname rejected", input: "dns.google:53", wantErr: true},
		{name: "bad port rejected", input: "8.8.8.8:99999", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeDNSServer(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeDNSServer(%q) succeeded, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeDNSServer(%q): %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("NormalizeDNSServer(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestNormalizeExitRouteCIDRs(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    []string
		wantErr bool
	}{
		{name: "default full tunnel", input: nil, want: []string{"0.0.0.0/0", "::/0"}},
		{name: "single host", input: []string{"1.2.3.4/32"}, want: []string{"1.2.3.4/32"}},
		{name: "masked network", input: []string{"1.2.3.4/24"}, want: []string{"1.2.3.0/24"}},
		{name: "dedupe", input: []string{"1.2.3.4/32", "1.2.3.4/32"}, want: []string{"1.2.3.4/32"}},
		{name: "reject invalid", input: []string{"1.2.3.4"}, wantErr: true},
		{name: "reject empty", input: []string{""}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeExitRouteCIDRs(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeExitRouteCIDRs(%v) succeeded, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeExitRouteCIDRs(%v): %v", test.input, err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("NormalizeExitRouteCIDRs(%v) = %v, want %v", test.input, got, test.want)
			}
			for i := range test.want {
				if got[i] != test.want[i] {
					t.Fatalf("NormalizeExitRouteCIDRs(%v) = %v, want %v", test.input, got, test.want)
				}
			}
		})
	}
}

func TestRouteCIDRsUseFullTunnel(t *testing.T) {
	if !RouteCIDRsUseFullTunnel(nil) {
		t.Fatalf("nil CIDRs should default to full tunnel")
	}
	if !RouteCIDRsUseFullTunnel([]string{"0.0.0.0/0"}) {
		t.Fatalf("0.0.0.0/0 should use full tunnel")
	}
	if RouteCIDRsUseFullTunnel([]string{"1.2.3.4/32"}) {
		t.Fatalf("single host route should not use full tunnel")
	}
}
