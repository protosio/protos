package installer

import (
	"testing"

	"github.com/protosio/protos/capability"
	"github.com/protosio/protos/util"
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
