package p2p

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	p2pgrpc "github.com/birros/go-libp2p-grpc"
	"github.com/go-playground/validator/v10"
	"github.com/protosio/protos/internal/db"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"google.golang.org/grpc"
	swarmionapp "swarmion.dev/runtime/app"
)

var _ proto.PingerServer = (*Server)(nil)
var _ proto.PeerDBServer = (*Server)(nil)
var _ proto.AppsServer = (*Server)(nil)
var _ proto.InstanceServer = (*Server)(nil)

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
			Hash:      commit.Hash,
			Committer: commit.Committer,
			Message:   commit.Message,
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
		var cidrsJSON string
		if err := rows.Scan(&route.ID, &route.DeviceID, &route.InstanceID, &route.DesiredStatus, &route.DNSServer, &cidrsJSON); err != nil {
			return nil, fmt.Errorf("scan exit route: %w", err)
		}
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
	return out, nil
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

// HandlerInit does the initialisation on the server side
func (s *Server) Init(ctx context.Context, req *proto.InitRequest) (*proto.InitResponse, error) {

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

	_, err = s.p2p.AddPeer(&im)
	if err != nil {
		return nil, fmt.Errorf("failed to add init device as rpc client: %w", err)
	}

	err = s.DB.InitFromPeer(im.GetID(), req.OriginSwarmionAddrs)
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
