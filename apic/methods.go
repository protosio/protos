package apic

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/netip"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/network"
	networkmodule "github.com/protosio/protos/internal/network/module"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
)

//
// Initialization
//

func (b *Backend) requireProvisionerCapability(action string) error {
	if b.protosClient.CanProvision {
		return nil
	}
	return fmt.Errorf("cannot %s: this protosd instance does not have the provisioner capability", action)
}

func (b *Backend) Init(ctx context.Context, in *pbApic.InitRequest) (*pbApic.InitResponse, error) {

	err := b.protosClient.Init(in.Username, in.Name, in.Organization)
	if err != nil {
		return nil, fmt.Errorf("failed to do local init: %w", err)
	}
	return &pbApic.InitResponse{}, nil
}

//
// User
//

func (b *Backend) GetUserDevices(ctx context.Context, in *pbApic.GetUserDevicesRequest) (*pbApic.GetUserDevicesResponse, error) {
	log.Debugf("Retrieving user devices")
	userDevices, err := b.protosClient.Manager.GetAllDevices(false)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user devices: %w", err)
	}
	resp := pbApic.GetUserDevicesResponse{}

	for _, device := range userDevices {
		wgPubKey := "n/a"
		wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(device.PublicKey)
		if err != nil {
			log.Error(err.Error())
		} else {
			wgPubKey = wgPublicKey.String()
		}

		respDevice := pbApic.UserDevice{
			Name:               device.Name,
			Id:                 device.ID,
			PublicKey:          device.PublicKey,
			PublicKeyWireguard: wgPubKey,
		}
		resp.Devices = append(resp.Devices, &respDevice)
	}

	return &resp, nil
}

func (b *Backend) GetUserInfo(ctx context.Context, in *pbApic.GetUserInfoRequest) (*pbApic.GetUserInfoResponse, error) {
	log.Debugf("Retrieving user info")
	adminUser, err := b.protosClient.Manager.GetAdmin()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user info: %w", err)
	}

	resp := pbApic.GetUserInfoResponse{
		Username: adminUser.Username,
		Name:     adminUser.Name,
		IsAdmin:  adminUser.IsAdmin(),
	}

	return &resp, nil
}

func (b *Backend) GetLocalSSHKey(ctx context.Context, in *pbApic.GetLocalSSHKeyRequest) (*pbApic.GetLocalSSHKeyResponse, error) {
	log.Debugf("Retrieving user info")
	key, err := b.protosClient.KeyManager.GetLocalKey()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve local key: %w", err)
	}

	resp := pbApic.GetLocalSSHKeyResponse{
		Public:  key.AuthorizedKey(),
		Private: key.EncodePrivateKeytoPEM(),
	}

	return &resp, nil
}

//
// App methods
//

func (b *Backend) GetApps(ctx context.Context, in *pbApic.GetAppsRequest) (*pbApic.GetAppsResponse, error) {

	log.Debugf("Retrieving apps")
	apps, err := b.protosClient.AppManager.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve apps: %w", err)
	}

	resp := pbApic.GetAppsResponse{}
	for _, app := range apps {
		status := "n/a"
		client, err := b.protosClient.P2PManager.GetClient(app.InstanceID)
		if err != nil {
			log.Errorf("Failed to retrieve status for app '%s': %s", app.Name, err.Error())
		} else {
			// FIXME: run this in parallel for all apps
			resp, err := client.GetAppStatus(context.TODO(), &p2pproto.GetAppStatusRequest{AppName: app.Name})
			if err != nil {
				log.Errorf("Failed to retrieve status for app '%s': %s", app.Name, err.Error())
			} else {
				status = resp.Status
			}
		}

		respApp := pbApic.App{
			Id:           app.ID,
			Name:         app.Name,
			Version:      app.GetVersion(),
			Status:       fmt.Sprintf("%s (%s)", status, app.DesiredStatus),
			InstanceName: app.InstanceID,
			Ip:           app.IPString(),
			Installer:    app.InstallerRef,
			Persistence:  app.Persistence,
		}
		resp.Apps = append(resp.Apps, &respApp)
	}

	return &resp, nil
}

func (b *Backend) CreateApp(ctx context.Context, in *pbApic.CreateAppRequest) (*pbApic.CreateAppResponse, error) {

	log.Debugf("Running app '%s' based on installer '%s', on instance '%s'", in.Name, in.InstallerId, in.InstanceId)
	_, err := b.protosClient.CloudManager.GetInstance(in.InstanceId)
	if err != nil {
		return nil, fmt.Errorf("failed to run app %s: %w", in.Name, err)
	}

	// FIXME: read the installer params from the command line
	app, err := b.protosClient.AppManager.Create(in.InstallerId, in.Name, in.InstanceId, in.Persistence, map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to run app %s: %w", in.Name, err)
	}

	return &pbApic.CreateAppResponse{Id: app.ID}, nil
}

func (b *Backend) StartApp(ctx context.Context, in *pbApic.StartAppRequest) (*pbApic.StartAppResponse, error) {
	log.Debugf("Starting app '%s'", in.Name)
	err := b.protosClient.AppManager.Start(in.Name)
	if err != nil {
		return nil, err
	}

	return &pbApic.StartAppResponse{}, nil
}

func (b *Backend) StopApp(ctx context.Context, in *pbApic.StopAppRequest) (*pbApic.StopAppResponse, error) {
	log.Debugf("Stopping app '%s'", in.Name)
	err := b.protosClient.AppManager.Stop(in.Name)
	if err != nil {
		return nil, err
	}

	return &pbApic.StopAppResponse{}, nil
}

func (b *Backend) RemoveApp(ctx context.Context, in *pbApic.RemoveAppRequest) (*pbApic.RemoveAppResponse, error) {
	log.Debugf("Removing app '%s'", in.Name)
	err := b.protosClient.AppManager.Remove(in.Name)
	if err != nil {
		return nil, err
	}

	return &pbApic.RemoveAppResponse{}, nil
}

func (b *Backend) GetAppLogs(ctx context.Context, in *pbApic.GetAppLogsRequest) (*pbApic.GetAppLogsResponse, error) {
	log.Debugf("Retrieveing logs for app '%s'", in.Name)

	app, err := b.protosClient.AppManager.Get(in.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve logs for app '%s': %w", in.Name, err)
	}

	client, err := b.protosClient.P2PManager.GetClient(app.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve logs for app '%s': %w", in.Name, err)
	}

	resp, err := client.GetAppLogs(context.TODO(), &p2pproto.GetAppLogsRequest{AppName: app.Name})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve logs for app '%s': %w", in.Name, err)
	}

	base64Logs, err := base64.StdEncoding.DecodeString(resp.Logs)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve logs for app '%s': %w", in.Name, err)
	}

	return &pbApic.GetAppLogsResponse{Logs: []byte(base64Logs)}, nil
}

//
// Cloud provider methods
//

func (b *Backend) GetSupportedCloudProviders(ctx context.Context, in *pbApic.GetSupportedCloudProvidersRequest) (*pbApic.GetSupportedCloudProvidersResponse, error) {
	log.Debug("Retrieving supported cloud providers")
	supportedCloudProviders := b.protosClient.CloudManager.SupportedProviders()

	resp := pbApic.GetSupportedCloudProvidersResponse{}
	for _, supportedCloudProvider := range supportedCloudProviders {
		authFields, err := b.protosClient.CloudManager.ProviderAuthFields(supportedCloudProvider)
		if err != nil {
			return nil, err
		}
		respCloudType := pbApic.CloudType{
			Name:                 supportedCloudProvider,
			AuthenticationFields: authFields,
		}
		resp.CloudTypes = append(resp.CloudTypes, &respCloudType)
	}

	return &resp, nil
}

func (b *Backend) GetCloudProviders(ctx context.Context, in *pbApic.GetCloudProvidersRequest) (*pbApic.GetCloudProvidersResponse, error) {
	log.Debug("Retrieving cloud providers")
	cloudProviders, err := b.protosClient.CloudManager.GetProviders()
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetCloudProvidersResponse{}
	for _, cloudProvider := range cloudProviders {
		respCloudProvider := pbApic.CloudProvider{
			Name: cloudProvider.NameStr(),
			Type: &pbApic.CloudType{
				Name:                 cloudProvider.TypeStr(),
				AuthenticationFields: cloudProvider.AuthFields(),
			},
		}
		resp.CloudProviders = append(resp.CloudProviders, &respCloudProvider)
	}

	return &resp, nil
}

func (b *Backend) GetCloudProvider(ctx context.Context, in *pbApic.GetCloudProviderRequest) (*pbApic.GetCloudProviderResponse, error) {
	log.Debugf("Retrieving cloud provider '%s'", in.Name)
	cloudProvider, err := b.protosClient.CloudManager.GetProvider(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud provider: %w", err)
	}

	computeProvider, ok := cloudProvider.(provisioners.ComputeProvider)
	if !ok {
		return nil, fmt.Errorf("cloud provider '%s'(%s) does not support compute operations", in.Name, cloudProvider.TypeStr())
	}
	// initialize cloud provider before use
	err = cloudProvider.Init()
	if err != nil {
		return nil, fmt.Errorf("error reaching cloud provider '%s'(%s) API: %w", in.Name, cloudProvider.TypeStr(), err)
	}

	supportedLocations := computeProvider.SupportedLocations()
	if len(supportedLocations) == 0 {
		return nil, fmt.Errorf("cloud provider '%s'(%s) does not report any supported locations", in.Name, cloudProvider.TypeStr())
	}
	supportedMachines, err := computeProvider.SupportedMachines(supportedLocations[0])
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve supported machines: %w", err)
	}

	respSupportedMachines := map[string]*pbApic.CloudMachineSpec{}
	for name, supportedMachine := range supportedMachines {
		respSupportedMachines[name] = &pbApic.CloudMachineSpec{
			Cores:                int32(supportedMachine.Cores),
			Memory:               int32(supportedMachine.Memory),
			DefaultStorage:       int32(supportedMachine.DefaultStorage),
			Bandwidth:            int32(supportedMachine.Bandwidth),
			IncludedDataTransfer: int32(supportedMachine.IncludedDataTransfer),
			Baremetal:            supportedMachine.Baremetal,
			PriceMonthly:         supportedMachine.PriceMonthly,
		}
	}

	resp := pbApic.GetCloudProviderResponse{
		CloudProvider: &pbApic.CloudProvider{
			Name:               cloudProvider.NameStr(),
			SupportedLocations: supportedLocations,
			SupportedMachines:  respSupportedMachines,
			Type: &pbApic.CloudType{
				Name:                 cloudProvider.TypeStr(),
				AuthenticationFields: cloudProvider.AuthFields(),
			},
		},
	}
	return &resp, nil
}

func (b *Backend) AddCloudProvider(ctx context.Context, in *pbApic.AddCloudProviderRequest) (*pbApic.AddCloudProviderResponse, error) {
	if err := b.requireProvisionerCapability("add cloud provider"); err != nil {
		return nil, err
	}
	if err := b.protosClient.CloudManager.AddProvider(in.Name, in.Type, in.Credentials); err != nil {
		return nil, err
	}
	return &pbApic.AddCloudProviderResponse{}, nil
}

func (b *Backend) RemoveCloudProvider(ctx context.Context, in *pbApic.RemoveCloudProviderRequest) (*pbApic.RemoveCloudProviderResponse, error) {
	if err := b.requireProvisionerCapability("remove cloud provider"); err != nil {
		return nil, err
	}
	// delete existing cloud provider
	err := b.protosClient.CloudManager.DeleteProvider(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete cloud provider '%s': %w", in.Name, err)
	}

	return &pbApic.RemoveCloudProviderResponse{}, nil
}

func (b *Backend) GetSupportedProvisioners(ctx context.Context, in *pbApic.GetSupportedProvisionersRequest) (*pbApic.GetSupportedProvisionersResponse, error) {
	log.Debug("Retrieving supported provisioners")
	supportedProvisioners := b.protosClient.CloudManager.SupportedProvisioners()

	resp := pbApic.GetSupportedProvisionersResponse{}
	for _, supportedProvisioner := range supportedProvisioners {
		authFields, err := b.protosClient.CloudManager.ProvisionerAuthFields(supportedProvisioner)
		if err != nil {
			return nil, err
		}
		resp.ProvisionerTypes = append(resp.ProvisionerTypes, &pbApic.ProvisionerType{
			Name:                 supportedProvisioner,
			AuthenticationFields: authFields,
		})
	}

	return &resp, nil
}

func (b *Backend) GetProvisioners(ctx context.Context, in *pbApic.GetProvisionersRequest) (*pbApic.GetProvisionersResponse, error) {
	log.Debug("Retrieving provisioners")
	provisioners, err := b.protosClient.CloudManager.GetProvisioners()
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetProvisionersResponse{}
	for _, provisioner := range provisioners {
		resp.Provisioners = append(resp.Provisioners, &pbApic.Provisioner{
			Name: provisioner.NameStr(),
			Type: &pbApic.ProvisionerType{
				Name:                 provisioner.TypeStr(),
				AuthenticationFields: provisioner.AuthFields(),
			},
		})
	}

	return &resp, nil
}

func (b *Backend) GetProvisioner(ctx context.Context, in *pbApic.GetProvisionerRequest) (*pbApic.GetProvisionerResponse, error) {
	log.Debugf("Retrieving provisioner '%s'", in.Name)
	provisioner, err := b.protosClient.CloudManager.GetProvisioner(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioner: %w", err)
	}

	computeProvisioner, ok := provisioner.(provisioners.ComputeProvisioner)
	if !ok {
		return nil, fmt.Errorf("provisioner '%s'(%s) does not support compute operations", in.Name, provisioner.TypeStr())
	}
	if err := provisioner.Init(); err != nil {
		return nil, fmt.Errorf("error reaching provisioner '%s'(%s) API: %w", in.Name, provisioner.TypeStr(), err)
	}

	supportedLocations := computeProvisioner.SupportedLocations()
	if len(supportedLocations) == 0 {
		return nil, fmt.Errorf("provisioner '%s'(%s) does not report any supported locations", in.Name, provisioner.TypeStr())
	}
	supportedMachines, err := computeProvisioner.SupportedMachines(supportedLocations[0])
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve supported machines: %w", err)
	}

	return &pbApic.GetProvisionerResponse{
		Provisioner: &pbApic.Provisioner{
			Name:               provisioner.NameStr(),
			SupportedLocations: supportedLocations,
			SupportedMachines:  provisionerMachineSpecs(supportedMachines),
			Type: &pbApic.ProvisionerType{
				Name:                 provisioner.TypeStr(),
				AuthenticationFields: provisioner.AuthFields(),
			},
		},
	}, nil
}

func (b *Backend) AddProvisioner(ctx context.Context, in *pbApic.AddProvisionerRequest) (*pbApic.AddProvisionerResponse, error) {
	if err := b.requireProvisionerCapability("add provisioner"); err != nil {
		return nil, err
	}
	if err := b.protosClient.CloudManager.AddProvisioner(in.Name, in.Type, in.Credentials); err != nil {
		return nil, err
	}
	return &pbApic.AddProvisionerResponse{}, nil
}

func (b *Backend) RemoveProvisioner(ctx context.Context, in *pbApic.RemoveProvisionerRequest) (*pbApic.RemoveProvisionerResponse, error) {
	if err := b.requireProvisionerCapability("remove provisioner"); err != nil {
		return nil, err
	}
	if err := b.protosClient.CloudManager.DeleteProvisioner(in.Name); err != nil {
		return nil, fmt.Errorf("failed to delete provisioner '%s': %w", in.Name, err)
	}
	return &pbApic.RemoveProvisionerResponse{}, nil
}

func provisionerMachineSpecs(machineSpecs map[string]provisioners.MachineSpec) map[string]*pbApic.ProvisionerMachineSpec {
	respSupportedMachines := map[string]*pbApic.ProvisionerMachineSpec{}
	for name, supportedMachine := range machineSpecs {
		respSupportedMachines[name] = &pbApic.ProvisionerMachineSpec{
			Cores:                int32(supportedMachine.Cores),
			Memory:               int32(supportedMachine.Memory),
			DefaultStorage:       int32(supportedMachine.DefaultStorage),
			Bandwidth:            int32(supportedMachine.Bandwidth),
			IncludedDataTransfer: int32(supportedMachine.IncludedDataTransfer),
			Baremetal:            supportedMachine.Baremetal,
			PriceMonthly:         supportedMachine.PriceMonthly,
		}
	}
	return respSupportedMachines
}

//
// Cloud instance methods
//

func (b *Backend) GetInstances(ctx context.Context, in *pbApic.GetInstancesRequest) (*pbApic.GetInstancesResponse, error) {
	log.Debugf("Retrieving instances")
	var (
		instances []provisioners.InstanceInfo
		err       error
	)
	if b.protosClient.CanProvision {
		instances, err = b.protosClient.CloudManager.GetInstancesWithUpdatedStatus()
	} else {
		instances, err = b.protosClient.CloudManager.GetInstances(false)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	resp := pbApic.GetInstancesResponse{}
	for _, instance := range instances {

		wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			log.Error(err.Error())
		}

		pubKey, err := pcrypto.CreatePublicKeyFromBase64(instance.PublicKey)
		if err != nil {
			log.Error(err.Error())
		}

		cloudName := "local"
		if instance.Kind == provisioners.KindCloudVM {
			provider, err := b.protosClient.CloudManager.GetProvider(instance.KindID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve cloud provider: %w", err)
			}
			cloudName = provider.NameStr()
		}
		respInstance := pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         pubKey.IPv6Address().StringExpanded(),
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey.String(),
			Architecture:       instance.Architecture,
			Status:             instance.Status,
			CloudName:          cloudName,
		}
		resp.Instances = append(resp.Instances, &respInstance)
	}

	return &resp, nil
}

func (b *Backend) GetInstance(ctx context.Context, in *pbApic.GetInstanceRequest) (*pbApic.GetInstanceResponse, error) {
	log.Debugf("Retrieving instance '%s'", in.Name)
	instance, err := b.protosClient.CloudManager.GetInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve instance '%s': %w", in.Name, err)
	}

	wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
	if err != nil {
		log.Error(err.Error())
	}

	pubKey, err := pcrypto.CreatePublicKeyFromBase64(instance.PublicKey)
	if err != nil {
		log.Error(err.Error())
	}

	var status string
	peers := map[string]string{}
	client, err := b.protosClient.P2PManager.GetClient(instance.Name)
	if err != nil {
		log.Error(err.Error())
		if peerState, found := b.protosClient.P2PManager.GetPeerState(instance.Name); found {
			status = fmt.Sprintf("%s (%s)", instance.Status, peerState.Reachability())
			if peerState.LastError != "" {
				log.Debugf("last p2p error for instance '%s': %s", instance.Name, peerState.LastError)
			}
		} else {
			status = fmt.Sprintf("%s (%s)", instance.Status, "unreachable")
		}
	} else {
		resp, err := client.GetPeers(context.TODO(), &p2pproto.GetPeersRequest{})
		if err != nil {
			log.Error(err.Error())
			status = fmt.Sprintf("%s (%s)", instance.Status, "unreachable")
		} else {
			status = fmt.Sprintf("%s (%s)", instance.Status, "reachable")
			for name, peer := range resp.Peers {
				peers[name] = peer
			}
		}
	}

	resp := pbApic.GetInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         pubKey.IPv6Address().StringExpanded(),
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey.String(),
			Status:             status,
			Architecture:       instance.Architecture,
			Peers:              peers,
		},
	}

	return &resp, nil
}

func (b *Backend) DeployInstance(ctx context.Context, in *pbApic.DeployInstanceRequest) (*pbApic.DeployInstanceResponse, error) {
	if err := b.requireProvisionerCapability("deploy instance"); err != nil {
		return nil, err
	}
	log.Debugf("Deploying new instance '%s'", in.Name)

	rls := release.Release{}
	var err error
	if in.DevImg != "" {
		rls.Version = in.DevImg
	} else {
		releases, err := b.protosClient.GetProtosAvailableReleases()
		if err != nil {
			return nil, err
		}
		if in.ProtosVersion != "" {
			rls, err = releases.GetVersion(in.ProtosVersion)
			if err != nil {
				return nil, err
			}
		} else {
			rls, err = releases.GetLatest()
			if err != nil {
				return nil, err
			}
		}
	}

	instance, err := b.protosClient.CloudManager.DeployInstance(in.Name, in.CloudName, in.CloudLocation, rls, in.MachineType)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy instance '%s': %w", in.Name, err)
	}

	wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy instance '%s': %w", in.Name, err)
	}

	pubKey, err := pcrypto.CreatePublicKeyFromBase64(instance.PublicKey)
	if err != nil {
		log.Error(err.Error())
	}

	resp := pbApic.DeployInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         pubKey.IPv6Address().StringExpanded(),
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey.String(),
			Status:             instance.Status,
		},
	}

	return &resp, nil
}

func (b *Backend) RemoveInstance(ctx context.Context, in *pbApic.RemoveInstanceRequest) (*pbApic.RemoveInstanceResponse, error) {
	if err := b.requireProvisionerCapability("remove instance"); err != nil {
		return nil, err
	}
	log.Debugf("Removing instance '%s'", in.Name)
	var err error
	if in.LocalOnly {
		err = b.protosClient.CloudManager.DeleteInstanceLocal(in.Name)
	} else {
		err = b.protosClient.CloudManager.DeleteInstance(in.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove instance '%s': %w", in.Name, err)
	}

	return &pbApic.RemoveInstanceResponse{}, nil
}

func (b *Backend) StartInstance(ctx context.Context, in *pbApic.StartInstanceRequest) (*pbApic.StartInstanceResponse, error) {
	if err := b.requireProvisionerCapability("start instance"); err != nil {
		return nil, err
	}
	log.Debugf("Starting instance '%s'", in.Name)
	err := b.protosClient.CloudManager.StartInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to start instance '%s': %w", in.Name, err)
	}
	return &pbApic.StartInstanceResponse{}, nil
}

func (b *Backend) StopInstance(ctx context.Context, in *pbApic.StopInstanceRequest) (*pbApic.StopInstanceResponse, error) {
	if err := b.requireProvisionerCapability("stop instance"); err != nil {
		return nil, err
	}
	log.Debugf("Stopping instance '%s'", in.Name)
	err := b.protosClient.CloudManager.StopInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to stop instance '%s': %w", in.Name, err)
	}
	return &pbApic.StopInstanceResponse{}, nil
}

func (b *Backend) GetInstanceKey(ctx context.Context, in *pbApic.GetInstanceKeyRequest) (*pbApic.GetInstanceKeyResponse, error) {
	log.Debugf("Retrieving key for instance '%s'", in.Name)
	key, err := b.protosClient.CloudManager.GetInstanceSSHKey(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve key for instance '%s': %w", in.Name, err)
	}
	return &pbApic.GetInstanceKeyResponse{Key: key}, nil
}

func (b *Backend) GetInstanceLogs(ctx context.Context, in *pbApic.GetInstanceLogsRequest) (*pbApic.GetInstanceLogsResponse, error) {
	log.Debugf("Retrieving logs for instance '%s'", in.Name)

	client, err := b.protosClient.P2PManager.GetClient(in.Name)
	if err != nil {
		return b.getInstanceLogsViaSSH(in.Name, err)
	}

	logs, err := client.GetLogs(context.TODO(), &p2pproto.GetLogsRequest{})
	if err != nil {
		return b.getInstanceLogsViaSSH(in.Name, err)
	}
	base64Logs, err := base64.StdEncoding.DecodeString(logs.Logs)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve instance '%s' logs: %w", in.Name, err)
	}

	return &pbApic.GetInstanceLogsResponse{Logs: string(base64Logs)}, nil
}

func (b *Backend) getInstanceLogsViaSSH(instanceName string, p2pErr error) (*pbApic.GetInstanceLogsResponse, error) {
	log.Debugf("Falling back to SSH logs for instance '%s' after p2p log retrieval failed: %s", instanceName, p2pErr.Error())
	logs, err := b.protosClient.CloudManager.LogsRemoteInstance(instanceName)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve instance '%s' logs: p2p failed: %w; ssh fallback failed: %w", instanceName, p2pErr, err)
	}
	return &pbApic.GetInstanceLogsResponse{Logs: logs}, nil
}

func (b *Backend) InitInstance(ctx context.Context, in *pbApic.InitInstanceRequest) (*pbApic.InitInstanceResponse, error) {
	if err := b.requireProvisionerCapability("initialize instance"); err != nil {
		return nil, err
	}
	log.Debugf("Initializing local instance '%s' at '%s'", in.Name, in.Ip)

	err := b.protosClient.CloudManager.InitInstance(in.Name, provisioners.KindLocalVM, "local-id", "local", in.Ip)
	if err != nil {
		return nil, fmt.Errorf("could not initialize instance '%s': %w", in.Name, err)
	}
	return &pbApic.InitInstanceResponse{}, nil
}

func (b *Backend) UpdateInstance(ctx context.Context, in *pbApic.UpdateInstanceRequest) (*pbApic.UpdateInstanceResponse, error) {
	if err := b.requireProvisionerCapability("update instance"); err != nil {
		return nil, err
	}
	log.Debugf("Updating instance '%s' to ip '%s'", in.Id, in.Ip)

	err := b.protosClient.CloudManager.UpdateInstance(in.Id, in.Ip)
	if err != nil {
		return nil, fmt.Errorf("failed to update instance '%s': %w", in.Id, err)
	}

	return &pbApic.UpdateInstanceResponse{}, nil
}

func (b *Backend) GetNetworkState(ctx context.Context, in *pbApic.GetNetworkStateRequest) (*pbApic.GetNetworkStateResponse, error) {
	instanceName := in.GetInstance()
	if instanceName == "" || instanceName == "local" {
		if b.protosClient.NetworkManager == nil {
			return nil, fmt.Errorf("network manager is not configured")
		}
		state, err := b.protosClient.NetworkManager.State()
		if err != nil {
			return nil, err
		}
		return &pbApic.GetNetworkStateResponse{State: networkStateToProto(state)}, nil
	}

	client, err := b.protosClient.P2PManager.GetClient(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance '%s' admin API: %w", instanceName, err)
	}
	resp, err := client.GetNetworkState(ctx, &p2pproto.GetNetworkStateRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve network state from instance '%s': %w", instanceName, err)
	}
	return &pbApic.GetNetworkStateResponse{State: networkStateFromP2PProto(resp.GetState())}, nil
}

func (b *Backend) GetExitRoutes(ctx context.Context, in *pbApic.GetExitRoutesRequest) (*pbApic.GetExitRoutesResponse, error) {
	instanceName := in.GetInstance()
	if instanceName != "" && instanceName != "local" {
		client, err := b.protosClient.P2PManager.GetClient(instanceName)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to instance '%s' admin API: %w", instanceName, err)
		}
		remote, err := client.GetExitRoutes(ctx, &p2pproto.GetExitRoutesRequest{})
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve exit routes from instance '%s': %w", instanceName, err)
		}
		resp := &pbApic.GetExitRoutesResponse{}
		for _, route := range remote.GetRoutes() {
			resp.Routes = append(resp.Routes, b.exitRouteToProto(network.ExitRoute{
				ID:            route.GetId(),
				DeviceID:      route.GetDeviceId(),
				InstanceID:    route.GetInstanceId(),
				DesiredStatus: route.GetStatus(),
				DNSServer:     route.GetDnsServer(),
				CIDRs:         append([]string(nil), route.GetCidrs()...),
			}))
		}
		return resp, nil
	}

	routes, err := network.GetExitRoutes(b.protosClient.DB)
	if err != nil {
		return nil, err
	}

	resp := &pbApic.GetExitRoutesResponse{}
	for _, route := range routes {
		resp.Routes = append(resp.Routes, b.exitRouteToProto(route))
	}
	return resp, nil
}

func (b *Backend) GetRuntimeState(ctx context.Context, in *pbApic.GetRuntimeStateRequest) (*pbApic.GetRuntimeStateResponse, error) {
	instanceName := in.GetInstance()
	if instanceName == "" || instanceName == "local" {
		state, err := b.localRuntimeState(ctx)
		if err != nil {
			return nil, err
		}
		return &pbApic.GetRuntimeStateResponse{State: state}, nil
	}

	client, err := b.protosClient.P2PManager.GetClient(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance '%s' admin API: %w", instanceName, err)
	}
	resp, err := client.GetRuntimeState(ctx, &p2pproto.GetRuntimeStateRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve runtime state from instance '%s': %w", instanceName, err)
	}
	return &pbApic.GetRuntimeStateResponse{State: runtimeStateFromP2PProto(resp.GetState())}, nil
}

func (b *Backend) WatchChanges(in *pbApic.WatchChangesRequest, stream pbApic.ProtosClientApi_WatchChangesServer) error {
	if b.protosClient == nil || b.protosClient.DB == nil {
		return fmt.Errorf("database is not configured")
	}

	ctx := stream.Context()
	changes, cancel := b.protosClient.DB.WatchChanges(ctx)
	defer cancel()

	var sequence uint64
	send := func(reason string, tableNames []string, runtimeChanged bool) error {
		sequence++
		return stream.Send(&pbApic.WatchChangesResponse{
			Sequence:       sequence,
			TableNames:     append([]string(nil), tableNames...),
			RuntimeChanged: runtimeChanged,
			Reason:         reason,
		})
	}

	if in.GetIncludeSnapshot() {
		if err := send("initial", nil, true); err != nil {
			return err
		}
	}

	var ticker *time.Ticker
	var ticks <-chan time.Time
	if heartbeatMs := in.GetHeartbeatIntervalMs(); heartbeatMs > 0 {
		interval := time.Duration(heartbeatMs) * time.Millisecond
		if interval < time.Second {
			interval = time.Second
		}
		ticker = time.NewTicker(interval)
		ticks = ticker.C
		defer ticker.Stop()
	}

	for {
		select {
		case event, ok := <-changes:
			if !ok {
				return nil
			}
			runtimeChanged := len(event.TableNames) == 0
			reason := "db"
			if runtimeChanged {
				reason = "runtime"
			}
			if err := send(reason, event.TableNames, runtimeChanged); err != nil {
				return err
			}
		case <-ticks:
			if err := send("heartbeat", nil, true); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func networkStateToProto(state networkmodule.State) *pbApic.NetworkState {
	out := &pbApic.NetworkState{
		Module:        state.Module,
		Up:            state.Up,
		InterfaceName: state.InterfaceName,
		Messages:      append([]string(nil), state.Messages...),
	}
	for _, item := range state.Interfaces {
		out.Interfaces = append(out.Interfaces, &pbApic.NetworkInterface{
			Name:       item.Name,
			Type:       item.Type,
			Index:      int32(item.Index),
			Mtu:        int32(item.MTU),
			Up:         item.Up,
			Master:     item.Master,
			MacAddress: item.MacAddress,
			Kind:       item.Kind,
		})
	}
	for _, item := range state.Addresses {
		out.Addresses = append(out.Addresses, &pbApic.NetworkAddress{
			InterfaceName: item.InterfaceName,
			Cidr:          item.CIDR,
			Scope:         item.Scope,
		})
	}
	for _, item := range state.Routes {
		out.Routes = append(out.Routes, &pbApic.NetworkRoute{
			InterfaceName: item.InterfaceName,
			Destination:   item.Destination,
			Gateway:       item.Gateway,
			Source:        item.Source,
			Family:        item.Family,
			Table:         item.Table,
			Protocol:      item.Protocol,
			Scope:         item.Scope,
			Priority:      item.Priority,
			Kind:          item.Kind,
		})
	}
	for _, item := range state.WireGuardPeers {
		out.WireguardPeers = append(out.WireguardPeers, &pbApic.WireGuardPeer{
			PublicKey:       item.PublicKey,
			Endpoint:        item.Endpoint,
			AllowedIps:      append([]string(nil), item.AllowedIPs...),
			LatestHandshake: item.LatestHandshake,
			RxBytes:         item.RxBytes,
			TxBytes:         item.TxBytes,
		})
	}
	for _, table := range state.FirewallTables {
		tableProto := &pbApic.FirewallTable{
			Family: table.Family,
			Name:   table.Name,
		}
		for _, chain := range table.Chains {
			chainProto := &pbApic.FirewallChain{
				Name:     chain.Name,
				Type:     chain.Type,
				Hook:     chain.Hook,
				Priority: chain.Priority,
			}
			for _, rule := range chain.Rules {
				chainProto.Rules = append(chainProto.Rules, &pbApic.FirewallRule{
					Expressions: append([]string(nil), rule.Expressions...),
					Packets:     rule.Packets,
					Bytes:       rule.Bytes,
				})
			}
			tableProto.Chains = append(tableProto.Chains, chainProto)
		}
		out.FirewallTables = append(out.FirewallTables, tableProto)
	}
	for _, item := range state.DNS {
		out.Dns = append(out.Dns, &pbApic.DNSState{
			Scope:   item.Scope,
			Domain:  item.Domain,
			Servers: append([]string(nil), item.Servers...),
			Port:    int32(item.Port),
			Active:  item.Active,
			Source:  item.Source,
		})
	}
	return out
}

func networkStateFromP2PProto(state *p2pproto.NetworkState) *pbApic.NetworkState {
	if state == nil {
		return nil
	}
	out := &pbApic.NetworkState{
		Module:        state.GetModule(),
		Up:            state.GetUp(),
		InterfaceName: state.GetInterfaceName(),
		Messages:      append([]string(nil), state.GetMessages()...),
	}
	for _, item := range state.GetInterfaces() {
		out.Interfaces = append(out.Interfaces, &pbApic.NetworkInterface{
			Name:       item.GetName(),
			Type:       item.GetType(),
			Index:      item.GetIndex(),
			Mtu:        item.GetMtu(),
			Up:         item.GetUp(),
			Master:     item.GetMaster(),
			MacAddress: item.GetMacAddress(),
			Kind:       item.GetKind(),
		})
	}
	for _, item := range state.GetAddresses() {
		out.Addresses = append(out.Addresses, &pbApic.NetworkAddress{
			InterfaceName: item.GetInterfaceName(),
			Cidr:          item.GetCidr(),
			Scope:         item.GetScope(),
		})
	}
	for _, item := range state.GetRoutes() {
		out.Routes = append(out.Routes, &pbApic.NetworkRoute{
			InterfaceName: item.GetInterfaceName(),
			Destination:   item.GetDestination(),
			Gateway:       item.GetGateway(),
			Source:        item.GetSource(),
			Family:        item.GetFamily(),
			Table:         item.GetTable(),
			Protocol:      item.GetProtocol(),
			Scope:         item.GetScope(),
			Priority:      item.GetPriority(),
			Kind:          item.GetKind(),
		})
	}
	for _, item := range state.GetWireguardPeers() {
		out.WireguardPeers = append(out.WireguardPeers, &pbApic.WireGuardPeer{
			PublicKey:       item.GetPublicKey(),
			Endpoint:        item.GetEndpoint(),
			AllowedIps:      append([]string(nil), item.GetAllowedIps()...),
			LatestHandshake: item.GetLatestHandshake(),
			RxBytes:         item.GetRxBytes(),
			TxBytes:         item.GetTxBytes(),
		})
	}
	for _, table := range state.GetFirewallTables() {
		tableProto := &pbApic.FirewallTable{
			Family: table.GetFamily(),
			Name:   table.GetName(),
		}
		for _, chain := range table.GetChains() {
			chainProto := &pbApic.FirewallChain{
				Name:     chain.GetName(),
				Type:     chain.GetType(),
				Hook:     chain.GetHook(),
				Priority: chain.GetPriority(),
			}
			for _, rule := range chain.GetRules() {
				chainProto.Rules = append(chainProto.Rules, &pbApic.FirewallRule{
					Expressions: append([]string(nil), rule.GetExpressions()...),
					Packets:     rule.GetPackets(),
					Bytes:       rule.GetBytes(),
				})
			}
			tableProto.Chains = append(tableProto.Chains, chainProto)
		}
		out.FirewallTables = append(out.FirewallTables, tableProto)
	}
	for _, item := range state.GetDns() {
		out.Dns = append(out.Dns, &pbApic.DNSState{
			Scope:   item.GetScope(),
			Domain:  item.GetDomain(),
			Servers: append([]string(nil), item.GetServers()...),
			Port:    item.GetPort(),
			Active:  item.GetActive(),
			Source:  item.GetSource(),
		})
	}
	return out
}

func (b *Backend) localRuntimeState(ctx context.Context) (*pbApic.RuntimeState, error) {
	if err := b.protosClient.DB.CatchUpFinalized(ctx, "apic get runtime state"); err != nil {
		return nil, err
	}
	status, ok := b.protosClient.DB.SwarmionStatus()
	if !ok {
		return nil, fmt.Errorf("swarmion status is not available")
	}
	out := &pbApic.RuntimeState{
		PeerId:                       status.PeerID,
		ManifestDigest:               status.ManifestDigest,
		FinalizedRootHash:            status.FinalizedRootHash.String(),
		TentativeRootHash:            status.TentativeRootHash.String(),
		ProtocolFinalizedRootHash:    status.RuntimeFinalizedDesiredRootHash.String(),
		DurableMainRootHash:          status.DurableMainRootHash.String(),
		ActiveEpochId:                status.ActiveEpochID,
		ActiveWitnessIds:             append([]string(nil), status.ActiveWitnessIDs...),
		EligibleWitnessIds:           append([]string(nil), status.EligibleWitnessIDs...),
		StateProviders:               append([]string(nil), status.StateProviders...),
		ConnectedPeers:               append([]string(nil), status.ConnectedPeers...),
		RuntimeRefreshPending:        status.RuntimeRefreshPending,
		RuntimeRefreshLastError:      status.RuntimeRefreshLastError,
		RuntimeFinalizedPending:      status.RuntimeFinalizedMaterializePending,
		RuntimeFinalizedLastError:    status.RuntimeFinalizedMaterializeLastError,
		RuntimeMaterializationPolicy: status.RuntimeFinalizedMaterializationPolicy.String(),
	}
	if status.Fatal != nil {
		out.FatalState = status.Fatal.State
	} else {
		out.FatalState = status.FatalState.String()
	}

	peerStatuses, err := b.protosClient.DB.SwarmionPeerStatus(ctx)
	if err != nil {
		return nil, err
	}
	for _, peerStatus := range peerStatuses {
		out.PeerStatuses = append(out.PeerStatuses, &pbApic.RuntimePeerStatus{
			PeerId:          peerStatus.PeerID,
			Connected:       peerStatus.Connected,
			Dialable:        peerStatus.Dialable,
			StateProvider:   peerStatus.StateProvider,
			Witness:         peerStatus.Witness,
			EligibleWitness: peerStatus.EligibleWitness,
			Compatible:      peerStatus.Compatible,
			Incompatible:    peerStatus.Incompatible,
			Ignored:         peerStatus.Ignored,
			RelayOnly:       peerStatus.RelayOnly,
			Addresses:       append([]string(nil), peerStatus.Addresses...),
			LastDialErrors:  cloneStringMap(peerStatus.LastDialErrors),
			Reason:          peerStatus.Reason,
		})
	}
	compatibility, err := b.protosClient.DB.SwarmionCompatibility(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range compatibility {
		out.Compatibility = append(out.Compatibility, &pbApic.RuntimeCompatibility{
			PeerId:       item.PeerID,
			LocalDigest:  item.LocalDigest,
			RemoteDigest: item.RemoteDigest,
			Compatible:   item.Compatible,
			Blocking:     item.Blocking,
			Reason:       item.Reason,
		})
	}
	if trace, ok := b.protosClient.DB.SwarmionContentSyncTrace(); ok {
		out.ContentSyncTrace = append([]string(nil), trace...)
	}
	return out, nil
}

func runtimeStateFromP2PProto(state *p2pproto.RuntimeState) *pbApic.RuntimeState {
	if state == nil {
		return nil
	}
	out := &pbApic.RuntimeState{
		PeerId:                       state.GetPeerId(),
		ManifestDigest:               state.GetManifestDigest(),
		FinalizedRootHash:            state.GetFinalizedRootHash(),
		TentativeRootHash:            state.GetTentativeRootHash(),
		ProtocolFinalizedRootHash:    state.GetProtocolFinalizedRootHash(),
		DurableMainRootHash:          state.GetDurableMainRootHash(),
		ActiveEpochId:                state.GetActiveEpochId(),
		ActiveWitnessIds:             append([]string(nil), state.GetActiveWitnessIds()...),
		EligibleWitnessIds:           append([]string(nil), state.GetEligibleWitnessIds()...),
		StateProviders:               append([]string(nil), state.GetStateProviders()...),
		ConnectedPeers:               append([]string(nil), state.GetConnectedPeers()...),
		FatalState:                   state.GetFatalState(),
		RuntimeRefreshPending:        state.GetRuntimeRefreshPending(),
		RuntimeRefreshLastError:      state.GetRuntimeRefreshLastError(),
		RuntimeFinalizedPending:      state.GetRuntimeFinalizedPending(),
		RuntimeFinalizedLastError:    state.GetRuntimeFinalizedLastError(),
		RuntimeMaterializationPolicy: state.GetRuntimeMaterializationPolicy(),
		ContentSyncTrace:             append([]string(nil), state.GetContentSyncTrace()...),
	}
	for _, peerStatus := range state.GetPeerStatuses() {
		out.PeerStatuses = append(out.PeerStatuses, &pbApic.RuntimePeerStatus{
			PeerId:          peerStatus.GetPeerId(),
			Connected:       peerStatus.GetConnected(),
			Dialable:        peerStatus.GetDialable(),
			StateProvider:   peerStatus.GetStateProvider(),
			Witness:         peerStatus.GetWitness(),
			EligibleWitness: peerStatus.GetEligibleWitness(),
			Compatible:      peerStatus.GetCompatible(),
			Incompatible:    peerStatus.GetIncompatible(),
			Ignored:         peerStatus.GetIgnored(),
			RelayOnly:       peerStatus.GetRelayOnly(),
			Addresses:       append([]string(nil), peerStatus.GetAddresses()...),
			LastDialErrors:  cloneStringMap(peerStatus.GetLastDialErrors()),
			Reason:          peerStatus.GetReason(),
		})
	}
	for _, item := range state.GetCompatibility() {
		out.Compatibility = append(out.Compatibility, &pbApic.RuntimeCompatibility{
			PeerId:       item.GetPeerId(),
			LocalDigest:  item.GetLocalDigest(),
			RemoteDigest: item.GetRemoteDigest(),
			Compatible:   item.GetCompatible(),
			Blocking:     item.GetBlocking(),
			Reason:       item.GetReason(),
		})
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func (b *Backend) SetExitRoute(ctx context.Context, in *pbApic.SetExitRouteRequest) (*pbApic.SetExitRouteResponse, error) {
	instanceRef := in.GetInstance()
	if instanceRef == "" {
		return nil, fmt.Errorf("instance is required")
	}
	instance, err := b.protosClient.CloudManager.GetInstance(instanceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve exit instance '%s': %w", instanceRef, err)
	}
	if !isPublicExitIP(instance.PublicIP) {
		return nil, fmt.Errorf("instance '%s' does not have a routable public IP", instance.Name)
	}

	deviceID := in.GetDeviceId()
	if deviceID == "" {
		currentDevice, err := b.protosClient.Manager.GetCurrentDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve current device: %w", err)
		}
		deviceID = currentDevice.ID
	}

	route, err := network.SetExitRoute(b.protosClient.DB, deviceID, instance.ID, in.GetDnsServer(), in.GetCidrs())
	if err != nil {
		return nil, fmt.Errorf("failed to set exit route: %w", err)
	}
	return &pbApic.SetExitRouteResponse{Route: b.exitRouteToProto(route)}, nil
}

func isPublicExitIP(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() {
		return false
	}
	for _, prefix := range nonPublicExitIPPrefixes {
		if prefix.Contains(addr) {
			return false
		}
	}
	return true
}

var nonPublicExitIPPrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("2001:db8::/32"),
}

func (b *Backend) ClearExitRoute(ctx context.Context, in *pbApic.ClearExitRouteRequest) (*pbApic.ClearExitRouteResponse, error) {
	deviceID := in.GetDeviceId()
	if deviceID == "" {
		currentDevice, err := b.protosClient.Manager.GetCurrentDevice()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve current device: %w", err)
		}
		deviceID = currentDevice.ID
	}
	if err := network.ClearExitRoute(b.protosClient.DB, deviceID); err != nil {
		return nil, fmt.Errorf("failed to clear exit route: %w", err)
	}
	return &pbApic.ClearExitRouteResponse{}, nil
}

func (b *Backend) exitRouteToProto(route network.ExitRoute) *pbApic.ExitRoute {
	resp := &pbApic.ExitRoute{
		Id:         route.ID,
		DeviceId:   route.DeviceID,
		InstanceId: route.InstanceID,
		Status:     route.DesiredStatus,
		DnsServer:  route.DNSServer,
		Cidrs:      route.CIDRs,
	}
	if b.protosClient == nil || b.protosClient.CloudManager == nil {
		return resp
	}
	instance, err := b.protosClient.CloudManager.GetInstance(route.InstanceID)
	if err != nil {
		log.Debugf("failed to enrich exit route %s: %s", route.ID, err.Error())
		return resp
	}
	resp.InstanceName = instance.Name
	resp.PublicIp = instance.PublicIP
	resp.Location = instance.Location
	return resp
}

//
// Releases methods
//

func (b *Backend) GetProtosdReleases(ctx context.Context, in *pbApic.GetProtosdReleasesRequest) (*pbApic.GetProtosdReleasesResponse, error) {
	log.Debug("Retrieving Protosd releases")
	releases, err := b.protosClient.GetProtosAvailableReleases()
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetProtosdReleasesResponse{}
	for _, release := range releases.Releases {
		respCloudImages := map[string]*pbApic.CloudImage{}
		for n, ci := range release.CloudImages {
			respCloudImage := pbApic.CloudImage{
				Provider:    ci.Provider,
				Digest:      ci.Digest,
				Url:         ci.URL,
				ReleaseDate: ci.ReleaseDate.Unix(),
			}
			respCloudImages[n] = &respCloudImage
		}
		respRelease := pbApic.Release{
			CloudImages: respCloudImages,
			Version:     release.Version,
			Description: release.Description,
			ReleaseDate: release.ReleaseDate.Unix(),
		}
		resp.Releases = append(resp.Releases, &respRelease)
	}
	return &resp, nil
}

func (b *Backend) GetCloudImages(ctx context.Context, in *pbApic.GetCloudImagesRequest) (*pbApic.GetCloudImagesResponse, error) {
	log.Debugf("Retrieving cloud images from cloud '%s'", in.Name)
	provider, err := b.protosClient.CloudManager.GetProvider(in.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve cloud '%s': %w", in.Name, err)
	}

	imageProvider, ok := provider.(provisioners.ImageProvider)
	if !ok {
		return nil, fmt.Errorf("cloud provider '%s'(%s) does not support image operations", in.Name, provider.TypeStr())
	}
	err = provider.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cloud provider '%s'(%s) API: %w", in.Name, provider.TypeStr(), err)
	}
	images, err := imageProvider.GetProtosImages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve cloud images from cloud '%s': %w", in.Name, err)
	}
	resp := pbApic.GetCloudImagesResponse{CloudImages: map[string]*pbApic.CloudSpecificImage{}}
	for id, image := range images {
		respImage := pbApic.CloudSpecificImage{
			Id:       image.ID,
			Name:     image.Name,
			Location: image.Location,
		}
		resp.CloudImages[id] = &respImage
	}
	return &resp, nil
}

func (b *Backend) GetProvisionerImages(ctx context.Context, in *pbApic.GetProvisionerImagesRequest) (*pbApic.GetProvisionerImagesResponse, error) {
	log.Debugf("Retrieving images from provisioner '%s'", in.Name)
	provisioner, err := b.protosClient.CloudManager.GetProvisioner(in.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve provisioner '%s': %w", in.Name, err)
	}

	imageProvisioner, ok := provisioner.(provisioners.ImageProvisioner)
	if !ok {
		return nil, fmt.Errorf("provisioner '%s'(%s) does not support image operations", in.Name, provisioner.TypeStr())
	}
	if err := provisioner.Init(); err != nil {
		return nil, fmt.Errorf("failed to connect to provisioner '%s'(%s) API: %w", in.Name, provisioner.TypeStr(), err)
	}
	images, err := imageProvisioner.GetProtosImages()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve images from provisioner '%s': %w", in.Name, err)
	}
	resp := pbApic.GetProvisionerImagesResponse{Images: map[string]*pbApic.CloudSpecificImage{}}
	for id, image := range images {
		resp.Images[id] = &pbApic.CloudSpecificImage{
			Id:       image.ID,
			Name:     image.Name,
			Location: image.Location,
		}
	}
	return &resp, nil
}

func (b *Backend) UploadCloudImage(ctx context.Context, in *pbApic.UploadCloudImageRequest) (*pbApic.UploadCloudImageResponse, error) {
	if err := b.requireProvisionerCapability("upload cloud image"); err != nil {
		return nil, err
	}
	log.Debugf("Uploading cloud image '%s'(%s) to cloud '%s'", in.ImageName, in.ImagePath, in.CloudName)
	return &pbApic.UploadCloudImageResponse{}, b.protosClient.CloudManager.UploadLocalImage(in.ImagePath, in.ImageName, in.CloudName, in.CloudLocation, time.Duration(in.Timeout)*time.Minute)
}

func (b *Backend) UploadProvisionerImage(ctx context.Context, in *pbApic.UploadProvisionerImageRequest) (*pbApic.UploadProvisionerImageResponse, error) {
	if err := b.requireProvisionerCapability("upload provisioner image"); err != nil {
		return nil, err
	}
	log.Debugf("Uploading image '%s'(%s) to provisioner '%s'", in.ImageName, in.ImagePath, in.ProvisionerName)
	return &pbApic.UploadProvisionerImageResponse{}, b.protosClient.CloudManager.UploadLocalImage(in.ImagePath, in.ImageName, in.ProvisionerName, in.Location, time.Duration(in.Timeout)*time.Minute)
}

func (b *Backend) RemoveCloudImage(ctx context.Context, in *pbApic.RemoveCloudImageRequest) (*pbApic.RemoveCloudImageResponse, error) {
	if err := b.requireProvisionerCapability("remove cloud image"); err != nil {
		return nil, err
	}
	log.Debugf("Removing cloud image '%s' from cloud '%s'", in.ImageName, in.CloudName)
	errMsg := fmt.Sprintf("failed to delete image '%s' from cloud '%s'", in.ImageName, in.CloudLocation)
	provider, err := b.protosClient.CloudManager.GetProvider(in.CloudName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvider, ok := provider.(provisioners.ImageProvider)
	if !ok {
		return nil, fmt.Errorf("%s: cloud provider '%s'(%s) does not support image operations", errMsg, in.CloudName, provider.TypeStr())
	}
	err = provider.Init()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	// delete image
	err = imageProvider.RemoveImage(in.ImageName, in.CloudLocation)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}
	return &pbApic.RemoveCloudImageResponse{}, nil
}

func (b *Backend) RemoveProvisionerImage(ctx context.Context, in *pbApic.RemoveProvisionerImageRequest) (*pbApic.RemoveProvisionerImageResponse, error) {
	if err := b.requireProvisionerCapability("remove provisioner image"); err != nil {
		return nil, err
	}
	log.Debugf("Removing image '%s' from provisioner '%s'", in.ImageName, in.ProvisionerName)
	errMsg := fmt.Sprintf("failed to delete image '%s' from provisioner '%s'", in.ImageName, in.ProvisionerName)
	provisioner, err := b.protosClient.CloudManager.GetProvisioner(in.ProvisionerName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvisioner, ok := provisioner.(provisioners.ImageProvisioner)
	if !ok {
		return nil, fmt.Errorf("%s: provisioner '%s'(%s) does not support image operations", errMsg, in.ProvisionerName, provisioner.TypeStr())
	}
	if err := provisioner.Init(); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := imageProvisioner.RemoveImage(in.ImageName, in.Location); err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}
	return &pbApic.RemoveProvisionerImageResponse{}, nil
}

//
// DVC methods
//

func (b *Backend) GetLocalCommits(ctx context.Context, in *pbApic.GetLocalCommitsRequest) (*pbApic.GetLocalCommitsResponse, error) {
	log.Debug("Retrieving local commits")
	finalizedCommits, err := b.protosClient.DB.GetCommits("main")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve finalized commits: %w", err)
	}
	tentativeCommits, err := b.protosClient.DB.GetCommits("tentative")
	if err != nil {
		log.Debugf("failed to retrieve tentative commits: %s", err.Error())
		tentativeCommits = nil
	}

	type localCommitRow struct {
		hash      string
		committer string
		message   string
		states    map[string]struct{}
	}
	rows := map[string]*localCommitRow{}
	order := []string{}
	mergeCommit := func(hash, committer, message, state string) {
		if hash == "" {
			return
		}
		row, ok := rows[hash]
		if !ok {
			row = &localCommitRow{
				hash:      hash,
				committer: committer,
				message:   message,
				states:    map[string]struct{}{},
			}
			rows[hash] = row
			order = append(order, hash)
		}
		row.states[state] = struct{}{}
	}

	for _, commit := range finalizedCommits {
		mergeCommit(commit.Hash, commit.Committer, commit.Message, "finalized")
	}
	for _, commit := range tentativeCommits {
		mergeCommit(commit.Hash, commit.Committer, commit.Message, "tentative")
	}

	resp := pbApic.GetLocalCommitsResponse{}
	for _, hash := range order {
		commit := rows[hash]
		respCommit := pbApic.Commit{
			Hash:      commit.hash,
			Committer: commit.committer,
			Message:   commit.message,
			States:    commitStates(commit.states),
		}
		resp.Commits = append(resp.Commits, &respCommit)
	}

	return &resp, nil
}

func commitStates(states map[string]struct{}) []string {
	out := []string{}
	if _, ok := states["finalized"]; ok {
		out = append(out, "finalized")
	}
	if _, ok := states["tentative"]; ok {
		out = append(out, "tentative")
	}
	return out
}

func (b *Backend) GetRemoteCommits(ctx context.Context, in *pbApic.GetRemoteCommitsRequest) (*pbApic.GetRemoteCommitsResponse, error) {
	log.Debugf("Retrieving commits from instance '%s'", in.Remote)

	client, err := b.protosClient.P2PManager.GetClient(in.Remote)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commits from remote '%s': %w", in.Remote, err)
	}

	respRemote, err := client.GetAllCommits(ctx, &p2pproto.GetAllCommitsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commits from remote '%s': %w", in.Remote, err)
	}

	resp := pbApic.GetRemoteCommitsResponse{}
	for _, commit := range respRemote.Commits {
		respCommit := pbApic.Commit{
			Hash:      commit.Hash,
			Committer: commit.Committer,
			Message:   commit.Message,
		}
		resp.Commits = append(resp.Commits, &respCommit)
	}

	return &resp, nil
}
