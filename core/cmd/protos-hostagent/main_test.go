package main

import "testing"

func TestVMRunnerManifestArg(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "space separated",
			args: []string{"--run-vm", "-manifest", "/tmp/protos-local-macos-e2e-1/local-macos-vms/instances/vm/manifest.json"},
			want: "/tmp/protos-local-macos-e2e-1/local-macos-vms/instances/vm/manifest.json",
		},
		{
			name: "equals separated",
			args: []string{"--run-vm", "--manifest=/tmp/protos-local-macos-e2e-1/local-macos-vms/instances/vm/manifest.json"},
			want: "/tmp/protos-local-macos-e2e-1/local-macos-vms/instances/vm/manifest.json",
		},
		{
			name: "missing value",
			args: []string{"--run-vm", "-manifest"},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vmRunnerManifestArg(tt.args); got != tt.want {
				t.Fatalf("vmRunnerManifestArg() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPathHasPrefixVariantHandlesMacOSTmpAlias(t *testing.T) {
	manifest := "/private/tmp/protos-local-macos-e2e-1/local-macos-vms/instances/vm/manifest.json"
	prefix := "/tmp/protos-local-macos-e2e-"
	if !pathHasPrefixVariant(manifest, prefix) {
		t.Fatalf("expected %q to match prefix alias %q", manifest, prefix)
	}

	if pathHasPrefixVariant("/private/tmp/other-e2e-1/manifest.json", prefix) {
		t.Fatal("unexpected match for unrelated manifest path")
	}
}
