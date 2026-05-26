package hetzner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	scp "github.com/bramvdbogaerde/go-scp"
	pb "github.com/cheggaaa/pb/v3"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"github.com/hetznercloud/hcloud-go/v2/hcloud/exp/actionutil"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/util"
	"golang.org/x/crypto/ssh"
)

const (
	hetznerAuthAPIKey = "API_KEY"

	hetznerUploadSSHKeyName    = "protos-upload-key"
	hetznerUploadServerName    = "protos-image-uploader"
	hetznerUploadServerType    = "cpx11"
	hetznerUploadBaseImageName = "ubuntu-24.04"

	hetznerLabelManaged       = "protos-managed"
	hetznerLabelImage         = "protos-image"
	hetznerLabelImageName     = "protos-image-name"
	hetznerLabelImageLocation = "protos-image-location"
	hetznerLabelTemporary     = "protos-temporary"
)

var (
	hetznerAuthFields       = []string{hetznerAuthAPIKey}
	hetznerDefaultLocations = []string{"fsn1", "nbg1", "hel1", "ash", "hil", "sin"}
	log                     = util.GetLogger("provisioner.hetzner")
)

const Type = provisioners.Type("hetzner")

type hetznerCredentials struct {
	apiKey string
}

type hetzner struct {
	provisioners.ProvisionerMetadata

	sm          *pcrypto.Manager
	credentials *hetznerCredentials
	client      *hcloud.Client
	locations   []string
}

func NewFactory() provisioners.ProvisionerFactory {
	return provisioners.NewTypedProvisionerFactory(Type, hetznerAuthFields, decodeHetznerCredentials, newHetznerClient)
}

func newHetznerClient(metadata provisioners.ProvisionerMetadata, deps provisioners.ProvisionerDeps, credentials hetznerCredentials) (provisioners.Provisioner, error) {
	if deps.SecretManager == nil {
		return nil, errors.New("secret manager is nil")
	}
	return &hetzner{
		ProvisionerMetadata: metadata,
		sm:                  deps.SecretManager,
		credentials:         &credentials,
	}, nil
}

func decodeHetznerCredentials(auth map[string]string) (hetznerCredentials, error) {
	if len(auth) == 0 {
		return hetznerCredentials{}, errors.New("Hetzner credentials are required")
	}

	credentials := hetznerCredentials{}
	for key, value := range auth {
		if strings.TrimSpace(value) == "" {
			return hetznerCredentials{}, errors.Errorf("Credentials field '%s' is empty", key)
		}
		switch key {
		case hetznerAuthAPIKey:
			credentials.apiKey = strings.TrimSpace(value)
		default:
			return hetznerCredentials{}, errors.Errorf("Credentials field '%s' not supported by Hetzner cloud provider", key)
		}
	}

	if credentials.apiKey == "" {
		return hetznerCredentials{}, errors.Errorf("Credentials field '%s' is required", hetznerAuthAPIKey)
	}
	return credentials, nil
}

func (hz *hetzner) Init() error {
	hz.client = hcloud.NewClient(
		hcloud.WithToken(hz.credentials.apiKey),
		hcloud.WithApplication("protos", "0.1.0"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	locations, err := hz.client.Location.All(ctx)
	if err != nil {
		return errors.Wrap(err, "Failed to init Hetzner client")
	}
	hz.locations = hz.locations[:0]
	for _, location := range locations {
		hz.locations = append(hz.locations, location.Name)
	}
	sort.Strings(hz.locations)
	return nil
}

func (hz *hetzner) SupportedLocations() []string {
	if len(hz.locations) == 0 {
		return append([]string(nil), hetznerDefaultLocations...)
	}
	return append([]string(nil), hz.locations...)
}

func (hz *hetzner) SupportedMachines(location string) (map[string]provisioners.MachineSpec, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	serverTypes, err := hz.client.ServerType.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve Hetzner server types")
	}

	vms := map[string]provisioners.MachineSpec{}
	for _, serverType := range serverTypes {
		if serverType.Architecture != hcloud.ArchitectureX86 {
			continue
		}
		if !hetznerServerTypeAvailableIn(serverType, location) {
			continue
		}
		vms[serverType.Name] = provisioners.MachineSpec{
			Cores:                uint32(serverType.Cores),
			Memory:               uint32(serverType.Memory * 1024),
			DefaultStorage:       uint32(serverType.Disk),
			Baremetal:            false,
			Bandwidth:            0,
			IncludedDataTransfer: hetznerIncludedTrafficGiB(serverType, location),
			PriceMonthly:         hetznerMonthlyPrice(serverType, location),
		}
	}
	return vms, nil
}

func (hz *hetzner) NewInstance(name string, imageID string, pubKey string, machineType string, location string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if existing, _, err := hz.client.Server.GetByName(ctx, name); err != nil {
		return "", errors.Wrap(err, "Failed to retrieve Hetzner servers")
	} else if existing != nil {
		return "", errors.Errorf("There is already an instance with name '%s' on Hetzner", name)
	}

	sshKey, err := hz.createSSHKey(ctx, name, pubKey, map[string]string{hetznerLabelManaged: "true"})
	if err != nil {
		return "", err
	}

	image, err := hz.getImage(ctx, imageID)
	if err != nil {
		return "", err
	}
	serverType, _, err := hz.client.ServerType.Get(ctx, machineType)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to retrieve Hetzner server type '%s'", machineType)
	}
	if serverType == nil {
		return "", errors.Errorf("Hetzner server type '%s' not found", machineType)
	}
	serverLocation, _, err := hz.client.Location.Get(ctx, location)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to retrieve Hetzner location '%s'", location)
	}
	if serverLocation == nil {
		return "", errors.Errorf("Hetzner location '%s' not found", location)
	}

	startAfterCreate := false
	log.Infof("Deploying Hetzner VM using image '%s'", imageID)
	userData, err := hetznerAuthorizedKeysUserData(pubKey)
	if err != nil {
		return "", errors.Wrap(err, "Failed to build Hetzner user data")
	}
	result, _, err := hz.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:             name,
		ServerType:       serverType,
		Image:            image,
		SSHKeys:          []*hcloud.SSHKey{sshKey},
		Location:         serverLocation,
		UserData:         userData,
		StartAfterCreate: &startAfterCreate,
		Labels: map[string]string{
			hetznerLabelManaged: "true",
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create Hetzner VM")
	}
	if err := hz.waitForActions(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
		return "", errors.Wrap(err, "Failed waiting for Hetzner VM creation")
	}
	log.Infof("Created Hetzner server '%s' (%d)", result.Server.Name, result.Server.ID)
	return strconv.FormatInt(result.Server.ID, 10), nil
}

func (hz *hetzner) DeleteInstance(id string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	server, err := hz.getServer(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.Errorf("Hetzner instance '%s' not found", id)
	}

	if _, _, err := hz.client.Server.DeleteWithResult(ctx, server); err != nil {
		return errors.Wrapf(err, "Failed to delete Hetzner instance '%s'", id)
	}
	if err := hz.deleteSSHKeysByName(ctx, server.Name); err != nil {
		log.Warnf("Failed to delete Hetzner SSH key for instance '%s': %s", server.Name, err.Error())
	}
	return nil
}

func (hz *hetzner) StartInstance(id string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	server, err := hz.getServer(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.Errorf("Hetzner instance '%s' not found", id)
	}
	if server.Status == hcloud.ServerStatusRunning {
		return nil
	}
	action, _, err := hz.client.Server.Poweron(ctx, server)
	if err != nil {
		return errors.Wrapf(err, "Failed to start Hetzner instance '%s'", id)
	}
	return hz.waitForActions(ctx, action)
}

func (hz *hetzner) StopInstance(id string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	server, err := hz.getServer(ctx, id)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.Errorf("Hetzner instance '%s' not found", id)
	}
	if server.Status == hcloud.ServerStatusOff {
		return nil
	}
	action, _, err := hz.client.Server.Poweroff(ctx, server)
	if err != nil {
		return errors.Wrapf(err, "Failed to stop Hetzner instance '%s'", id)
	}
	return hz.waitForActions(ctx, action)
}

func (hz *hetzner) GetInstanceInfo(id string, location string) (provisioners.InstanceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	server, err := hz.getServer(ctx, id)
	if err != nil {
		return provisioners.InstanceInfo{}, err
	}
	if server == nil {
		return provisioners.InstanceInfo{}, errors.Errorf("Hetzner instance '%s' not found", id)
	}

	info := provisioners.InstanceInfo{
		ID:                 strconv.FormatInt(server.ID, 10),
		Name:               server.Name,
		Kind:               provisioners.KindCloudVM,
		ProviderResourceID: strconv.FormatInt(server.ID, 10),
		Status:             transformStatus(server.Status),
	}
	if server.Location != nil {
		info.Location = server.Location.Name
	} else {
		info.Location = location
	}
	if !server.PublicNet.IPv4.IsUnspecified() {
		info.PublicIP = server.PublicNet.IPv4.IP.String()
	} else if !server.PublicNet.IPv6.IsUnspecified() {
		info.PublicIP = server.PublicNet.IPv6.IP.String()
	}
	for _, volume := range server.Volumes {
		info.Volumes = append(info.Volumes, provisioners.VolumeInfo{
			VolumeID: strconv.FormatInt(volume.ID, 10),
			Name:     volume.Name,
			Size:     uint64(volume.Size) * 1024 * 1024 * 1024,
		})
	}
	return info, nil
}

func (hz *hetzner) GetImages() (map[string]provisioners.ImageInfo, error) {
	return hz.getImages(false)
}

func (hz *hetzner) GetProtosImages() (map[string]provisioners.ImageInfo, error) {
	return hz.getImages(true)
}

func (hz *hetzner) AddImage(url string, hash string, version string, location string) (string, error) {
	errMsg := "Failed to add Protos image to Hetzner"
	protosImage := "protos-" + version

	key, sshKey, server, err := hz.createImageUploadServer(location)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer hz.cleanImageUploadServer(server)
	defer hz.cleanImageSSHKey(sshKey)

	sshClient, err := hz.connectImageUploadServer(server, key)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer sshClient.Close()

	remoteImage := "/tmp/" + sanitizeRemoteName(protosImage)
	log.Info("Downloading Protos image on Hetzner upload server")
	downloadCmd := fmt.Sprintf("(command -v wget >/dev/null && wget -O %s %s) || (command -v curl >/dev/null && curl -L -o %s %s)",
		shellQuote(remoteImage), shellQuote(url), shellQuote(remoteImage), shellQuote(url))
	out, err := pcrypto.ExecuteCommand(downloadCmd, sshClient)
	if err != nil {
		log.Errorf("Error downloading Protos VM image: %s", out)
		return "", errors.Wrap(err, errMsg+". Error downloading Protos VM image")
	}
	if err := verifyRemoteDigest(sshClient, remoteImage, hash); err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	if err := writeRemoteImageToDisk(sshClient, remoteImage); err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	imageID, err := hz.snapshotImageServer(server, version, location)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	return imageID, nil
}

func (hz *hetzner) UploadLocalImage(imagePath string, imageName string, location string, timeout time.Duration) (id string, err error) {
	errMsg := "Failed to upload Protos image to Hetzner"
	protosImage := "protos-" + imageName

	imageHash, err := fileSHA256(imagePath)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	key, sshKey, server, err := hz.createImageUploadServer(location)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	defer hz.cleanImageUploadServer(server)
	defer hz.cleanImageSSHKey(sshKey)

	if err := util.WaitForPort(hetznerServerPublicIP(server), "22", 30); err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			key.SSHAuth(),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client := scp.NewClient(hetznerServerPublicIP(server)+":22", sshConfig)
	if timeout > 0 {
		client.Timeout = timeout
	}
	log.Infof("Connecting via SSH and starting SCP transfer to '%s'", hetznerServerPublicIP(server)+":22")
	if err := client.Connect(); err != nil {
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

	remoteImage := "/tmp/" + sanitizeRemoteName(protosImage)
	log.Info("Uploading image to Hetzner rescue server. This can take a while...")
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

	sshClient, err := pcrypto.NewConnection(hetznerServerPublicIP(server), "root", key.SSHAuth(), 10)
	if err != nil {
		return "", errors.Wrap(err, errMsg+". Failed to connect to Hetzner rescue server")
	}
	defer sshClient.Close()

	if err := verifyRemoteDigest(sshClient, remoteImage, imageHash); err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	if err := writeRemoteImageToDisk(sshClient, remoteImage); err != nil {
		return "", errors.Wrap(err, errMsg)
	}

	imageID, err := hz.snapshotImageServer(server, imageName, location)
	if err != nil {
		return "", errors.Wrap(err, errMsg)
	}
	return imageID, nil
}

func (hz *hetzner) RemoveImage(name string, location string) error {
	errMsg := fmt.Sprintf("Failed to remove image '%s' in '%s'", name, location)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	images, err := hz.listProtosHcloudImages(ctx)
	if err != nil {
		return errors.Wrap(err, errMsg)
	}
	for _, image := range images {
		imageName := hetznerImageName(image)
		imageLocation := image.Labels[hetznerLabelImageLocation]
		if imageName == name && (location == "" || imageLocation == "" || imageLocation == location) {
			if _, err := hz.client.Image.Delete(ctx, image); err != nil {
				return errors.Wrap(err, errMsg)
			}
			return nil
		}
	}
	return fmt.Errorf("%s: could not find image '%s'", errMsg, name)
}

func (hz *hetzner) NewVolume(name string, size int, location string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	serverLocation, _, err := hz.client.Location.Get(ctx, location)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to retrieve Hetzner location '%s'", location)
	}
	if serverLocation == nil {
		return "", errors.Errorf("Hetzner location '%s' not found", location)
	}

	result, _, err := hz.client.Volume.Create(ctx, hcloud.VolumeCreateOpts{
		Name:     name,
		Size:     volumeSizeGiB(size),
		Location: serverLocation,
		Labels: map[string]string{
			hetznerLabelManaged: "true",
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create Hetzner volume")
	}
	if err := hz.waitForActions(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
		return "", errors.Wrap(err, "Failed waiting for Hetzner volume")
	}
	return strconv.FormatInt(result.Volume.ID, 10), nil
}

func (hz *hetzner) DeleteVolume(id string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	volume, err := hz.getVolume(ctx, id)
	if err != nil {
		return err
	}
	if volume == nil {
		return nil
	}
	if volume.Server != nil {
		action, _, err := hz.client.Volume.Detach(ctx, volume)
		if err != nil {
			return errors.Wrapf(err, "Failed to detach Hetzner volume '%s'", id)
		}
		if err := hz.waitForActions(ctx, action); err != nil {
			return errors.Wrapf(err, "Failed waiting for Hetzner volume '%s' detach", id)
		}
	}
	if _, err := hz.client.Volume.Delete(ctx, volume); err != nil {
		return errors.Wrapf(err, "Failed to delete Hetzner volume '%s'", id)
	}
	return nil
}

func (hz *hetzner) AttachVolume(volumeID string, instanceID string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	volume, err := hz.getVolume(ctx, volumeID)
	if err != nil {
		return err
	}
	if volume == nil {
		return errors.Errorf("Hetzner volume '%s' not found", volumeID)
	}
	server, err := hz.getServer(ctx, instanceID)
	if err != nil {
		return err
	}
	if server == nil {
		return errors.Errorf("Hetzner instance '%s' not found", instanceID)
	}
	action, _, err := hz.client.Volume.Attach(ctx, volume, server)
	if err != nil {
		return errors.Wrapf(err, "Failed to attach Hetzner volume '%s' to instance '%s'", volumeID, instanceID)
	}
	return hz.waitForActions(ctx, action)
}

func (hz *hetzner) DettachVolume(volumeID string, instanceID string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	volume, err := hz.getVolume(ctx, volumeID)
	if err != nil {
		return err
	}
	if volume == nil {
		return errors.Errorf("Hetzner volume '%s' not found", volumeID)
	}
	action, _, err := hz.client.Volume.Detach(ctx, volume)
	if err != nil {
		return errors.Wrapf(err, "Failed to detach Hetzner volume '%s'", volumeID)
	}
	return hz.waitForActions(ctx, action)
}

func (hz *hetzner) getImages(protosOnly bool) (map[string]provisioners.ImageInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var (
		images []*hcloud.Image
		err    error
	)
	if protosOnly {
		images, err = hz.listProtosHcloudImages(ctx)
	} else {
		images, err = hz.client.Image.AllWithOpts(ctx, hcloud.ImageListOpts{
			Type:              []hcloud.ImageType{hcloud.ImageTypeSnapshot},
			IncludeDeprecated: true,
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve account images from Hetzner")
	}

	result := map[string]provisioners.ImageInfo{}
	for _, image := range images {
		if image == nil {
			continue
		}
		if image.Status != hcloud.ImageStatusAvailable {
			continue
		}
		imageName := hetznerImageName(image)
		if protosOnly && imageName == "" {
			continue
		}
		if imageName == "" {
			imageName = firstNonEmpty(image.Description, image.Name, strconv.FormatInt(image.ID, 10))
		}
		result[strconv.FormatInt(image.ID, 10)] = provisioners.ImageInfo{
			ID:       strconv.FormatInt(image.ID, 10),
			Name:     imageName,
			Location: image.Labels[hetznerLabelImageLocation],
		}
	}
	return result, nil
}

func (hz *hetzner) listProtosHcloudImages(ctx context.Context) ([]*hcloud.Image, error) {
	images, err := hz.client.Image.AllWithOpts(ctx, hcloud.ImageListOpts{
		Type:              []hcloud.ImageType{hcloud.ImageTypeSnapshot},
		IncludeDeprecated: true,
	})
	if err != nil {
		return nil, err
	}
	filtered := make([]*hcloud.Image, 0, len(images))
	for _, image := range images {
		if image == nil {
			continue
		}
		if image.Labels[hetznerLabelImage] == "true" || strings.HasPrefix(image.Description, "protos-") {
			filtered = append(filtered, image)
		}
	}
	return filtered, nil
}

func (hz *hetzner) createImageUploadServer(location string) (*pcrypto.Key, *hcloud.SSHKey, *hcloud.Server, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	key, err := hz.sm.GenerateKey()
	if err != nil {
		return nil, nil, nil, err
	}
	pubKey := strings.TrimSuffix(key.AuthorizedKey(), "\n") + " root@protos.io"

	sshKey, err := hz.createSSHKey(ctx, hetznerUploadSSHKeyName, pubKey, map[string]string{
		hetznerLabelManaged:   "true",
		hetznerLabelTemporary: "true",
	})
	if err != nil {
		return nil, nil, nil, err
	}

	var server *hcloud.Server
	cleanupOnError := true
	defer func() {
		if cleanupOnError {
			hz.cleanImageUploadServer(server)
			hz.cleanImageSSHKey(sshKey)
		}
	}()

	serverType, err := hz.selectUploadServerType(ctx, location)
	if err != nil {
		return nil, nil, nil, err
	}
	baseImage, _, err := hz.client.Image.GetForArchitecture(ctx, hetznerUploadBaseImageName, hcloud.ArchitectureX86)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "Failed to retrieve Hetzner image '%s'", hetznerUploadBaseImageName)
	}
	serverLocation, _, err := hz.client.Location.Get(ctx, location)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "Failed to retrieve Hetzner location '%s'", location)
	}
	if baseImage == nil || serverLocation == nil {
		return nil, nil, nil, errors.New("Failed to resolve Hetzner upload server inputs")
	}

	startAfterCreate := false
	serverName := fmt.Sprintf("%s-%d", hetznerUploadServerName, time.Now().Unix())
	log.Infof("Creating Hetzner image upload server '%s'", serverName)
	result, _, err := hz.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:             serverName,
		ServerType:       serverType,
		Image:            baseImage,
		SSHKeys:          []*hcloud.SSHKey{sshKey},
		Location:         serverLocation,
		StartAfterCreate: &startAfterCreate,
		Labels: map[string]string{
			hetznerLabelManaged:   "true",
			hetznerLabelTemporary: "true",
		},
	})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed to create Hetzner image upload server")
	}
	server = result.Server
	if err := hz.waitForActions(ctx, actionutil.AppendNext(result.Action, result.NextActions)...); err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed waiting for Hetzner image upload server creation")
	}

	server, _, err = hz.client.Server.GetByID(ctx, server.ID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed to refresh Hetzner image upload server")
	}

	log.Infof("Enabling rescue mode for Hetzner image upload server '%s'", server.Name)
	rescue, _, err := hz.client.Server.EnableRescue(ctx, server, hcloud.ServerEnableRescueOpts{
		Type:    hcloud.ServerRescueTypeLinux64,
		SSHKeys: []*hcloud.SSHKey{sshKey},
	})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed to enable Hetzner rescue mode")
	}
	if err := hz.waitForActions(ctx, rescue.Action); err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed waiting for Hetzner rescue mode")
	}

	log.Infof("Starting Hetzner image upload server '%s' in rescue mode", server.Name)
	action, _, err := hz.client.Server.Poweron(ctx, server)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed to start Hetzner image upload server")
	}
	if err := hz.waitForActions(ctx, action); err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed waiting for Hetzner image upload server start")
	}
	server, _, err = hz.client.Server.GetByID(ctx, server.ID)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "Failed to refresh Hetzner image upload server")
	}
	log.Infof("Hetzner image upload server '%s' is reachable at '%s'", server.Name, hetznerServerPublicIP(server))

	cleanupOnError = false
	return key, sshKey, server, nil
}

func (hz *hetzner) selectUploadServerType(ctx context.Context, location string) (*hcloud.ServerType, error) {
	if serverType, _, err := hz.client.ServerType.Get(ctx, hetznerUploadServerType); err == nil && serverType != nil && hetznerServerTypeAvailableIn(serverType, location) {
		return serverType, nil
	}

	serverTypes, err := hz.client.ServerType.All(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to retrieve Hetzner server types")
	}
	candidates := make([]*hcloud.ServerType, 0, len(serverTypes))
	for _, serverType := range serverTypes {
		if serverType.Architecture != hcloud.ArchitectureX86 {
			continue
		}
		if !hetznerServerTypeAvailableIn(serverType, location) {
			continue
		}
		candidates = append(candidates, serverType)
	}
	sort.Slice(candidates, func(i int, j int) bool {
		iPrice := hetznerMonthlyPrice(candidates[i], location)
		jPrice := hetznerMonthlyPrice(candidates[j], location)
		if iPrice != jPrice {
			return iPrice < jPrice
		}
		if candidates[i].Memory != candidates[j].Memory {
			return candidates[i].Memory < candidates[j].Memory
		}
		return candidates[i].Name < candidates[j].Name
	})
	if len(candidates) == 0 {
		return nil, errors.Errorf("No available x86 Hetzner server type found in location '%s'", location)
	}
	log.Infof("Using Hetzner upload server type '%s' in '%s'", candidates[0].Name, location)
	return candidates[0], nil
}

func (hz *hetzner) connectImageUploadServer(server *hcloud.Server, key *pcrypto.Key) (*ssh.Client, error) {
	publicIP := hetznerServerPublicIP(server)
	log.Infof("Waiting for SSH service to be reachable at '%s'", publicIP+":22")
	if err := util.WaitForPort(publicIP, "22", 30); err != nil {
		return nil, err
	}
	log.Info("Trying to connect to Hetzner rescue server over SSH")
	sshClient, err := pcrypto.NewConnection(publicIP, "root", key.SSHAuth(), 10)
	if err != nil {
		return nil, err
	}
	log.Info("SSH connection initiated")
	return sshClient, nil
}

func (hz *hetzner) snapshotImageServer(server *hcloud.Server, imageName string, location string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	if err := hz.poweroffServer(ctx, server); err != nil {
		return "", err
	}

	description := "protos-" + imageName
	log.Infof("Creating Hetzner snapshot image '%s'", description)
	result, _, err := hz.client.Server.CreateImage(ctx, server, &hcloud.ServerCreateImageOpts{
		Type:        hcloud.ImageTypeSnapshot,
		Description: &description,
		Labels: map[string]string{
			hetznerLabelManaged:       "true",
			hetznerLabelImage:         "true",
			hetznerLabelImageName:     imageName,
			hetznerLabelImageLocation: location,
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "Failed to create Hetzner snapshot image")
	}
	if err := hz.waitForActions(ctx, result.Action); err != nil {
		return "", errors.Wrap(err, "Failed waiting for Hetzner snapshot image")
	}
	log.Infof("Protos image '%s' created as Hetzner snapshot '%d'", description, result.Image.ID)
	return strconv.FormatInt(result.Image.ID, 10), nil
}

func (hz *hetzner) poweroffServer(ctx context.Context, server *hcloud.Server) error {
	refreshed, _, err := hz.client.Server.GetByID(ctx, server.ID)
	if err != nil {
		return errors.Wrap(err, "Failed to refresh Hetzner server")
	}
	if refreshed == nil || refreshed.Status == hcloud.ServerStatusOff {
		return nil
	}
	log.Infof("Powering off Hetzner server '%s' (%d)", refreshed.Name, refreshed.ID)
	action, _, err := hz.client.Server.Poweroff(ctx, refreshed)
	if err != nil {
		return errors.Wrap(err, "Failed to power off Hetzner server")
	}
	return hz.waitForActions(ctx, action)
}

func (hz *hetzner) cleanImageUploadServer(server *hcloud.Server) {
	if server == nil || server.ID == 0 || hz.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := hz.poweroffServer(ctx, server); err != nil {
		log.Warnf("Failed to stop Hetzner image upload server '%s': %s", server.Name, err.Error())
	}
	if _, _, err := hz.client.Server.DeleteWithResult(ctx, server); err != nil {
		log.Warnf("Failed to delete Hetzner image upload server '%s': %s", server.Name, err.Error())
		return
	}
	log.Infof("Deleted Hetzner image upload server '%s' (%d)", server.Name, server.ID)
}

func (hz *hetzner) cleanImageSSHKey(sshKey *hcloud.SSHKey) {
	if sshKey == nil || sshKey.ID == 0 || hz.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := hz.client.SSHKey.Delete(ctx, sshKey); err != nil {
		log.Warnf("Failed to delete Hetzner image upload key '%s': %s", sshKey.Name, err.Error())
		return
	}
	log.Infof("Deleted Hetzner SSH key '%s'", sshKey.Name)
}

func (hz *hetzner) createSSHKey(ctx context.Context, name string, pubKey string, labels map[string]string) (*hcloud.SSHKey, error) {
	if err := hz.deleteSSHKeysByName(ctx, name); err != nil {
		return nil, err
	}
	pubKey = strings.TrimSpace(pubKey)
	sshKey, _, err := hz.client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      name,
		PublicKey: pubKey,
		Labels:    labels,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Hetzner SSH key '%s'", name)
	}
	return sshKey, nil
}

func (hz *hetzner) deleteSSHKeysByName(ctx context.Context, name string) error {
	keys, _, err := hz.client.SSHKey.List(ctx, hcloud.SSHKeyListOpts{Name: name})
	if err != nil {
		return errors.Wrap(err, "Failed to list Hetzner SSH keys")
	}
	for _, key := range keys {
		if key.Name != name {
			continue
		}
		if _, err := hz.client.SSHKey.Delete(ctx, key); err != nil {
			return errors.Wrapf(err, "Failed to delete Hetzner SSH key '%s'", name)
		}
	}
	return nil
}

func (hz *hetzner) getServer(ctx context.Context, idOrName string) (*hcloud.Server, error) {
	server, _, err := hz.client.Server.Get(ctx, idOrName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve Hetzner instance '%s'", idOrName)
	}
	return server, nil
}

func (hz *hetzner) getImage(ctx context.Context, idOrName string) (*hcloud.Image, error) {
	image, _, err := hz.client.Image.GetForArchitecture(ctx, idOrName, hcloud.ArchitectureX86)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve Hetzner image '%s'", idOrName)
	}
	if image == nil {
		return nil, errors.Errorf("Hetzner image '%s' not found", idOrName)
	}
	return image, nil
}

func (hz *hetzner) getVolume(ctx context.Context, idOrName string) (*hcloud.Volume, error) {
	volume, _, err := hz.client.Volume.Get(ctx, idOrName)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to retrieve Hetzner volume '%s'", idOrName)
	}
	return volume, nil
}

func (hz *hetzner) waitForActions(ctx context.Context, actions ...*hcloud.Action) error {
	filtered := make([]*hcloud.Action, 0, len(actions))
	for _, action := range actions {
		if action != nil {
			filtered = append(filtered, action)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return hz.client.Action.WaitFor(ctx, filtered...)
}

func transformStatus(status hcloud.ServerStatus) string {
	switch status {
	case hcloud.ServerStatusRunning:
		return provisioners.ServerStateRunning
	case hcloud.ServerStatusOff:
		return provisioners.ServerStateStopped
	case hcloud.ServerStatusStarting, hcloud.ServerStatusStopping, hcloud.ServerStatusInitializing:
		return provisioners.ServerStateChanging
	default:
		return provisioners.ServerStateOther
	}
}

func hetznerServerTypeAvailableIn(serverType *hcloud.ServerType, location string) bool {
	for _, serverLocation := range serverType.Locations {
		if serverLocation.Location != nil && serverLocation.Location.Name == location {
			return serverLocation.Available
		}
	}
	return false
}

func hetznerIncludedTrafficGiB(serverType *hcloud.ServerType, location string) uint32 {
	for _, pricing := range serverType.Pricings {
		if pricing.Location != nil && pricing.Location.Name == location {
			return uint32(pricing.IncludedTraffic / 1024 / 1024 / 1024)
		}
	}
	return 0
}

func hetznerMonthlyPrice(serverType *hcloud.ServerType, location string) float32 {
	for _, pricing := range serverType.Pricings {
		if pricing.Location == nil || pricing.Location.Name != location {
			continue
		}
		value, err := strconv.ParseFloat(pricing.Monthly.Net, 32)
		if err != nil {
			return 0
		}
		return float32(value)
	}
	return 0
}

func hetznerServerPublicIP(server *hcloud.Server) string {
	if server == nil {
		return ""
	}
	if !server.PublicNet.IPv4.IsUnspecified() {
		return server.PublicNet.IPv4.IP.String()
	}
	if !server.PublicNet.IPv6.IsUnspecified() {
		return server.PublicNet.IPv6.IP.String()
	}
	return ""
}

func hetznerImageName(image *hcloud.Image) string {
	if image == nil {
		return ""
	}
	if name := strings.TrimSpace(image.Labels[hetznerLabelImageName]); name != "" {
		return name
	}
	if strings.HasPrefix(image.Description, "protos-") {
		return strings.TrimPrefix(image.Description, "protos-")
	}
	return ""
}

func fileSHA256(path string) (string, error) {
	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	h := sha256.New()
	if _, err := io.Copy(h, fd); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifyRemoteDigest(sshClient *ssh.Client, remoteImage string, expectedHash string) error {
	log.Info("Checking image integrity")
	cmdString := fmt.Sprintf("openssl dgst -r -sha256 %s | awk '{ print $1 }' | { read digest; if [ \"$digest\" = %s ]; then true; else echo \"digest $digest does not match expected %s\"; false; fi }",
		shellQuote(remoteImage), shellQuote(expectedHash), shellQuote(expectedHash))
	out, err := pcrypto.ExecuteCommand(cmdString, sshClient)
	if err != nil {
		log.Errorf("Image integrity check failed: %s: %s", out, err.Error())
		return errors.New("image integrity check failed")
	}
	return nil
}

func writeRemoteImageToDisk(sshClient *ssh.Client, remoteImage string) error {
	log.Info("Writing Protos image to Hetzner server disk")
	cmdString := fmt.Sprintf(`set -eu
disk=$(lsblk -dpno NAME,TYPE | awk '$2=="disk" && $1 !~ /loop|ram/ { print $1; exit }')
if [ -z "$disk" ]; then
  echo "No target disk found" >&2
  exit 1
fi
echo "Writing %s to $disk"
dd if=%s of="$disk" bs=16M conv=fsync status=none
sync
`, shellQuote(remoteImage), shellQuote(remoteImage))
	out, err := pcrypto.ExecuteCommand(cmdString, sshClient)
	if err != nil {
		log.Errorf("Error while writing image to server disk: %s", out)
		return errors.New("error while writing image to server disk")
	}
	return nil
}

func volumeSizeGiB(sizeMiB int) int {
	if sizeMiB <= 0 {
		return 10
	}
	sizeGiB := (sizeMiB + 1023) / 1024
	if sizeGiB < 10 {
		return 10
	}
	return sizeGiB
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeRemoteName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "protos-image"
	}
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", "'", "-", "\"", "-")
	return replacer.Replace(name)
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func hetznerAuthorizedKeysUserData(pubKey string) (string, error) {
	authorizedKeys := strings.TrimSpace(pubKey) + "\n"
	return fmt.Sprintf(`#!/bin/sh
set -eu
mkdir -p /run/config/ssh
cat > /run/config/ssh/authorized_keys <<'PROTOS_AUTHORIZED_KEYS'
%sPROTOS_AUTHORIZED_KEYS
chmod 0600 /run/config/ssh/authorized_keys
`, authorizedKeys), nil
}
