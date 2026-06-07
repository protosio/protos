package apic

import (
	"context"
	"testing"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/invitations"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestBaseInstanceDeployFieldsFollowDeployRequestDescriptor(t *testing.T) {
	t.Parallel()

	fields := baseInstanceDeployFields()
	descriptor := (&pbApic.DeployInstanceRequest{}).ProtoReflect().Descriptor()
	if len(fields) != descriptor.Fields().Len() {
		t.Fatalf("field count = %d, want %d", len(fields), descriptor.Fields().Len())
	}
	for i, field := range fields {
		want := string(descriptor.Fields().Get(i).Name())
		if field.GetName() != want {
			t.Fatalf("field[%d] = %q, want %q", i, field.GetName(), want)
		}
	}
}

func TestInstanceDeployImageLessPrefersNewestUpdatedAt(t *testing.T) {
	t.Parallel()

	oldImage := provisioners.ImageInfo{Name: "z-old", UpdatedAt: time.Unix(100, 0)}
	newImage := provisioners.ImageInfo{Name: "a-new", UpdatedAt: time.Unix(200, 0)}
	if !instanceDeployImageLess(newImage, oldImage) {
		t.Fatal("newer image should sort before older image")
	}
	if instanceDeployImageLess(oldImage, newImage) {
		t.Fatal("older image should not sort before newer image")
	}
	if !instanceDeployImageLess(provisioners.ImageInfo{Name: "a"}, provisioners.ImageInfo{Name: "b"}) {
		t.Fatal("images without update times should sort by name")
	}
}

func TestPreferredInstanceDeployMachineUsesLocalMacOS2GBDefault(t *testing.T) {
	t.Parallel()

	options := []*pbApic.InstanceDeployFieldOption{
		{Value: "vz-1c-1g"},
		{Value: "vz-2c-2g"},
	}
	if got := preferredInstanceDeployMachine("local_macos", options); got != "vz-2c-2g" {
		t.Fatalf("preferred local macOS machine = %q, want vz-2c-2g", got)
	}
	if got := preferredInstanceDeployMachine("hetzner", options); got != "vz-1c-1g" {
		t.Fatalf("preferred non-local machine = %q, want first option", got)
	}
}

func TestIsPublicExitIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{name: "public IPv4", ip: "5.161.52.86", want: true},
		{name: "public IPv6", ip: "2606:4700:4700::1111", want: true},
		{name: "private IPv4", ip: "10.0.0.1", want: false},
		{name: "carrier NAT IPv4", ip: "100.64.0.1", want: false},
		{name: "documentation IPv4", ip: "203.0.113.10", want: false},
		{name: "documentation IPv6", ip: "2001:db8::1", want: false},
		{name: "loopback", ip: "127.0.0.1", want: false},
		{name: "invalid", ip: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPublicExitIP(tt.ip); got != tt.want {
				t.Fatalf("isPublicExitIP(%q) = %t, want %t", tt.ip, got, tt.want)
			}
		})
	}
}

func TestAddKnownRuntimePeerStatusesAddsDbPeersAndSelf(t *testing.T) {
	t.Parallel()

	state := &pbApic.RuntimeState{
		PeerId:             "local-peer",
		ConnectedPeers:     []string{"connected-peer"},
		ActiveWitnessIds:   []string{"witness-peer"},
		EligibleWitnessIds: []string{"eligible-peer"},
		StateProviders:     []string{"provider-peer"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"connected-peer": {},
		"witness-peer":   {},
		"eligible-peer":  {},
		"provider-peer":  {},
	})

	statuses := map[string]*pbApic.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}

	if len(statuses) != 5 {
		t.Fatalf("peer statuses count = %d, want 5: %#v", len(statuses), statuses)
	}
	if self := statuses["local-peer"]; self == nil || !self.GetConnected() || !self.GetDialable() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want connected dialable compatible self row", self)
	}
	if witness := statuses["witness-peer"]; witness == nil || !witness.GetWitness() {
		t.Fatalf("witness status = %#v, want witness=true", witness)
	}
	if eligible := statuses["eligible-peer"]; eligible == nil || !eligible.GetEligibleWitness() {
		t.Fatalf("eligible status = %#v, want eligible_witness=true", eligible)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	}
	if connected := statuses["connected-peer"]; connected == nil || !connected.GetConnected() {
		t.Fatalf("connected status = %#v, want existing connected status preserved", connected)
	}
}

func TestFilterRuntimePeerSurfaceRemovesUnknownCachedPeers(t *testing.T) {
	t.Parallel()

	state := &pbApic.RuntimeState{
		PeerId:         "local-peer",
		StateProviders: []string{"provider-peer", "deleted-peer", "provider-peer"},
		ConnectedPeers: []string{"provider-peer", "deleted-peer", "provider-peer"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true},
			{PeerId: "deleted-peer", Connected: true, Dialable: true, StateProvider: true},
		},
	}

	filterRuntimePeerSurface(state, map[string]struct{}{
		"provider-peer": {},
	})

	if got, want := state.GetStateProviders(), []string{"provider-peer"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("state providers = %#v, want %#v", got, want)
	}
	if got, want := state.GetConnectedPeers(), []string{"provider-peer"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("connected peers = %#v, want %#v", got, want)
	}
	if !state.GetPeerStatuses()[0].GetStateProvider() {
		t.Fatal("known provider lost provider flag")
	}
	if len(state.GetPeerStatuses()) != 1 {
		t.Fatalf("peer statuses count = %d, want 1", len(state.GetPeerStatuses()))
	}
}

func TestEffectiveJoinModeRejectsModeMismatch(t *testing.T) {
	got, err := effectiveJoinMode(invitations.InviteJoinModeNewDevice, invitations.InviteJoinModeNewUser)

	if grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %s, want %s: %v", grpcstatus.Code(err), codes.InvalidArgument, err)
	}
	if got != "" {
		t.Fatalf("effective join mode = %q, want empty on error", got)
	}
}

func TestEffectiveJoinModeUsesAdvertisedModeWhenRequestOmitsMode(t *testing.T) {
	got, err := effectiveJoinMode(invitations.InviteJoinModeNewUser, "")
	if err != nil {
		t.Fatalf("effectiveJoinMode returned error: %v", err)
	}
	if got != invitations.InviteJoinModeNewUser {
		t.Fatalf("effective join mode = %q, want %q", got, invitations.InviteJoinModeNewUser)
	}
}

func TestStartInviteJoinModeDefaultsToNewDevice(t *testing.T) {
	got, err := startInviteJoinMode("")
	if err != nil {
		t.Fatalf("startInviteJoinMode returned error: %v", err)
	}
	if got != invitations.InviteJoinModeNewDevice {
		t.Fatalf("start invite join mode = %q, want %q", got, invitations.InviteJoinModeNewDevice)
	}
}

func TestStartInviteJoinModeRejectsAny(t *testing.T) {
	got, err := startInviteJoinMode(invitations.InviteJoinModeAny)

	if grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %s, want %s: %v", grpcstatus.Code(err), codes.InvalidArgument, err)
	}
	if got != "" {
		t.Fatalf("start invite join mode = %q, want empty on error", got)
	}
}

func TestEnsureJoinedUserDeviceAddsDeviceForExistingUser(t *testing.T) {
	backend, store, manager := newUserDeviceTestBackend(t)
	existingUser, err := manager.CreateUser("alex", "Alex", false)
	if err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	if err := backend.ensureJoinedUserDevice("alex", "", invitations.InviteJoinModeNewDevice, ""); err != nil {
		t.Fatalf("ensureJoinedUserDevice: %v", err)
	}

	if count := countUsersByUsername(t, store, "alex"); count != 1 {
		t.Fatalf("users named alex = %d, want 1", count)
	}
	devices, err := manager.GetAllDevices(false)
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1: %#v", len(devices), devices)
	}
	if devices[0].UserID != existingUser.ID {
		t.Fatalf("device UserID = %q, want existing user %q", devices[0].UserID, existingUser.ID)
	}
}

func TestEnsureJoinedUserDeviceAddsDeviceForInviteTargetUser(t *testing.T) {
	backend, store, manager := newUserDeviceTestBackend(t)
	existingUser, err := manager.CreateUser("alex", "Alex", false)
	if err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	if err := backend.ensureJoinedUserDevice("", "", invitations.InviteJoinModeNewDevice, existingUser.ID); err != nil {
		t.Fatalf("ensureJoinedUserDevice: %v", err)
	}

	if count := countUsersByUsername(t, store, "alex"); count != 1 {
		t.Fatalf("users named alex = %d, want 1", count)
	}
	devices, err := manager.GetAllDevices(false)
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1: %#v", len(devices), devices)
	}
	if devices[0].UserID != existingUser.ID {
		t.Fatalf("device UserID = %q, want existing user %q", devices[0].UserID, existingUser.ID)
	}
}

func TestEnsureJoinedUserDeviceRejectsMismatchedInviteTargetUser(t *testing.T) {
	backend, _, manager := newUserDeviceTestBackend(t)
	existingUser, err := manager.CreateUser("alex", "Alex", false)
	if err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	err = backend.ensureJoinedUserDevice("bob", "", invitations.InviteJoinModeNewDevice, existingUser.ID)

	if grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %s, want %s: %v", grpcstatus.Code(err), codes.InvalidArgument, err)
	}
}

func TestEnsureJoinedUserDeviceCreatesMissingUser(t *testing.T) {
	backend, store, manager := newUserDeviceTestBackend(t)

	if err := backend.ensureJoinedUserDevice("sam", "Sam", invitations.InviteJoinModeNewUser, ""); err != nil {
		t.Fatalf("ensureJoinedUserDevice: %v", err)
	}

	if count := countUsersByUsername(t, store, "sam"); count != 1 {
		t.Fatalf("users named sam = %d, want 1", count)
	}
	createdUser, err := manager.GetUser("sam")
	if err != nil {
		t.Fatalf("get created user: %v", err)
	}
	devices, err := manager.GetAllDevices(false)
	if err != nil {
		t.Fatalf("get devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %d, want 1: %#v", len(devices), devices)
	}
	if devices[0].UserID != createdUser.ID {
		t.Fatalf("device UserID = %q, want created user %q", devices[0].UserID, createdUser.ID)
	}
}

func TestEnsureJoinedUserDeviceRejectsExistingUserForNewUserInvite(t *testing.T) {
	backend, _, manager := newUserDeviceTestBackend(t)
	if _, err := manager.CreateUser("alex", "Alex", false); err != nil {
		t.Fatalf("create existing user: %v", err)
	}

	err := backend.ensureJoinedUserDevice("alex", "Alex G", invitations.InviteJoinModeNewUser, "")

	if grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %s, want %s: %v", grpcstatus.Code(err), codes.InvalidArgument, err)
	}
}

func TestEnsureJoinedUserDeviceRejectsMissingUserForNewDeviceInvite(t *testing.T) {
	backend, _, _ := newUserDeviceTestBackend(t)

	err := backend.ensureJoinedUserDevice("sam", "", invitations.InviteJoinModeNewDevice, "")

	if grpcstatus.Code(err) != codes.InvalidArgument {
		t.Fatalf("error code = %s, want %s: %v", grpcstatus.Code(err), codes.InvalidArgument, err)
	}
}

func newUserDeviceTestBackend(t *testing.T) (*Backend, *db.DB, *user.Manager) {
	t.Helper()

	cfg := config.Get()
	previousWorkDir := cfg.WorkDir
	previousP2PPort := cfg.P2PPort
	workDir := t.TempDir()
	cfg.WorkDir = workDir
	cfg.P2PPort = 0
	t.Cleanup(func() {
		cfg.WorkDir = previousWorkDir
		cfg.P2PPort = previousP2PPort
	})

	localKey, err := pcrypto.GetLocalKey(workDir)
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}
	store, err := db.Open(workDir, "protos_test", localKey)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	if err := store.Init(); err != nil {
		t.Fatalf("init db: %v", err)
	}

	keyManager := pcrypto.CreateManager(store)
	manager := user.CreateManager(store, keyManager)
	backend := &Backend{
		protosClient: &Services{
			DB:         store,
			Manager:    manager,
			KeyManager: keyManager,
		},
	}
	return backend, store, manager
}

func countUsersByUsername(t *testing.T, store *db.DB, username string) int {
	t.Helper()

	rows, err := store.QueryContext(context.Background(), "SELECT COUNT(*) FROM users WHERE username = ?", username)
	if err != nil {
		t.Fatalf("query users count: %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatal("query users count returned no rows")
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		t.Fatalf("scan users count: %v", err)
	}
	return count
}
