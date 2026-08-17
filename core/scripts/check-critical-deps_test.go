package main

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestResolveDependencyQueriesResolvesSwarmionRootBranchOnce(t *testing.T) {
	const (
		repository = "https://example.test/swarmion"
		branch     = "codex/r44-cascade-attribution"
		commit     = "e3a94518969305edf957c707a4bae5f95eb83cd9"
	)
	dependencies := []dependency{
		{Name: "Swarmion", Path: "example.test/swarmion", Query: branch, Repository: repository},
		{Name: "unrelated", Path: "example.test/unrelated", Query: "latest"},
	}
	var calls [][]string
	run := func(name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return []byte(commit + "\trefs/heads/" + branch + "\n"), nil
	}

	resolved, err := resolveDependencyQueries(dependencies, run)
	if err != nil {
		t.Fatalf("resolve dependency queries: %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("git invocation count = %d, want 1", len(calls))
	}
	wantCall := []string{"git", "ls-remote", "--exit-code", "--refs", repository, "refs/heads/" + branch}
	if !reflect.DeepEqual(calls[0], wantCall) {
		t.Fatalf("git invocation = %#v, want %#v", calls[0], wantCall)
	}
	if got := resolved[0].ResolvedQuery; got != commit {
		t.Errorf("resolved Swarmion query = %q, want %q", got, commit)
	}
	if got := resolved[0].Query; got != branch {
		t.Errorf("Swarmion query = %q, want original branch %q", got, branch)
	}
	if got := dependencyQuery(resolved[0]); got != commit {
		t.Errorf("dependencyQuery(Swarmion) = %q, want %q", got, commit)
	}
	if dependencies[0].ResolvedQuery != "" {
		t.Fatalf("input dependency was mutated: %#v", dependencies[0])
	}
	if got := resolved[1].ResolvedQuery; got != "" {
		t.Fatalf("unrelated resolved query = %q, want empty", got)
	}
}

func TestCriticalDependenciesContainOneSwarmionRootModule(t *testing.T) {
	var paths []string
	for _, dep := range criticalDependencies {
		if dep.Path == swarmionModulePath || strings.HasPrefix(dep.Path, swarmionModulePath+"/") {
			paths = append(paths, dep.Path)
		}
	}
	if !reflect.DeepEqual(paths, []string{swarmionModulePath}) {
		t.Fatalf("Swarmion dependency paths = %v, want only root module", paths)
	}
}

func TestValidateSwarmionPackageOwnershipRequiresRootModule(t *testing.T) {
	root := &moduleInfo{Path: swarmionModulePath, Version: "v0.0.0-20260817000000-deadbeefdead"}
	packages := []packageInfo{
		{ImportPath: swarmionModulePath + "/runtime", Module: root},
		{ImportPath: swarmionModulePath + "/protocol", Module: root},
		{ImportPath: "example.test/unrelated", Module: &moduleInfo{Path: "example.test/unrelated"}},
	}
	if err := validateSwarmionPackageOwnership(packages); err != nil {
		t.Fatalf("validate root ownership: %v", err)
	}

	packages[1].Module = &moduleInfo{Path: swarmionModulePath + "/protocol", Version: "v0.0.0"}
	if err := validateSwarmionPackageOwnership(packages); err == nil || !strings.Contains(err.Error(), "split module") {
		t.Fatalf("split ownership error = %v, want split-module rejection", err)
	}
}

func TestResolveDependencyQueriesLeavesImmutableQueriesAlone(t *testing.T) {
	const commit = "e3a94518969305edf957c707a4bae5f95eb83cd9"
	dependencies := []dependency{
		{Path: "example.test/swarmion", Query: commit, Repository: "https://example.test/swarmion"},
		{Path: "example.test/other", Query: "v1.2.3", Repository: "https://example.test/other"},
	}
	run := func(string, ...string) ([]byte, error) {
		t.Fatal("immutable queries must not invoke git")
		return nil, nil
	}

	resolved, err := resolveDependencyQueries(dependencies, run)
	if err != nil {
		t.Fatalf("resolve dependency queries: %v", err)
	}
	for i := range resolved {
		if resolved[i].ResolvedQuery != "" {
			t.Errorf("resolved[%d].ResolvedQuery = %q, want empty", i, resolved[i].ResolvedQuery)
		}
	}
}

func TestResolveRemoteBranchRejectsUnexpectedOutput(t *testing.T) {
	tests := map[string]string{
		"wrong ref":       "e3a94518969305edf957c707a4bae5f95eb83cd9\trefs/heads/main\n",
		"short object id": "e3a9451\trefs/heads/codex/r44-cascade-attribution\n",
		"missing fields":  "e3a94518969305edf957c707a4bae5f95eb83cd9\n",
		"empty":           "",
	}
	for name, output := range tests {
		t.Run(name, func(t *testing.T) {
			run := func(string, ...string) ([]byte, error) { return []byte(output), nil }
			if _, err := resolveRemoteBranch("https://example.test/swarmion", "codex/r44-cascade-attribution", run); err == nil {
				t.Fatal("resolveRemoteBranch succeeded, want error")
			}
		})
	}
}

func TestResolveRemoteBranchIncludesGitFailure(t *testing.T) {
	run := func(string, ...string) ([]byte, error) { return nil, errors.New("remote unavailable") }
	_, err := resolveRemoteBranch("https://example.test/swarmion", "codex/r44-cascade-attribution", run)
	if err == nil || !strings.Contains(err.Error(), "remote unavailable") {
		t.Fatalf("resolveRemoteBranch error = %v, want wrapped git failure", err)
	}
}

func TestFormatQueryShowsRequestedBranchAndResolvedCommit(t *testing.T) {
	dep := dependency{
		Query:         "codex/r44-cascade-attribution",
		ResolvedQuery: "e3a94518969305edf957c707a4bae5f95eb83cd9",
	}
	want := "codex/r44-cascade-attribution (resolved e3a94518969305edf957c707a4bae5f95eb83cd9)"
	if got := formatQuery(dep); got != want {
		t.Fatalf("formatQuery() = %q, want %q", got, want)
	}
}
