package p2p

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/go-playground/validator/v10"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/imageregistry"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"google.golang.org/grpc"
	swarmionapp "swarmion.dev/runtime/app"
)

var _ proto.PingerServer = (*Server)(nil)
var _ proto.PeerDBServer = (*Server)(nil)
var _ proto.AppsServer = (*Server)(nil)
var _ proto.ImagesServer = (*Server)(nil)
var _ proto.InstanceServer = (*Server)(nil)

const imageArchiveUploadMaxChunkSize = 1024 * 1024

type ExternalDB interface {
	AddPeer(peerID string, conn *grpc.ClientConn) error
	RemovePeer(peerID string) error
	GetAllCommits() ([]db.Commit, error)
	ExecSQLAndCommit(statement string, commitMsg string) (string, error)
	GetLastCommit(branch string) (db.Commit, error)
	CatchUpFinalized(ctx context.Context, reason string) error
	InitFromPeer(peerID string, bootstrapPeers []string) error
	EnableGRPCServers(server *grpc.Server) error
	Initialized() bool
}

type initMachine struct {
	id        string
	publicIP  string
	publicKey string
}

func (im *initMachine) GetID() string {
	return im.id
}
func (im *initMachine) GetPublicKey() string {
	return im.publicKey
}
func (im *initMachine) GetPublicIP() string {
	return im.publicIP
}
func (im *initMachine) GetName() string {
	return im.id
}

type Server struct {
	DB  ExternalDB
	p2p *P2P
}

type swarmionRuntimeReader interface {
	SwarmionStatus() (swarmionapp.Status, bool)
	SwarmionCompatibility(context.Context) ([]swarmionapp.ManifestCompatibility, error)
	SwarmionPeerStatus(context.Context) ([]swarmionapp.PeerStatus, error)
	SwarmionContentSyncTrace() ([]string, bool)
}

func (s *Server) Ping(ctx context.Context, req *proto.PingRequest) (*proto.PingResponse, error) {
	_, ok := p2pgrpc.RemotePeerFromContext(ctx)
	if !ok {
		return nil, errors.New("no AuthInfo in context")
	}

	res := &proto.PingResponse{
		Pong: "Ping: " + req.Ping + "!",
	}
	return res, nil
}

func (s *Server) ExecSQL(ctx context.Context, req *proto.ExecSQLRequest) (*proto.ExecSQLResponse, error) {
	commit, err := s.DB.ExecSQLAndCommit(req.Statement, req.Msg)
	if err != nil {
		return nil, err
	}
	return &proto.ExecSQLResponse{Result: "", Commit: commit}, nil
}

func (s *Server) GetAllCommits(ctx context.Context, _ *proto.GetAllCommitsRequest) (*proto.GetAllCommitsResponse, error) {
	if err := s.DB.CatchUpFinalized(ctx, "p2p get all commits"); err != nil {
		return nil, err
	}
	commits, err := s.DB.GetAllCommits()
	if err != nil {
		return nil, err
	}

	res := &proto.GetAllCommitsResponse{}
	for _, commit := range commits {
		respCommit := proto.Commit{
			Hash:         commit.Hash,
			Committer:    commit.Committer,
			Message:      commit.Message,
			ParentHashes: append([]string(nil), commit.ParentHashes...),
			Refs:         append([]string(nil), commit.Refs...),
		}
		if !commit.Date.IsZero() {
			respCommit.DateUnix = commit.Date.Unix()
		}
		res.Commits = append(res.Commits, &respCommit)
	}

	return res, nil
}

func (s *Server) GetHead(ctx context.Context, _ *proto.GetHeadRequest) (*proto.GetHeadResponse, error) {
	if err := s.DB.CatchUpFinalized(ctx, "p2p get head"); err != nil {
		return nil, err
	}
	commit, err := s.DB.GetLastCommit("main")
	if err != nil {
		return nil, err
	}
	return &proto.GetHeadResponse{Commit: commit.Hash}, nil
}

// HandlerGetInstanceLogs retrieves logs for the local instance
func (s *Server) GetLogs(context.Context, *proto.GetLogsRequest) (*proto.GetLogsResponse, error) {

	var logs []byte
	var err error
	for _, logPath := range []string{"/var/log/protos.log", "/proc/1/root/var/log/protos.log"} {
		logs, err = os.ReadFile(logPath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read protos logs: %w", err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)

	return &proto.GetLogsResponse{Logs: encodedLogs}, nil
}

func (s *Server) GetNetworkState(context.Context, *proto.GetNetworkStateRequest) (*proto.GetNetworkStateResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.network == nil {
		return nil, fmt.Errorf("network inspector is not configured")
	}
	state, err := s.p2p.network.State()
	if err != nil {
		return nil, err
	}
	return &proto.GetNetworkStateResponse{State: networkStateToProto(state)}, nil
}

func (s *Server) GetExitRoutes(ctx context.Context, _ *proto.GetExitRoutesRequest) (*proto.GetExitRoutesResponse, error) {
	database, ok := s.DB.(*db.DB)
	if !ok || database == nil {
		return nil, fmt.Errorf("exit route reader is not configured")
	}
	if err := s.DB.CatchUpFinalized(ctx, "p2p get exit routes"); err != nil {
		return nil, err
	}
	routes, err := readExitRoutes(ctx, database)
	if err != nil {
		return nil, err
	}
	resp := &proto.GetExitRoutesResponse{}
	for _, route := range routes {
		resp.Routes = append(resp.Routes, exitRouteToP2PProto(route))
	}
	return resp, nil
}

func (s *Server) GetRuntimeState(ctx context.Context, _ *proto.GetRuntimeStateRequest) (*proto.GetRuntimeStateResponse, error) {
	reader, ok := s.DB.(swarmionRuntimeReader)
	if !ok || reader == nil {
		return nil, fmt.Errorf("runtime state reader is not configured")
	}
	if err := s.DB.CatchUpFinalized(ctx, "p2p get runtime state"); err != nil {
		return nil, err
	}
	state, err := runtimeStateToP2PProto(ctx, reader)
	if err != nil {
		return nil, err
	}
	return &proto.GetRuntimeStateResponse{State: state}, nil
}

func (s *Server) DescribeImage(ctx context.Context, req *proto.DescribeImageRequest) (*proto.DescribeImageResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	imageRef, targetDigest, platform, labels, found, err := s.p2p.imageManager.DescribeImage(ctx, req.GetImageRef())
	if err != nil {
		return nil, err
	}
	return &proto.DescribeImageResponse{
		Found:        found,
		ImageRef:     imageRef,
		TargetDigest: targetDigest,
		Platform:     platform,
		Labels:       labels,
	}, nil
}

func (s *Server) GetImageContent(ctx context.Context, req *proto.GetImageContentRequest) (*proto.GetImageContentResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	content, found, err := s.p2p.imageManager.GetImageContent(ctx, req.GetImageRef())
	if err != nil {
		return nil, err
	}
	return &proto.GetImageContentResponse{
		Found:       found,
		ImageRef:    content.ImageRef,
		Target:      imageDescriptorToProto(content.Target),
		Platform:    content.Platform,
		Labels:      content.Labels,
		Descriptors: imageDescriptorsToProto(content.Descriptors),
	}, nil
}

func (s *Server) GetImageBlobChunk(ctx context.Context, req *proto.GetImageBlobChunkRequest) (*proto.GetImageBlobChunkResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	data, eof, err := s.p2p.imageManager.ReadImageBlobChunk(ctx, req.GetDigest(), req.GetOffset(), req.GetLength())
	if err != nil {
		return nil, err
	}
	return &proto.GetImageBlobChunkResponse{
		Digest: req.GetDigest(),
		Offset: req.GetOffset(),
		Data:   data,
		Eof:    eof,
	}, nil
}

func (s *Server) LoadImageArchive(ctx context.Context, req *proto.LoadImageArchiveRequest) (*proto.LoadImageArchiveResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	loaded, err := s.p2p.imageManager.LoadImageArchive(ctx, req.GetArchivePath(), req.GetImageRef())
	if err != nil {
		return nil, err
	}
	return &proto.LoadImageArchiveResponse{
		ImageRef:     loaded.ImageRef,
		TargetDigest: loaded.TargetDigest,
		Platform:     loaded.Platform,
	}, nil
}

func (s *Server) UploadImageArchiveChunk(ctx context.Context, req *proto.UploadImageArchiveChunkRequest) (*proto.UploadImageArchiveChunkResponse, error) {
	if s == nil || s.p2p == nil || s.p2p.imageManager == nil {
		return nil, fmt.Errorf("image manager is not configured")
	}
	if len(req.GetData()) > imageArchiveUploadMaxChunkSize {
		return nil, fmt.Errorf("image archive chunk is too large: %d bytes", len(req.GetData()))
	}
	archivePath, err := imageArchiveUploadPath(req.GetUploadId())
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(archivePath), 0700); err != nil {
		return nil, err
	}

	offset := req.GetOffset()
	flags := os.O_CREATE | os.O_WRONLY
	if offset == 0 {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(archivePath, flags, 0600)
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		info, statErr := file.Stat()
		if statErr != nil {
			_ = file.Close()
			return nil, statErr
		}
		if uint64(info.Size()) != offset {
			_ = file.Close()
			return nil, fmt.Errorf("image archive upload offset mismatch: got %d, current size %d", offset, info.Size())
		}
		if _, seekErr := file.Seek(int64(offset), io.SeekStart); seekErr != nil {
			_ = file.Close()
			return nil, seekErr
		}
	}
	written, err := file.Write(req.GetData())
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if written != len(req.GetData()) {
		return nil, io.ErrShortWrite
	}

	received := offset + uint64(written)
	resp := &proto.UploadImageArchiveChunkResponse{ReceivedBytes: received}
	if !req.GetEof() {
		return resp, nil
	}
	loaded, err := s.p2p.imageManager.LoadImageArchive(ctx, archivePath, req.GetImageRef())
	_ = os.Remove(archivePath)
	if err != nil {
		return nil, err
	}
	resp.Loaded = true
	resp.ImageRef = loaded.ImageRef
	resp.TargetDigest = loaded.TargetDigest
	resp.Platform = loaded.Platform
	return resp, nil
}

func imageArchiveUploadPath(uploadID string) (string, error) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return "", fmt.Errorf("image archive upload id is empty")
	}
	if len(uploadID) > 96 {
		return "", fmt.Errorf("image archive upload id is too long")
	}
	for _, r := range uploadID {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			continue
		}
		return "", fmt.Errorf("image archive upload id contains invalid character %q", r)
	}
	return filepath.Join(os.TempDir(), "protos-p2p-image-upload-"+uploadID+".tar.gz"), nil
}

func imageDescriptorToProto(desc imageregistry.Descriptor) *proto.ImageContentDescriptor {
	if desc.Digest == "" {
		return nil
	}
	return &proto.ImageContentDescriptor{
		MediaType:   desc.MediaType,
		Digest:      desc.Digest,
		SizeBytes:   desc.SizeBytes,
		Platform:    desc.Platform,
		Annotations: desc.Annotations,
	}
}

func imageDescriptorsToProto(descs []imageregistry.Descriptor) []*proto.ImageContentDescriptor {
	out := make([]*proto.ImageContentDescriptor, 0, len(descs))
	for _, desc := range descs {
		out = append(out, imageDescriptorToProto(desc))
	}
	return out
}

// HandlerGetInstancePeers retrieves the peers for the local instance
func (s *Server) GetPeers(context.Context, *proto.GetPeersRequest) (*proto.GetPeersResponse, error) {

	peers := map[string]string{}
	for id, machine := range s.p2p.machines.Snapshot() {
		peerStatus := string(PeerStatusDesired)
		if state, found := s.p2p.peerStates.Get(id); found && state.Status != "" {
			peerStatus = string(state.Status)
		}
		if client, found := s.p2p.clients.Get(id); found && client != nil {
			peerStatus = string(PeerStatusConnected)
		}
		if machine.GetName() != "" {
			peers[machine.GetName()] = peerStatus
			continue
		}
		peers[id] = peerStatus
	}

	return &proto.GetPeersResponse{Peers: peers}, nil
}

func networkStateToProto(state networkmodule.State) *proto.NetworkState {
	out := &proto.NetworkState{
		Module:        state.Module,
		Up:            state.Up,
		InterfaceName: state.InterfaceName,
		Messages:      append([]string(nil), state.Messages...),
	}
	for _, item := range state.Interfaces {
		out.Interfaces = append(out.Interfaces, &proto.NetworkInterface{
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
		out.Addresses = append(out.Addresses, &proto.NetworkAddress{
			InterfaceName: item.InterfaceName,
			Cidr:          item.CIDR,
			Scope:         item.Scope,
		})
	}
	for _, item := range state.Routes {
		out.Routes = append(out.Routes, &proto.NetworkRoute{
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
		out.WireguardPeers = append(out.WireguardPeers, &proto.WireGuardPeer{
			PublicKey:       item.PublicKey,
			Endpoint:        item.Endpoint,
			AllowedIps:      append([]string(nil), item.AllowedIPs...),
			LatestHandshake: item.LatestHandshake,
			RxBytes:         item.RxBytes,
			TxBytes:         item.TxBytes,
		})
	}
	for _, table := range state.FirewallTables {
		tableProto := &proto.FirewallTable{
			Family: table.Family,
			Name:   table.Name,
		}
		for _, chain := range table.Chains {
			chainProto := &proto.FirewallChain{
				Name:     chain.Name,
				Type:     chain.Type,
				Hook:     chain.Hook,
				Priority: chain.Priority,
			}
			for _, rule := range chain.Rules {
				chainProto.Rules = append(chainProto.Rules, &proto.FirewallRule{
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
		out.Dns = append(out.Dns, &proto.DNSState{
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

type p2pExitRoute struct {
	ID            string
	DeviceID      string
	InstanceID    string
	DesiredStatus string
	DNSServer     string
	CIDRs         []string
}

func readExitRoutes(ctx context.Context, database *db.DB) ([]p2pExitRoute, error) {
	rows, err := database.QueryContext(ctx, "SELECT id, device_id, instance_id, desired_status, dns_server, cidrs FROM exit_routes")
	if err != nil {
		return nil, fmt.Errorf("retrieve exit routes: %w", err)
	}
	defer rows.Close()

	var routes []p2pExitRoute
	for rows.Next() {
		var route p2pExitRoute
		var id, deviceID, instanceID []byte
		var cidrsJSON string
		if err := rows.Scan(&id, &deviceID, &instanceID, &route.DesiredStatus, &route.DNSServer, &cidrsJSON); err != nil {
			return nil, fmt.Errorf("scan exit route: %w", err)
		}
		route.ID = db.UUIDString(id)
		route.DeviceID = db.UUIDString(deviceID)
		route.InstanceID = db.UUIDString(instanceID)
		route.CIDRs = parseExitRouteCIDRS(cidrsJSON)
		routes = append(routes, route)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read exit routes: %w", err)
	}
	return routes, nil
}

func parseExitRouteCIDRS(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	var cidrs []string
	if err := json.Unmarshal([]byte(value), &cidrs); err != nil {
		return []string{value}
	}
	return cidrs
}

func exitRouteToP2PProto(route p2pExitRoute) *proto.ExitRoute {
	return &proto.ExitRoute{
		Id:         route.ID,
		DeviceId:   route.DeviceID,
		InstanceId: route.InstanceID,
		Status:     route.DesiredStatus,
		DnsServer:  route.DNSServer,
		Cidrs:      append([]string(nil), route.CIDRs...),
	}
}

func runtimeStateToP2PProto(ctx context.Context, reader swarmionRuntimeReader) (*proto.RuntimeState, error) {
	status, ok := reader.SwarmionStatus()
	if !ok {
		return nil, fmt.Errorf("swarmion status is not available")
	}
	out := &proto.RuntimeState{
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
		KnownEpochIds:                append([]string(nil), status.KnownEpochIDs...),
		EpochDescriptorDigestById:    cloneStringMap(status.EpochDescriptorDigestByID),
		EpochFinalizedDigestById:     cloneStringMap(status.EpochFinalizedDigestByID),
		ProtocolFinalizedDigest:      formatRuntimeDigest(status.ProtocolFinalizedDigest),
	}
	if status.Fatal != nil {
		out.FatalState = status.Fatal.State
	} else {
		out.FatalState = status.FatalState.String()
	}

	peerStatuses, err := reader.SwarmionPeerStatus(ctx)
	if err != nil {
		return nil, err
	}
	for _, peerStatus := range peerStatuses {
		out.PeerStatuses = append(out.PeerStatuses, &proto.RuntimePeerStatus{
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
	if database, ok := reader.(*db.DB); ok {
		peerIDs, err := db.GetPeerIDs(database)
		if err != nil {
			return nil, err
		}
		filterRuntimePeerSurface(out, peerIDs)
		addKnownRuntimePeerStatuses(out, peerIDs)
	}

	compatibility, err := reader.SwarmionCompatibility(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range compatibility {
		out.Compatibility = append(out.Compatibility, &proto.RuntimeCompatibility{
			PeerId:       item.PeerID,
			LocalDigest:  item.LocalDigest,
			RemoteDigest: item.RemoteDigest,
			Compatible:   item.Compatible,
			Blocking:     item.Blocking,
			Reason:       item.Reason,
		})
	}
	if trace, ok := reader.SwarmionContentSyncTrace(); ok {
		out.ContentSyncTrace = append([]string(nil), trace...)
	}
	sanitizeRuntimeStateStrings(out)
	return out, nil
}

func sanitizeRuntimeStateStrings(out *proto.RuntimeState) {
	if out == nil {
		return
	}
	out.PeerId = validProtoString(out.GetPeerId())
	out.ManifestDigest = validProtoString(out.GetManifestDigest())
	out.FinalizedRootHash = validProtoString(out.GetFinalizedRootHash())
	out.TentativeRootHash = validProtoString(out.GetTentativeRootHash())
	out.ProtocolFinalizedRootHash = validProtoString(out.GetProtocolFinalizedRootHash())
	out.DurableMainRootHash = validProtoString(out.GetDurableMainRootHash())
	out.ActiveEpochId = validProtoString(out.GetActiveEpochId())
	out.ActiveWitnessIds = validProtoStrings(out.GetActiveWitnessIds())
	out.EligibleWitnessIds = validProtoStrings(out.GetEligibleWitnessIds())
	out.StateProviders = validProtoStrings(out.GetStateProviders())
	out.ConnectedPeers = validProtoStrings(out.GetConnectedPeers())
	out.FatalState = validProtoString(out.GetFatalState())
	out.RuntimeRefreshLastError = validProtoString(out.GetRuntimeRefreshLastError())
	out.RuntimeFinalizedLastError = validProtoString(out.GetRuntimeFinalizedLastError())
	out.RuntimeMaterializationPolicy = validProtoString(out.GetRuntimeMaterializationPolicy())
	out.ContentSyncTrace = validProtoStrings(out.GetContentSyncTrace())
	out.KnownEpochIds = validProtoStrings(out.GetKnownEpochIds())
	out.EpochDescriptorDigestById = validProtoStringMap(out.GetEpochDescriptorDigestById())
	out.EpochFinalizedDigestById = validProtoStringMap(out.GetEpochFinalizedDigestById())
	out.ProtocolFinalizedDigest = validProtoString(out.GetProtocolFinalizedDigest())

	for _, item := range out.GetCompatibility() {
		if item == nil {
			continue
		}
		item.PeerId = validProtoString(item.GetPeerId())
		item.LocalDigest = validProtoString(item.GetLocalDigest())
		item.RemoteDigest = validProtoString(item.GetRemoteDigest())
		item.Reason = validProtoString(item.GetReason())
	}
	for _, item := range out.GetPeerStatuses() {
		if item == nil {
			continue
		}
		item.PeerId = validProtoString(item.GetPeerId())
		item.Addresses = validProtoStrings(item.GetAddresses())
		item.LastDialErrors = validProtoStringMap(item.GetLastDialErrors())
		item.Reason = validProtoString(item.GetReason())
	}
}

func validProtoString(value string) string {
	return strings.ToValidUTF8(value, "?")
}

func validProtoStrings(values []string) []string {
	for i, value := range values {
		values[i] = validProtoString(value)
	}
	return values
}

func validProtoStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return values
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[validProtoString(key)] = validProtoString(value)
	}
	return out
}

func filterRuntimePeerSurface(out *proto.RuntimeState, peerIDs map[string]struct{}) {
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
	for _, peerStatus := range out.GetPeerStatuses() {
		if peerStatus == nil {
			continue
		}
		peerID := strings.TrimSpace(peerStatus.GetPeerId())
		if _, found := allowed[peerID]; !found {
			continue
		}
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

func addKnownRuntimePeerStatuses(out *proto.RuntimeState, peerIDs map[string]struct{}) {
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

func knownRuntimePeerStatus(peerID string, state *proto.RuntimeState) *proto.RuntimePeerStatus {
	isSelf := peerID == strings.TrimSpace(state.GetPeerId())
	status := &proto.RuntimePeerStatus{
		PeerId:          peerID,
		Connected:       isSelf || stringInList(peerID, state.GetConnectedPeers()),
		Dialable:        isSelf,
		StateProvider:   stringInList(peerID, state.GetStateProviders()),
		Witness:         stringInList(peerID, state.GetActiveWitnessIds()),
		EligibleWitness: stringInList(peerID, state.GetEligibleWitnessIds()),
		Compatible:      isSelf,
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

// HandlerInit does the initialisation on the server side
func (s *Server) Init(ctx context.Context, req *proto.InitRequest) (*proto.InitResponse, error) {
	if s.DB.Initialized() {
		return nil, errors.New("initialisation RPC is disabled after database initialisation")
	}

	remotePeer, ok := p2pgrpc.RemotePeerFromContext(ctx)
	if !ok {
		return nil, errors.New("cannot perform initialization: missing remote peer identity")
	}

	validate := validator.New()
	err := validate.Struct(req)
	if err != nil {
		return nil, fmt.Errorf("failed to validate init request: %w", err)
	}

	pubKey, err := pcrypto.CreatePublicKeyFromBase64(req.OriginDevicePublicKey)
	if err != nil {
		return nil, fmt.Errorf("cannot perform initialization: %w", err)
	}

	im := initMachine{
		id:        pubKey.GetID(),
		publicKey: req.OriginDevicePublicKey,
	}
	if remotePeer.String() != im.GetID() {
		return nil, fmt.Errorf("cannot perform initialization: request origin %s does not match authenticated peer %s", im.GetID(), remotePeer.String())
	}
	if !s.p2p.initPeerAllowed(remotePeer) {
		return nil, fmt.Errorf("cannot perform initialization: peer %s is not the configured init peer", remotePeer.String())
	}

	_, err = s.p2p.AddPeer(&im)
	if err != nil {
		return nil, fmt.Errorf("failed to add init device as rpc client: %w", err)
	}

	originSwarmionAddrs := append([]string(nil), req.OriginSwarmionAddrs...)
	tunnelAddr, closeTunnel, tunnelErr := s.p2p.StartSwarmionBootstrapTunnel(ctx, im.GetID())
	if tunnelErr != nil {
		log.Warnf("failed to create libp2p Swarmion bootstrap tunnel: %v", tunnelErr)
	} else {
		defer closeTunnel()
		originSwarmionAddrs = append([]string{tunnelAddr}, originSwarmionAddrs...)
	}

	err = s.DB.InitFromPeer(im.GetID(), originSwarmionAddrs)
	if err != nil {
		err = fmt.Errorf("failed to initialize database from peer %s(%s): %w", im.GetID(), pubKey.GetID(), err)
		log.Error(err.Error())
		return nil, err
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		s.p2p.restartServerSignal <- im
		close(s.p2p.restartServerSignal)
	}()

	return &proto.InitResponse{Architecture: runtime.GOARCH}, nil
}

func (s *Server) GetAppLogs(ctx context.Context, req *proto.GetAppLogsRequest) (*proto.GetAppLogsResponse, error) {

	logs, err := s.p2p.appManager.GetLogs(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for app '%s': %w", req.AppName, err)
	}

	encodedLogs := base64.StdEncoding.EncodeToString(logs)
	return &proto.GetAppLogsResponse{Logs: encodedLogs}, nil
}

func (s *Server) GetAppStatus(ctx context.Context, req *proto.GetAppStatusRequest) (*proto.GetAppStatusResponse, error) {
	if err := s.DB.CatchUpFinalized(ctx, "p2p get app status"); err != nil {
		return nil, err
	}

	status, err := s.p2p.appManager.GetStatus(req.AppName)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve status for app '%s': %w", req.AppName, err)
	}

	return &proto.GetAppStatusResponse{Status: status}, nil
}
