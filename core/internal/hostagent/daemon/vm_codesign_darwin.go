//go:build darwin

package daemon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const virtualizationEntitlement = "com.apple.security.virtualization"

const virtualizationEntitlementsPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>com.apple.security.virtualization</key>
	<true/>
</dict>
</plist>
`

func ensureVMRunnerEntitled(executable string) error {
	if strings.TrimSpace(executable) == "" {
		return fmt.Errorf("VM runner executable is empty")
	}
	if executableHasVirtualizationEntitlement(executable) {
		return nil
	}

	entitlements, err := os.CreateTemp("", "protos-hostagent-entitlements-*.plist")
	if err != nil {
		return fmt.Errorf("create temporary VM runner entitlements: %w", err)
	}
	entitlementsPath := entitlements.Name()
	defer os.Remove(entitlementsPath)
	if _, err := entitlements.WriteString(virtualizationEntitlementsPlist); err != nil {
		_ = entitlements.Close()
		return fmt.Errorf("write temporary VM runner entitlements: %w", err)
	}
	if err := entitlements.Close(); err != nil {
		return fmt.Errorf("close temporary VM runner entitlements: %w", err)
	}

	cmd := exec.Command("codesign", "--force", "--sign", "-", "--entitlements", entitlementsPath, executable)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sign VM runner %s with virtualization entitlement: %w: %s", executable, err, strings.TrimSpace(string(out)))
	}
	if !executableHasVirtualizationEntitlement(executable) {
		return fmt.Errorf("sign VM runner %s with virtualization entitlement: entitlement still missing", executable)
	}
	return nil
}

func executableHasVirtualizationEntitlement(executable string) bool {
	cmd := exec.Command("codesign", "-d", "--entitlements", ":-", executable)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return bytes.Contains(out, []byte(virtualizationEntitlement))
}
