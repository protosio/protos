package apic

import (
	"context"
	"fmt"
	"os"

	"github.com/denisbrodbeck/machineid"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/auth"
)

func (b *Backend) GetApps(ctx context.Context, in *pbApic.GetAppsRequest) (*pbApic.GetAppsResponse, error) {

	log.Debugf("Retrieving apps")
	appsResponse := pbApic.GetAppsResponse{}

	return &appsResponse, nil
}

func (b *Backend) Init(ctx context.Context, in *pbApic.InitRequest) (*pbApic.InitResponse, error) {

	log.Debugf("Performing initialization")

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("Failed to add user. Could not retrieve hostname: %w", err)
	}
	key, err := b.protosClient.KeyManager.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("Failed to add user. Could not generate key: %w", err)
	}

	machineID, err := machineid.ProtectedID("protos")
	if err != nil {
		return nil, fmt.Errorf("Failed to add user. Error while generating machine id: %w", err)
	}

	devices := []auth.UserDevice{{Name: host, PublicKey: key.PublicWG().String(), MachineID: machineID, Network: "10.100.0.1/24"}}

	_, err = b.protosClient.UserManager.CreateUser(in.Username, in.Password, in.Name, in.Domain, true, devices)
	if err != nil {
		return nil, err
	}

	// saving the key to disk
	key.Save()

	return &pbApic.InitResponse{}, nil
}
