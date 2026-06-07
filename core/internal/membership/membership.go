package membership

import (
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
)

func FilterInstances(instances []provisioners.InstanceInfo, peerIDs map[string]struct{}) []provisioners.InstanceInfo {
	filtered := make([]provisioners.InstanceInfo, 0, len(instances))
	for _, instance := range instances {
		peerID, err := instance.GetPeerID()
		if err != nil {
			continue
		}
		if _, found := peerIDs[peerID]; found {
			filtered = append(filtered, instance)
		}
	}
	return filtered
}

func FilterDevices(devices []user.UserDevice, peerIDs map[string]struct{}) []user.UserDevice {
	filtered := make([]user.UserDevice, 0, len(devices))
	for _, device := range devices {
		peerID, err := db.PeerIDFromPublicKeyString(device.GetPublicKey())
		if err != nil {
			continue
		}
		if _, found := peerIDs[peerID]; found {
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

func WitnessCandidates(instances []provisioners.InstanceInfo, devices []user.UserDevice) []db.WitnessCandidate {
	candidates := make([]db.WitnessCandidate, 0, len(instances)+len(devices))
	for _, instance := range instances {
		peerID, err := instance.GetPeerID()
		if err != nil {
			continue
		}
		candidates = append(candidates, db.WitnessCandidate{
			PeerID:     peerID,
			DeviceType: witnessDeviceTypeForInstance(instance),
			Rank:       instance.WitnessRank,
		})
	}
	for _, device := range devices {
		peerID, err := db.PeerIDFromPublicKeyString(device.GetPublicKey())
		if err != nil {
			continue
		}
		candidates = append(candidates, db.WitnessCandidate{
			PeerID:     peerID,
			DeviceType: witnessDeviceTypeForUserDevice(device),
			Rank:       device.WitnessRank,
		})
	}
	return candidates
}

func witnessDeviceTypeForInstance(instance provisioners.InstanceInfo) string {
	return db.WitnessDeviceTypeForMachine(instance.Kind, instance.KindID)
}

func witnessDeviceTypeForUserDevice(device user.UserDevice) string {
	return db.WitnessDeviceTypeForUserDeviceName(device.GetName())
}
