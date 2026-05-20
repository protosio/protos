package apic

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
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
		return nil, fmt.Errorf("could not retrieve instance '%s' logs: p2p failed: %w; ssh fallback failed: %v", instanceName, p2pErr, err)
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
	commits, err := b.protosClient.DB.GetAllCommits()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve local commits: %w", err)
	}

	resp := pbApic.GetLocalCommitsResponse{}
	for _, commit := range commits {
		respCommit := pbApic.Commit{
			Hash:      commit.Hash,
			Committer: commit.Committer,
			Message:   commit.Message,
		}
		resp.Commits = append(resp.Commits, &respCommit)
	}

	return &resp, nil
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
