package cloud

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	pb "github.com/cheggaaa/pb/v3"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/util"
	account "github.com/scaleway/scaleway-sdk-go/api/account/v3"
	iam "github.com/scaleway/scaleway-sdk-go/api/iam/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"golang.org/x/crypto/ssh"
)

const (
	scalewayArch          = "x86_64"
	scalewayUploadSSHkey  = "protos-upload-key"
	scalewayImageDisk     = "/dev/vdb"
	scalewayUploadImageID = "ubuntu_focal"

	scalewayAuthOrganisationID = "ORGANISATION_ID"
	scalewayAuthProjectID      = "PROJECT_ID"
	scalewayAuthAccessKey      = "ACCESS_KEY"
	scalewayAuthSecretKey      = "SECRET_KEY"
)

var scalewayAuthFields = []string{scalewayAuthOrganisationID, scalewayAuthAccessKey, scalewayAuthSecretKey}

type scalewayCredentials struct {
	organisationID string
	projectID      string
	accessKey      string
	secretKey      string
}

type scaleway struct {
	providerMetadata

	sm          *pcrypto.Manager
	credentials *scalewayCredentials
	client      *scw.Client
	instanceAPI *instance.API
	projectAPI  *account.ProjectAPI
	iamAPI      *iam.API
}

func newScalewayFactory() ProviderFactory {
	return typedProviderFactory[scalewayCredentials]{
		providerType: Scaleway,
		authFields:   scalewayAuthFields,
		decodeAuth:   decodeScalewayCredentials,
		newClient:    newScalewayClient,
	}
}

func newScalewayClient(metadata providerMetadata, deps ProviderDeps, credentials scalewayCredentials) (ProviderClient, error) {
	if deps.SecretManager == nil {
		return nil, errors.New("secret manager is nil")
	}
	return &scaleway{
		providerMetadata: metadata,
		sm:               deps.SecretManager,
		credentials:      &credentials,
	}, nil
}

func decodeScalewayCredentials(auth map[string]string) (scalewayCredentials, error) {
	if len(auth) == 0 {
		return scalewayCredentials{}, errors.New("Scaleway credentials are required")
	}

	credentials := scalewayCredentials{}
	for key, value := range auth {
		if strings.TrimSpace(value) == "" {
			return scalewayCredentials{}, errors.Errorf("Credentials field '%s' is empty", key)
		}
		switch key {
		case scalewayAuthOrganisationID:
			credentials.organisationID = value
		case scalewayAuthProjectID:
			credentials.projectID = value
		case scalewayAuthAccessKey:
			credentials.accessKey = value
		case scalewayAuthSecretKey:
			credentials.secretKey = value
		default:
			return scalewayCredentials{}, errors.Errorf("Credentials field '%s' not supported by Scaleway cloud provider", key)
		}
	}

	for _, requiredField := range scalewayAuthFields {
		if strings.TrimSpace(auth[requiredField]) == "" {
			return scalewayCredentials{}, errors.Errorf("Credentials field '%s' is required", requiredField)
		}
	}
	return credentials, nil
}

func transformStatus(status instance.ServerState) string {
	var instanceStatus string
	switch status {
	case instance.ServerStateRunning:
		instanceStatus = ServerStateRunning
	case instance.ServerStateStopped:
		instanceStatus = ServerStateStopped
	case instance.ServerStateStarting:
		instanceStatus = ServerStateChanging
	case instance.ServerStateStopping:
		instanceStatus = ServerStateChanging
	default:
		instanceStatus = ServerStateOther
	}
	return instanceStatus
}

//
// Config methods
//

func (sw *scaleway) SupportedLocations() []string {
	return []string{string(scw.ZoneFrPar1), string(scw.ZoneNlAms1)}
}

func (sw *scaleway) Init() error {
	var err error
	clientOptions := []scw.ClientOption{
		scw.WithDefaultOrganizationID(sw.credentials.organisationID),
		scw.WithAuth(sw.credentials.accessKey, sw.credentials.secretKey),
	}
	if sw.credentials.projectID != "" {
		clientOptions = append(clientOptions, scw.WithDefaultProjectID(sw.credentials.projectID))
	}

	sw.client, err = scw.NewClient(clientOptions...)
	if err != nil {
		return errors.Wrap(err, "Failed to init Scaleway client")
	}

	sw.instanceAPI = instance.NewAPI(sw.client)
	sw.projectAPI = account.NewProjectAPI(sw.client)
	sw.iamAPI = iam.NewAPI(sw.client)
	sw.credentials.projectID, err = sw.resolveProjectID()
	if err != nil {
		return err
	}
	return nil
}

func (sw *scaleway) resolveProjectID() (string, error) {
	if sw.credentials.projectID != "" {
		return sw.credentials.projectID, nil
	}
	if projectID, found := sw.client.GetDefaultProjectID(); found && projectID != "" {
		return projectID, nil
	}

	projects, err := sw.projectAPI.ListProjects(&account.ProjectAPIListProjectsRequest{OrganizationID: sw.credentials.organisationID}, scw.WithAllPages())
	if err != nil {
		return "", errors.Wrap(err, "Failed to retrieve Scaleway projects")
	}
	if len(projects.Projects) == 0 {
		return "", errors.Errorf("Scaleway organization '%s' has no projects", sw.credentials.organisationID)
	}
	return projects.Projects[0].ID, nil
}

func (sw *scaleway) SupportedMachines(location string) (map[string]MachineSpec, error) {
	vms := map[string]MachineSpec{}
	inst, err := sw.instanceAPI.ListServersTypes(&instance.ListServersTypesRequest{Zone: scw.Zone(location)})
	if err != nil {
		return vms, errors.Wrap(err, "Failed to retrieve Scaleway instance types")
	}
	for id, instance := range inst.Servers {
		if instance.Arch == "x86_64" && strings.Contains(id, "DEV") {
			vms[id] = MachineSpec{
				Cores:                instance.Ncpus,
				Memory:               uint32(instance.RAM / 1048576),
				DefaultStorage:       uint32(instance.VolumesConstraint.MinSize / 1000000000),
				Baremetal:            false,
				Bandwidth:            uint32(*instance.Network.SumInternetBandwidth),
				IncludedDataTransfer: 0,
				PriceMonthly:         instance.HourlyPrice * 24 * 30,
			}
		}
	}
	return vms, nil
}

//
// Instance methods
//

func (sw *scaleway) deleteSSHkey(name string) error {
	keysResp, err := sw.iamAPI.ListSSHKeys(&iam.ListSSHKeysRequest{ProjectID: &sw.credentials.projectID}, scw.WithAllPages())
	if err != nil {
		return errors.Wrap(err, "Failed to get SSH keys")
	}
	for _, k := range keysResp.SSHKeys {
		if k.Name == name {
			log.Infof("Deleting SSH key '%s' (%s)", name, k.ID)
			err = sw.iamAPI.DeleteSSHKey(&iam.DeleteSSHKeyRequest{SSHKeyID: k.ID})
			if err != nil {
				return errors.Wrapf(err, "Failed to delete SSH key '%s'", name)
			}
			return nil
		}
	}
	return errors.Errorf("Could not find an SSH key named '%s'", name)
}

// NewInstance creates a new Protos instance on Scaleway
func (sw *scaleway) NewInstance(name string, imageID string, pubKey string, machineType string, location string) (string, error) {

	//
	// create SSH key
	//

	keysResp, err := sw.iamAPI.ListSSHKeys(&iam.ListSSHKeysRequest{ProjectID: &sw.credentials.projectID}, scw.WithAllPages())
	if err != nil {
		return "", errors.Wrap(err, "Failed to get SSH keys")
	}
	for _, k := range keysResp.SSHKeys {
		if k.Name == name {
			log.Infof("Found an SSH key with the same name as the instance (%s). Deleting it and creating a new key for the current instance.", name)
			if err := sw.iamAPI.DeleteSSHKey(&iam.DeleteSSHKeyRequest{SSHKeyID: k.ID}); err != nil {
				return "", errors.Wrapf(err, "Failed to delete existing SSH key '%s'", k.ID)
			}
		}
	}

	pubKey = strings.TrimSuffix(pubKey, "\n") + " root@protos.io"
	_, err = sw.iamAPI.CreateSSHKey(&iam.CreateSSHKeyRequest{Name: name, ProjectID: sw.credentials.projectID, PublicKey: pubKey})
	if err != nil {
		return "", errors.Wrap(err, "Failed to add SSH key for instance")
	}

	//
	// create server

	// checking if there is a server with the same name
	serversResp, err := sw.instanceAPI.ListServers(&instance.ListServersRequest{Zone: scw.Zone(location)})
	if err != nil {
		return "", errors.Wrap(err, "Failed to retrieve servers")
	}
	for _, srv := range serversResp.Servers {
		if srv.Name == name {
			return "", errors.Errorf("There is already an instance with name '%s' on Scaleway, in zone '%s'", name, scw.Zone(location))
		}
	}

	// deploying the instance
	volumeMap := make(map[string]*instance.VolumeServerTemplate)
	log.Infof("Deploing VM using image '%s'", imageID)
	ipreq := true
	enableIPv6 := false
	bootType := instance.BootTypeLocal
	req := &instance.CreateServerRequest{
		Name:              name,
		Zone:              scw.Zone(location),
		CommercialType:    machineType,
		DynamicIPRequired: &ipreq,
		EnableIPv6:        &enableIPv6,
		BootType:          &bootType,
		Image:             &imageID,
		Volumes:           volumeMap,
		Project:           &sw.credentials.projectID,
	}

	srvResp, err := sw.instanceAPI.CreateServer(req)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create VM")
	}
	log.Infof("Created server '%s' (%s)", srvResp.Server.Name, srvResp.Server.ID)

	return srvResp.Server.ID, nil
}

func (sw *scaleway) DeleteInstance(id string, location string) error {
	info, err := sw.GetInstanceInfo(id, location)
	if err != nil {
		return errors.Wrapf(err, "Failed to retrieve instance '%s'", id)
	}
	err = sw.instanceAPI.DeleteServer(&instance.DeleteServerRequest{Zone: scw.Zone(location), ServerID: id})
	if err != nil {
		return errors.Wrapf(err, "Failed to delete instance '%s'", id)
	}
	err = sw.deleteSSHkey(info.Name)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete SSH key for instance '%s'", id)
	}
	return nil
}

func (sw *scaleway) StartInstance(id string, location string) error {
	startReq := &instance.ServerActionAndWaitRequest{
		ServerID: id,
		Zone:     scw.Zone(location),
		Action:   instance.ServerActionPoweron,
	}
	err := sw.instanceAPI.ServerActionAndWait(startReq)
	if err != nil {
		return errors.Wrap(err, "Failed to start Scaleway instance")
	}
	return nil
}

func (sw *scaleway) StopInstance(id string, location string) error {
	stopReq := &instance.ServerActionAndWaitRequest{
		ServerID: id,
		Zone:     scw.Zone(scw.Zone(location)),
		Action:   instance.ServerActionPoweroff,
	}
	err := sw.instanceAPI.ServerActionAndWait(stopReq)
	if err != nil {
		return errors.Wrap(err, "Failed to stop Scaleway instance")
	}
	return nil
}

func (sw *scaleway) GetInstanceInfo(id string, location string) (InstanceInfo, error) {
	resp, err := sw.instanceAPI.GetServer(&instance.GetServerRequest{ServerID: id, Zone: scw.Zone(location)})
	if err != nil {
		return InstanceInfo{}, errors.Wrapf(err, "Failed to retrieve Scaleway instance (%s) information", id)
	}
	info := InstanceInfo{ID: id, Name: resp.Server.Name, Kind: KindCloudVM, Location: string(scw.Zone(location)), Status: transformStatus(resp.Server.State)}
	publicIP, err := scalewayServerPublicIP(resp.Server)
	if err != nil {
		return InstanceInfo{}, err
	}
	info.PublicIP = publicIP
	for _, svol := range resp.Server.Volumes {
		name := ""
		if svol.Name != nil {
			name = *svol.Name
		}
		var size uint64
		if svol.Size != nil {
			size = uint64(*svol.Size)
		}
		info.Volumes = append(info.Volumes, VolumeInfo{VolumeID: svol.ID, Name: name, Size: size})
	}
	return info, nil
}

//
// Images methods
//

func (sw *scaleway) GetImages() (map[string]ImageInfo, error) {
	images := map[string]ImageInfo{}
	locations := sw.SupportedLocations()
	for _, location := range locations {
		resp, err := sw.instanceAPI.ListImages(&instance.ListImagesRequest{Zone: scw.Zone(location)})
		if err != nil {
			return images, errors.Wrap(err, "Failed to retrieve account images from Scaleway")
		}
		for _, img := range resp.Images {
			if strings.Contains(img.Name, "protos-") {
				imgName := strings.TrimPrefix(img.Name, "protos-")
				images[img.ID] = ImageInfo{Name: imgName, ID: img.ID, Location: location}
			} else {
				images[img.ID] = ImageInfo{Name: img.Name, ID: img.ID, Location: location}
			}
		}
	}
	return images, nil
}

func (sw *scaleway) GetProtosImages() (map[string]ImageInfo, error) {
	images := map[string]ImageInfo{}
	locations := sw.SupportedLocations()
	for _, location := range locations {
		resp, err := sw.instanceAPI.ListImages(&instance.ListImagesRequest{Zone: scw.Zone(location)})
		if err != nil {
			return images, errors.Wrap(err, "Failed to retrieve account images from Scaleway")
		}
		for _, img := range resp.Images {
			if strings.Contains(img.Name, "protos-") {
				imgName := strings.TrimPrefix(img.Name, "protos-")
				images[img.ID] = ImageInfo{Name: imgName, ID: img.ID, Location: location}
			}
		}
	}

	return images, nil
}

func (sw *scaleway) AddImage(url string, hash string, version string, location string) (string, error) {

	//
	// create and add ssh key to account
	//

	key, err := sw.sm.GenerateKey()
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway")
	}
	pubKey := strings.TrimSuffix(key.AuthorizedKey(), "\n") + " root@protos.io"

	sshKey, err := sw.iamAPI.CreateSSHKey(&iam.CreateSSHKeyRequest{Name: scalewayUploadSSHkey, ProjectID: sw.credentials.projectID, PublicKey: pubKey})
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway: Failed to add temporary SSH key")
	}
	defer sw.cleanImageSSHkeys(sshKey.ID)

	//
	// create upload server
	//

	srv, vol, err := sw.createImageUploadVM(scalewayUploadImageID, location)
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway")
	}
	defer sw.cleanImageUploadVM(srv, location)

	//
	// connect via SSH, download Protos image and write it to a volume
	//

	srvPublicIP, err := scalewayServerPublicIP(srv)
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway")
	}
	log.Infof("Waiting for SSH service to be reachable at '%s'", srvPublicIP+":22")
	err = util.WaitForPort(srvPublicIP, "22", 25)
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway")
	}

	log.Info("Trying to connect to Scaleway upload instance over SSH")

	sshClient, err := pcrypto.NewConnection(srvPublicIP, "root", key.SSHAuth(), 10)
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Failed to deploy VM to Scaleway")
	}
	log.Info("SSH connection initiated")

	localISO := "/tmp/protos-scaleway.iso"

	log.Info("Downloading Protos image")
	out, err := pcrypto.ExecuteCommand("wget -O "+localISO+" "+url, sshClient)
	if err != nil {
		log.Errorf("Error downloading Protos VM image: %s", out)
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error downloading Protos VM image")
	}

	log.Info("Checking image integrity")
	cmdString := fmt.Sprintf("openssl dgst -r -sha256 %s | awk '{ print $1 }' | { read digest; if [ \"$digest\" = \"%s\" ]; then true; else false; fi }", localISO, hash)
	out, err = pcrypto.ExecuteCommand(cmdString, sshClient)
	if err != nil {
		log.Errorf("Image integrity check failed: %s: %s", out, err.Error())
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error downloading Protos VM image. Integrity check failed")
	}

	//
	// wite Protos image to volume
	//

	out, err = pcrypto.ExecuteCommand(fmt.Sprintf("ls %s", scalewayImageDisk), sshClient)
	if err != nil {
		log.Errorf("Snapshot volume not found: %s", out)
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Snapshot volume not found")
	}

	log.Info("Writing Protos image to volume")
	out, err = pcrypto.ExecuteCommand(fmt.Sprintf("dd if=%s of=%s", localISO, scalewayImageDisk), sshClient)
	if err != nil {
		log.Errorf("Error while writing image to volume: %s", out)
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error while writing image to volume")
	}

	//
	// turn off upload VM and dettach volume
	//

	log.Infof("Stopping upload server '%s' (%s)", srv.Name, srv.ID)
	stopReq := &instance.ServerActionAndWaitRequest{
		ServerID: srv.ID,
		Zone:     scw.Zone(location),
		Action:   instance.ServerActionPoweroff,
	}
	err = sw.instanceAPI.ServerActionAndWait(stopReq)
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error while stopping upload server")
	}

	_, err = sw.instanceAPI.DetachVolume(&instance.DetachVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error while detaching image volume")
	}

	//
	// create snapshot and image
	//

	log.Info("Creating snapshot from volume")
	snapshotResp, err := sw.instanceAPI.CreateSnapshot(&instance.CreateSnapshotRequest{
		VolumeID: &vol.ID,
		Name:     "protos-snapshot-" + version,
		Zone:     scw.Zone(location),
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error while creating snapshot from volume")
	}

	log.Info("Creating image from snapshot")
	imageResp, err := sw.instanceAPI.CreateImage(&instance.CreateImageRequest{
		Name:       "protos-" + version,
		Arch:       instance.ArchX86_64,
		RootVolume: snapshotResp.Snapshot.ID,
		Zone:       scw.Zone(location),
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to add Protos image to Scaleway. Error while creating image from snapshot")
	}
	log.Infof("Protos image '%s' created", imageResp.Image.ID)

	log.Infof("Deleting protos image volume '%s'", vol.ID)
	err = sw.instanceAPI.DeleteVolume(&instance.DeleteVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
	if err != nil {
		return "", errors.Wrap(err, "Error while removing protos image volume. Manual clean might be needed")
	}

	return imageResp.Image.ID, nil
}

func (sw *scaleway) UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (id string, err error) {

	errMsg := "Failed to upload Protos image to Scaleway"
	protosImage := "protos-" + imageName

	fdHash, err := os.Open(imagePath)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer fdHash.Close()

	h := sha256.New()
	if _, err := io.Copy(h, fdHash); err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	imageHash := hex.EncodeToString(h.Sum(nil))

	//
	// Create and add a temporary ssh key to account
	//

	key, err := sw.sm.GenerateKey()
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	pubKey := strings.TrimSuffix(key.AuthorizedKey(), "\n") + " root@protos.io"

	sshKey, err := sw.iamAPI.CreateSSHKey(&iam.CreateSSHKeyRequest{Name: scalewayUploadSSHkey, ProjectID: sw.credentials.projectID, PublicKey: pubKey})
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Failed to add temporary SSH key")
	}
	defer sw.cleanImageSSHkeys(sshKey.ID)

	//
	// Create upload server
	//

	srv, vol, err := sw.createImageUploadVM(scalewayUploadImageID, location)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer sw.cleanImageUploadVM(srv, location)

	//
	// Upload image via SCP
	//

	srvPublicIP, err := scalewayServerPublicIP(srv)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	log.Infof("Waiting for SSH service to be reachable at '%s'", srvPublicIP+":22")
	err = util.WaitForPort(srvPublicIP, "22", 25)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			key.SSHAuth(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client := scp.NewClient(srvPublicIP+":22", sshConfig)
	log.Infof("Connecting via SSH and starting SCP transfer to '%s'", srvPublicIP+":22")
	client.Timeout = timeout
	err = client.Connect()
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer client.Close()

	fdUpload, err := os.Open(imagePath)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer fdUpload.Close()

	fInfo, err := fdUpload.Stat()
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	log.Info("Uploading image. This can take a while...")
	remoteImage := "/tmp/" + protosImage

	bar := pb.Full.Start(0)
	passThru := func(r io.Reader, total int64) io.Reader {
		bar.SetTotal(total)
		return bar.NewProxyReader(r)
	}
	err = client.CopyPassThru(context.TODO(), fdUpload, remoteImage, "0655", fInfo.Size(), passThru)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	bar.Finish()

	//
	// connect via SSH and check the integrity of the image
	//

	log.Info("Trying to connect to Scaleway upload instance over SSH")

	sshClient, err := pcrypto.NewConnection(srvPublicIP, "root", key.SSHAuth(), 10)
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Failed to deploy VM to Scaleway")
	}
	log.Info("SSH connection initiated")

	log.Info("Checking image integrity")
	cmdString := fmt.Sprintf("openssl dgst -r -sha256 %s | awk '{ print $1 }' | { read digest; if [ \"$digest\" = \"%s\" ]; then true; else false; fi }", remoteImage, imageHash)
	out, err := pcrypto.ExecuteCommand(cmdString, sshClient)
	if err != nil {
		log.Errorf("Image integrity check failed: %s: %s", out, err.Error())
		return "", errors.Wrap(err, errMsg+". Integrity check failed")
	}

	//
	// wite Protos image to volume
	//

	out, err = pcrypto.ExecuteCommand(fmt.Sprintf("ls %s", scalewayImageDisk), sshClient)
	if err != nil {
		log.Errorf("Snapshot volume not found: %s", out)
		return "", errors.Wrap(err, errMsg+". Snapshot volume not found")
	}

	log.Info("Writing Protos image to volume")
	out, err = pcrypto.ExecuteCommand(fmt.Sprintf("dd if=%s of=%s", remoteImage, scalewayImageDisk), sshClient)
	if err != nil {
		log.Errorf("Error while writing image to volume: %s", out)
		return "", errors.Wrap(err, errMsg+". Error while writing image to volume")
	}

	//
	// turn off upload VM and dettach volume
	//

	log.Infof("Stopping upload server '%s' (%s)", srv.Name, srv.ID)
	stopReq := &instance.ServerActionAndWaitRequest{
		ServerID: srv.ID,
		Zone:     scw.Zone(location),
		Action:   instance.ServerActionPoweroff,
	}
	err = sw.instanceAPI.ServerActionAndWait(stopReq)
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Error while stopping upload server")
	}

	_, err = sw.instanceAPI.DetachVolume(&instance.DetachVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Error while detaching image volume")
	}

	//
	// create snapshot and image
	//

	log.Info("Creating snapshot from volume")
	snapshotResp, err := sw.instanceAPI.CreateSnapshot(&instance.CreateSnapshotRequest{
		VolumeID: &vol.ID,
		Name:     "protos-snapshot-" + imageName,
		Zone:     scw.Zone(location),
	})
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Error while creating snapshot from volume")
	}

	log.Info("Creating image from snapshot")
	imageResp, err := sw.instanceAPI.CreateImage(&instance.CreateImageRequest{
		Name:       protosImage,
		Arch:       instance.ArchX86_64,
		RootVolume: snapshotResp.Snapshot.ID,
		Zone:       scw.Zone(location),
	})
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Error while creating image from snapshot")
	}
	log.Infof("Protos image '%s(%s)' created", protosImage, imageResp.Image.ID)

	log.Infof("Deleting protos image volume '%s'", vol.ID)
	err = sw.instanceAPI.DeleteVolume(&instance.DeleteVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
	if err != nil {
		return "", errors.Wrap(err, "Error while removing protos image volume. Manual clean might be needed")
	}

	return imageResp.Image.ID, nil
}

func (sw *scaleway) RemoveImage(name string, location string) error {
	errMsg := fmt.Sprintf("Failed to remove image '%s' in '%s'", name, location)
	if location == "" {
		return errors.Wrap(fmt.Errorf("location is required for Scaleway"), errMsg)
	}
	// find image
	images, err := sw.GetProtosImages()
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	id := ""
	for _, img := range images {
		if img.Location == location && img.Name == name {
			id = img.ID
			break
		}
	}
	if id == "" {
		return fmt.Errorf("%s: could not find image '%s'", errMsg, name)
	}
	img, err := sw.instanceAPI.GetImage(&instance.GetImageRequest{ImageID: id, Zone: scw.Zone(location)})
	if err != nil {
		return errors.Wrap(err, errMsg)
	}

	err = sw.instanceAPI.DeleteImage(&instance.DeleteImageRequest{ImageID: id, Zone: scw.Zone(location)})
	if err != nil {
		return errors.Wrap(err, errMsg)
	}

	err = sw.instanceAPI.DeleteSnapshot(&instance.DeleteSnapshotRequest{SnapshotID: img.Image.RootVolume.ID, Zone: scw.Zone(location)})
	if err != nil {
		return errors.Wrap(err, errMsg)
	}

	return nil
}

//
// Volumes methods
//

func (sw *scaleway) NewVolume(name string, size int, location string) (string, error) {
	sizeVolume := scw.Size(uint64(size * 1048576))
	createVolumeReq := &instance.CreateVolumeRequest{
		Name:       name,
		VolumeType: "b_ssd",
		Size:       &sizeVolume,
		Zone:       scw.Zone(location),
	}

	targetVolumeResp, err := sw.instanceAPI.CreateVolume(createVolumeReq)
	if err != nil {
		return "", errors.Wrap(err, "Failed to create Scaleway volume")
	}
	return targetVolumeResp.Volume.ID, nil
}

func (sw *scaleway) DeleteVolume(id string, location string) error {
	deleteVolumeReq := &instance.DeleteVolumeRequest{
		VolumeID: id,
		Zone:     scw.Zone(location),
	}
	err := sw.instanceAPI.DeleteVolume(deleteVolumeReq)
	if err != nil {
		return errors.Wrapf(err, "Failed to delete Scaleway volume '%s'", id)
	}
	return nil
}

func (sw *scaleway) AttachVolume(volumeID string, instanceID string, location string) error {
	attachVolumeReq := &instance.AttachVolumeRequest{
		Zone:     scw.Zone(location),
		VolumeID: volumeID,
		ServerID: instanceID,
	}
	_, err := sw.instanceAPI.AttachVolume(attachVolumeReq)
	if err != nil {
		return errors.Wrapf(err, "Failed to attach Scaleway volume '%s' to instance '%s'", volumeID, instanceID)
	}
	return nil
}

func (sw *scaleway) DettachVolume(volumeID string, instanceID string, location string) error {
	detachVolumeReq := &instance.DetachVolumeRequest{
		Zone:     scw.Zone(location),
		VolumeID: volumeID,
	}
	_, err := sw.instanceAPI.DetachVolume(detachVolumeReq)
	if err != nil {
		return errors.Wrapf(err, "Failed to detach Scaleway volume '%s' from instance '%s'", volumeID, instanceID)
	}
	return nil
}

//
// helper methods
//

func scalewayServerPublicIP(server *instance.Server) (string, error) {
	if server == nil {
		return "", errors.New("Scaleway server is nil")
	}
	for _, publicIP := range server.PublicIPs {
		if publicIP != nil && publicIP.Address != nil {
			return publicIP.Address.String(), nil
		}
	}
	return "", errors.Errorf("Scaleway server '%s' has no public IP", server.Name)
}

func (sw *scaleway) cleanImageSSHkeys(keyID string) {
	err := sw.iamAPI.DeleteSSHKey(&iam.DeleteSSHKeyRequest{SSHKeyID: keyID})
	if err != nil {
		log.Error(errors.Wrapf(err, "Failed to clean up Scaleway image upload key with id '%s'", keyID))
	}
	log.Infof("Deleted SSH key '%s'", keyID)
}

func (sw *scaleway) createImageUploadVM(imageID string, location string) (*instance.Server, *instance.Volume, error) {

	//
	// create volume
	//

	sizeTargetDisk := scw.Size(uint64(1)) * scw.GB
	createVolumeReq := &instance.CreateVolumeRequest{
		Name:       "protos-image-uploader-target",
		VolumeType: instance.VolumeVolumeTypeLSSD,
		Size:       &sizeTargetDisk,
		Zone:       scw.Zone(location),
	}

	log.Info("Creating image volume")
	targetVolumeResp, err := sw.instanceAPI.CreateVolume(createVolumeReq)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create image volume")
	}

	//
	// create server
	//
	trueBool := true
	sizeOSDisk := scw.Size(uint64(19)) * scw.GB
	volumeMap := make(map[string]*instance.VolumeServerTemplate, 1)
	osVolumeTemplate := &instance.VolumeServerTemplate{
		VolumeType: instance.VolumeVolumeTypeLSSD,
		Size:       &sizeOSDisk,
		Boot:       &trueBool,
	}
	volumeMap["0"] = osVolumeTemplate

	bootType := instance.BootTypeLocal
	enableIPv6 := false
	req := &instance.CreateServerRequest{
		Name:              "protos-image-uploader",
		Zone:              scw.Zone(location),
		CommercialType:    "DEV1-S",
		DynamicIPRequired: &trueBool,
		EnableIPv6:        &enableIPv6,
		BootType:          &bootType,
		Image:             &imageID,
		Volumes:           volumeMap,
		Project:           &sw.credentials.projectID,
	}

	srvResp, err := sw.instanceAPI.CreateServer(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to create upload VM")
	}
	log.Infof("Created server '%s' (%s)", srvResp.Server.Name, srvResp.Server.ID)

	//
	// attach volume
	//

	log.Info("Attaching snapshot volume to upload VM")
	attachVolumeReq := &instance.AttachVolumeRequest{
		ServerID: srvResp.Server.ID,
		VolumeID: targetVolumeResp.Volume.ID,
		Zone:     scw.Zone(location),
	}

	_, err = sw.instanceAPI.AttachVolume(attachVolumeReq)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to attach volume to upload VM")
	}

	//
	// start server
	//

	// default timeout is 5 minutes
	log.Infof("Starting and waiting for server '%s' (%s)", srvResp.Server.Name, srvResp.Server.ID)
	startReq := &instance.ServerActionAndWaitRequest{
		ServerID: srvResp.Server.ID,
		Zone:     scw.Zone(location),
		Action:   instance.ServerActionPoweron,
	}
	err = sw.instanceAPI.ServerActionAndWait(startReq)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to start upload server")
	}
	log.Infof("Server '%s' (%s) started successfully", srvResp.Server.Name, srvResp.Server.ID)

	//
	// refresh IP info
	//

	srvStatusResp, err := sw.instanceAPI.GetServer(&instance.GetServerRequest{ServerID: srvResp.Server.ID, Zone: scw.Zone(location)})
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to retrieve upload VM details")
	}

	return srvStatusResp.Server, targetVolumeResp.Volume, nil
}

func (sw *scaleway) cleanImageUploadVM(srv *instance.Server, location string) {
	srvStatusResp, err := sw.instanceAPI.GetServer(&instance.GetServerRequest{ServerID: srv.ID, Zone: scw.Zone(location)})
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to refresh upload server info"))
		return
	}
	srv = srvStatusResp.Server

	if srv.State == instance.ServerStateRunning {
		// default timeout is 5 minutes
		log.Infof("Stopping and waiting for server '%s' (%s)", srv.Name, srv.ID)
		stopReq := &instance.ServerActionAndWaitRequest{
			ServerID: srv.ID,
			Zone:     scw.Zone(location),
			Action:   instance.ServerActionPoweroff,
		}
		err = sw.instanceAPI.ServerActionAndWait(stopReq)
		if err != nil {
			log.Error(errors.Wrap(err, "Failed to stop upload server"))
			return
		}
		log.Infof("Server '%s' (%s) stopped successfully", srv.Name, srv.ID)
	}

	for _, vol := range srv.Volumes {
		log.Infof("Deleting volume '%s' for server '%s' (%s)", vol.ID, srv.Name, srv.ID)
		_, err = sw.instanceAPI.DetachVolume(&instance.DetachVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
		if err != nil {
			log.Errorf("Failed to dettach volume '%s' for server '%s' (%s): %s", vol.ID, srv.Name, srv.ID, err.Error())
			continue
		}
		err = sw.instanceAPI.DeleteVolume(&instance.DeleteVolumeRequest{Zone: scw.Zone(location), VolumeID: vol.ID})
		if err != nil {
			log.Errorf("Failed to delete volume '%s' for server '%s' (%s): %s", vol.ID, srv.Name, srv.ID, err.Error())
		}
	}

	log.Infof("Deleting server '%s' (%s)", srv.Name, srv.ID)
	err = sw.instanceAPI.DeleteServer(&instance.DeleteServerRequest{ServerID: srv.ID, Zone: scw.Zone(location)})
	if err != nil {
		log.Error(errors.Wrap(err, "Failed to add Protos image to Scaleway"))
		return
	}
}
