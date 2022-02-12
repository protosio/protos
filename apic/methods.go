package apic

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/denisbrodbeck/machineid"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
)

func (b *Backend) Init(ctx context.Context, in *pbApic.InitRequest) (*pbApic.InitResponse, error) {

	log.Debugf("Performing initialization")

	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to add user. Could not retrieve hostname: %w", err)
	}

	key, err := b.protosClient.Meta.GetPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to add user. Could not retrieve key: %w", err)
	}

	machineID, err := machineid.ProtectedID("protos")
	if err != nil {
		return nil, fmt.Errorf("failed to add user. Error while generating machine id: %w", err)
	}

	adminUser, err := b.protosClient.UserManager.CreateUser(in.Username, in.Password, in.Name, true)
	if err != nil {
		return nil, err
	}

	err = adminUser.AddDevice(machineID, host, key.Public(), "10.100.0.1/24")
	if err != nil {
		return nil, fmt.Errorf("failed to add user. Error while creating user device: %w", err)
	}

	// saving the key to disk
	key.Save()
	b.protosClient.SetInitialized()

	return &pbApic.InitResponse{}, nil
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
		respApp := pbApic.App{
			Id:            app.ID,
			Name:          app.Name,
			Version:       app.Version,
			DesiredStatus: app.DesiredStatus,
			InstanceName:  app.InstanceName,
			Ip:            app.IP.String(),
			Installer:     app.InstallerRef,
		}
		resp.Apps = append(resp.Apps, &respApp)
	}

	return &resp, nil
}

func (b *Backend) CreateApp(ctx context.Context, in *pbApic.CreateAppRequest) (*pbApic.CreateAppResponse, error) {

	log.Debugf("Running app '%s' based on installer '%s', on instance '%s'", in.Name, in.InstallerId, in.InstanceId)
	installer, err := b.protosClient.AppStore.GetInstaller(in.InstallerId)
	if err != nil {
		return nil, fmt.Errorf("failed to run app %s: %w", in.Name, err)
	}

	instance, err := b.protosClient.CloudManager.GetInstance(in.InstanceId)
	if err != nil {
		return nil, fmt.Errorf("failed to run app %s: %w", in.Name, err)
	}

	if !installer.SupportsArchitecture(instance.Architecture) {
		return nil, fmt.Errorf("failed to run app %s: installer '%s' does support architecture of target instance '%s'(%s)", in.Name, in.InstallerId, instance.Name, instance.Architecture)
	}

	// FIXME: read the installer params from the command line
	app, err := b.protosClient.AppManager.Create(installer, in.Name, in.InstanceId, instance.Network, map[string]string{})
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
	log.Debugf("Removing app '%s'", in.Name)
	logs, err := b.protosClient.AppManager.GetLogs(in.Name)
	if err != nil {
		return nil, err
	}

	return &pbApic.GetAppLogsResponse{Logs: logs}, nil
}

//
// App store methods
//

func (b *Backend) GetInstallers(ctx context.Context, in *pbApic.GetInstallersRequest) (*pbApic.GetInstallersResponse, error) {
	log.Debugf("Retrieving installers from app store")
	installers, err := b.protosClient.AppStore.GetInstallers()
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetInstallersResponse{}
	for _, installer := range installers {
		respInstaller := pbApic.Installer{
			Id:          installer.ID,
			Name:        installer.Name,
			Version:     installer.Version,
			Description: installer.GetDescription(),
		}
		resp.Installers = append(resp.Installers, &respInstaller)
	}

	return &resp, nil
}

func (b *Backend) GetInstaller(ctx context.Context, in *pbApic.GetInstallerRequest) (*pbApic.GetInstallerResponse, error) {
	log.Debugf("Retrieving installer '%s' from app store", in.Id)
	installer, err := b.protosClient.AppStore.GetInstaller(in.Id)
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetInstallerResponse{
		Installer: &pbApic.Installer{
			Id:                installer.ID,
			Name:              installer.Name,
			Version:           installer.Version,
			Description:       installer.GetDescription(),
			RequiresResources: installer.GetRequires(),
			ProvidesResources: installer.GetProvides(),
			Capabilities:      installer.GetCapabilities(),
		},
	}

	return &resp, nil
}

//
// Cloud provider methods
//

func (b *Backend) GetSupportedCloudProviders(ctx context.Context, in *pbApic.GetSupportedCloudProvidersRequest) (*pbApic.GetSupportedCloudProvidersResponse, error) {
	log.Debug("Retrieving supported cloud providers")
	supportedCloudProviders := b.protosClient.CloudManager.SupportedProviders()

	resp := pbApic.GetSupportedCloudProvidersResponse{}
	for _, supportedCloudProvider := range supportedCloudProviders {
		// create new temporary cloud provider to retrieve the required auth fields
		tempCloud, err := b.protosClient.CloudManager.NewProvider("tempCloud", supportedCloudProvider)
		if err != nil {
			return nil, err
		}
		respCloudType := pbApic.CloudType{
			Name:                 supportedCloudProvider,
			AuthenticationFields: tempCloud.AuthFields(),
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

	// initialize cloud provider before use
	err = cloudProvider.Init()
	if err != nil {
		return nil, fmt.Errorf("error reaching cloud provider '%s'(%s) API: %w", in.Name, cloudProvider.TypeStr(), err)
	}

	supportedLocations := cloudProvider.SupportedLocations()
	supportedMachines, err := cloudProvider.SupportedMachines(supportedLocations[0])
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
	// create new cloud provider
	provider, err := b.protosClient.CloudManager.NewProvider(in.Name, in.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to create cloud provider: %w", err)
	}

	// set authentication
	err = provider.SetAuth(in.Credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to set credentials for cloud provider: %w", err)
	}

	// init cloud client
	err = provider.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloud provider: %w", err)
	}

	// save the cloud provider in the db
	err = provider.Save()
	if err != nil {
		return nil, fmt.Errorf("failed to save cloud provider: %w", err)
	}
	return &pbApic.AddCloudProviderResponse{}, nil
}

func (b *Backend) RemoveCloudProvider(ctx context.Context, in *pbApic.RemoveCloudProviderRequest) (*pbApic.RemoveCloudProviderResponse, error) {
	// delete existing cloud provider
	err := b.protosClient.CloudManager.DeleteProvider(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to delete cloud provider '%s': %w", in.Name, err)
	}

	return &pbApic.RemoveCloudProviderResponse{}, nil
}

//
// Cloud instance methods
//

func (b *Backend) GetInstances(ctx context.Context, in *pbApic.GetInstancesRequest) (*pbApic.GetInstancesResponse, error) {
	log.Debugf("Retrieving instances")
	instances, err := b.protosClient.CloudManager.GetInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	resp := pbApic.GetInstancesResponse{}
	for _, instance := range instances {

		wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
		if err != nil {
			log.Error(err.Error())
		}

		respInstance := pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         instance.InternalIP,
			Network:            instance.Network,
			CloudName:          instance.CloudName,
			CloudType:          instance.CloudType,
			VmId:               instance.VMID,
			Location:           instance.Location,
			PublicKey:          base64.StdEncoding.EncodeToString(instance.PublicKey),
			PublicKeyWireguard: wgPublicKey.String(),
			ProtosVersion:      instance.ProtosVersion,
			Architecture:       instance.Architecture,
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

	resp := pbApic.GetInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         instance.InternalIP,
			Network:            instance.Network,
			CloudName:          instance.CloudName,
			CloudType:          instance.CloudType,
			VmId:               instance.VMID,
			Location:           instance.Location,
			PublicKey:          base64.StdEncoding.EncodeToString(instance.PublicKey),
			PublicKeyWireguard: wgPublicKey.String(),
			ProtosVersion:      instance.ProtosVersion,
			Status:             instance.Status,
			Architecture:       instance.Architecture,
		},
	}

	return &resp, nil
}

func (b *Backend) DeployInstance(ctx context.Context, in *pbApic.DeployInstanceRequest) (*pbApic.DeployInstanceResponse, error) {
	log.Debugf("Deploying new instance '%s'", in.Name)

	releases, err := b.protosClient.GetProtosAvailableReleases()
	if err != nil {
		return nil, err
	}
	rls := release.Release{}
	if in.DevImg != "" {
		rls.Version = in.DevImg
	} else if in.ProtosVersion != "" {
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

	instance, err := b.protosClient.CloudManager.DeployInstance(in.Name, in.CloudName, in.CloudLocation, rls, in.MachineType)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy instance '%s': %w", in.Name, err)
	}

	wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(instance.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy instance '%s': %w", in.Name, err)
	}

	resp := pbApic.DeployInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         instance.InternalIP,
			Network:            instance.Network,
			CloudName:          instance.CloudName,
			CloudType:          instance.CloudType,
			VmId:               instance.VMID,
			Location:           instance.Location,
			PublicKey:          base64.StdEncoding.EncodeToString(instance.PublicKey),
			PublicKeyWireguard: wgPublicKey.String(),
			ProtosVersion:      instance.ProtosVersion,
			Status:             instance.Status,
		},
	}

	return &resp, nil
}

func (b *Backend) RemoveInstance(ctx context.Context, in *pbApic.RemoveInstanceRequest) (*pbApic.RemoveInstanceResponse, error) {
	log.Debugf("Removing instance '%s'", in.Name)
	err := b.protosClient.CloudManager.DeleteInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to remove instance '%s': %w", in.Name, err)
	}

	return &pbApic.RemoveInstanceResponse{}, nil
}

func (b *Backend) StartInstance(ctx context.Context, in *pbApic.StartInstanceRequest) (*pbApic.StartInstanceResponse, error) {
	log.Debugf("Starting instance '%s'", in.Name)
	err := b.protosClient.CloudManager.StartInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to start instance '%s': %w", in.Name, err)
	}
	return &pbApic.StartInstanceResponse{}, nil
}

func (b *Backend) StopInstance(ctx context.Context, in *pbApic.StopInstanceRequest) (*pbApic.StopInstanceResponse, error) {
	log.Debugf("Stopping instance '%s'", in.Name)
	err := b.protosClient.CloudManager.StopInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to stop instance '%s': %w", in.Name, err)
	}
	return &pbApic.StopInstanceResponse{}, nil
}

func (b *Backend) GetInstanceKey(ctx context.Context, in *pbApic.GetInstanceKeyRequest) (*pbApic.GetInstanceKeyResponse, error) {
	log.Debugf("Retrieving key for instance '%s'", in.Name)
	instance, err := b.protosClient.CloudManager.GetInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve instance '%s' key: %w", in.Name, err)
	}
	if len(instance.SSHKeySeed) == 0 {
		return nil, fmt.Errorf("instance '%s' is missing its SSH key", in.Name)
	}
	key, err := b.protosClient.KeyManager.NewKeyFromSeed(instance.SSHKeySeed)
	if err != nil {
		return nil, fmt.Errorf("instance '%s' has an invalid SSH key: %w", in.Name, err)
	}
	return &pbApic.GetInstanceKeyResponse{Key: key.EncodePrivateKeytoPEM()}, nil
}

func (b *Backend) GetInstanceLogs(ctx context.Context, in *pbApic.GetInstanceLogsRequest) (*pbApic.GetInstanceLogsResponse, error) {
	log.Debugf("Retrieving logs for instance '%s'", in.Name)
	logs, err := b.protosClient.CloudManager.LogsInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve instance '%s' logs: %w", in.Name, err)
	}

	return &pbApic.GetInstanceLogsResponse{Logs: logs}, nil
}

func (b *Backend) InitDevInstance(ctx context.Context, in *pbApic.InitDevInstanceRequest) (*pbApic.InitDevInstanceResponse, error) {
	log.Debugf("Initializing dev instance '%s' at '%s'", in.Name, in.Ip)

	err := b.protosClient.CloudManager.InitDevInstance(in.Name, "local", "local", in.KeyFile, in.Ip)
	if err != nil {
		return nil, fmt.Errorf("could not initialize dev instance '%s': %w", in.Name, err)
	}
	return &pbApic.InitDevInstanceResponse{}, nil
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

	err = provider.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cloud provider '%s'(%s) API: %w", in.Name, provider.TypeStr(), err)
	}

	images, err := provider.GetProtosImages()
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

func (b *Backend) UploadCloudImage(ctx context.Context, in *pbApic.UploadCloudImageRequest) (*pbApic.UploadCloudImageResponse, error) {
	log.Debugf("Uploading cloud image '%s'(%s) to cloud '%s'", in.ImageName, in.ImagePath, in.CloudName)
	return &pbApic.UploadCloudImageResponse{}, b.protosClient.CloudManager.UploadLocalImage(in.ImagePath, in.ImageName, in.CloudName, in.CloudLocation, time.Duration(in.Timeout)*time.Minute)
}

func (b *Backend) RemoveCloudImage(ctx context.Context, in *pbApic.RemoveCloudImageRequest) (*pbApic.RemoveCloudImageResponse, error) {
	log.Debugf("Removing cloud image '%s' from cloud '%s'", in.ImageName, in.CloudName)
	errMsg := fmt.Sprintf("failed to delete image '%s' from cloud '%s'", in.ImageName, in.CloudLocation)
	provider, err := b.protosClient.CloudManager.GetProvider(in.CloudName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	err = provider.Init()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}

	// delete image
	err = provider.RemoveImage(in.ImageName, in.CloudLocation)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errMsg, err)
	}
	return &pbApic.RemoveCloudImageResponse{}, nil
}
