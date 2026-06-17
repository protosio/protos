package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type preflight struct {
	failures []string
	warnings []string
}

func main() {
	var (
		hostAgentBin    = flag.String("hostagent-bin", "../bin/protos-hostagent", "protos-hostagent binary path")
		localImage      = flag.String("local-image", "../cloud-provisioning/targets/output/mactest", "local macOS image directory or file")
		hetznerImage    = flag.String("hetzner-image", "../cloud-provisioning/targets/output/hetzner/hetzner-bios.img", "Hetzner raw image file")
		scalewayImage   = flag.String("scaleway-image", "../cloud-provisioning/targets/output/scaleway/scaleway-efi.iso", "Scaleway ISO image file")
		hetznerEnv      = flag.String("hetzner-env", "../.env-hetzner", "Hetzner credential env file")
		scalewayEnv     = flag.String("scaleway-env", "../.env-scaleway", "Scaleway credential env file")
		scalewayZone    = flag.String("scaleway-zone", "fr-par-1", "Scaleway zone used for API reachability checks")
		probeArchive    = flag.String("probe-archive", ".tmp/protos-e2e-probe.tar.gz", "probe image archive path")
		summaryArtifact = flag.String("summary", ".tmp/mixed-cloud-e2e-summary.json", "mixed-cloud summary artifact path")
		minFreeGiB      = flag.Uint64("min-free-gib", 20, "minimum free disk space in GiB")
	)
	flag.Parse()

	check := &preflight{}
	check.darwin()
	check.commands("docker", "task", "cue", "protoc")
	check.docker()
	check.freeDisk(".", *minFreeGiB)
	check.hostAgent(*hostAgentBin)
	check.imagePath("local macOS image", *localImage, "mactest-disk.img")
	check.regularFile("Hetzner image", *hetznerImage)
	check.regularFile("Scaleway image", *scalewayImage)
	check.outputPath("probe image archive", *probeArchive, false)
	check.outputPath("mixed-cloud summary artifact", *summaryArtifact, true)

	hetznerValues := check.envFile(*hetznerEnv)
	hetznerToken := firstNonEmpty(os.Getenv("API_KEY"), hetznerValues["API_KEY"])
	check.required("Hetzner API key", hetznerToken)
	if hetznerToken != "" {
		check.http("Hetzner API", "https://api.hetzner.cloud/v1/locations", map[string]string{
			"Authorization": "Bearer " + hetznerToken,
		})
	}

	scalewayValues := check.envFile(*scalewayEnv)
	scalewaySecret := firstNonEmpty(os.Getenv("SECRET_KEY"), scalewayValues["SECRET_KEY"])
	check.required("Scaleway organization id", firstNonEmpty(os.Getenv("ORGANISATION_ID"), os.Getenv("ORG_ID"), os.Getenv("SCW_DEFAULT_ORGANIZATION_ID"), scalewayValues["ORGANISATION_ID"], scalewayValues["ORG_ID"], scalewayValues["SCW_DEFAULT_ORGANIZATION_ID"], scalewayConfigValue("default_organization_id"), scalewayConfigValue("organization_id")))
	check.required("Scaleway access key", firstNonEmpty(os.Getenv("ACCESS_KEY"), scalewayValues["ACCESS_KEY"]))
	check.required("Scaleway secret key", scalewaySecret)
	if scalewaySecret != "" {
		check.http("Scaleway API", fmt.Sprintf("https://api.scaleway.com/instance/v1/zones/%s/products/servers", strings.TrimSpace(*scalewayZone)), map[string]string{
			"X-Auth-Token": scalewaySecret,
		})
	}

	for _, warning := range check.warnings {
		fmt.Printf("warn %s\n", warning)
	}
	if len(check.failures) > 0 {
		fmt.Fprintln(os.Stderr, "e2e preflight failed:")
		for _, failure := range check.failures {
			fmt.Fprintf(os.Stderr, "- %s\n", failure)
		}
		os.Exit(1)
	}
	fmt.Println("e2e preflight ok")
}

func (p *preflight) darwin() {
	if runtime.GOOS != "darwin" {
		p.failures = append(p.failures, fmt.Sprintf("local macOS e2e requires darwin, got %s", runtime.GOOS))
		return
	}
	fmt.Println("ok host platform is darwin")
}

func (p *preflight) commands(names ...string) {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			p.failures = append(p.failures, fmt.Sprintf("required command %s was not found", name))
			continue
		}
		fmt.Printf("ok command %s: %s\n", name, path)
	}
}

func (p *preflight) docker() {
	if err := runQuiet("docker", "info"); err != nil {
		p.failures = append(p.failures, fmt.Sprintf("Docker is not available: %v", err))
		return
	}
	if err := runQuiet("docker", "buildx", "version"); err != nil {
		p.failures = append(p.failures, fmt.Sprintf("Docker buildx is not available: %v", err))
		return
	}
	fmt.Println("ok Docker and buildx are available")
}

func (p *preflight) hostAgent(path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("resolve host-agent path: %v", err))
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("host-agent binary %s is not available: %v", abs, err))
		return
	}
	if info.IsDir() {
		p.failures = append(p.failures, fmt.Sprintf("host-agent binary %s is a directory", abs))
		return
	}
	if err := runQuiet("sudo", "-n", abs, "--help"); err != nil {
		p.failures = append(p.failures, fmt.Sprintf("host-agent sudo rule is not usable for %s: %v", abs, err))
		return
	}
	fmt.Printf("ok host-agent sudo access: %s\n", abs)
}

func (p *preflight) imagePath(label string, path string, defaultFile string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s path cannot be resolved: %v", label, err))
		return
	}
	resolved, err := resolveImageFile(abs, defaultFile)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s is not uploadable: %v", label, err))
		return
	}
	fmt.Printf("ok %s: %s\n", label, resolved)
}

func (p *preflight) regularFile(label string, path string) {
	abs, err := filepath.Abs(path)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s path cannot be resolved: %v", label, err))
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s is missing: %v", label, err))
		return
	}
	if !info.Mode().IsRegular() {
		p.failures = append(p.failures, fmt.Sprintf("%s is not a regular file: %s", label, abs))
		return
	}
	if info.Size() == 0 {
		p.failures = append(p.failures, fmt.Sprintf("%s is empty: %s", label, abs))
		return
	}
	fmt.Printf("ok %s: %s\n", label, abs)
}

func (p *preflight) outputPath(label string, path string, expectExisting bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s path cannot be resolved: %v", label, err))
		return
	}
	if expectExisting {
		if _, err := os.Stat(abs); err != nil {
			p.warnings = append(p.warnings, fmt.Sprintf("%s does not exist yet: %s", label, abs))
		}
	}
	dir := filepath.Dir(abs)
	if err := os.MkdirAll(dir, 0755); err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s parent directory is not writable: %v", label, err))
		return
	}
	probe, err := os.CreateTemp(dir, ".preflight-*")
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s parent directory is not writable: %v", label, err))
		return
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	fmt.Printf("ok %s parent: %s\n", label, dir)
}

func (p *preflight) freeDisk(path string, minGiB uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		p.failures = append(p.failures, fmt.Sprintf("read free disk space for %s: %v", path, err))
		return
	}
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	freeGiB := freeBytes / (1024 * 1024 * 1024)
	if freeGiB < minGiB {
		p.failures = append(p.failures, fmt.Sprintf("free disk space is %dGiB, need at least %dGiB", freeGiB, minGiB))
		return
	}
	fmt.Printf("ok free disk space: %dGiB\n", freeGiB)
}

func (p *preflight) envFile(path string) map[string]string {
	values, err := loadEnvFile(path)
	if err != nil {
		p.failures = append(p.failures, err.Error())
		return map[string]string{}
	}
	fmt.Printf("ok env file: %s\n", path)
	return values
}

func (p *preflight) required(label string, value string) {
	if strings.TrimSpace(value) == "" {
		p.failures = append(p.failures, label+" is missing")
	}
}

func (p *preflight) http(label string, url string, headers map[string]string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s request cannot be built: %v", label, err))
		return
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.failures = append(p.failures, fmt.Sprintf("%s is not reachable: %v", label, err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		p.failures = append(p.failures, fmt.Sprintf("%s returned HTTP %d", label, resp.StatusCode))
		return
	}
	fmt.Printf("ok %s reachable\n", label)
}

func runQuiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func resolveImageFile(imagePath string, defaultFile string) (string, error) {
	info, err := os.Stat(imagePath)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		if info.Mode().IsRegular() && info.Size() > 0 {
			return imagePath, nil
		}
		return "", fmt.Errorf("%s is not a non-empty regular file", imagePath)
	}
	base := filepath.Base(imagePath)
	candidates := []string{
		filepath.Join(imagePath, defaultFile),
		filepath.Join(imagePath, base+"-disk.img"),
		filepath.Join(imagePath, base+"-efi.iso"),
		filepath.Join(imagePath, "disk.img"),
		filepath.Join(imagePath, "efi.iso"),
		filepath.Join(imagePath, base+"-root.raw"),
		filepath.Join(imagePath, "root.raw"),
	}
	for _, candidate := range candidates {
		if isRegularFile(candidate) {
			return candidate, nil
		}
	}
	for _, pattern := range []string{"*-disk.img", "*-root.raw"} {
		matches, err := filepath.Glob(filepath.Join(imagePath, pattern))
		if err != nil {
			return "", err
		}
		for _, match := range matches {
			if isRegularFile(match) {
				return match, nil
			}
		}
	}
	return "", fmt.Errorf("directory %s does not contain an uploadable image file", imagePath)
}

func isRegularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Size() > 0
}

func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", path, err)
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			values[key] = value
		}
	}
	return values, nil
}

func scalewayConfigValue(key string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, ".config", "scw", "config.yaml"))
	if err != nil {
		return ""
	}
	prefix := key + ":"
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.TrimPrefix(line, prefix)), `"'`)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
