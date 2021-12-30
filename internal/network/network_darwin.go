package network

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/protosio/protos/internal/auth"
	"github.com/protosio/protos/internal/cloud"
	"github.com/protosio/protos/internal/ssh"
)

const (
	protosNetworkInterface = "protos0"
	wgProtosBinary         = "wg-protos"
)

func (m *Manager) Up() error {

	cmd := exec.Command("sudo", wgProtosBinary, "wg", "up", protosNetworkInterface, m.network.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	cmd = exec.Command("sudo", wgProtosBinary, "domain", "add", m.domain, "127.0.0.1")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add domain using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	return nil
}

func (m *Manager) Down() error {
	cmd := exec.Command("sudo", wgProtosBinary, "wg", "down", protosNetworkInterface)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	// delete domain DNS
	cmd = exec.Command("sudo", wgProtosBinary, "domain", "del", m.domain)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete domain using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
	}

	return nil
}

func (m *Manager) ConfigurePeers(instances []cloud.InstanceInfo, devices []auth.UserDevice) error {

	log.Debug("Configuring network")
	peerConfigs := []string{}

	for _, instance := range instances {
		if len(instance.PublicKey) == 0 || instance.PublicIP == "" || instance.InternalIP == "" || instance.Network == "" || instance.Name == "" {
			continue
		}

		pubkey, err := ssh.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			return fmt.Errorf("failed to configure network (%s): %w", instance.Name, err)
		}

		_, instanceNetwork, err := net.ParseCIDR(instance.Network)
		if err != nil {
			return fmt.Errorf("failed to parse network for instance '%s': %w", instance.Name, err)
		}

		peerConf := fmt.Sprintf("%s:%s:%s:%s:%s", instance.Name, pubkey.String(), instance.PublicIP, instance.InternalIP, instanceNetwork.String())
		peerConfigs = append(peerConfigs, peerConf)
	}

	if len(peerConfigs) > 0 {
		configureArgs := []string{wgProtosBinary, "wg", "configure", protosNetworkInterface, m.privateKey.String()}
		configureArgs = append(configureArgs, peerConfigs...)
		cmd := exec.Command("sudo", configureArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to configure link using wg-protos: \n---- wg-protos output ----\n%s-------------------", string(output))
		}
	}

	return nil
}
