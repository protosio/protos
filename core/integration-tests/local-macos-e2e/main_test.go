//go:build darwin

package main

import (
	"errors"
	"strings"
	"testing"
)

func TestPathPrefixVariantsHandlesMacOSTmpAlias(t *testing.T) {
	got := pathPrefixVariants("/tmp/protos-local-macos-e2e-1")
	want := map[string]bool{
		"/tmp/protos-local-macos-e2e-1":         false,
		"/private/tmp/protos-local-macos-e2e-1": false,
	}
	for _, value := range got {
		if _, ok := want[value]; ok {
			want[value] = true
		}
	}
	for value, found := range want {
		if !found {
			t.Fatalf("pathPrefixVariants() missing %q; got %v", value, got)
		}
	}
}

func TestEquivalentPathHandlesMacOSTmpAlias(t *testing.T) {
	candidates := pathPrefixVariants("/private/tmp/protos-local-macos-e2e-1")
	if !equivalentPath("/tmp/protos-local-macos-e2e-1", candidates) {
		t.Fatal("expected /tmp and /private/tmp workdir spellings to be equivalent")
	}
	if equivalentPath("/tmp/protos-local-macos-e2e-2", candidates) {
		t.Fatal("unexpected equivalent path for different e2e workdir")
	}
}

func TestCleanupFailurePropagatesToRunErrorAndIsRetained(t *testing.T) {
	want := errors.New("peer drain did not finalize")
	cleanup := &localMacOSE2ECleanup{
		vmDir: "/tmp/protos-local-macos-e2e-test/local-macos-vms",
		cleanupHostAgentVMs: func(string) error {
			return want
		},
	}
	first := cleanup.run()
	second := cleanup.run()
	if !errors.Is(first, want) || !errors.Is(second, want) {
		t.Fatalf("cleanup errors = (%v, %v), want retained %v", first, second, want)
	}
	got := mergeCleanupError(nil, first)
	if !errors.Is(got, want) || !strings.Contains(got.Error(), "local macOS e2e cleanup failed") {
		t.Fatalf("merged run error = %v, want cleanup failure", got)
	}
}
