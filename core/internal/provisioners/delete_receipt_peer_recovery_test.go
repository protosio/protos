package provisioners

import (
	"context"
	"errors"
	"testing"
	"time"

	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestInstanceDeleteFreshPeerRecoversForeignAuthorOperationFromValidatedPendingEvent(t *testing.T) {
	// Use the short scheduler only while preparing a durable checkpoint that
	// contains the delete task's stable operation identity. The author and relay
	// are reopened below with a deliberately longer, non-adaptive deadline so
	// the delete itself has an observable uncheckpointed propagation window.
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")

	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() { cfg.P2PPort = previousP2PPort })

	const databaseName = "protos_delete_foreign_receipt_recovery_test"
	authorDir := t.TempDir()
	authorKey, err := pcrypto.GetLocalKey(authorDir)
	if err != nil {
		t.Fatal(err)
	}
	authorNetwork := testswarmion.New(t, authorKey)
	authorStore, err := db.Open(authorDir, databaseName, authorKey, authorNetwork.Link)
	if err != nil {
		t.Fatal(err)
	}
	activeAuthorStore := authorStore
	t.Cleanup(func() {
		if activeAuthorStore != nil {
			_ = activeAuthorStore.Close()
		}
	})
	if err := authorStore.Init(); err != nil {
		t.Fatal(err)
	}
	authorStatus, ok := authorStore.SwarmionStatus()
	if !ok || authorStatus.PeerID == "" {
		t.Fatal("author Swarmion peer identity is unavailable")
	}

	instance := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "foreign-author-delete-recovery",
		Kind:                "receipt_test_record",
		KindID:              "foreign-author-delete-recovery-test",
		ProviderResourceID:  "foreign-author-delete-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	insertInstanceForDeleteReceiptTest(t, authorStore, &instance)

	authorManager := newLifecycleTestManager(t, authorStore, newProvisionerRegistry())
	record, err := authorManager.QueueDeleteInstanceLocal(context.Background(), instance.ID)
	if err != nil {
		t.Fatal(err)
	}
	queuedPayload := decodeDeleteRestartPayload(t, record)
	if queuedPayload.DeleteOperation == nil {
		t.Fatal("queued delete task did not replicate its operation identity")
	}
	operationIdentity := *queuedPayload.DeleteOperation
	if operationIdentity.AuthorPeerID != authorStatus.PeerID {
		t.Fatalf("delete operation author=%q, want author peer A %q", operationIdentity.AuthorPeerID, authorStatus.PeerID)
	}
	if queuedPayload.OperationStateModel != instanceDeleteOperationFactsV1 {
		t.Fatalf("queued operation state model=%q, want %q", queuedPayload.OperationStateModel, instanceDeleteOperationFactsV1)
	}
	if queuedPayload.CheckpointAuthorPeerID != "" {
		t.Fatalf("queued immutable-fact delete unexpectedly retained mutable checkpoint owner %q", queuedPayload.CheckpointAuthorPeerID)
	}

	// A later same-author event can only be checkpointed after the task-create
	// event in its transition chain. Waiting for this barrier therefore makes
	// the replicated task/operation identity available to a bootstrapping relay
	// without checkpointing the delete operation under test.
	barrier := InstanceInfo{
		ID:                  db.MustNewUUIDv7(),
		Name:                "foreign-author-delete-recovery-barrier",
		Kind:                KindLocalVM,
		KindID:              "foreign-author-delete-recovery-barrier-test",
		ProviderResourceID:  "foreign-author-delete-recovery-barrier-provider-id",
		DesiredStatus:       ServerStateRunning,
		ReplicationPriority: 1,
		Location:            "test",
	}
	insertInstanceForDeleteReceiptTest(t, authorStore, &barrier)

	if err := authorStore.Close(); err != nil {
		t.Fatalf("close author after preparing task checkpoint: %v", err)
	}
	activeAuthorStore = nil

	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "")
	t.Setenv("SWARMION_CONTINUOUS_EVENT_DEADLINE", "4s")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_PERMIT_GAP", "10ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_GAP", "10ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_SLOTS", "1")
	t.Setenv("SWARMION_STABLE_PREFIX_EVIDENCE_TARGET", "16")
	t.Setenv("SWARMION_STABLE_PREFIX_PLAN_MATCH_TARGET", "8")
	t.Setenv("SWARMION_STABLE_PREFIX_ADAPTIVE_TARGETS", "false")

	authorStore, err = db.Open(authorDir, databaseName, authorKey, authorNetwork.Link)
	if err != nil {
		t.Fatalf("reopen author with pending-window scheduler: %v", err)
	}
	activeAuthorStore = authorStore
	reopenedAuthorStatus, ok := authorStore.SwarmionStatus()
	if !ok || reopenedAuthorStatus.PeerID != authorStatus.PeerID {
		t.Fatalf("reopened author identity=%q, want %q", reopenedAuthorStatus.PeerID, authorStatus.PeerID)
	}
	authorAddrInfo := libp2ppeer.AddrInfo{ID: authorNetwork.Host.ID(), Addrs: authorNetwork.Host.Addrs()}
	authorMultiaddrs, err := libp2ppeer.AddrInfoToP2pAddrs(&authorAddrInfo)
	if err != nil {
		t.Fatalf("build author bootstrap addresses: %v", err)
	}
	authorBootstrapAddrs := make([]string, 0, len(authorMultiaddrs))
	for _, addr := range authorMultiaddrs {
		authorBootstrapAddrs = append(authorBootstrapAddrs, addr.String())
	}
	if len(authorBootstrapAddrs) == 0 {
		t.Fatal("reopened author Swarmion peer has no bootstrap addresses")
	}

	relayDir := t.TempDir()
	relayKey, err := pcrypto.GetLocalKey(relayDir)
	if err != nil {
		t.Fatal(err)
	}
	relayNetwork := testswarmion.New(t, relayKey)
	testswarmion.Connect(t, relayNetwork, authorNetwork)
	relayStore, err := db.Open(relayDir, databaseName, relayKey, relayNetwork.Link)
	if err != nil {
		t.Fatal(err)
	}
	activeRelayStore := relayStore
	t.Cleanup(func() {
		if activeRelayStore != nil {
			_ = activeRelayStore.Close()
		}
	})
	bootstrapCtx, cancelBootstrap := context.WithTimeout(context.Background(), 30*time.Second)
	if err := relayStore.InitFromPeerContext(bootstrapCtx, authorStatus.PeerID, authorBootstrapAddrs); err != nil {
		cancelBootstrap()
		t.Fatalf("bootstrap relay peer C from author peer A: %v", err)
	}
	cancelBootstrap()
	relayStatus, ok := relayStore.SwarmionStatus()
	if !ok || relayStatus.PeerID == "" || relayStatus.PeerID == authorStatus.PeerID {
		t.Fatalf("relay did not use a fresh peer identity: author=%q relay=%q", authorStatus.PeerID, relayStatus.PeerID)
	}
	relayTaskManager := tasks.NewManager(relayStore)
	replicatedOnRelay, err := relayTaskManager.Get(record.ID)
	if err != nil {
		t.Fatalf("read replicated delete task on relay peer C: %v", err)
	}
	relayPayload := decodeDeleteRestartPayload(t, replicatedOnRelay)
	if relayPayload.DeleteOperation == nil || *relayPayload.DeleteOperation != operationIdentity || relayPayload.DeleteReceipt != nil ||
		relayPayload.OperationStateModel != instanceDeleteOperationFactsV1 || relayPayload.CheckpointAuthorPeerID != "" {
		t.Fatalf("relay task state=%+v, want exact operation identity and no receipt", relayPayload)
	}

	const simulatedAuthorCrash = "author peer exited before delete receipt fact"
	var accepted db.PublishedWriteReceipt
	var relayPending swarmionapp.BranchEventReceiptStatus
	authorManager = newLifecycleTestManager(t, authorStore, newProvisionerRegistry())
	authorManager.afterInstanceDeletePublished = func(receipt db.PublishedWriteReceipt) {
		accepted = receipt
		relayPending = waitForForeignDeletePendingEvent(t, relayStore, operationIdentity, receipt)
		panic(simulatedAuthorCrash)
	}
	var recoveredPanic any
	func() {
		defer func() { recoveredPanic = recover() }()
		_ = authorManager.tasks.RunPending(context.Background())
	}()
	if recoveredPanic != simulatedAuthorCrash {
		t.Fatalf("author delete interruption=%v, want simulated post-publication crash", recoveredPanic)
	}
	if accepted.EventID == "" || accepted.PublishedRootHash == "" || accepted.EventDigest == "" {
		t.Fatalf("author crash lost accepted exact receipt: %+v", accepted)
	}
	if accepted.AuthorPeerID != authorStatus.PeerID || accepted.OperationIntentDigest != operationIdentity.IntentDigest {
		t.Fatalf("accepted receipt lost operation provenance: receipt=%+v identity=%+v", accepted, operationIdentity)
	}
	interrupted, err := authorManager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	interruptedPayload := decodeDeleteRestartPayload(t, interrupted)
	if interruptedPayload.DeleteOperation == nil || *interruptedPayload.DeleteOperation != operationIdentity {
		t.Fatalf("interrupted task lost replicated operation identity: %+v", interruptedPayload.DeleteOperation)
	}
	if interruptedPayload.DeleteReceipt != nil {
		t.Fatalf("author crash unexpectedly checkpointed the returned receipt: %+v", interruptedPayload.DeleteReceipt)
	}
	if interruptedPayload.CheckpointAuthorPeerID != "" {
		t.Fatalf("author crash introduced mutable checkpoint owner %q", interruptedPayload.CheckpointAuthorPeerID)
	}
	if !relayPending.Known || relayPending.Checkpointed || relayPending.AppliedDurably {
		t.Fatalf("relay did not retain the delete as an exact uncheckpointed pending event: %+v", relayPending)
	}

	// A is now genuinely unavailable. C must retain and resolve A's validated
	// pending event; there is deliberately no delete checkpoint to recover from.
	if err := authorStore.Close(); err != nil {
		t.Fatalf("close crashed author peer A: %v", err)
	}
	activeAuthorStore = nil
	relayStatusAfterAuthorClose, err := relayStore.SwarmionEventReceiptStatus(
		context.Background(),
		accepted.EventID,
		accepted.PublishedRootHash,
	)
	if err != nil {
		t.Fatalf("inspect relay pending event after author stopped: %v", err)
	}
	if !relayStatusAfterAuthorClose.Known || relayStatusAfterAuthorClose.Checkpointed || relayStatusAfterAuthorClose.AppliedDurably {
		t.Fatalf("relay pending event changed before recovery lookup: %+v", relayStatusAfterAuthorClose)
	}
	relayResolved, err := relayStore.LookupPublishedWriteOperation(
		context.Background(),
		operationIdentity.publishedWriteOperation(),
	)
	if err != nil {
		t.Fatalf("relay peer C resolve A-authored delete after A stopped: %v", err)
	}
	assertForeignDeleteOperationReceipt(t, relayResolved, operationIdentity, accepted)
	repeatedResolved, err := relayStore.LookupPublishedWriteOperation(
		context.Background(),
		operationIdentity.publishedWriteOperation(),
	)
	if err != nil {
		t.Fatalf("repeat relay resolution of A-authored pending delete: %v", err)
	}
	if repeatedResolved != relayResolved {
		t.Fatalf("repeated pending receipt changed: first=%+v repeated=%+v", relayResolved, repeatedResolved)
	}

	unknownKey := operationIdentity.publishedWriteOperation()
	unknownKey.Key += "-unknown"
	assertForeignDeleteOperationUnavailable(t, relayStore, unknownKey, "unknown operation key for known foreign author")
	wrongAuthor := operationIdentity.publishedWriteOperation()
	wrongAuthor.AuthorPeerID = "unknown-foreign-author"
	assertForeignDeleteOperationUnavailable(t, relayStore, wrongAuthor, "known operation key for unknown foreign author")

	// The same operation identity scoped to C is authoritatively absent before
	// recovery. It must remain absent afterwards; finding it would prove that C
	// re-executed and published a duplicate delete instead of resuming A's event.
	relayAuthoredOperation := operationIdentity.publishedWriteOperation()
	relayAuthoredOperation.AuthorPeerID = relayStatus.PeerID
	locallyResolvedBefore, err := relayStore.LookupPublishedWriteOperation(context.Background(), relayAuthoredOperation)
	if err != nil {
		t.Fatalf("resolve relay-authored operation before recovery: %v", err)
	}
	if locallyResolvedBefore.Resolution != swarmionapp.BranchOperationReceiptAbsent {
		t.Fatalf("relay-authored operation before recovery=%+v, want authoritative absent", locallyResolvedBefore)
	}

	probeManager := newLifecycleTestManager(t, relayStore, newProvisionerRegistry())

	// Prove that receipt/status recovery itself can start immediately and does
	// not need to write task bookkeeping. A short context intentionally stops at
	// pending; every callback must still carry A's exact receipt, and the backend
	// transaction counters must remain unchanged.
	directMetricsBefore := relayStore.TransactionMetrics()
	directCtx, cancelDirect := context.WithTimeout(context.Background(), 300*time.Millisecond)
	var directlyRecovered []instanceDeleteOperationReceipt
	directErr := probeManager.deleteInstanceImperative(
		directCtx,
		func(int, string, any) error { return nil },
		instance.ID,
		true,
		record.ID,
		operationIdentity,
		nil,
		nil,
		func(receipt instanceDeleteOperationReceipt, _ int, _ string) error {
			directlyRecovered = append(directlyRecovered, receipt)
			return nil
		},
	)
	cancelDirect()
	if !errors.Is(directErr, ErrInstanceLifecycleOwnerConflict) {
		t.Fatalf("direct foreign receipt recovery error=%v, want lifecycle-owner conflict", directErr)
	}
	if len(directlyRecovered) != 0 {
		t.Fatalf("foreign executor exposed %d recoverable receipts, want none", len(directlyRecovered))
	}
	directMetricsAfter := relayStore.TransactionMetrics()
	if directMetricsAfter != directMetricsBefore {
		t.Fatalf("read-only direct receipt recovery opened a write transaction: before=%+v after=%+v", directMetricsBefore, directMetricsAfter)
	}
	if record.OwnerPeerID != authorStatus.PeerID {
		t.Fatalf("replicated delete task owner=%q, want immutable author %q", record.OwnerPeerID, authorStatus.PeerID)
	}
	if err := probeManager.tasks.RunPending(context.Background()); err != nil {
		t.Fatalf("foreign executor inspected owner-scoped queue: %v", err)
	}
	stillOwned, err := probeManager.tasks.Get(record.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillOwned.OwnerPeerID != authorStatus.PeerID || (stillOwned.Status != tasks.StatusPending && stillOwned.Status != tasks.StatusRunning) {
		t.Fatalf("lost-owner task changed under foreign executor: %+v", stillOwned)
	}
	// No takeover is inferred from elapsed time, receipt visibility, or owner
	// disappearance. The original identity may resume on the same PeerID; until
	// then the operation remains explicitly unavailable.
}

func waitForForeignDeletePendingEvent(
	t *testing.T,
	relay *db.DB,
	identity instanceDeleteOperationIdentity,
	want db.PublishedWriteReceipt,
) swarmionapp.BranchEventReceiptStatus {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var (
		lastStatus  swarmionapp.BranchEventReceiptStatus
		lastReceipt swarmionapp.BranchOperationReceipt
		lastErr     error
	)
	for {
		lastStatus, lastErr = relay.SwarmionEventReceiptStatus(
			context.Background(),
			want.EventID,
			want.PublishedRootHash,
		)
		if lastErr == nil && lastStatus.Known {
			if lastStatus.Checkpointed || lastStatus.AppliedDurably {
				t.Fatalf("foreign event checkpointed before pending recovery window: %+v", lastStatus)
			}
			lastReceipt, lastErr = relay.LookupPublishedWriteOperation(
				context.Background(),
				identity.publishedWriteOperation(),
			)
			if lastErr == nil && lastReceipt.Resolution == swarmionapp.BranchOperationReceiptFound {
				assertForeignDeleteOperationReceipt(t, lastReceipt, identity, want)
				return lastStatus
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf(
				"relay did not learn exact foreign pending receipt before checkpoint: status=%+v receipt=%+v error=%v",
				lastStatus,
				lastReceipt,
				lastErr,
			)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertForeignDeleteOperationUnavailable(
	t *testing.T,
	store *db.DB,
	operation db.PublishedWriteOperation,
	reason string,
) {
	t.Helper()
	resolved, err := store.LookupPublishedWriteOperation(context.Background(), operation)
	if err != nil {
		t.Fatalf("%s lookup failed: %v", reason, err)
	}
	if resolved.Resolution != swarmionapp.BranchOperationReceiptUnavailable {
		t.Fatalf("%s resolution=%q, want unavailable (never authoritative absent): %+v", reason, resolved.Resolution, resolved)
	}
}

func assertForeignDeleteOperationReceipt(
	t *testing.T,
	resolved swarmionapp.BranchOperationReceipt,
	identity instanceDeleteOperationIdentity,
	want db.PublishedWriteReceipt,
) {
	t.Helper()
	if resolved.Resolution != swarmionapp.BranchOperationReceiptFound ||
		resolved.EventID != want.EventID ||
		resolved.PublishedRootHash != want.PublishedRootHash ||
		resolved.EventDigest != want.EventDigest ||
		resolved.AuthorPeerID != identity.AuthorPeerID ||
		resolved.IntentDigest != identity.IntentDigest {
		t.Fatalf("foreign operation receipt=%+v, want exact accepted receipt %+v with identity %+v", resolved, want, identity)
	}
	if resolved.AuthorSeq == 0 {
		t.Fatalf("foreign operation receipt has no signed author sequence: %+v", resolved)
	}
	if resolved.EvidenceSource != swarmionapp.OperationReceiptEvidenceSignedEvent &&
		resolved.EvidenceSource != swarmionapp.OperationReceiptEvidenceCheckpointMetadata {
		t.Fatalf("foreign operation receipt has invalid evidence source %q: %+v", resolved.EvidenceSource, resolved)
	}
	if resolved.SafeToPublish || resolved.UnavailableReason != "" {
		t.Fatalf("found foreign operation receipt exposes unavailable/publication guidance: %+v", resolved)
	}
	got, err := db.PublishedWriteReceiptFromOperation(resolved)
	if err != nil {
		t.Fatalf("convert foreign operation receipt: %v", err)
	}
	if got.EventID != want.EventID ||
		got.PublishedRootHash != want.PublishedRootHash ||
		got.EventDigest != want.EventDigest ||
		got.AuthorPeerID != identity.AuthorPeerID ||
		got.OperationIntentDigest != identity.IntentDigest {
		t.Fatalf("foreign operation receipt=%+v, want exact accepted receipt %+v with identity %+v", got, want, identity)
	}
}
