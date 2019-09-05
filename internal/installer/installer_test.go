package installer

import (
	"testing"

	"protos/internal/capability"
	"protos/internal/util"
)

func TestInstaller(t *testing.T) {

	caps := parseInstallerCapabilities("ResourceProvider,WrongCap")
	if len(caps) != 1 {
		t.Errorf("Wrong number of capabilities returned. %d instead of 1", len(caps))
	}
	if caps[0] != capability.ResourceProvider {
		t.Errorf("Wrong capability returned by the parse function")
	}

	ports := parsePublicPorts("1/TCP,2/UDP,sfdsf,80000/TCP,50/SH")
	if len(ports) != 2 {
		t.Errorf("Wrong number of ports returned. %d instead of 2", len(caps))
	}
	if ports[0].Nr != 1 || ports[0].Type != util.TCP || ports[1].Nr != 2 || ports[1].Type != util.UDP {
		t.Errorf("Wrong data in the ports array returned by the parsePublicPorts: %v", ports)
	}

}

func TestInstallerMetadata(t *testing.T) {

	testMetadata := map[string]string{
		"protos.installer.metadata.capabilities": "ResourceProvider,ResourceConsumer,InternetAccess,GetInformation,PublicDNS,AuthUser",
		"protos.installer.metadata.requires":     "dns",
		"protos.installer.metadata.provides":     "mail,backup",
		"protos.installer.metadata.publicports":  "80/tcp,443/tcp,9999/udp",
		"protos.installer.metadata.name":         "testapp",
	}

	_, err := GetMetadata(testMetadata)
	if err == nil {
		t.Errorf("GetMetadata(testMetadata) should return an error on missing description")
	}

	testMetadata["protos.installer.metadata.description"] = "Small app description"

	metadata, err := GetMetadata(testMetadata)
	if err != nil {
		t.Errorf("GetMetadata(testMetadata) should not return an error, but it did: %s", err)
	}

	if len(metadata.PublicPorts) != 3 {
		t.Errorf("There should be %d publicports in the metadata. There are %d", 3, len(metadata.PublicPorts))
	}

	if (len(metadata.Requires) == 1 && metadata.Requires[0] != "dns") || len(metadata.Requires) != 1 {
		t.Errorf("metadata.Requires should only have 'dns' stored: %v", metadata.Requires)
	}

	if (len(metadata.Provides) == 2 && metadata.Provides[0] != "mail" && metadata.Provides[1] != "backup") || len(metadata.Provides) != 2 {
		t.Errorf("metadata.Provides should only have 'mail,backup' stored: %v", metadata.Requires)
	}

	if len(metadata.Capabilities) != 5 {
		t.Errorf("metadata.Capabilities should have 5 elements, but it has %d", len(metadata.Capabilities))
	}

}
