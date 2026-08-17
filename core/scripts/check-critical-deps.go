package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

type dependency struct {
	Name          string
	Path          string
	Query         string
	Repository    string
	ResolvedQuery string
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

const swarmionRepository = "https://github.com/nustiueudinastea/swarmion"
const swarmionModulePath = "github.com/nustiueudinastea/swarmion"

var swarmionDependencyQuery = envOrDefault("SWARMION_DEP_QUERY", "main")

var criticalDependencies = []dependency{
	{Name: "containerd runtime", Path: "github.com/containerd/containerd/v2", Query: "latest"},
	{Name: "Swarmion", Path: swarmionModulePath, Query: swarmionDependencyQuery, Repository: swarmionRepository},
	{Name: "gRPC", Path: "google.golang.org/grpc", Query: "latest"},
	{Name: "protobuf", Path: "google.golang.org/protobuf", Query: "latest"},
	{Name: "Hetzner SDK", Path: "github.com/hetznercloud/hcloud-go/v2", Query: "latest"},
	{Name: "Scaleway SDK", Path: "github.com/scaleway/scaleway-sdk-go", Query: "latest"},
	{Name: "Go crypto", Path: "golang.org/x/crypto", Query: "latest"},
	{Name: "Go network", Path: "golang.org/x/net", Query: "latest"},
	{Name: "Go sys", Path: "golang.org/x/sys", Query: "latest"},
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func main() {
	dependencies, err := resolveDependencyQueries(criticalDependencies, commandOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "critical dependency freshness check failed:\n- %v\n", err)
		os.Exit(1)
	}

	var results []checkResult
	var failures []string
	if err := checkSwarmionPackageOwnership(commandOutput); err != nil {
		failures = append(failures, err.Error())
	}
	for _, dep := range dependencies {
		result, err := checkDependency(dep)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		results = append(results, result)
		if result.Outdated {
			failures = append(failures, fmt.Sprintf("%s is outdated: current=%s latest=%s query=%s", dep.Path, result.Current, result.Latest, formatQuery(dep)))
		}
	}

	for _, result := range results {
		status := "ok"
		if result.Outdated {
			status = "outdated"
		}
		fmt.Printf("%-10s %-36s %-64s current=%s latest=%s query=%s\n", status, result.Dependency.Name, result.Dependency.Path, result.Current, result.Latest, formatQuery(result.Dependency))
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

type packageInfo struct {
	ImportPath string      `json:"ImportPath"`
	Module     *moduleInfo `json:"Module"`
}

func checkSwarmionPackageOwnership(run outputRunner) error {
	out, err := run("go", "list", "-deps", "-json", "./...")
	if err != nil {
		return fmt.Errorf("inspect Swarmion package ownership: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(out))
	var packages []packageInfo
	for {
		var pkg packageInfo
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("decode package ownership: %w", err)
		}
		packages = append(packages, pkg)
	}
	return validateSwarmionPackageOwnership(packages)
}

func validateSwarmionPackageOwnership(packages []packageInfo) error {
	found := false
	for _, pkg := range packages {
		if pkg.ImportPath != swarmionModulePath && !strings.HasPrefix(pkg.ImportPath, swarmionModulePath+"/") {
			continue
		}
		found = true
		if pkg.Module == nil {
			return fmt.Errorf("Swarmion package %s has no owning module", pkg.ImportPath)
		}
		if pkg.Module.Path != swarmionModulePath {
			return fmt.Errorf("Swarmion package %s is owned by split module %s, want coherent root module %s", pkg.ImportPath, pkg.Module.Path, swarmionModulePath)
		}
	}
	if !found {
		return fmt.Errorf("module graph contains no imported Swarmion packages")
	}
	return nil
}

func checkDependency(dep dependency) (checkResult, error) {
	current, err := module(dep.Path)
	if err != nil {
		return checkResult{}, fmt.Errorf("load current module %s: %w", dep.Path, err)
	}
	query := dependencyQuery(dep)
	latest, err := module(dep.Path + "@" + query)
	if err != nil {
		return checkResult{}, fmt.Errorf("load latest module %s@%s: %w", dep.Path, formatQuery(dep), err)
	}

	currentVersion := effectiveVersion(current)
	latestVersion := effectiveVersion(latest)
	if strings.TrimSpace(currentVersion) == "" {
		return checkResult{}, fmt.Errorf("current module %s did not report a version", dep.Path)
	}
	if strings.TrimSpace(latestVersion) == "" {
		return checkResult{}, fmt.Errorf("latest module %s@%s did not report a version", dep.Path, formatQuery(dep))
	}

	return checkResult{
		Dependency: dep,
		Current:    currentVersion,
		Latest:     latestVersion,
		Outdated:   versionLess(currentVersion, latestVersion),
	}, nil
}

type outputRunner func(name string, args ...string) ([]byte, error)

func commandOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func resolveDependencyQueries(dependencies []dependency, run outputRunner) ([]dependency, error) {
	resolved := append([]dependency(nil), dependencies...)
	cache := make(map[string]string)
	for i := range resolved {
		dep := &resolved[i]
		if strings.TrimSpace(dep.Repository) == "" || queryNeedsNoRefResolution(dep.Query) {
			continue
		}
		key := strings.TrimSpace(dep.Repository) + "\x00" + strings.TrimSpace(dep.Query)
		commit, ok := cache[key]
		if !ok {
			var err error
			commit, err = resolveRemoteBranch(dep.Repository, dep.Query, run)
			if err != nil {
				return nil, fmt.Errorf("resolve %s query %q: %w", dep.Repository, dep.Query, err)
			}
			cache[key] = commit
		}
		dep.ResolvedQuery = commit
	}
	return resolved, nil
}

func queryNeedsNoRefResolution(query string) bool {
	query = strings.TrimSpace(query)
	return query == "latest" || semver.IsValid(query) || isHexRevision(query)
}

func resolveRemoteBranch(repository string, branch string, run outputRunner) (string, error) {
	repository = strings.TrimSpace(repository)
	branch = strings.TrimSpace(branch)
	if repository == "" {
		return "", fmt.Errorf("repository is empty")
	}
	if branch == "" {
		return "", fmt.Errorf("branch is empty")
	}
	ref := "refs/heads/" + branch
	out, err := run("git", "ls-remote", "--exit-code", "--refs", repository, ref)
	if err != nil {
		return "", fmt.Errorf("git ls-remote %s %s: %w", repository, ref, err)
	}

	var commit string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 || fields[1] != ref || !isFullGitObjectID(fields[0]) {
			return "", fmt.Errorf("unexpected git ls-remote result %q", line)
		}
		if commit != "" && !strings.EqualFold(commit, fields[0]) {
			return "", fmt.Errorf("remote branch %s returned multiple object IDs", ref)
		}
		commit = strings.ToLower(fields[0])
	}
	if commit == "" {
		return "", fmt.Errorf("remote branch %s returned no object ID", ref)
	}
	return commit, nil
}

func dependencyQuery(dep dependency) string {
	if query := strings.TrimSpace(dep.ResolvedQuery); query != "" {
		return query
	}
	return strings.TrimSpace(dep.Query)
}

func formatQuery(dep dependency) string {
	requested := strings.TrimSpace(dep.Query)
	resolved := strings.TrimSpace(dep.ResolvedQuery)
	if resolved == "" || requested == resolved {
		return requested
	}
	return requested + " (resolved " + resolved + ")"
}

func isHexRevision(value string) bool {
	value = strings.TrimSpace(value)
	return len(value) >= 7 && len(value) <= 64 && isHex(value)
}

func isFullGitObjectID(value string) bool {
	value = strings.TrimSpace(value)
	return (len(value) == 40 || len(value) == 64) && isHex(value)
}

func isHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
			return false
		}
	}
	return value != ""
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
