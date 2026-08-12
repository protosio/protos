package apic

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	swarmion "github.com/nustiueudinastea/swarmion/runtime"
	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/invitations"
	p2pproto "github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/testswarmion"
	"github.com/protosio/protos/internal/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

func TestErrorLoggingUnaryInterceptorPreservesUnresolvedWriteReceipt(t *testing.T) {
	t.Parallel()

	cause := errors.New("publisher acknowledgement was interrupted")
	unresolved := &db.PublishedWriteConfirmationUnresolvedError{
		Confirmation: db.PublishedWriteConfirmation{
			Receipt: db.PublishedWriteReceipt{
				EventID:           "event-unresolved",
				PublishedRootHash: "root-unresolved",
			},
		},
		Cause: cause,
	}
	_, err := errorLoggingUnaryInterceptor(
		context.Background(),
		&pbApic.GetTasksRequest{},
		&grpc.UnaryServerInfo{FullMethod: "/apic.ProtosClientApi/TestWrite"},
		func(context.Context, interface{}) (interface{}, error) {
			return nil, fmt.Errorf("save desired state: %w", unresolved)
		},
	)
	if grpcstatus.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %s, want %s: %v", grpcstatus.Code(err), codes.FailedPrecondition, err)
	}
	statusValue, ok := grpcstatus.FromError(err)
	if !ok {
		t.Fatalf("error has no gRPC status: %v", err)
	}
	details := statusValue.Details()
	if len(details) != 1 {
		t.Fatalf("details = %#v, want one exact receipt", details)
	}
	confirmation, ok := details[0].(*pbApic.WriteConfirmation)
	if !ok {
		t.Fatalf("detail type = %T, want *WriteConfirmation", details[0])
	}
	if confirmation.GetStage() != "" || confirmation.GetEventId() != "event-unresolved" || confirmation.GetPublishedRootHash() != "root-unresolved" {
		t.Fatalf("unexpected unresolved receipt detail: %+v", confirmation)
	}
	if confirmation.GetAvailabilityPending() {
		t.Fatalf("unresolved local acceptance must not be reported as accepted availability pending: %+v", confirmation)
	}
}

func TestPublishedWriteConfirmationToProtoPreservesMachineReadableBoundary(t *testing.T) {
	t.Parallel()

	eligiblePeerIDs := []string{"peer-b"}
	got := publishedWriteConfirmationToProto(db.PublishedWriteConfirmation{
		Receipt: db.PublishedWriteReceipt{
			EventID:           "event-1",
			PublishedRootHash: "root-1",
		},
		Stage: db.PublishedWriteConfirmationOtherPeerAvailable,
		Availability: swarmion.ReceiptAvailabilityStatus{
			RequiredOtherPeers:  1,
			ConfirmedOtherPeers: 1,
			CandidateScope:      swarmion.ReceiptAvailabilityCandidateScopeCurrentLogicalPeers,
			EligiblePeerIDs:     eligiblePeerIDs,
		},
		AvailabilityPending: true,
		AvailabilityError:   "internal prose must not cross APIC",
	})
	if got == nil {
		t.Fatal("confirmation is nil")
	}
	if got.GetStage() != "other_peer_available" || got.GetEventId() != "event-1" || got.GetPublishedRootHash() != "root-1" {
		t.Fatalf("unexpected receipt boundary: %+v", got)
	}
	if got.GetRequiredOtherPeers() != 1 || got.GetConfirmedOtherPeers() != 1 || !got.GetAvailabilityPending() {
		t.Fatalf("unexpected availability counters: %+v", got)
	}
	if got.GetCandidateScope() != "current_logical_peers" || !slices.Equal(got.GetEligiblePeerIds(), []string{"peer-b"}) ||
		got.GetNoCurrentEligiblePeers() || got.GetReasonCode() != "" {
		t.Fatalf("unexpected topology summary: %+v", got)
	}
	eligiblePeerIDs[0] = "mutated-after-mapping"
	if !slices.Equal(got.GetEligiblePeerIds(), []string{"peer-b"}) {
		t.Fatalf("eligible peers were aliased across APIC mapping: %+v", got)
	}
	if publishedWriteConfirmationToProto(db.PublishedWriteConfirmation{}) != nil {
		t.Fatal("empty confirmation should remain absent")
	}
}

func TestTaskWriteConfirmationToProtoPreservesMachineReadableBoundary(t *testing.T) {
	t.Parallel()

	got := taskWriteConfirmationToProto(tasks.WriteConfirmation{
		Stage:                  db.PublishedWriteConfirmationLocalAccepted,
		EventID:                "event-2",
		PublishedRootHash:      "root-2",
		RequiredOtherPeers:     1,
		ConfirmedOtherPeers:    0,
		AvailabilityPending:    true,
		CandidateScope:         "current_logical_peers",
		NoCurrentEligiblePeers: true,
		ReasonCode:             "no_current_eligible_peers",
		AvailabilityError:      "internal prose must not cross APIC",
	})
	if got == nil || got.GetStage() != "local_accepted" || got.GetEventId() != "event-2" || !got.GetAvailabilityPending() {
		t.Fatalf("unexpected task confirmation: %+v", got)
	}
	if got.GetCandidateScope() != "current_logical_peers" || !got.GetNoCurrentEligiblePeers() || got.GetReasonCode() != "no_current_eligible_peers" {
		t.Fatalf("unexpected task topology summary: %+v", got)
	}

	forwardedSource := &p2pproto.WriteConfirmation{
		Stage:               "other_peer_available",
		EventId:             "event-3",
		PublishedRootHash:   "root-3",
		RequiredOtherPeers:  1,
		ConfirmedOtherPeers: 1,
		CandidateScope:      "explicit_peer_ids",
		EligiblePeerIds:     []string{"peer-c"},
		ReasonCode:          "insufficient_other_peer_receipts",
	}
	forwarded := taskWriteConfirmationFromP2PProto(forwardedSource)
	if forwarded == nil || forwarded.GetStage() != "other_peer_available" || forwarded.GetEventId() != "event-3" || forwarded.GetConfirmedOtherPeers() != 1 {
		t.Fatalf("unexpected forwarded task confirmation: %+v", forwarded)
	}
	if forwarded.GetCandidateScope() != "explicit_peer_ids" || !slices.Equal(forwarded.GetEligiblePeerIds(), []string{"peer-c"}) ||
		forwarded.GetReasonCode() != "insufficient_other_peer_receipts" {
		t.Fatalf("unexpected forwarded topology summary: %+v", forwarded)
	}
	forwardedSource.EligiblePeerIds[0] = "mutated-after-forwarding"
	if !slices.Equal(forwarded.GetEligiblePeerIds(), []string{"peer-c"}) {
		t.Fatalf("eligible peers were aliased across P2P-to-APIC forwarding: %+v", forwarded)
	}
}

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
		PeerId:                 "local-peer",
		ConnectedPeers:         []string{"connected-peer"},
		RoutedPeers:            []string{"connected-peer"},
		ParticipatingPeers:     []string{"provider-peer"},
		LogicalPeers:           []string{"database-peer"},
		PhysicalConnectedPeers: []string{"provider-peer"},
		StateProviders:         []string{"provider-peer"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{
			{PeerId: "connected-peer", Connected: true, Dialable: true, Routed: true},
		},
	}

	addKnownRuntimePeerStatuses(state, map[string]struct{}{
		"connected-peer": {},
		"database-peer":  {},
		"provider-peer":  {},
	})

	statuses := map[string]*pbApic.RuntimePeerStatus{}
	for _, status := range state.GetPeerStatuses() {
		statuses[status.GetPeerId()] = status
	}

	if len(statuses) != 4 {
		t.Fatalf("peer statuses count = %d, want 4: %#v", len(statuses), statuses)
	}
	if self := statuses["local-peer"]; self == nil || self.GetConnected() || self.GetDialable() || self.GetRouted() || self.GetPhysicalConnected() || !self.GetCompatible() || self.GetReason() != "self" {
		t.Fatalf("self status = %#v, want compatible self row without inferred reachability", self)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || databasePeer.GetConnected() || databasePeer.GetDialable() || databasePeer.GetStateProvider() || databasePeer.GetReason() != "known database peer" {
		t.Fatalf("database peer status = %#v, want inert known database row", databasePeer)
	}
	if provider := statuses["provider-peer"]; provider == nil || !provider.GetStateProvider() {
		t.Fatalf("provider status = %#v, want state_provider=true", provider)
	} else if !provider.GetParticipating() || !provider.GetPhysicalConnected() {
		t.Fatalf("provider transport planes = %#v, want participating and physical", provider)
	}
	if connected := statuses["connected-peer"]; connected == nil || !connected.GetRouted() || !connected.GetConnected() || !connected.GetDialable() {
		t.Fatalf("connected status = %#v, want routed and legacy aliases", connected)
	}
	if databasePeer := statuses["database-peer"]; databasePeer == nil || !databasePeer.GetLogical() {
		t.Fatalf("database peer status = %#v, want logical plane", databasePeer)
	}
}

func TestFilterRuntimePeerSurfaceRemovesUnknownCachedPeers(t *testing.T) {
	t.Parallel()

	state := &pbApic.RuntimeState{
		PeerId:                 "local-peer",
		StateProviders:         []string{"provider-peer", "deleted-peer", "provider-peer"},
		ConnectedPeers:         []string{"provider-peer", "deleted-peer", "provider-peer"},
		RoutedPeers:            []string{"provider-peer", "deleted-peer", "provider-peer"},
		ParticipatingPeers:     []string{"provider-peer", "deleted-peer", "provider-peer"},
		LogicalPeers:           []string{"provider-peer", "deleted-peer", "provider-peer"},
		PhysicalConnectedPeers: []string{"provider-peer", "deleted-peer", "provider-peer"},
		PeerStatuses: []*pbApic.RuntimePeerStatus{
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true, LastRoutedAtUnixNano: 1234},
			{PeerId: "provider-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
			{PeerId: "deleted-peer", Connected: true, Dialable: true, StateProvider: true, Routed: true, Participating: true, Logical: true, PhysicalConnected: true},
		},
		Compatibility: []*pbApic.RuntimeCompatibility{
			{PeerId: "provider-peer", Compatible: true},
			{PeerId: "provider-peer", Compatible: true},
			{PeerId: "deleted-peer", Blocking: true},
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
	for name, got := range map[string][]string{
		"routed":        state.GetRoutedPeers(),
		"participating": state.GetParticipatingPeers(),
		"logical":       state.GetLogicalPeers(),
		"physical":      state.GetPhysicalConnectedPeers(),
	} {
		if len(got) != 1 || got[0] != "provider-peer" {
			t.Fatalf("%s peers = %#v, want provider-peer", name, got)
		}
	}
	if !state.GetPeerStatuses()[0].GetStateProvider() {
		t.Fatal("known provider lost provider flag")
	}
	if len(state.GetPeerStatuses()) != 1 {
		t.Fatalf("peer statuses count = %d, want 1", len(state.GetPeerStatuses()))
	}
	status := state.GetPeerStatuses()[0]
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() || !status.GetPhysicalConnected() || !status.GetConnected() || !status.GetDialable() || status.GetLastRoutedAtUnixNano() != 1234 {
		t.Fatalf("filtered transport status = %#v, want every explicit plane and timestamp retained", status)
	}
	if got := state.GetCompatibility(); len(got) != 1 || got[0].GetPeerId() != "provider-peer" {
		t.Fatalf("compatibility = %#v, want only provider peer", got)
	}
}

func TestRuntimePeerMapFromP2PStateUsesCanonicalRuntimeState(t *testing.T) {
	t.Parallel()

	peers := runtimePeerMapFromP2PState(&p2pproto.RuntimeState{
		ConnectedPeers:         []string{"peer-legacy-routed", "  "},
		RoutedPeers:            []string{"peer-routed-list"},
		ParticipatingPeers:     []string{"peer-participating-list"},
		PhysicalConnectedPeers: []string{"peer-physical-list"},
		PeerStatuses: []*p2pproto.RuntimePeerStatus{
			{PeerId: "peer-legacy-routed", Dialable: false, Reason: "old dial error"},
			{PeerId: "peer-routed-status", Routed: true},
			{PeerId: "peer-dialable", Dialable: true},
			{PeerId: "peer-unreachable", Reason: "dial failed"},
			{PeerId: "peer-ignored", Ignored: true},
			{PeerId: "peer-incompatible", Incompatible: true},
			{PeerId: "peer-relay", RelayOnly: true},
			{PeerId: "peer-logical", Logical: true},
			{PeerId: "peer-disconnected"},
			{PeerId: "  ", Connected: true},
		},
	})

	want := map[string]string{
		"peer-legacy-routed":      "routed",
		"peer-routed-list":        "routed",
		"peer-routed-status":      "routed",
		"peer-participating-list": "participating",
		"peer-physical-list":      "physical",
		"peer-dialable":           "dialable",
		"peer-unreachable":        "unreachable",
		"peer-ignored":            "ignored",
		"peer-incompatible":       "incompatible",
		"peer-relay":              "relay_only",
		"peer-logical":            "logical",
		"peer-disconnected":       "disconnected",
	}
	if len(peers) != len(want) {
		t.Fatalf("peer count = %d, want %d: %#v", len(peers), len(want), peers)
	}
	for peerID, label := range want {
		if peers[peerID] != label {
			t.Fatalf("peer %s label = %q, want %q (all=%#v)", peerID, peers[peerID], label, peers)
		}
	}
}

func TestRuntimePeerStatusFromSwarmionUsesRoutedAsLegacyReachabilityAlias(t *testing.T) {
	t.Parallel()

	lastRoutedAt := time.Unix(123, 456)
	status := runtimePeerStatusFromSwarmion(swarmion.PeerStatus{
		PeerID:        "peer-a",
		Routed:        true,
		Participating: true,
		Logical:       true,
		StateProvider: true,
		Compatible:    true,
		Addresses:     []string{"/ip4/192.0.2.10/tcp/1111"},
		LastRoutedAt:  lastRoutedAt,
	})
	if !status.GetConnected() || !status.GetDialable() {
		t.Fatalf("legacy reachability = connected:%t dialable:%t, want routed alias true", status.GetConnected(), status.GetDialable())
	}
	if !status.GetStateProvider() || !status.GetCompatible() {
		t.Fatalf("database status flags were not preserved: %#v", status)
	}
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() {
		t.Fatalf("explicit Swarmion status planes were not preserved: %#v", status)
	}
	if got := status.GetLastRoutedAtUnixNano(); got != lastRoutedAt.UnixNano() {
		t.Fatalf("last routed timestamp = %d, want %d", got, lastRoutedAt.UnixNano())
	}
}

func TestRuntimePeerStatusFromSwarmionDoesNotPresentOverlayMembershipAsReachability(t *testing.T) {
	t.Parallel()

	status := runtimePeerStatusFromSwarmion(swarmion.PeerStatus{
		PeerID:        "peer-a",
		Participating: true,
		Logical:       true,
	})
	if status.GetConnected() || status.GetDialable() {
		t.Fatalf("overlay-only peer surfaced as network reachable: %#v", status)
	}
	if !status.GetParticipating() || !status.GetLogical() || status.GetRouted() || status.GetPhysicalConnected() {
		t.Fatalf("explicit overlay planes were not kept separate: %#v", status)
	}
}

func TestP2PListenAddrsWithPeerIDCanBeSharedByControlAndSwarmion(t *testing.T) {
	t.Parallel()

	got := p2pListenAddrsWithPeerID([]string{
		"/ip4/192.0.2.10/tcp/1111",
		"/ip4/192.0.2.10/udp/1111/quic-v1/p2p/peer-a",
		"/ip4/198.51.100.5/tcp/2222/p2p/relay-a/p2p-circuit",
		"/ip4/192.0.2.10/tcp/1111",
	}, "peer-a")
	want := []string{
		"/ip4/192.0.2.10/tcp/1111/p2p/peer-a",
		"/ip4/192.0.2.10/udp/1111/quic-v1/p2p/peer-a",
		"/ip4/198.51.100.5/tcp/2222/p2p/relay-a/p2p-circuit/p2p/peer-a",
	}
	if len(got) != len(want) {
		t.Fatalf("shared addresses = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shared address[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRuntimeStateFromP2PProtoMapsTransportPlanesAndReceiptMetrics(t *testing.T) {
	t.Parallel()

	state := runtimeStateFromP2PProto(&p2pproto.RuntimeState{
		EventReceiptContentDissentObservations: 7,
		RoutedPeers:                            []string{"routed"},
		ParticipatingPeers:                     []string{"participating"},
		LogicalPeers:                           []string{"logical"},
		LogicalPeerTarget:                      8,
		PhysicalConnectedPeers:                 []string{"physical"},
		PeerStatuses: []*p2pproto.RuntimePeerStatus{{
			PeerId:               "peer-a",
			Routed:               true,
			Participating:        true,
			Logical:              true,
			PhysicalConnected:    true,
			LastRoutedAtUnixNano: 1234,
		}},
	})
	if got := state.GetEventReceiptContentDissentObservations(); got != 7 {
		t.Fatalf("event receipt content dissent observations = %d, want 7", got)
	}
	if got := state.GetLogicalPeerTarget(); got != 8 {
		t.Fatalf("logical peer target = %d, want 8", got)
	}
	for name, got := range map[string][]string{
		"routed":        state.GetRoutedPeers(),
		"participating": state.GetParticipatingPeers(),
		"logical":       state.GetLogicalPeers(),
		"physical":      state.GetPhysicalConnectedPeers(),
	} {
		if len(got) != 1 || got[0] != name {
			t.Fatalf("%s peers = %#v, want [%s]", name, got, name)
		}
	}
	status := state.GetPeerStatuses()[0]
	if !status.GetRouted() || !status.GetParticipating() || !status.GetLogical() || !status.GetPhysicalConnected() || status.GetLastRoutedAtUnixNano() != 1234 {
		t.Fatalf("peer transport status = %#v", status)
	}
}

func TestApplyEventReceiptMetricsMapsContentDissentObservations(t *testing.T) {
	t.Parallel()

	state := &pbApic.RuntimeState{}
	applyEventReceiptMetrics(state, db.EventReceiptMetrics{ContentDissentObservations: 11})
	if got := state.GetEventReceiptContentDissentObservations(); got != 11 {
		t.Fatalf("event receipt content dissent observations = %d, want 11", got)
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

func TestCommitViewToProtoResolvesSignerToUserDevice(t *testing.T) {
	backend, _, manager := newUserDeviceTestBackend(t)
	backend.commitIdentities = newCommitIdentityResolver(backend.protosClient)

	createdUser, err := manager.CreateUser("alex", "Alex", false)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	localKey, err := backend.protosClient.KeyManager.GetLocalKey()
	if err != nil {
		t.Fatalf("get local key: %v", err)
	}
	if err := manager.AddDevice(createdUser.Username, "baracuda", localKey); err != nil {
		t.Fatalf("add device: %v", err)
	}
	backend.commitIdentities.Notify()

	commit := db.CommitView{
		Commit: db.Commit{
			Hash:            "commit",
			Committer:       "swarmion-checkpoint",
			SignerPublicKey: libp2pPublicKeyString(t, localKey),
		},
	}

	if got := backend.commitViewToProto(commit).GetCommitter(); got != "alex (baracuda)" {
		t.Fatalf("committer = %q, want alex (baracuda)", got)
	}
}

func TestCommitViewToProtoUsesSystemFallbackForUnsignedSwarmionCheckpoint(t *testing.T) {
	backend, _, _ := newUserDeviceTestBackend(t)
	backend.commitIdentities = newCommitIdentityResolver(backend.protosClient)

	commit := db.CommitView{
		Commit: db.Commit{
			Hash:      "commit",
			Committer: "swarmion-checkpoint",
		},
	}

	if got := backend.commitViewToProto(commit).GetCommitter(); got != "system" {
		t.Fatalf("committer = %q, want system", got)
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
	store, err := db.Open(workDir, "protos_test", localKey, testswarmion.NewBorrowedLink(t, localKey))
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

func libp2pPublicKeyString(t *testing.T, key *pcrypto.Key) string {
	t.Helper()

	rawPublicKey, err := base64.StdEncoding.DecodeString(key.PublicString())
	if err != nil {
		t.Fatalf("decode public key: %v", err)
	}
	pub, err := libp2pcrypto.UnmarshalEd25519PublicKey(rawPublicKey)
	if err != nil {
		t.Fatalf("unmarshal public key: %v", err)
	}
	marshaled, err := libp2pcrypto.MarshalPublicKey(pub)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	return base64.StdEncoding.EncodeToString(marshaled)
}

func countUsersByUsername(t *testing.T, store *db.DB, username string) int {
	t.Helper()

	var count int
	if err := store.ReadRows(context.Background(), "SELECT COUNT(*) FROM users WHERE username = ?", []any{username}, func(rows *sql.Rows) error {
		if !rows.Next() {
			return errors.New("query users count returned no rows")
		}
		return rows.Scan(&count)
	}); err != nil {
		t.Fatalf("query users count: %v", err)
	}
	return count
}
