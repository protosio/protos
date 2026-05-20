package membership

import (
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
)

func FilterInstances(instances []provisioners.InstanceInfo, peerIDs map[string]struct{}) []provisioners.InstanceInfo {
	filtered := make([]provisioners.InstanceInfo, 0, len(instances))
	for _, instance := range instances {
		if _, found := peerIDs[instance.GetID()]; found {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func FilterDevices(devices []user.UserDevice, peerIDs map[string]struct{}) []user.UserDevice {
	filtered := make([]user.UserDevice, 0, len(devices))
	for _, device := range devices {
		if _, found := peerIDs[device.GetID()]; found {
			filtered = append(filtered, device)
		}
	}
	return filtered
}

func Machines(instances []provisioners.InstanceInfo, devices []user.UserDevice) []p2p.Machine {
	peers := make([]p2p.Machine, 0, len(instances)+len(devices))
	for _, instance := range instances {
		peers = append(peers, instance)
	}
	for i := range devices {
		peers = append(peers, &devices[i])
	}
	return peers
}
