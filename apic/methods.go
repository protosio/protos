package apic

import (
	"context"
	"fmt"
	"os"

	"github.com/denisbrodbeck/machineid"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/auth"
)

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

func (b *Backend) GetApps(ctx context.Context, in *pbApic.GetAppsRequest) (*pbApic.GetAppsResponse, error) {

	log.Debugf("Retrieving apps")
	apps, err := b.protosClient.AppManager.GetAll()
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve apps: %w", err)
	}

	resp := pbApic.GetAppsResponse{}
	for _, app := range apps {
		respApp := pbApic.App{
			Id:            app.ID,
			Name:          app.Name,
			Version:       app.InstallerVersion,
			DesiredStatus: app.DesiredStatus,
			InstanceName:  app.InstanceName,
			Ip:            app.IP,
		}
		resp.Apps = append(resp.Apps, &respApp)
	}

	return &resp, nil
}

func (b *Backend) RunApp(ctx context.Context, in *pbApic.RunAppRequest) (*pbApic.RunAppResponse, error) {

	log.Debugf("Running app '%s' based on installer '%s', on instance '%s'", in.Name, in.InstallerId, in.InstanceId)
	installer, err := b.protosClient.AppStore.GetInstaller(in.InstallerId)
	if err != nil {
		return nil, fmt.Errorf("Failed to run app %s: %w", in.Name, err)
	}

	instMetadata, err := installer.GetMetadata(installer.GetLastVersion())
	if err != nil {
		return nil, fmt.Errorf("Failed to run app %s: %w", in.Name, err)
	}

	// FIXME: read the installer params from the command line
	app, err := b.protosClient.AppManager.Create(in.InstallerId, installer.GetLastVersion(), in.Name, in.InstanceId, map[string]string{}, instMetadata)
	if err != nil {
		return nil, fmt.Errorf("Failed to run app %s: %w", in.Name, err)
	}

	return &pbApic.RunAppResponse{Id: app.ID}, nil
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

func (b *Backend) GetInstallers(ctx context.Context, in *pbApic.GetInstallersRequest) (*pbApic.GetInstallersResponse, error) {
	log.Debugf("Retrieving installers from app store")
	installers, err := b.protosClient.AppStore.GetInstallers()
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetInstallersResponse{}
	for _, installer := range installers {
		installerMetadata, err := installer.GetMetadata(installer.GetLastVersion())
		if err != nil {
			return nil, fmt.Errorf("Failed to get metadata for installer '%s': %w", installer.ID, err)
		}
		respInstaller := pbApic.Installer{
			Id:          installer.ID,
			Name:        installer.Name,
			Version:     installer.GetLastVersion(),
			Description: installerMetadata.Description,
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

	installerMetadata, err := installer.GetMetadata(installer.GetLastVersion())
	if err != nil {
		return nil, err
	}

	resp := pbApic.GetInstallerResponse{
		Installer: &pbApic.Installer{
			Id:                installer.ID,
			Name:              installer.Name,
			Version:           installer.GetLastVersion(),
			Description:       installerMetadata.Description,
			RequiresResources: installerMetadata.Requires,
			ProvidesResources: installerMetadata.Provides,
			Capabilities:      installerMetadata.Capabilities,
		},
	}

	return &resp, nil
}
