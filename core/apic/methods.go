package apic

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/netip"
	"os"
	"sort"
	"strings"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/invitations"
	"github.com/protosio/protos/internal/network"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/p2p"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/user"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
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

	err := b.protosClient.Init(in.Username, in.Name, in.Organisation)
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
	organisation, err := db.EnsureOrganisation(b.protosClient.DB, db.DefaultOrganisationName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve organisation info: %w", err)
	}

	resp := pbApic.GetUserInfoResponse{
		Username:         adminUser.Username,
		Name:             adminUser.Name,
		IsAdmin:          adminUser.IsAdmin(),
		OrganisationId:   organisation.ID,
		OrganisationName: organisation.Name,
	}

	return &resp, nil
}

func (b *Backend) ListOrganisations(ctx context.Context, in *pbApic.ListOrganisationsRequest) (*pbApic.ListOrganisationsResponse, error) {
	log.Debugf("Listing organisations")
	if _, err := db.EnsureOrganisation(b.protosClient.DB, db.DefaultOrganisationName); err != nil {
		return nil, fmt.Errorf("failed to ensure organisation info: %w", err)
	}
	organisations, err := db.ListOrganisations(b.protosClient.DB)
	if err != nil {
		return nil, err
	}
	resp := pbApic.ListOrganisationsResponse{}
	for _, organisation := range organisations {
		resp.Organisations = append(resp.Organisations, protoOrganisation(organisation))
	}
	return &resp, nil
}

func (b *Backend) StartDeviceInvite(ctx context.Context, in *pbApic.StartDeviceInviteRequest) (*pbApic.StartDeviceInviteResponse, error) {
	if b.protosClient == nil || b.protosClient.Invites == nil {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "local invite channels are not available")
	}
	if b.protosClient.DB == nil || !b.protosClient.DB.Initialized() {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "Protos must be initialized before inviting a device")
	}
	if b.protosClient.KeyManager == nil || b.protosClient.Manager == nil || b.protosClient.P2PManager == nil {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "local peer identity is not available")
	}

	organisation, err := b.organisationForInvite(in.GetOrganisationId())
	if err != nil {
		return nil, err
	}
	joinMode, err := startInviteJoinMode(in.GetJoinMode())
	if err != nil {
		return nil, err
	}
	targetUserID, err := b.inviteTargetUserID(joinMode, in.GetUsername())
	if err != nil {
		return nil, err
	}
	localKey, err := b.protosClient.KeyManager.GetLocalKey()
	if err != nil {
		return nil, fmt.Errorf("load local key: %w", err)
	}
	peerID := strings.TrimSpace(b.protosClient.P2PManager.PeerID())
	if peerID == "" {
		peerID = localKey.GetID()
	}
	deviceName := localDeviceName(b.protosClient.Manager)
	localIPs := localInviteIPs()
	swarmionAddrs := b.protosClient.DB.DialableListenMultiaddrs(localIPs)
	if len(swarmionAddrs) == 0 {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "no local database bootstrap addresses are available")
	}
	log.Infof(
		"starting %s invite for organisation %q via channel %q with %d local IPs and %d bootstrap addresses",
		joinMode,
		organisation.ID,
		in.GetChannel(),
		len(localIPs),
		len(swarmionAddrs),
	)
	invite, err := b.protosClient.Invites.StartInvite(ctx, in.GetChannel(), invitations.Invite{
		OrganisationID:   organisation.ID,
		OrganisationName: organisation.Name,
		DeviceName:       deviceName,
		PeerID:           peerID,
		PublicKey:        localKey.PublicString(),
		JoinMode:         joinMode,
		TargetUserID:     targetUserID,
		Port:             b.protosClient.P2PPort,
		P2PAddrs:         p2pListenAddrsWithPeerID(b.protosClient.P2PManager.ListenAddresses(), peerID),
		SwarmionAddrs:    swarmionAddrs,
	})
	if err != nil {
		return nil, err
	}
	log.Infof(
		"device invite %q active via %q as %q with verification code %q",
		invite.InviteID,
		invite.Channel,
		invite.AdvertiseName,
		invite.VerificationCode,
	)
	return &pbApic.StartDeviceInviteResponse{
		InviteId:         invite.InviteID,
		ExpiresAtUnix:    invite.ExpiresAt.Unix(),
		AdvertiseName:    invite.AdvertiseName,
		AdvertiseService: invite.AdvertiseService,
		Channel:          invite.Channel,
		VerificationCode: invite.VerificationCode,
		JoinMode:         invite.JoinMode,
	}, nil
}

func (b *Backend) ListNearbyOrganisations(ctx context.Context, in *pbApic.ListNearbyOrganisationsRequest) (*pbApic.ListNearbyOrganisationsResponse, error) {
	if b.protosClient == nil || b.protosClient.Invites == nil {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "local invite channels are not available")
	}
	nearby, err := b.protosClient.Invites.Browse(ctx, in.GetChannel(), 3*time.Second)
	if err != nil {
		return nil, err
	}
	localPeerID := ""
	if b.protosClient.P2PManager != nil {
		localPeerID = strings.TrimSpace(b.protosClient.P2PManager.PeerID())
	}
	resp := pbApic.ListNearbyOrganisationsResponse{}
	for _, item := range nearby {
		if item.PeerID == localPeerID {
			continue
		}
		resp.Organisations = append(resp.Organisations, &pbApic.NearbyOrganisation{
			OrganisationId:   item.OrganisationID,
			OrganisationName: item.OrganisationName,
			DeviceName:       item.DeviceName,
			PeerId:           item.PeerID,
			InviteId:         item.InviteID,
			Channel:          item.Channel,
			JoinMode:         item.JoinMode,
		})
	}
	return &resp, nil
}

func (b *Backend) JoinOrganisation(ctx context.Context, in *pbApic.JoinOrganisationRequest) (*pbApic.JoinOrganisationResponse, error) {
	if b.protosClient == nil || b.protosClient.Invites == nil {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "local invite channels are not available")
	}
	if b.protosClient.DB == nil || b.protosClient.Manager == nil || b.protosClient.KeyManager == nil {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "local initialization services are not available")
	}
	if b.protosClient.DB.Initialized() {
		return nil, grpcstatus.Error(codes.FailedPrecondition, "this device is already initialized")
	}
	nearby, err := b.protosClient.Invites.Find(ctx, in.GetChannel(), in.GetOrganisationId(), in.GetPeerId(), in.GetInviteId())
	if err != nil {
		return nil, err
	}
	if err := invitations.VerifyNearbyInviteCode(nearby, in.GetVerificationCode()); err != nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}
	if err := verifyInvitePeerKey(nearby); err != nil {
		return nil, grpcstatus.Error(codes.InvalidArgument, err.Error())
	}
	joinMode, err := effectiveJoinMode(nearby.JoinMode, in.GetJoinMode())
	if err != nil {
		return nil, err
	}
	if err := b.protosClient.DB.InitFromPeer(nearby.PeerID, nearby.SwarmionAddrs); err != nil {
		return nil, fmt.Errorf("join organisation from %s: %w", nearby.PeerID, err)
	}
	if err := b.protosClient.DB.CatchUpCheckpoint(ctx, "join organisation user/device registration"); err != nil {
		return nil, fmt.Errorf("sync organisation state after joining: %w", err)
	}
	if err := b.ensureJoinedUserDevice(in.GetUsername(), in.GetName(), joinMode, nearby.TargetUserID); err != nil {
		return nil, err
	}
	b.protosClient.MarkInitializedIfNeeded()
	return &pbApic.JoinOrganisationResponse{}, nil
}

func startInviteJoinMode(rawMode string) (string, error) {
	joinMode := invitations.NormalizeInviteJoinMode(rawMode)
	if strings.TrimSpace(rawMode) == "" {
		joinMode = invitations.InviteJoinModeNewDevice
	}
	switch joinMode {
	case invitations.InviteJoinModeNewUser, invitations.InviteJoinModeNewDevice:
		return joinMode, nil
	default:
		return "", grpcstatus.Errorf(codes.InvalidArgument, "invite join mode must be %q or %q", invitations.InviteJoinModeNewUser, invitations.InviteJoinModeNewDevice)
	}
}

func effectiveJoinMode(inviteMode string, requestMode string) (string, error) {
	inviteMode = invitations.NormalizeInviteJoinMode(inviteMode)
	if !invitations.ValidInviteJoinMode(inviteMode) {
		return "", grpcstatus.Errorf(codes.InvalidArgument, "invite advertised invalid join mode %q", inviteMode)
	}
	requestMode = strings.TrimSpace(requestMode)
	if requestMode != "" {
		requestMode = invitations.NormalizeInviteJoinMode(requestMode)
		if !invitations.ValidInviteJoinMode(requestMode) {
			return "", grpcstatus.Errorf(codes.InvalidArgument, "join mode must be %q or %q", invitations.InviteJoinModeNewUser, invitations.InviteJoinModeNewDevice)
		}
	}
	if inviteMode != invitations.InviteJoinModeAny {
		if requestMode != "" && requestMode != invitations.InviteJoinModeAny && requestMode != inviteMode {
			return "", grpcstatus.Errorf(codes.InvalidArgument, "invite is for %s, not %s", inviteMode, requestMode)
		}
		return inviteMode, nil
	}
	if requestMode != "" {
		return requestMode, nil
	}
	return invitations.InviteJoinModeAny, nil
}

func (b *Backend) inviteTargetUserID(joinMode string, username string) (string, error) {
	if joinMode != invitations.InviteJoinModeNewDevice {
		return "", nil
	}
	username = strings.TrimSpace(username)
	if username != "" {
		targetUser, err := b.protosClient.Manager.GetUser(username)
		if err != nil {
			return "", grpcstatus.Errorf(codes.InvalidArgument, "user %q does not exist", username)
		}
		return targetUser.ID, nil
	}
	if currentUser, err := b.protosClient.Manager.GetCurrentUser(); err == nil {
		return currentUser.ID, nil
	}
	adminUser, err := b.protosClient.Manager.GetAdmin()
	if err != nil {
		return "", fmt.Errorf("resolve target user for device invite: %w", err)
	}
	return adminUser.ID, nil
}

func protoOrganisation(organisation db.Organisation) *pbApic.Organisation {
	return &pbApic.Organisation{
		Id:        organisation.ID,
		Name:      organisation.Name,
		CreatedAt: organisation.CreatedAt,
	}
}

func (b *Backend) organisationForInvite(id string) (db.Organisation, error) {
	id = strings.TrimSpace(id)
	organisations, err := db.ListOrganisations(b.protosClient.DB)
	if err != nil {
		return db.Organisation{}, fmt.Errorf("list organisations: %w", err)
	}
	if id != "" {
		for _, organisation := range organisations {
			if organisation.ID == id {
				return organisation, nil
			}
		}
		return db.Organisation{}, grpcstatus.Errorf(codes.NotFound, "organisation %s was not found", id)
	}
	if len(organisations) == 0 {
		return db.EnsureOrganisation(b.protosClient.DB, db.DefaultOrganisationName)
	}
	sort.Slice(organisations, func(i, j int) bool {
		return organisations[i].CreatedAt < organisations[j].CreatedAt
	})
	return organisations[0], nil
}

func (b *Backend) ensureJoinedUserDevice(username string, name string, joinMode string, targetUserID string) error {
	username = strings.TrimSpace(username)
	name = strings.TrimSpace(name)
	targetUserID = strings.TrimSpace(targetUserID)
	joinMode = invitations.NormalizeInviteJoinMode(joinMode)
	if !invitations.ValidInviteJoinMode(joinMode) {
		return grpcstatus.Errorf(codes.InvalidArgument, "join mode must be %q or %q", invitations.InviteJoinModeNewUser, invitations.InviteJoinModeNewDevice)
	}
	switch joinMode {
	case invitations.InviteJoinModeNewUser:
		if username == "" {
			return grpcstatus.Error(codes.InvalidArgument, "username is required")
		}
		if name == "" {
			return grpcstatus.Error(codes.InvalidArgument, "name is required")
		}
		if _, err := b.protosClient.Manager.GetUser(username); err == nil {
			return grpcstatus.Errorf(codes.InvalidArgument, "user %q already exists", username)
		}
		createdUser, err := b.protosClient.Manager.CreateUser(username, name, false)
		if err != nil {
			return fmt.Errorf("create joined user: %w", err)
		}
		return b.addJoinedLocalDevice(createdUser.ID)
	case invitations.InviteJoinModeNewDevice:
		if targetUserID != "" {
			targetUser, err := b.protosClient.Manager.GetUserByID(targetUserID)
			if err != nil {
				return grpcstatus.Errorf(codes.InvalidArgument, "target user for device invite does not exist")
			}
			if username != "" && username != targetUser.Username {
				return grpcstatus.Errorf(codes.InvalidArgument, "invite is for user %q, not %q", targetUser.Username, username)
			}
			return b.addJoinedLocalDevice(targetUser.ID)
		}
		if username == "" {
			return grpcstatus.Error(codes.InvalidArgument, "username is required")
		}
		existingUser, err := b.protosClient.Manager.GetUser(username)
		if err != nil {
			return grpcstatus.Errorf(codes.InvalidArgument, "user %q does not exist", username)
		}
		return b.addJoinedLocalDevice(existingUser.ID)
	default:
		if username == "" {
			return grpcstatus.Error(codes.InvalidArgument, "username is required")
		}
		var userRef string
		if existingUser, err := b.protosClient.Manager.GetUser(username); err == nil {
			userRef = existingUser.ID
		} else {
			if name == "" {
				return grpcstatus.Error(codes.InvalidArgument, "name is required")
			}
			createdUser, err := b.protosClient.Manager.CreateUser(username, name, false)
			if err != nil {
				return fmt.Errorf("create joined user: %w", err)
			}
			userRef = createdUser.ID
		}
		return b.addJoinedLocalDevice(userRef)
	}
}

func (b *Backend) addJoinedLocalDevice(userRef string) error {
	if _, found, err := b.protosClient.Manager.GetCurrentDeviceIfExists(); err != nil {
		return err
	} else if found {
		return nil
	}
	localKey, err := b.protosClient.KeyManager.GetLocalKey()
	if err != nil {
		return fmt.Errorf("load local key: %w", err)
	}
	return b.protosClient.Manager.AddDevice(userRef, localDeviceName(b.protosClient.Manager), localKey)
}

func verifyInvitePeerKey(nearby invitations.NearbyInvite) error {
	derivedPeerID, err := pcrypto.PeerIDFromPublicKeyString(nearby.PublicKey)
	if err != nil {
		return fmt.Errorf("invalid invite public key: %w", err)
	}
	if derivedPeerID != nearby.PeerID {
		return fmt.Errorf("invite public key does not match peer id")
	}
	return nil
}

func localDeviceName(manager *user.Manager) string {
	if manager != nil {
		if device, found, err := manager.GetCurrentDeviceIfExists(); err == nil && found && strings.TrimSpace(device.GetName()) != "" {
			return strings.TrimSpace(device.GetName())
		}
	}
	hostname, err := os.Hostname()
	if err != nil || strings.TrimSpace(hostname) == "" {
		return "device"
	}
	return strings.TrimSpace(hostname)
}

func localInviteIPs() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var ips []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() {
				continue
			}
			ipString := ip.String()
			if _, found := seen[ipString]; found {
				continue
			}
			seen[ipString] = struct{}{}
			ips = append(ips, ipString)
		}
	}
	sort.Strings(ips)
	return ips
}

func p2pListenAddrsWithPeerID(addrs []string, peerID string) []string {
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if !strings.Contains(addr, "/p2p/") {
			addr += "/p2p/" + peerID
		}
		if _, found := seen[addr]; found {
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, addr)
	}
	return out
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
		instanceName := app.InstanceID
		peerID, instance, err := b.appInstancePeerID(app.InstanceID)
		if err != nil {
			log.Errorf("Failed to resolve instance for app '%s': %s", app.Name, err.Error())
		} else {
			instanceName = instance.Name
			client, err := b.protosClient.P2PManager.GetClient(peerID)
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
		}

		respApp := pbApic.App{
			Id:           app.ID,
			Name:         app.Name,
			Version:      app.GetVersion(),
			Status:       fmt.Sprintf("%s (%s)", status, app.DesiredStatus),
			InstanceName: instanceName,
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

	peerID, _, err := b.appInstancePeerID(app.InstanceID)
	if err != nil {
		return nil, fmt.Errorf("could not resolve instance for app '%s': %w", in.Name, err)
	}
	client, err := b.protosClient.P2PManager.GetClient(peerID)
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

func (b *Backend) appInstancePeerID(instanceID string) (string, provisioners.InstanceInfo, error) {
	if b.protosClient == nil || b.protosClient.CloudManager == nil {
		return "", provisioners.InstanceInfo{}, fmt.Errorf("cloud manager is not available")
	}
	instance, err := b.protosClient.CloudManager.GetInstance(instanceID)
	if err != nil {
		return "", provisioners.InstanceInfo{}, err
	}
	peerID, err := instance.GetPeerID()
	if err != nil {
		return "", provisioners.InstanceInfo{}, fmt.Errorf("derive peer id for instance %s: %w", instance.Name, err)
	}
	return peerID, instance, nil
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
		internalIP, wgPublicKey := instanceIdentityStrings(instance.PublicKey)
		cloudName, cloudType := b.instanceProvisionerLabels(instance)
		respInstance := pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         internalIP,
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey,
			Architecture:       instance.Architecture,
			Status:             instance.Status,
			CloudName:          cloudName,
			CloudType:          cloudType,
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

	internalIP, wgPublicKey := instanceIdentityStrings(instance.PublicKey)
	var status string
	peers := map[string]string{}
	if strings.TrimSpace(instance.PublicKey) == "" || !provisioners.IsActiveInstance(instance) {
		status = instance.Status
	} else {
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
				for peerID, peerStatus := range resp.GetPeers() {
					peers[peerID] = peerStatus
				}
			}
		}
	}

	cloudName, cloudType := b.instanceProvisionerLabels(instance)
	resp := pbApic.GetInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         internalIP,
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey,
			Status:             status,
			Architecture:       instance.Architecture,
			CloudName:          cloudName,
			CloudType:          cloudType,
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

	internalIP, wgPublicKey := instanceIdentityStrings(instance.PublicKey)

	resp := pbApic.DeployInstanceResponse{
		Instance: &pbApic.CloudInstance{
			Name:               instance.Name,
			PublicIp:           instance.PublicIP,
			InternalIp:         internalIP,
			VmId:               instance.ID,
			Location:           instance.Location,
			PublicKey:          instance.PublicKey,
			PublicKeyWireguard: wgPublicKey,
			Status:             instance.Status,
		},
	}

	return &resp, nil
}

func (b *Backend) instanceProvisionerLabels(instance provisioners.InstanceInfo) (string, string) {
	provisionerName := strings.TrimSpace(instance.KindID)
	provisionerType := strings.TrimSpace(instance.Kind)
	if provisionerName == "" || provisionerName == "local-id" {
		provisionerName = "local"
	}
	if b.protosClient == nil || b.protosClient.CloudManager == nil || strings.TrimSpace(instance.KindID) == "" {
		return provisionerName, provisionerType
	}
	provisioner, err := b.protosClient.CloudManager.GetProvisioner(instance.KindID)
	if err != nil {
		return provisionerName, provisionerType
	}
	return provisioner.NameStr(), provisioner.TypeStr()
}

func instanceIdentityStrings(publicKey string) (internalIP string, wireGuardPublicKey string) {
	publicKey = strings.TrimSpace(publicKey)
	if publicKey == "" {
		return "", ""
	}
	wgPublicKey, err := pcrypto.ConvertPublicEd25519ToCurve25519(publicKey)
	if err != nil {
		log.Error(err.Error())
	} else {
		wireGuardPublicKey = wgPublicKey.String()
	}
	pubKey, err := pcrypto.CreatePublicKeyFromBase64(publicKey)
	if err != nil {
		log.Error(err.Error())
		return internalIP, wireGuardPublicKey
	}
	return pubKey.IPv6Address().StringExpanded(), wireGuardPublicKey
}

func (b *Backend) RemoveInstance(ctx context.Context, in *pbApic.RemoveInstanceRequest) (*pbApic.RemoveInstanceResponse, error) {
	if err := b.requireProvisionerCapability("remove instance"); err != nil {
		return nil, err
	}
	log.Debugf("Removing instance '%s'", in.Name)
	var task tasks.Record
	var err error
	if in.LocalOnly {
		task, err = b.protosClient.CloudManager.QueueDeleteInstanceLocal(ctx, in.Name)
	} else {
		task, err = b.protosClient.CloudManager.QueueDeleteInstance(ctx, in.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to remove instance '%s': %w", in.Name, err)
	}

	return &pbApic.RemoveInstanceResponse{TaskId: task.ID}, nil
}

func (b *Backend) StartInstance(ctx context.Context, in *pbApic.StartInstanceRequest) (*pbApic.StartInstanceResponse, error) {
	if err := b.requireProvisionerCapability("start instance"); err != nil {
		return nil, err
	}
	log.Debugf("Starting instance '%s'", in.Name)
	task, err := b.protosClient.CloudManager.QueueStartInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to start instance '%s': %w", in.Name, err)
	}
	return &pbApic.StartInstanceResponse{TaskId: task.ID}, nil
}

func (b *Backend) StopInstance(ctx context.Context, in *pbApic.StopInstanceRequest) (*pbApic.StopInstanceResponse, error) {
	if err := b.requireProvisionerCapability("stop instance"); err != nil {
		return nil, err
	}
	log.Debugf("Stopping instance '%s'", in.Name)
	task, err := b.protosClient.CloudManager.QueueStopInstance(in.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to stop instance '%s': %w", in.Name, err)
	}
	return &pbApic.StopInstanceResponse{TaskId: task.ID}, nil
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
		if b.protosClient.NetworkControl == nil {
			return &pbApic.GetNetworkStateResponse{State: networkStateToProto(networkmodule.State{
				Module:   "none",
				Up:       false,
				Messages: []string{"network control is not available"},
			})}, nil
		}
		state, err := b.protosClient.NetworkControl.NetworkState(ctx)
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
	state := runtimeStateFromP2PProto(resp.GetState())
	metadata, metadataErr := b.runtimePeerReplicationMetadata()
	if metadataErr != nil {
		return nil, metadataErr
	}
	annotateRuntimePeerReplication(state, metadata)
	return &pbApic.GetRuntimeStateResponse{State: state}, nil
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

func (b *Backend) GetTasks(ctx context.Context, in *pbApic.GetTasksRequest) (*pbApic.GetTasksResponse, error) {
	manager, err := b.taskManager()
	if err != nil {
		return nil, err
	}
	maxResults := int(in.GetMaxResults())
	if maxResults <= 0 {
		maxResults = 200
	}
	if maxResults > 1000 {
		maxResults = 1000
	}
	records, truncated, err := manager.List(tasks.ListOptions{
		Stream:      in.GetStream(),
		SubjectType: in.GetSubjectType(),
		SubjectID:   in.GetSubjectId(),
		Status:      tasks.Status(in.GetStatus()),
		MaxResults:  maxResults,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tasks: %w", err)
	}
	resp := &pbApic.GetTasksResponse{Truncated: truncated}
	for _, record := range records {
		resp.Tasks = append(resp.Tasks, taskRecordToProto(record))
	}
	return resp, nil
}

func (b *Backend) GetTask(ctx context.Context, in *pbApic.GetTaskRequest) (*pbApic.GetTaskResponse, error) {
	manager, err := b.taskManager()
	if err != nil {
		return nil, err
	}
	record, err := manager.Get(in.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task '%s': %w", in.GetId(), err)
	}
	resp := &pbApic.GetTaskResponse{Task: taskRecordToProto(record)}
	if in.GetIncludeEvents() {
		events, err := manager.Events(record.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve task events for '%s': %w", record.ID, err)
		}
		for _, event := range events {
			resp.Events = append(resp.Events, taskEventToProto(event))
		}
	}
	return resp, nil
}

func (b *Backend) WatchTask(in *pbApic.WatchTaskRequest, stream pbApic.ProtosClientApi_WatchTaskServer) error {
	manager, err := b.taskManager()
	if err != nil {
		return err
	}
	id := strings.TrimSpace(in.GetId())
	if id == "" {
		return grpcstatus.Error(codes.InvalidArgument, "task id is empty")
	}
	updates, cancel, err := manager.Subscribe(id)
	if err != nil {
		return err
	}
	defer cancel()

	record, err := manager.Get(id)
	if err != nil {
		return grpcstatus.Errorf(codes.NotFound, "failed to retrieve task %q: %v", id, err)
	}
	if in.GetIncludeSnapshot() {
		resp := &pbApic.WatchTaskResponse{Task: taskRecordToProto(record)}
		if in.GetIncludeEvents() {
			events, err := manager.Events(record.ID)
			if err != nil {
				return fmt.Errorf("failed to retrieve task events for '%s': %w", record.ID, err)
			}
			for _, event := range events {
				resp.Events = append(resp.Events, taskEventToProto(event))
			}
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}

	var lastSequence uint64
	if latest, found := manager.LatestProgress(id); found {
		lastSequence = latest.Sequence
		if err := stream.Send(&pbApic.WatchTaskResponse{
			Sequence: latest.Sequence,
			Update:   taskProgressUpdateToProto(latest),
		}); err != nil {
			return err
		}
	}

	var ticks <-chan time.Time
	var ticker *time.Ticker
	if intervalMillis := in.GetHeartbeatIntervalMs(); intervalMillis > 0 {
		ticker = time.NewTicker(time.Duration(intervalMillis) * time.Millisecond)
		defer ticker.Stop()
		ticks = ticker.C
	}

	ctx := stream.Context()
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Sequence <= lastSequence {
				continue
			}
			lastSequence = update.Sequence
			if err := stream.Send(&pbApic.WatchTaskResponse{
				Sequence: update.Sequence,
				Update:   taskProgressUpdateToProto(update),
			}); err != nil {
				return err
			}
		case <-ticks:
			if err := stream.Send(&pbApic.WatchTaskResponse{Heartbeat: true}); err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
}

func (b *Backend) taskManager() (*tasks.Manager, error) {
	if b.protosClient == nil || b.protosClient.TaskManager == nil {
		return nil, fmt.Errorf("task manager is not configured")
	}
	return b.protosClient.TaskManager, nil
}

func taskRecordToProto(record tasks.Record) *pbApic.Task {
	return &pbApic.Task{
		Id:           record.ID,
		Stream:       record.Stream,
		SubjectType:  record.SubjectType,
		SubjectId:    record.SubjectID,
		Status:       string(record.Status),
		Title:        record.Title,
		Message:      record.Message,
		Progress:     int32(record.Progress),
		PayloadJson:  rawJSONText(record.Payload),
		ResultJson:   rawJSONText(record.Result),
		ErrorMessage: record.ErrorMessage,
		Attempts:     int32(record.Attempts),
		MaxAttempts:  int32(record.MaxAttempts),
		CreatedAt:    formatTaskTime(record.CreatedAt),
		UpdatedAt:    formatTaskTime(record.UpdatedAt),
		StartedAt:    formatTaskTime(record.StartedAt),
		FinishedAt:   formatTaskTime(record.FinishedAt),
	}
}

func taskEventToProto(event tasks.Event) *pbApic.TaskEvent {
	return &pbApic.TaskEvent{
		Id:          event.ID,
		TaskId:      event.TaskID,
		Status:      string(event.Status),
		Message:     event.Message,
		Progress:    int32(event.Progress),
		DetailsJson: rawJSONText(event.Details),
		CreatedAt:   formatTaskTime(event.CreatedAt),
	}
}

func taskProgressUpdateToProto(update tasks.ProgressUpdate) *pbApic.TaskProgressUpdate {
	return &pbApic.TaskProgressUpdate{
		TaskId:      update.TaskID,
		Status:      string(update.Status),
		Message:     update.Message,
		Progress:    int32(update.Progress),
		DetailsJson: rawJSONText(update.Details),
		CreatedAt:   formatTaskTime(update.CreatedAt),
		Durable:     update.Durable,
	}
}

func rawJSONText(raw []byte) string {
	if len(raw) == 0 {
		return "{}"
	}
	return string(raw)
}

func formatTaskTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
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
	if err := b.protosClient.DB.CatchUpCheckpoint(ctx, "apic get runtime state"); err != nil {
		return nil, err
	}
	status, ok := b.protosClient.DB.SwarmionStatus()
	if !ok {
		return nil, fmt.Errorf("swarmion status is not available")
	}
	out := &pbApic.RuntimeState{
		PeerId:                       status.PeerID,
		ManifestDigest:               status.ManifestDigest,
		CheckpointRootHash:           status.CheckpointRootHash.String(),
		TentativeRootHash:            status.TentativeRootHash.String(),
		ProtocolCheckpointRootHash:   status.RuntimeCheckpointDesiredRootHash.String(),
		DurableMainRootHash:          status.DurableMainRootHash.String(),
		StateProviders:               append([]string(nil), status.StateProviders...),
		ConnectedPeers:               append([]string(nil), status.ConnectedPeers...),
		RuntimeRefreshPending:        status.RuntimeRefreshPending,
		RuntimeRefreshLastError:      status.RuntimeRefreshLastError,
		RuntimeCheckpointPending:     status.RuntimeCheckpointMaterializePending,
		RuntimeCheckpointLastError:   status.RuntimeCheckpointMaterializeLastError,
		RuntimeMaterializationPolicy: status.RuntimeCheckpointMaterializationPolicy.String(),
		ProtocolCheckpointDigest:     formatRuntimeDigest(status.ProtocolCheckpointDigest),
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
			PeerId:         peerStatus.PeerID,
			Connected:      peerStatus.Connected,
			Dialable:       peerStatus.Dialable,
			StateProvider:  peerStatus.StateProvider,
			Compatible:     peerStatus.Compatible,
			Incompatible:   peerStatus.Incompatible,
			Ignored:        peerStatus.Ignored,
			RelayOnly:      peerStatus.RelayOnly,
			Addresses:      append([]string(nil), peerStatus.Addresses...),
			LastDialErrors: cloneStringMap(peerStatus.LastDialErrors),
			Reason:         peerStatus.Reason,
		})
	}
	peerIDs, err := db.GetActiveRuntimePeerIDs(b.protosClient.DB)
	if err != nil {
		return nil, err
	}
	metadata, err := b.runtimePeerReplicationMetadata()
	if err != nil {
		return nil, err
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
	filterRuntimePeerSurface(out, peerIDs)
	addKnownRuntimePeerStatuses(out, peerIDs)
	annotateRuntimePeerReplication(out, metadata)
	if trace, ok := b.protosClient.DB.SwarmionContentSyncTrace(); ok {
		out.ContentSyncTrace = append([]string(nil), trace...)
	}
	return out, nil
}

func filterRuntimePeerSurface(out *pbApic.RuntimeState, peerIDs map[string]struct{}) {
	if out == nil {
		return
	}
	allowed := make(map[string]struct{}, len(peerIDs)+1)
	for peerID := range peerIDs {
		peerID = strings.TrimSpace(peerID)
		if peerID != "" {
			allowed[peerID] = struct{}{}
		}
	}
	if localPeerID := strings.TrimSpace(out.GetPeerId()); localPeerID != "" {
		allowed[localPeerID] = struct{}{}
	}
	out.StateProviders = filterStringsBySet(out.GetStateProviders(), allowed)
	out.ConnectedPeers = filterStringsBySet(out.GetConnectedPeers(), allowed)
	providers := make(map[string]struct{}, len(out.GetStateProviders()))
	for _, peerID := range out.GetStateProviders() {
		providers[peerID] = struct{}{}
	}
	connected := make(map[string]struct{}, len(out.GetConnectedPeers())+1)
	for _, peerID := range out.GetConnectedPeers() {
		connected[peerID] = struct{}{}
	}
	if localPeerID := strings.TrimSpace(out.GetPeerId()); localPeerID != "" {
		connected[localPeerID] = struct{}{}
	}
	filteredStatuses := out.GetPeerStatuses()[:0]
	seenStatuses := make(map[string]struct{}, len(out.GetPeerStatuses()))
	for _, peerStatus := range out.GetPeerStatuses() {
		if peerStatus == nil {
			continue
		}
		peerID := strings.TrimSpace(peerStatus.GetPeerId())
		if _, found := allowed[peerID]; !found {
			continue
		}
		if _, found := seenStatuses[peerID]; found {
			continue
		}
		seenStatuses[peerID] = struct{}{}
		if _, found := providers[peerID]; !found {
			peerStatus.StateProvider = false
		}
		_, isConnected := connected[peerID]
		peerStatus.Connected = isConnected
		if !isConnected {
			peerStatus.Dialable = false
		}
		filteredStatuses = append(filteredStatuses, peerStatus)
	}
	out.PeerStatuses = filteredStatuses
	filteredCompatibility := out.GetCompatibility()[:0]
	seenCompatibility := make(map[string]struct{}, len(out.GetCompatibility()))
	for _, item := range out.GetCompatibility() {
		if item == nil {
			continue
		}
		peerID := strings.TrimSpace(item.GetPeerId())
		if _, found := allowed[peerID]; !found {
			continue
		}
		if _, found := seenCompatibility[peerID]; found {
			continue
		}
		seenCompatibility[peerID] = struct{}{}
		filteredCompatibility = append(filteredCompatibility, item)
	}
	out.Compatibility = filteredCompatibility
}

func filterStringsBySet(values []string, allowed map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, found := allowed[value]; !found {
			continue
		}
		if _, found := seen[value]; found {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func runtimeStateFromP2PProto(state *p2pproto.RuntimeState) *pbApic.RuntimeState {
	if state == nil {
		return nil
	}
	out := &pbApic.RuntimeState{
		PeerId:                       state.GetPeerId(),
		ManifestDigest:               state.GetManifestDigest(),
		CheckpointRootHash:           state.GetCheckpointRootHash(),
		TentativeRootHash:            state.GetTentativeRootHash(),
		ProtocolCheckpointRootHash:   state.GetProtocolCheckpointRootHash(),
		DurableMainRootHash:          state.GetDurableMainRootHash(),
		StateProviders:               append([]string(nil), state.GetStateProviders()...),
		ConnectedPeers:               append([]string(nil), state.GetConnectedPeers()...),
		FatalState:                   state.GetFatalState(),
		RuntimeRefreshPending:        state.GetRuntimeRefreshPending(),
		RuntimeRefreshLastError:      state.GetRuntimeRefreshLastError(),
		RuntimeCheckpointPending:     state.GetRuntimeCheckpointPending(),
		RuntimeCheckpointLastError:   state.GetRuntimeCheckpointLastError(),
		RuntimeMaterializationPolicy: state.GetRuntimeMaterializationPolicy(),
		ContentSyncTrace:             append([]string(nil), state.GetContentSyncTrace()...),
		ProtocolCheckpointDigest:     state.GetProtocolCheckpointDigest(),
	}
	for _, peerStatus := range state.GetPeerStatuses() {
		out.PeerStatuses = append(out.PeerStatuses, &pbApic.RuntimePeerStatus{
			PeerId:         peerStatus.GetPeerId(),
			Connected:      peerStatus.GetConnected(),
			Dialable:       peerStatus.GetDialable(),
			StateProvider:  peerStatus.GetStateProvider(),
			Compatible:     peerStatus.GetCompatible(),
			Incompatible:   peerStatus.GetIncompatible(),
			Ignored:        peerStatus.GetIgnored(),
			RelayOnly:      peerStatus.GetRelayOnly(),
			Addresses:      append([]string(nil), peerStatus.GetAddresses()...),
			LastDialErrors: cloneStringMap(peerStatus.GetLastDialErrors()),
			Reason:         peerStatus.GetReason(),
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

func addKnownRuntimePeerStatuses(out *pbApic.RuntimeState, peerIDs map[string]struct{}) {
	if out == nil {
		return
	}
	wanted := make(map[string]struct{}, len(peerIDs)+1)
	for peerID := range peerIDs {
		peerID = strings.TrimSpace(peerID)
		if peerID != "" {
			wanted[peerID] = struct{}{}
		}
	}
	if localPeerID := strings.TrimSpace(out.GetPeerId()); localPeerID != "" {
		wanted[localPeerID] = struct{}{}
	}
	if len(wanted) == 0 {
		return
	}

	existing := make(map[string]struct{}, len(out.GetPeerStatuses()))
	for _, peerStatus := range out.GetPeerStatuses() {
		peerID := strings.TrimSpace(peerStatus.GetPeerId())
		if peerID != "" {
			existing[peerID] = struct{}{}
		}
	}

	missing := make([]string, 0, len(wanted))
	for peerID := range wanted {
		if _, found := existing[peerID]; !found {
			missing = append(missing, peerID)
		}
	}
	sort.Strings(missing)
	for _, peerID := range missing {
		out.PeerStatuses = append(out.PeerStatuses, knownRuntimePeerStatus(peerID, out))
	}
}

type runtimePeerReplicationMetadata struct {
	deviceClass string
	priority    int
}

func (b *Backend) runtimePeerReplicationMetadata() (map[string]runtimePeerReplicationMetadata, error) {
	out := map[string]runtimePeerReplicationMetadata{}
	if b == nil || b.protosClient == nil {
		return out, nil
	}
	if b.protosClient.CloudManager != nil {
		instances, err := b.protosClient.CloudManager.GetInstances(false)
		if err != nil {
			return nil, fmt.Errorf("load runtime instance replication metadata: %w", err)
		}
		for _, instance := range instances {
			if !provisioners.IsActiveInstance(instance) {
				continue
			}
			peerID, err := instance.GetPeerID()
			if err != nil {
				continue
			}
			out[peerID] = runtimePeerReplicationMetadata{
				deviceClass: db.ReplicationDeviceClassForMachine(instance.Kind, instance.KindID),
				priority:    instance.ReplicationPriority,
			}
		}
	}
	if b.protosClient.Manager != nil {
		devices, err := b.protosClient.Manager.GetAllDevices(false)
		if err != nil {
			return nil, fmt.Errorf("load runtime user-device replication metadata: %w", err)
		}
		for _, device := range devices {
			peerID, err := db.PeerIDFromPublicKeyString(device.PublicKey)
			if err != nil {
				continue
			}
			out[peerID] = runtimePeerReplicationMetadata{
				deviceClass: db.ReplicationDeviceClassForUserDeviceName(device.Name),
				priority:    device.ReplicationPriority,
			}
		}
	}
	return out, nil
}

func annotateRuntimePeerReplication(state *pbApic.RuntimeState, metadata map[string]runtimePeerReplicationMetadata) {
	if state == nil || len(metadata) == 0 {
		return
	}
	for _, peerStatus := range state.GetPeerStatuses() {
		if peerStatus == nil {
			continue
		}
		item, found := metadata[peerStatus.GetPeerId()]
		if !found {
			continue
		}
		peerStatus.ReplicationDeviceClass = item.deviceClass
		peerStatus.ReplicationPriority = int32(item.priority)
	}
}

func knownRuntimePeerStatus(peerID string, state *pbApic.RuntimeState) *pbApic.RuntimePeerStatus {
	isSelf := peerID == strings.TrimSpace(state.GetPeerId())
	status := &pbApic.RuntimePeerStatus{
		PeerId:        peerID,
		Connected:     isSelf || stringInList(peerID, state.GetConnectedPeers()),
		Dialable:      isSelf,
		StateProvider: stringInList(peerID, state.GetStateProviders()),
		Compatible:    isSelf,
	}
	if isSelf {
		status.Reason = "self"
	} else {
		status.Reason = "known database peer"
	}
	return status
}

func stringInList(value string, values []string) bool {
	for _, item := range values {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
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

func formatRuntimeDigest(digest [32]byte) string {
	if digest == ([32]byte{}) {
		return ""
	}
	return fmt.Sprintf("%x", digest[:])
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
		resp.CloudImages[id] = cloudSpecificImageProto(image)
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
		resp.Images[id] = cloudSpecificImageProto(image)
	}
	return &resp, nil
}

func cloudSpecificImageProto(image provisioners.ImageInfo) *pbApic.CloudSpecificImage {
	resp := &pbApic.CloudSpecificImage{
		Id:          image.ID,
		Name:        image.Name,
		LogicalName: image.LogicalName,
		DateSuffix:  image.DateSuffix,
		Location:    image.Location,
		Canonical:   image.Canonical,
	}
	if !image.UpdatedAt.IsZero() {
		resp.UpdatedAtUnix = image.UpdatedAt.Unix()
	}
	return resp
}

func (b *Backend) UploadCloudImage(ctx context.Context, in *pbApic.UploadCloudImageRequest) (*pbApic.UploadCloudImageResponse, error) {
	if err := b.requireProvisionerCapability("upload cloud image"); err != nil {
		return nil, err
	}
	log.Debugf("Queueing cloud image upload '%s'(%s) to cloud '%s'", in.ImageName, in.ImagePath, in.CloudName)
	task, err := b.protosClient.CloudManager.QueueUploadLocalImage(in.ImagePath, in.ImageName, in.CloudName, in.CloudLocation, time.Duration(in.Timeout)*time.Minute)
	if err != nil {
		return nil, err
	}
	return &pbApic.UploadCloudImageResponse{TaskId: task.ID}, nil
}

func (b *Backend) UploadProvisionerImage(ctx context.Context, in *pbApic.UploadProvisionerImageRequest) (*pbApic.UploadProvisionerImageResponse, error) {
	if err := b.requireProvisionerCapability("upload provisioner image"); err != nil {
		return nil, err
	}
	log.Debugf("Queueing image upload '%s'(%s) to provisioner '%s'", in.ImageName, in.ImagePath, in.ProvisionerName)
	task, err := b.protosClient.CloudManager.QueueUploadLocalImage(in.ImagePath, in.ImageName, in.ProvisionerName, in.Location, time.Duration(in.Timeout)*time.Minute)
	if err != nil {
		return nil, err
	}
	return &pbApic.UploadProvisionerImageResponse{TaskId: task.ID}, nil
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

func (b *Backend) GetInstanceImage(ctx context.Context, in *pbApic.GetInstanceImageRequest) (*pbApic.GetInstanceImageResponse, error) {
	client, err := b.instanceAdminClient(in.GetInstance(), "get instance image")
	if err != nil {
		return nil, err
	}
	describeResp, err := client.DescribeImage(ctx, &p2pproto.DescribeImageRequest{ImageRef: in.GetImageRef()})
	if err != nil {
		return nil, fmt.Errorf("describe image %q on instance %q: %w", in.GetImageRef(), in.GetInstance(), err)
	}
	resp := &pbApic.GetInstanceImageResponse{
		Found:        describeResp.GetFound(),
		ImageRef:     describeResp.GetImageRef(),
		TargetDigest: describeResp.GetTargetDigest(),
		Platform:     describeResp.GetPlatform(),
		Labels:       cloneStringMap(describeResp.GetLabels()),
	}
	if !describeResp.GetFound() {
		return resp, nil
	}
	contentResp, err := client.GetImageContent(ctx, &p2pproto.GetImageContentRequest{ImageRef: in.GetImageRef()})
	if err != nil {
		return nil, fmt.Errorf("get image content %q on instance %q: %w", in.GetImageRef(), in.GetInstance(), err)
	}
	resp.HasContent = contentResp.GetFound()
	resp.Target = imageContentDescriptorFromP2P(contentResp.GetTarget())
	for _, descriptor := range contentResp.GetDescriptors() {
		resp.Descriptors = append(resp.Descriptors, imageContentDescriptorFromP2P(descriptor))
	}
	if contentResp.GetImageRef() != "" {
		resp.ImageRef = contentResp.GetImageRef()
	}
	if contentResp.GetPlatform() != "" {
		resp.Platform = contentResp.GetPlatform()
	}
	if len(contentResp.GetLabels()) > 0 {
		resp.Labels = cloneStringMap(contentResp.GetLabels())
	}
	return resp, nil
}

func (b *Backend) UploadInstanceImageArchiveChunk(ctx context.Context, in *pbApic.UploadInstanceImageArchiveChunkRequest) (*pbApic.UploadInstanceImageArchiveChunkResponse, error) {
	client, err := b.instanceAdminClient(in.GetInstance(), "upload instance image archive")
	if err != nil {
		return nil, err
	}
	resp, err := client.UploadImageArchiveChunk(ctx, &p2pproto.UploadImageArchiveChunkRequest{
		UploadId: in.GetUploadId(),
		ImageRef: in.GetImageRef(),
		Offset:   in.GetOffset(),
		Data:     append([]byte(nil), in.GetData()...),
		Eof:      in.GetEof(),
	})
	if err != nil {
		return nil, fmt.Errorf("upload image archive chunk to instance %q: %w", in.GetInstance(), err)
	}
	return &pbApic.UploadInstanceImageArchiveChunkResponse{
		ReceivedBytes: resp.GetReceivedBytes(),
		Loaded:        resp.GetLoaded(),
		ImageRef:      resp.GetImageRef(),
		TargetDigest:  resp.GetTargetDigest(),
		Platform:      resp.GetPlatform(),
	}, nil
}

func (b *Backend) instanceAdminClient(instance string, action string) (*p2p.Client, error) {
	instance = strings.TrimSpace(instance)
	if instance == "" || instance == "local" {
		return nil, fmt.Errorf("cannot %s: instance is required", action)
	}
	if b.protosClient == nil || b.protosClient.P2PManager == nil {
		return nil, fmt.Errorf("cannot %s: p2p manager is not configured", action)
	}
	client, err := b.protosClient.P2PManager.GetClient(instance)
	if err != nil {
		return nil, fmt.Errorf("cannot %s for instance %q: %w", action, instance, err)
	}
	return client, nil
}

func imageContentDescriptorFromP2P(desc *p2pproto.ImageContentDescriptor) *pbApic.ImageContentDescriptor {
	if desc == nil {
		return nil
	}
	return &pbApic.ImageContentDescriptor{
		MediaType:   desc.GetMediaType(),
		Digest:      desc.GetDigest(),
		SizeBytes:   desc.GetSizeBytes(),
		Platform:    desc.GetPlatform(),
		Annotations: cloneStringMap(desc.GetAnnotations()),
	}
}

//
// DVC methods
//

func (b *Backend) GetLocalCommits(ctx context.Context, in *pbApic.GetLocalCommitsRequest) (*pbApic.GetLocalCommitsResponse, error) {
	log.Debug("Retrieving local commits")
	commits, err := b.protosClient.DB.GetCombinedCommits("main", "tentative")
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve local commits: %w", err)
	}

	resp := pbApic.GetLocalCommitsResponse{}
	for _, commit := range commits {
		resp.Commits = append(resp.Commits, b.commitViewToProto(commit))
	}
	graph := db.BuildCommitGraph(commits)
	resp.Graph = b.commitGraphToProto(graph)

	return &resp, nil
}

func (b *Backend) commitViewToProto(commit db.CommitView) *pbApic.Commit {
	committer := commit.Committer
	if b != nil && b.commitIdentities != nil {
		committer = b.commitIdentities.displayCommitter(commit.Commit)
	}
	resp := &pbApic.Commit{
		Hash:         commit.Hash,
		Committer:    committer,
		Message:      commit.Message,
		States:       append([]string(nil), commit.States...),
		ParentHashes: append([]string(nil), commit.ParentHashes...),
		Refs:         append([]string(nil), commit.Refs...),
	}
	if !commit.Date.IsZero() {
		resp.DateUnix = commit.Date.Unix()
	}
	return resp
}

func (b *Backend) commitGraphToProto(graph db.CommitGraph) *pbApic.CommitGraph {
	resp := &pbApic.CommitGraph{
		LaneCount: int32(graph.LaneCount),
	}
	for _, item := range graph.Items {
		respItem := &pbApic.CommitGraphItem{
			Commit: b.commitViewToProto(item.Commit),
			Row:    int32(item.Row),
			Lane:   int32(item.Lane),
		}
		for _, lane := range item.ActiveLanes {
			respItem.ActiveLanes = append(respItem.ActiveLanes, int32(lane))
		}
		for _, relation := range item.Relations {
			respItem.Relations = append(respItem.Relations, &pbApic.CommitGraphRelation{
				ParentHash: relation.ParentHash,
				ParentRow:  int32(relation.ParentRow),
				FromLane:   int32(relation.FromLane),
				ToLane:     int32(relation.ToLane),
				Visible:    relation.Visible,
			})
		}
		resp.Items = append(resp.Items, respItem)
	}
	return resp
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
	commitViews := make([]db.CommitView, 0, len(respRemote.Commits))
	for _, commit := range respRemote.Commits {
		commitView := db.CommitView{
			Commit: db.Commit{
				Hash:            commit.Hash,
				Committer:       commit.Committer,
				Message:         commit.Message,
				SignerPublicKey: db.ExtractCommitSignerPublicKey(commit.Message),
				ParentHashes:    append([]string(nil), commit.ParentHashes...),
				Refs:            append([]string(nil), commit.Refs...),
			},
			States: []string{db.CommitStateFinalized},
		}
		if commit.DateUnix > 0 {
			commitView.Date = time.Unix(commit.DateUnix, 0)
		}
		commitViews = append(commitViews, commitView)
		resp.Commits = append(resp.Commits, b.commitViewToProto(commitView))
	}
	graph := db.BuildCommitGraph(commitViews)
	resp.Graph = b.commitGraphToProto(graph)

	return &resp, nil
}

func (b *Backend) ExecuteSql(ctx context.Context, in *pbApic.ExecuteSqlRequest) (*pbApic.ExecuteSqlResponse, error) {
	// This endpoint is only for the user-facing SQL console. Do not build
	// internal product features on SQL as an ad hoc API surface.
	log.Debug("Executing SQL from client API")
	if b.protosClient == nil || b.protosClient.DB == nil {
		return nil, fmt.Errorf("database is not configured")
	}
	result, err := b.protosClient.DB.ExecuteSQL(ctx, in.GetSql(), int(in.GetMaxRows()))
	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %w", err)
	}
	resp := &pbApic.ExecuteSqlResponse{
		Columns:      append([]string(nil), result.Columns...),
		RowsAffected: result.RowsAffected,
		Truncated:    result.Truncated,
		Message:      result.Message,
	}
	for _, row := range result.Rows {
		respRow := &pbApic.SqlRow{Cells: make([]*pbApic.SqlCell, 0, len(row.Cells))}
		for _, cell := range row.Cells {
			respRow.Cells = append(respRow.Cells, &pbApic.SqlCell{
				Value:  cell.Value,
				IsNull: cell.Null,
			})
		}
		resp.Rows = append(resp.Rows, respRow)
	}
	return resp, nil
}
