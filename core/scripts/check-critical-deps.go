package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

type dependency struct {
	Name  string
	Path  string
	Query string
}

type moduleInfo struct {
	Path    string      `json:"Path"`
	Version string      `json:"Version"`
	Replace *moduleInfo `json:"Replace"`
}

type checkResult struct {
	Dependency dependency
	Current    string
	Latest     string
	Outdated   bool
}

var criticalDependencies = []dependency{
	{Name: "containerd runtime", Path: "github.com/containerd/containerd/v2", Query: "latest"},
	{Name: "Swarmion protocol", Path: "github.com/nustiueudinastea/swarmion/protocol", Query: "main"},
	{Name: "Swarmion runtime", Path: "github.com/nustiueudinastea/swarmion/runtime", Query: "main"},
	{Name: "Swarmion CUE schema engine", Path: "github.com/nustiueudinastea/swarmion/schema-engines/cue", Query: "main"},
	{Name: "Swarmion declarative schema engine", Path: "github.com/nustiueudinastea/swarmion/schema-engines/declarative", Query: "main"},
	{Name: "Swarmion transports", Path: "github.com/nustiueudinastea/swarmion/transports", Query: "main"},
	{Name: "gRPC", Path: "google.golang.org/grpc", Query: "latest"},
	{Name: "protobuf", Path: "google.golang.org/protobuf", Query: "latest"},
	{Name: "Hetzner SDK", Path: "github.com/hetznercloud/hcloud-go/v2", Query: "latest"},
	{Name: "Scaleway SDK", Path: "github.com/scaleway/scaleway-sdk-go", Query: "latest"},
	{Name: "Go crypto", Path: "golang.org/x/crypto", Query: "latest"},
	{Name: "Go network", Path: "golang.org/x/net", Query: "latest"},
	{Name: "Go sys", Path: "golang.org/x/sys", Query: "latest"},
}

func main() {
	var results []checkResult
	var failures []string
	for _, dep := range criticalDependencies {
		result, err := checkDependency(dep)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		results = append(results, result)
		if result.Outdated {
			failures = append(failures, fmt.Sprintf("%s is outdated: current=%s latest=%s query=%s", dep.Path, result.Current, result.Latest, dep.Query))
		}
	}

	for _, result := range results {
		status := "ok"
		if result.Outdated {
			status = "outdated"
		}
		fmt.Printf("%-10s %-36s %-64s current=%s latest=%s query=%s\n", status, result.Dependency.Name, result.Dependency.Path, result.Current, result.Latest, result.Dependency.Query)
	}

	if len(failures) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "critical dependency freshness check failed:")
	for _, failure := range failures {
		fmt.Fprintf(os.Stderr, "- %s\n", failure)
	}
	os.Exit(1)
}

func checkDependency(dep dependency) (checkResult, error) {
	current, err := module(dep.Path)
	if err != nil {
		return checkResult{}, fmt.Errorf("load current module %s: %w", dep.Path, err)
	}
	latest, err := module(dep.Path + "@" + dep.Query)
	if err != nil {
		return checkResult{}, fmt.Errorf("load latest module %s@%s: %w", dep.Path, dep.Query, err)
	}

	currentVersion := effectiveVersion(current)
	latestVersion := effectiveVersion(latest)
	if strings.TrimSpace(currentVersion) == "" {
		return checkResult{}, fmt.Errorf("current module %s did not report a version", dep.Path)
	}
	if strings.TrimSpace(latestVersion) == "" {
		return checkResult{}, fmt.Errorf("latest module %s@%s did not report a version", dep.Path, dep.Query)
	}

	return checkResult{
		Dependency: dep,
		Current:    currentVersion,
		Latest:     latestVersion,
		Outdated:   versionLess(currentVersion, latestVersion),
	}, nil
}

func module(arg string) (moduleInfo, error) {
	cmd := exec.Command("go", "list", "-m", "-json", arg)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return moduleInfo{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	var info moduleInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return moduleInfo{}, err
	}
	return info, nil
}

func effectiveVersion(info moduleInfo) string {
	if info.Replace != nil && strings.TrimSpace(info.Replace.Version) != "" {
		return strings.TrimSpace(info.Replace.Version)
	}
	return strings.TrimSpace(info.Version)
}

func versionLess(current string, latest string) bool {
	current = strings.TrimSpace(current)
	latest = strings.TrimSpace(latest)
	if current == latest {
		return false
	}
	if semver.IsValid(current) && semver.IsValid(latest) {
		return semver.Compare(current, latest) < 0
	}
	return current < latest
}
