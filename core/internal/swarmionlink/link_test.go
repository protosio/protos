package swarmionlink

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	libp2pnode "github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p/core/network"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/nustiueudinastea/swarmion/transports"
	"github.com/nustiueudinastea/swarmion/transports/transporttest"
)

func TestNewRejectsNilHost(t *testing.T) {
	link, err := New(nil)
	if link != nil {
		t.Fatal("nil host returned a Link")
	}
	if !errors.Is(err, transports.ErrInvalidArgument) {
		t.Fatalf("nil host error = %v, want ErrInvalidArgument", err)
	}
}

func TestRouteFenceWithdrawsLinkRejectsStreamsAndAdvancesGeneration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fence, err := NewRouteFence()
	if err != nil {
		t.Fatalf("new route fence: %v", err)
	}
	aHost := newTestHost(t)
	bHost, err := libp2pnode.New(
		libp2pnode.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2pnode.ConnectionGater(fence),
	)
	if err != nil {
		t.Fatalf("new fenced host: %v", err)
	}
	t.Cleanup(func() { _ = bHost.Close() })
	a := newTestLink(t, aHost)
	bRegistry, err := NewRegistryWithRouteFence(bHost, fence)
	if err != nil {
		t.Fatalf("new fenced registry: %v", err)
	}
	b := bRegistry.Link()
	ensureTestRoute(t, ctx, a, b)

	snapshot, subscription, err := b.SubscribeRoutes(ctx)
	if err != nil {
		t.Fatalf("subscribe routes: %v", err)
	}
	defer subscription.Close(ctx)
	initial, found := snapshot.Routes[a.LocalPeer()]
	if !found {
		t.Fatalf("initial route missing from snapshot: %+v", snapshot)
	}

	firstGeneration, err := fence.FencePeer(aHost.ID())
	if err != nil {
		t.Fatalf("fence peer: %v", err)
	}
	event, err := subscription.Next(ctx)
	if err != nil {
		t.Fatalf("read fence route event: %v", err)
	}
	if event.Reachable || event.Peer != a.LocalPeer() {
		t.Fatalf("fence route event = %+v, want unreachable %s", event, a.LocalPeer())
	}
	if _, err := b.OpenStream(ctx, a.LocalPeer(), "/swarmion/test/1"); !errors.Is(err, transports.ErrPolicyDenied) {
		t.Fatalf("fenced OpenStream error = %v, want ErrPolicyDenied", err)
	}
	if err := b.EnsureRoute(ctx, transports.RouteRequest{Peer: a.LocalPeer()}); !errors.Is(err, transports.ErrPolicyDenied) {
		t.Fatalf("fenced EnsureRoute error = %v, want ErrPolicyDenied", err)
	}
	if err := bHost.Network().ClosePeer(aHost.ID()); err != nil {
		t.Fatalf("close fenced physical connection: %v", err)
	}
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		if aHost.Network().Connectedness(bHost.ID()) == libp2pnetwork.NotConnected &&
			bHost.Network().Connectedness(aHost.ID()) == libp2pnetwork.NotConnected {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if aHost.Network().Connectedness(bHost.ID()) != libp2pnetwork.NotConnected {
		t.Fatal("fenced physical connection did not close on the dialing peer")
	}
	_ = aHost.Connect(ctx, libp2ppeer.AddrInfo{ID: bHost.ID(), Addrs: bHost.Addrs()})
	time.Sleep(100 * time.Millisecond)
	if aHost.Network().Connectedness(bHost.ID()) != libp2pnetwork.NotConnected ||
		bHost.Network().Connectedness(aHost.ID()) != libp2pnetwork.NotConnected {
		t.Fatal("fenced peer redial established a physical connection")
	}

	if err := fence.ReopenPeer(aHost.ID()); err != nil {
		t.Fatalf("readmit peer: %v", err)
	}
	if err := aHost.Connect(ctx, libp2ppeer.AddrInfo{ID: bHost.ID(), Addrs: bHost.Addrs()}); err != nil {
		t.Fatalf("connect readmitted peer: %v", err)
	}
	restored, err := subscription.Next(ctx)
	if err != nil {
		t.Fatalf("read restored route event: %v", err)
	}
	if !restored.Reachable || restored.Route.Generation <= initial.Generation {
		t.Fatalf("restored route = %+v, initial = %+v", restored.Route, initial)
	}
	secondGeneration, err := fence.FencePeer(aHost.ID())
	if err != nil {
		t.Fatalf("refence peer: %v", err)
	}
	if secondGeneration == firstGeneration {
		t.Fatalf("route-fence generation was reused: %s", secondGeneration)
	}
	if fence.GenerationMatches(aHost.ID(), firstGeneration) {
		t.Fatal("old route-fence generation still matched")
	}
	if !fence.GenerationMatches(aHost.ID(), secondGeneration) {
		t.Fatal("current route-fence generation did not match")
	}
}

func TestRouteFenceRevisionRejectsStaleAdmissionTrackerCallback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fence, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	aHost := newTestHost(t)
	bHost, err := libp2pnode.New(
		libp2pnode.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2pnode.ConnectionGater(fence),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = bHost.Close() })
	a := newTestLink(t, aHost)
	registry, err := NewRegistryWithRouteFence(bHost, fence)
	if err != nil {
		t.Fatal(err)
	}
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	admissionComputed := make(chan struct{})
	releaseAdmission := make(chan struct{})
	fence.beforeTrackerUpdateForTest = func(remote libp2ppeer.ID, _ uint64, blocked bool) {
		if remote == aHost.ID() && !blocked {
			select {
			case <-admissionComputed:
			default:
				close(admissionComputed)
			}
			<-releaseAdmission
		}
	}
	admitDone := make(chan error, 1)
	go func() { admitDone <- fence.AdmitPeer(aHost.ID()) }()
	select {
	case <-admissionComputed:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := fence.FencePeer(aHost.ID()); err != nil {
		t.Fatalf("newer fence: %v", err)
	}
	if !fence.LinkRouteWithdrawn(aHost.ID()) {
		t.Fatal("newer fence did not withdraw the Link route")
	}
	close(releaseAdmission)
	if err := <-admitDone; err != nil {
		t.Fatalf("older admission result: %v", err)
	}
	if !fence.LinkRouteWithdrawn(aHost.ID()) {
		t.Fatal("stale admission callback re-exposed the fenced Link route")
	}
}

func TestIdentityProbeIsDialOnlyScopedAndDeletionFenceWins(t *testing.T) {
	_, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	_, realPublicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	realPeerID, err := libp2ppeer.IDFromPublicKey(realPublicKey)
	if err != nil {
		t.Fatal(err)
	}
	peerID, err := libp2ppeer.IDFromPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	fence.ReconcileAdmittedPeers(nil, nil)
	if !fence.IsPeerFenced(peerID) || fence.InterceptPeerDial(peerID) {
		t.Fatal("managed unknown peer was dialable before identity probe")
	}

	first, err := fence.BeginIdentityProbe(peerID)
	if err != nil {
		t.Fatalf("begin first identity probe: %v", err)
	}
	second, err := fence.BeginIdentityProbe(peerID)
	if err != nil {
		t.Fatalf("begin second identity probe: %v", err)
	}
	if !fence.InterceptPeerDial(peerID) || !fence.InterceptAddrDial(peerID, nil) {
		t.Fatal("active identity probe did not permit outbound placeholder dial")
	}
	if fence.InterceptPeerDial(realPeerID) || fence.InterceptAddrDial(realPeerID, nil) {
		t.Fatal("identity probe permitted the authenticated real unknown peer")
	}
	if !fence.IsPeerFenced(peerID) || fence.InterceptSecured(libp2pnetwork.DirOutbound, peerID, nil) {
		t.Fatal("identity probe admitted an unknown secured peer or Link route")
	}
	if !fence.IsPeerFenced(realPeerID) || fence.InterceptSecured(libp2pnetwork.DirOutbound, realPeerID, nil) {
		t.Fatal("identity probe admitted the authenticated real unknown peer")
	}
	first.Close()
	first.Close()
	if !fence.InterceptPeerDial(peerID) {
		t.Fatal("closing one of two identity probes revoked the live probe")
	}

	if _, err := fence.FencePeer(peerID); err != nil {
		t.Fatalf("establish deletion fence: %v", err)
	}
	if fence.InterceptPeerDial(peerID) || fence.InterceptAddrDial(peerID, nil) {
		t.Fatal("identity probe overrode an explicit deletion fence")
	}
	second.Close()
	if _, err := fence.BeginIdentityProbe(peerID); !errors.Is(err, ErrPeerFenced) {
		t.Fatalf("probe across deletion fence error = %v, want ErrPeerFenced", err)
	}
}

func TestTemporaryPeerAdmissionIsPhysicalOnlyRefCountedAndPromotable(t *testing.T) {
	_, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peerID, err := libp2ppeer.IDFromPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	fence.ReconcileAdmittedPeers(nil, nil)

	first, err := fence.BeginTemporaryPeerAdmission(peerID)
	if err != nil {
		t.Fatalf("begin first temporary admission: %v", err)
	}
	second, err := fence.BeginTemporaryPeerAdmission(peerID)
	if err != nil {
		t.Fatalf("begin second temporary admission: %v", err)
	}
	if !fence.IsPeerFenced(peerID) {
		t.Fatal("temporary admission exposed the peer to protocol/Link policy")
	}
	if !fence.IsPeerConnectionAllowed(peerID) || !fence.InterceptPeerDial(peerID) ||
		!fence.InterceptSecured(libp2pnetwork.DirOutbound, peerID, nil) {
		t.Fatal("temporary admission did not permit the authenticated physical connection")
	}

	first.Close()
	first.Close()
	if !fence.IsPeerConnectionAllowed(peerID) {
		t.Fatal("closing one scope revoked another live temporary admission")
	}
	if err := second.Promote(); err != nil {
		t.Fatalf("promote second temporary admission: %v", err)
	}
	second.Close()
	if fence.IsPeerFenced(peerID) || !fence.IsPeerConnectionAllowed(peerID) {
		t.Fatal("promotion did not establish permanent peer admission")
	}

	if _, err := fence.FencePeer(peerID); err != nil {
		t.Fatalf("fence promoted peer: %v", err)
	}
	if fence.IsPeerConnectionAllowed(peerID) || fence.InterceptPeerDial(peerID) {
		t.Fatal("explicit deletion fence did not override promoted admission")
	}
	if _, err := fence.BeginTemporaryPeerAdmission(peerID); !errors.Is(err, ErrPeerFenced) {
		t.Fatalf("temporary admission across deletion fence error = %v, want ErrPeerFenced", err)
	}
}

func TestTemporaryPeerAdmissionReleaseDoesNotRemoveConcurrentPermanentAdmission(t *testing.T) {
	_, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peerID, err := libp2ppeer.IDFromPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	fence, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	fence.ReconcileAdmittedPeers(nil, nil)
	temporary, err := fence.BeginTemporaryPeerAdmission(peerID)
	if err != nil {
		t.Fatal(err)
	}
	if err := fence.AdmitPeer(peerID); err != nil {
		t.Fatal(err)
	}
	temporary.Close()
	if fence.IsPeerFenced(peerID) || !fence.IsPeerConnectionAllowed(peerID) {
		t.Fatal("temporary release removed a concurrent permanent admission")
	}
}

func TestRouteFenceRejectsPostFenceInboundProtocolStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	bRegistry := newTestRegistry(t, bHost)
	b := bRegistry.Link()
	a := newTestLink(t, aHost)
	ensureTestRoute(t, ctx, a, b)

	const protocolID = transports.ProtocolID("/swarmion/fence-inbound/1")
	called := make(chan struct{}, 1)
	registration, err := b.RegisterProtocols(ctx, transports.ProtocolBundle{Handlers: []transports.ProtocolHandler{{
		ID: protocolID,
		Handler: func(context.Context, transports.Stream) {
			called <- struct{}{}
		},
	}}})
	if err != nil {
		t.Fatalf("register protocol: %v", err)
	}
	defer registration.Close(ctx)
	if _, err := bRegistry.RouteFence().FencePeer(aHost.ID()); err != nil {
		t.Fatalf("fence peer: %v", err)
	}
	raw, err := aHost.NewStream(ctx, bHost.ID(), protocol.ID(protocolID))
	if err == nil {
		_ = raw.SetDeadline(time.Now().Add(time.Second))
		_, err = raw.Write([]byte("heartbeat"))
		if err == nil {
			var response [1]byte
			_, err = raw.Read(response[:])
		}
		_ = raw.Close()
	}
	if err == nil {
		t.Fatal("post-fence inbound protocol stream unexpectedly succeeded")
	}
	select {
	case <-called:
		t.Fatal("post-fence protocol handler was invoked")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestRouteFenceProcessRestartAllocatesFreshOpaqueGeneration(t *testing.T) {
	_, publicKey, err := libp2pcrypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	peerID, err := libp2ppeer.IDFromPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	beforeRestart, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	afterRestart, err := NewRouteFence()
	if err != nil {
		t.Fatal(err)
	}
	first, err := beforeRestart.FencePeer(peerID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := afterRestart.FencePeer(peerID)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("fresh process reused opaque route-fence generation %q", first)
	}
}

func TestApplicationFirstCollisionIsAtomicAndDoesNotOverwrite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	a := newTestLink(t, aHost)
	registry := newTestRegistry(t, bHost)
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	const (
		colliding protocol.ID = "/protos/application-first/1"
		fresh                 = transports.ProtocolID("/swarmion/must-not-install/1")
	)
	application, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{applicationResponseHandler(colliding, "application")},
	})
	if err != nil {
		t.Fatalf("register application protocol: %v", err)
	}
	defer application.Close(ctx)

	failed, err := b.RegisterProtocols(ctx, transports.ProtocolBundle{Handlers: []transports.ProtocolHandler{
		{ID: fresh, Handler: discardTransportHandler},
		{ID: transports.ProtocolID(colliding), Handler: discardTransportHandler},
	}})
	if failed != nil || !errors.Is(err, transports.ErrProtocolInUse) {
		t.Fatalf("Swarmion collision = (%v, %v), want nil, ErrProtocolInUse", failed, err)
	}
	if _, err := a.OpenStream(ctx, b.LocalPeer(), fresh); !errors.Is(err, transports.ErrUnsupportedProtocol) {
		t.Fatalf("fresh bundle prefix survived collision: %v", err)
	}
	if got := readApplicationProtocol(t, ctx, aHost, bHost.ID(), colliding); got != "application" {
		t.Fatalf("application handler overwritten, response = %q", got)
	}
}

func TestSwarmionFirstCollisionIsAtomicAndDoesNotOverwrite(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	a := newTestLink(t, aHost)
	registry := newTestRegistry(t, bHost)
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	const (
		colliding = transports.ProtocolID("/swarmion/swarmion-first/1")
		fresh     = protocol.ID("/protos/must-not-install/1")
	)
	swarmion, err := b.RegisterProtocols(ctx, transports.ProtocolBundle{Handlers: []transports.ProtocolHandler{{
		ID: colliding,
		Handler: func(_ context.Context, stream transports.Stream) {
			_, _ = stream.Write([]byte("swarmion"))
			_ = stream.CloseWrite()
		},
	}}})
	if err != nil {
		t.Fatalf("register Swarmion protocol: %v", err)
	}
	defer swarmion.Close(ctx)

	failed, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{
			applicationResponseHandler(fresh, "partial"),
			applicationResponseHandler(protocol.ID(colliding), "overwrite"),
		},
	})
	if failed != nil || !errors.Is(err, transports.ErrProtocolInUse) {
		t.Fatalf("application collision = (%v, %v), want nil, ErrProtocolInUse", failed, err)
	}
	if _, err := aHost.NewStream(ctx, bHost.ID(), fresh); err == nil {
		t.Fatal("fresh application bundle prefix survived collision")
	}
	if got := readTransportProtocol(t, ctx, a, b.LocalPeer(), colliding); got != "swarmion" {
		t.Fatalf("Swarmion handler overwritten, response = %q", got)
	}
}

func TestApplicationRegistrationCloseDrainsAndOldGenerationCannotRemoveReplacement(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	a := newTestLink(t, aHost)
	registry := newTestRegistry(t, bHost)
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	const reusable protocol.ID = "/protos/generation-safe/1"
	first, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{applicationResponseHandler(reusable, "first")},
	})
	if err != nil {
		t.Fatalf("register first generation: %v", err)
	}
	if err := first.Close(ctx); err != nil {
		t.Fatalf("close first generation: %v", err)
	}
	second, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{applicationResponseHandler(reusable, "second")},
	})
	if err != nil {
		t.Fatalf("register second generation: %v", err)
	}
	defer second.Close(ctx)
	if err := first.Close(ctx); err != nil {
		t.Fatalf("repeat old-generation close: %v", err)
	}
	if got := readApplicationProtocol(t, ctx, aHost, bHost.ID(), reusable); got != "second" {
		t.Fatalf("old close removed replacement generation, response = %q", got)
	}

	const draining protocol.ID = "/protos/draining/1"
	entered := make(chan struct{})
	released := make(chan struct{})
	drainRegistration, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{{
			ID: draining,
			Handler: func(handlerCtx context.Context, _ libp2pnetwork.Stream) {
				close(entered)
				<-handlerCtx.Done()
				close(released)
			},
		}},
	})
	if err != nil {
		t.Fatalf("register draining handler: %v", err)
	}
	raw, err := aHost.NewStream(ctx, bHost.ID(), draining)
	if err != nil {
		t.Fatalf("open draining stream: %v", err)
	}
	defer raw.Close()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("application handler did not start")
	}
	if err := drainRegistration.Close(ctx); err != nil {
		t.Fatalf("close draining registration: %v", err)
	}
	select {
	case <-released:
	default:
		t.Fatal("registration close returned before callback drained")
	}
}

func TestApplicationListenerIsAtomicAndKeepsAcceptedStreamAlive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	a := newTestLink(t, aHost)
	registry := newTestRegistry(t, bHost)
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	const (
		listenerProtocol protocol.ID = "/protos/listener/1"
		siblingProtocol              = protocol.ID("/protos/listener-sibling/1")
	)
	listener, listenerHandler, err := registry.NewApplicationListener(
		listenerProtocol,
		func(_ context.Context, stream libp2pnetwork.Stream) error {
			if stream.Conn().RemotePeer() != aHost.ID() {
				return fmt.Errorf("unexpected authenticated peer %s", stream.Conn().RemotePeer())
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("create application listener: %v", err)
	}
	defer listener.Close()
	registration, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{
			listenerHandler,
			applicationResponseHandler(siblingProtocol, "sibling"),
		},
	})
	if err != nil {
		t.Fatalf("register application listener bundle: %v", err)
	}
	defer registration.Close(ctx)

	acceptedResult := make(chan net.Conn, 1)
	acceptError := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			acceptError <- acceptErr
			return
		}
		acceptedResult <- conn
	}()
	raw, err := aHost.NewStream(ctx, bHost.ID(), listenerProtocol)
	if err != nil {
		t.Fatalf("open application-listener stream: %v", err)
	}
	defer raw.Close()
	if err := raw.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set raw stream deadline: %v", err)
	}
	// BasicHost may negotiate a previously identified protocol lazily on the
	// first I/O, so write before waiting for the remote callback.
	if _, err := raw.Write([]byte("request")); err != nil {
		t.Fatalf("write request: %v", err)
	}

	var accepted net.Conn
	select {
	case accepted = <-acceptedResult:
	case err := <-acceptError:
		t.Fatalf("accept application-listener stream: %v", err)
	case <-ctx.Done():
		t.Fatal("application listener did not accept stream")
	}
	if _, ok := accepted.(libp2pnetwork.Stream); !ok {
		t.Fatalf("accepted connection type %T does not expose authenticated libp2p stream", accepted)
	}
	if err := accepted.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set accepted connection deadline: %v", err)
	}
	request := make([]byte, len("request"))
	if _, err := io.ReadFull(accepted, request); err != nil {
		t.Fatalf("read accepted request: %v", err)
	}

	// Listener shutdown must not end an already accepted connection or remove
	// the application bundle. Its registration remains the protocol owner.
	if err := listener.Close(); err != nil {
		t.Fatalf("close application listener: %v", err)
	}
	if _, err := accepted.Write([]byte("response")); err != nil {
		t.Fatalf("accepted connection died with listener: %v", err)
	}
	response := make([]byte, len("response"))
	if _, err := io.ReadFull(raw, response); err != nil {
		t.Fatalf("read response after listener close: %v", err)
	}
	if got := readApplicationProtocol(t, ctx, aHost, bHost.ID(), siblingProtocol); got != "sibling" {
		t.Fatalf("listener close removed sibling protocol, response = %q", got)
	}
	if err := accepted.Close(); err != nil {
		t.Fatalf("close accepted connection: %v", err)
	}
}

func TestApplicationListenerRegistrationDrainsAndIsGenerationSafe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	a := newTestLink(t, aHost)
	registry := newTestRegistry(t, bHost)
	b := registry.Link()
	ensureTestRoute(t, ctx, a, b)

	const id protocol.ID = "/protos/listener-generation/1"
	firstListener, firstHandler, err := registry.NewApplicationListener(id, nil)
	if err != nil {
		t.Fatalf("create first listener: %v", err)
	}
	defer firstListener.Close()
	first, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{firstHandler},
	})
	if err != nil {
		t.Fatalf("register first listener: %v", err)
	}

	acceptedResult := make(chan net.Conn, 1)
	go func() {
		conn, _ := firstListener.Accept()
		acceptedResult <- conn
	}()
	raw, err := aHost.NewStream(ctx, bHost.ID(), id)
	if err != nil {
		t.Fatalf("open first-generation stream: %v", err)
	}
	defer raw.Close()
	if _, err := raw.Write([]byte{1}); err != nil {
		t.Fatalf("activate first-generation stream: %v", err)
	}
	select {
	case conn := <-acceptedResult:
		if conn == nil {
			t.Fatal("first listener returned nil connection")
		}
		// Leave the accepted connection live. Registration.Close must reset it
		// and wait for the registry callback to drain.
	case <-ctx.Done():
		t.Fatal("first listener did not accept stream")
	}
	if err := first.Close(ctx); err != nil {
		t.Fatalf("close first listener registration: %v", err)
	}

	secondListener, secondHandler, err := registry.NewApplicationListener(id, nil)
	if err != nil {
		t.Fatalf("create replacement listener: %v", err)
	}
	defer secondListener.Close()
	second, err := registry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{secondHandler},
	})
	if err != nil {
		t.Fatalf("register replacement listener: %v", err)
	}
	defer second.Close(ctx)
	if err := first.Close(ctx); err != nil {
		t.Fatalf("repeat old listener-generation close: %v", err)
	}

	secondAccepted := make(chan net.Conn, 1)
	go func() {
		conn, _ := secondListener.Accept()
		secondAccepted <- conn
	}()
	replacementRaw, err := aHost.NewStream(ctx, bHost.ID(), id)
	if err != nil {
		t.Fatalf("old close removed replacement listener: %v", err)
	}
	defer replacementRaw.Close()
	if _, err := replacementRaw.Write([]byte{2}); err != nil {
		t.Fatalf("activate replacement listener: %v", err)
	}
	select {
	case conn := <-secondAccepted:
		if conn == nil {
			t.Fatal("replacement listener returned nil connection")
		}
		_ = conn.Close()
	case <-ctx.Done():
		t.Fatal("replacement listener did not accept stream")
	}
	if aHost.Network().Connectedness(bHost.ID()) == libp2pnetwork.NotConnected {
		t.Fatal("application registration close closed the shared physical connection")
	}
}

func TestSwarmionSessionAndApplicationRegistrationCoexist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	aHost := newTestHost(t)
	bHost := newTestHost(t)
	aRegistry := newTestRegistry(t, aHost)
	bRegistry := newTestRegistry(t, bHost)
	a := aRegistry.Link()
	b := bRegistry.Link()
	ensureTestRoute(t, ctx, a, b)

	const applicationProtocol protocol.ID = "/protos/coexists/1"
	application, err := bRegistry.RegisterApplicationProtocols(ctx, ApplicationProtocolBundle{
		Handlers: []ApplicationProtocolHandler{applicationResponseHandler(applicationProtocol, "alive")},
	})
	if err != nil {
		t.Fatalf("register application protocol: %v", err)
	}
	defer application.Close(ctx)

	openSession := func(link transports.Link, host libp2phost.Host) *transports.Session {
		identity := &conformanceIdentity{private: host.Peerstore().PrivKey(host.ID())}
		session, err := transports.OpenSession(ctx, transports.SessionConfig{
			Link: link, Identity: identity,
			DerivePeerID: conformancePeerID, VerifySignature: verifyConformanceSignature,
		})
		if err != nil {
			t.Fatalf("open Swarmion Session: %v", err)
		}
		return session
	}
	aSession := openSession(a, aHost)
	bSession := openSession(b, bHost)
	if got := readApplicationProtocol(t, ctx, aHost, bHost.ID(), applicationProtocol); got != "alive" {
		t.Fatalf("application response while Session active = %q", got)
	}
	if err := aSession.Close(ctx); err != nil {
		t.Fatalf("close first Swarmion Session: %v", err)
	}
	if err := bSession.Close(ctx); err != nil {
		t.Fatalf("close second Swarmion Session: %v", err)
	}
	if got := readApplicationProtocol(t, ctx, aHost, bHost.ID(), applicationProtocol); got != "alive" {
		t.Fatalf("Session close removed application handler, response = %q", got)
	}
}

func TestBorrowedLinkConformance(t *testing.T) {
	transporttest.Run(t, func(t *testing.T) transporttest.Pair {
		t.Helper()
		aHost := newTestHost(t)
		bHost := newTestHost(t)
		a := newTestLink(t, aHost)
		b := newTestLink(t, bHost)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ensureTestRoute(t, ctx, a, b)
		cancel()
		return transporttest.Pair{
			A: a,
			B: b,
			SetRoute: func(t *testing.T, reachable bool) {
				t.Helper()
				if !reachable {
					closeApplicationConnections(t, aHost, bHost.ID())
					return
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				ensureTestRoute(t, ctx, a, b)
			},
			OverflowRoutes: func(t *testing.T) {
				t.Helper()
				overflowConformanceRoutes(t, a, b.LocalPeer())
			},
			AssertApplicationAlive: func(t *testing.T) {
				t.Helper()
				if aHost.Network().Connectedness(bHost.ID()) == libp2pnetwork.NotConnected {
					t.Fatal("scoped registration close closed the application connection")
				}
			},
		}
	})
}

func TestBorrowedLinkSessionConformance(t *testing.T) {
	transporttest.RunSession(t, func(t *testing.T) transporttest.SessionPair {
		t.Helper()
		aHost := newTestHost(t)
		bHost := newTestHost(t)
		a := newTestLink(t, aHost)
		b := newTestLink(t, bHost)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ensureTestRoute(t, ctx, a, b)
		cancel()
		privateA := aHost.Peerstore().PrivKey(aHost.ID())
		privateB := bHost.Peerstore().PrivKey(bHost.ID())
		if privateA == nil || privateB == nil {
			t.Fatal("caller-owned test host did not retain its signing identity")
		}
		identityA := &conformanceIdentity{private: privateA}
		identityB := &conformanceIdentity{private: privateB}
		config := func(link transports.Link, identity *conformanceIdentity) transports.SessionConfig {
			return transports.SessionConfig{
				Link: link, Identity: identity,
				DerivePeerID: conformancePeerID, VerifySignature: verifyConformanceSignature,
				CallTimeout: 2 * time.Second, HandlerTimeout: 2 * time.Second,
			}
		}
		return transporttest.SessionPair{
			ConfigA: config(a, identityA),
			ConfigB: config(b, identityB),
			AssertApplicationAlive: func(t *testing.T) {
				t.Helper()
				if aHost.Network().Connectedness(bHost.ID()) == libp2pnetwork.NotConnected {
					t.Fatal("closing a Swarmion Session closed the application connection")
				}
			},
		}
	})
}

type conformanceIdentity struct{ private libp2pcrypto.PrivKey }

func (i *conformanceIdentity) PublicKeyBytes() []byte {
	encoded, err := libp2pcrypto.MarshalPublicKey(i.private.GetPublic())
	if err != nil {
		panic(err)
	}
	return encoded
}

func (i *conformanceIdentity) SignBytes(ctx context.Context, payload []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return i.private.Sign(payload)
}

func conformancePeerID(public []byte) (transports.PeerID, error) {
	key, err := libp2pcrypto.UnmarshalPublicKey(public)
	if err != nil {
		return "", err
	}
	peerID, err := libp2ppeer.IDFromPublicKey(key)
	return transports.PeerID(peerID.String()), err
}

func verifyConformanceSignature(public, payload, signature []byte) error {
	key, err := libp2pcrypto.UnmarshalPublicKey(public)
	if err != nil {
		return err
	}
	verified, err := key.Verify(payload, signature)
	if err != nil {
		return err
	}
	if !verified {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}

func overflowConformanceRoutes(t *testing.T, link *Link, peer transports.PeerID) {
	t.Helper()
	tracker := link.routes
	tracker.mu.Lock()
	defer tracker.mu.Unlock()
	route, ok := tracker.routes[peer]
	if !ok {
		t.Fatalf("route %s missing before overflow", peer)
	}
	for index := 0; index < observationQueueSize+2; index++ {
		if route.Kind == transports.RouteKindDirect {
			route.Kind = transports.RouteKindRelayed
		} else {
			route.Kind = transports.RouteKindDirect
		}
		tracker.routes[peer] = route
		tracker.revision++
		tracker.emitLocked(transports.RouteEvent{
			Revision: tracker.revision, Peer: peer, Reachable: true, Route: route,
		})
	}
	connections := tracker.connections[peer]
	delete(tracker.connections, peer)
	delete(tracker.routes, peer)
	tracker.revision++
	tracker.emitLocked(transports.RouteEvent{
		Revision: tracker.revision, Peer: peer, Reachable: false, Route: route,
	})
	tracker.generations[peer]++
	restored := transports.RouteState{
		Peer: peer, Generation: tracker.generations[peer], Kind: transports.RouteKindDirect,
	}
	tracker.connections[peer] = connections
	tracker.routes[peer] = restored
	tracker.revision++
	tracker.emitLocked(transports.RouteEvent{
		Revision: tracker.revision, Peer: peer, Reachable: true, Route: restored,
	})
}

func newTestHost(t *testing.T) libp2phost.Host {
	t.Helper()
	host, err := libp2pnode.New(libp2pnode.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	if err != nil {
		t.Fatalf("create caller-owned libp2p host: %v", err)
	}
	t.Cleanup(func() {
		if err := host.Close(); err != nil {
			t.Errorf("close caller-owned host: %v", err)
		}
	})
	return host
}

func newTestLink(t *testing.T, host libp2phost.Host) *Link {
	t.Helper()
	link, err := New(host)
	if err != nil {
		t.Fatalf("wrap caller-owned host: %v", err)
	}
	return link
}

func newTestRegistry(t *testing.T, host libp2phost.Host) *Registry {
	t.Helper()
	registry, err := NewRegistry(host)
	if err != nil {
		t.Fatalf("create shared protocol registry: %v", err)
	}
	return registry
}

func ensureTestRoute(t *testing.T, ctx context.Context, local, remote *Link) {
	t.Helper()
	snapshot, subscription, err := remote.SubscribeLocalEndpoints(ctx)
	if err != nil {
		t.Fatalf("snapshot remote endpoints: %v", err)
	}
	defer subscription.Close(ctx)
	if err := local.EnsureRoute(ctx, transports.RouteRequest{
		Peer:      remote.LocalPeer(),
		Endpoints: snapshot.Endpoints,
		Purpose:   "test",
	}); err != nil {
		t.Fatalf("ensure test route: %v", err)
	}
}

func closeApplicationConnections(t *testing.T, host libp2phost.Host, remote libp2ppeer.ID) {
	t.Helper()
	connections := host.Network().ConnsToPeer(remote)
	if len(connections) == 0 {
		t.Fatal("application host has no physical connection to close")
	}
	for _, connection := range connections {
		if err := connection.Close(); err != nil {
			t.Fatalf("application closes physical connection: %v", err)
		}
	}
}

func applicationResponseHandler(id protocol.ID, response string) ApplicationProtocolHandler {
	return ApplicationProtocolHandler{
		ID: id,
		Handler: func(_ context.Context, stream libp2pnetwork.Stream) {
			_, _ = stream.Write([]byte(response))
			_ = stream.CloseWrite()
		},
	}
}

func discardTransportHandler(_ context.Context, stream transports.Stream) {
	_, _ = io.Copy(io.Discard, stream)
}

func readApplicationProtocol(
	t *testing.T,
	ctx context.Context,
	host libp2phost.Host,
	remote libp2ppeer.ID,
	id protocol.ID,
) string {
	t.Helper()
	stream, err := host.NewStream(ctx, remote, id)
	if err != nil {
		t.Fatalf("open application protocol %s: %v", id, err)
	}
	payload, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read application protocol %s: %v", id, err)
	}
	_ = stream.Close()
	return string(payload)
}

func readTransportProtocol(
	t *testing.T,
	ctx context.Context,
	link *Link,
	remote transports.PeerID,
	id transports.ProtocolID,
) string {
	t.Helper()
	stream, err := link.OpenStream(ctx, remote, id)
	if err != nil {
		t.Fatalf("open transport protocol %s: %v", id, err)
	}
	payload, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read transport protocol %s: %v", id, err)
	}
	_ = stream.Close()
	return string(payload)
}
