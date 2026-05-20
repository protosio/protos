package protosd

import "testing"

func TestParseCapabilitiesExplicitList(t *testing.T) {
	caps, err := ParseCapabilities("api,network,provisioner")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || !caps.Network || !caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesCanDisableDefaults(t *testing.T) {
	caps, err := ParseCapabilities("default,no-network,no-provisioner,no-app-runtime")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || caps.Network || caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesNoneResets(t *testing.T) {
	caps, err := ParseCapabilities("all,none,api")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || caps.Network || caps.Provision || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesReadsEnvWhenUnset(t *testing.T) {
	t.Setenv("PROTOS_CAPABILITIES", "api,provisioner")
	caps, err := ParseCapabilities("")
	if err != nil {
		t.Fatal(err)
	}
	if !caps.API || !caps.Provision || caps.Network || caps.AppRuntime {
		t.Fatalf("unexpected capabilities: %+v", caps)
	}
}

func TestParseCapabilitiesRejectsUnknown(t *testing.T) {
	if _, err := ParseCapabilities("api,unknown"); err == nil {
		t.Fatal("expected unknown capability to fail")
	}
}
