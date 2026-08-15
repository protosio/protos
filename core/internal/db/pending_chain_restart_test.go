package db

import (
	"context"
	cryptorand "crypto/rand"
	"testing"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/testswarmion"
)

func TestPendingWriteChainIsImmediatelyReadyAfterRestart(t *testing.T) {
	cfg := config.Get()
	previousP2PPort := cfg.P2PPort
	cfg.P2PPort = 0
	t.Cleanup(func() { cfg.P2PPort = previousP2PPort })

	privateKey, publicKey, err := libp2pcrypto.GenerateEd25519Key(cryptorand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer := testSwarmionRawSigner{privateKey: privateKey, publicKey: publicKey}
	link := testswarmion.NewBorrowedLink(t, signer)
	workDir := t.TempDir()
	const databaseName = "protos_pending_chain_restart_test"
	var store *DB
	t.Cleanup(func() {
		if store != nil {
			_ = store.Close()
		}
	})
	open := func() *DB {
		t.Helper()
		next, openErr := Open(workDir, databaseName, signer, link)
		if openErr != nil {
			t.Fatalf("open pending-chain database: %v", openErr)
		}
		if initErr := next.Init(); initErr != nil {
			_ = next.Close()
			t.Fatalf("initialize pending-chain database: %v", initErr)
		}
		store = next
		return next
	}

	// Establish the schema and migration checkpoint promptly, then reopen with
	// a deliberately long checkpoint deadline so B and C remain pending across
	// the restart without relying on a narrow scheduler race.
	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "single_node_development")
	store = open()
	if err := store.Close(); err != nil {
		t.Fatalf("close initialized pending-chain database: %v", err)
	}
	store = nil

	t.Setenv("SWARMION_CHECKPOINT_SCHEDULER_PROFILE", "")
	t.Setenv("SWARMION_CONTINUOUS_EVENT_DEADLINE", "5m")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_PERMIT_GAP", "15s")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_GAP", "2ms")
	t.Setenv("SWARMION_CONTINUOUS_DEADLINE_STAGGER_SLOTS", "1")
	t.Setenv("SWARMION_STABLE_PREFIX_ADAPTIVE_TARGETS", "false")
	t.Setenv("SWARMION_STABLE_PREFIX_EVIDENCE_TARGET", "2")
	t.Setenv("SWARMION_STABLE_PREFIX_PLAN_MATCH_TARGET", "2")
	store = open()

	initial := testRuntimeSnapshot(t, store)
	if initial == nil || initial.CheckpointCommitID.IsZero() || initial.CheckpointProllyRootHash.IsZero() {
		t.Fatalf("initial checkpoint boundary is incomplete: %+v", initial)
	}
	checkpointRoot := initial.CheckpointProllyRootHash
	primaryID := MustNewUUIDv7()
	bSideID := MustNewUUIDv7()
	cSideID := MustNewUUIDv7()
	dSideID := MustNewUUIDv7()

	bReceipt, err := InsertWithReceiptContext(
		context.Background(),
		store,
		transactionTestUserInsert(primaryID, "pending-restart-b"),
		transactionTestUserInsert(bSideID, "pending-restart-b-side"),
	)
	if err != nil {
		t.Fatalf("publish pending B: %v", err)
	}
	cReceipt, err := UpdateAndInsertWithReceiptContext(
		context.Background(),
		store,
		[]UpdateMapper{transactionTestUserUpdate(primaryID, "pending-restart-c")},
		[]InsertMapper{transactionTestUserInsert(cSideID, "pending-restart-c-side")},
	)
	if err != nil {
		t.Fatalf("publish pending C: %v", err)
	}

	bID := swarmionprotocol.ParseEventID(bReceipt.EventID)
	cID := swarmionprotocol.ParseEventID(cReceipt.EventID)
	bRoot := swarmionprotocol.ParseRootHash(bReceipt.PublishedRootHash)
	cRoot := swarmionprotocol.ParseRootHash(cReceipt.PublishedRootHash)
	beforeRestart := testRuntimeSnapshot(t, store)
	requirePendingRestartEvent(t, beforeRestart, bID, bRoot, checkpointRoot, swarmionprotocol.EventID{})
	requirePendingRestartEvent(t, beforeRestart, cID, cRoot, bRoot, bID)
	if beforeRestart.CheckpointProllyRootHash != checkpointRoot || beforeRestart.LocalWriteBaseRootHash != cRoot {
		t.Fatalf("pre-restart roots checkpoint=%s write_base=%s, want %s/%s", beforeRestart.CheckpointProllyRootHash, beforeRestart.LocalWriteBaseRootHash, checkpointRoot, cRoot)
	}
	if len(beforeRestart.HeadEventIDs) != 1 {
		t.Fatalf("pre-restart pending heads=%v, want only C=%s", swarmionprotocol.EventIDSetToSortedSlice(beforeRestart.HeadEventIDs), cID)
	}
	if _, ok := beforeRestart.HeadEventIDs[cID]; !ok {
		t.Fatalf("pre-restart pending head=%v, want C=%s", swarmionprotocol.EventIDSetToSortedSlice(beforeRestart.HeadEventIDs), cID)
	}
	sequenceBeforeRestart := beforeRestart.LocalAuthorSeq

	if err := store.Close(); err != nil {
		t.Fatalf("close database with B/C pending: %v", err)
	}
	store = nil
	restartStarted := time.Now()
	store = open()
	restartElapsed := time.Since(restartStarted)

	// No wait or application retry is allowed before these checks. Open/Init is
	// the local readiness boundary promised by Swarmion.
	afterRestart := testRuntimeSnapshot(t, store)
	if afterRestart == nil {
		t.Fatal("pending-chain snapshot is nil immediately after restart")
	}
	if afterRestart.CheckpointProllyRootHash != checkpointRoot || afterRestart.LocalWriteBaseRootHash != cRoot {
		t.Fatalf("post-restart roots checkpoint=%s write_base=%s, want %s/%s", afterRestart.CheckpointProllyRootHash, afterRestart.LocalWriteBaseRootHash, checkpointRoot, cRoot)
	}
	if afterRestart.LocalAuthorSeq != sequenceBeforeRestart {
		t.Fatalf("restart changed author sequence=%d, want %d", afterRestart.LocalAuthorSeq, sequenceBeforeRestart)
	}
	requirePendingRestartEvent(t, afterRestart, bID, bRoot, checkpointRoot, swarmionprotocol.EventID{})
	requirePendingRestartEvent(t, afterRestart, cID, cRoot, bRoot, bID)
	if len(afterRestart.HeadEventIDs) != 1 {
		t.Fatalf("post-restart pending heads=%v, want only C=%s", swarmionprotocol.EventIDSetToSortedSlice(afterRestart.HeadEventIDs), cID)
	}
	if _, ok := afterRestart.HeadEventIDs[cID]; !ok {
		t.Fatalf("post-restart pending head=%v, want C=%s", swarmionprotocol.EventIDSetToSortedSlice(afterRestart.HeadEventIDs), cID)
	}

	rows, err := store.GetSqlDB().QueryContext(
		context.Background(),
		"SELECT name FROM users WHERE id = ?",
		MustUUIDBytes(primaryID),
	)
	if err != nil {
		t.Fatalf("immediate direct SQL read after restart: %v", err)
	}
	defer rows.Close()
	var visibleName string
	if !rows.Next() {
		t.Fatal("immediate direct SQL read did not return pending C")
	}
	if err := rows.Scan(&visibleName); err != nil {
		t.Fatalf("scan immediate pending C: %v", err)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate immediate pending C rows: %v", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("close immediate pending C rows: %v", err)
	}
	if visibleName != "pending-restart-c" {
		t.Fatalf("immediate post-restart value=%q, want pending C", visibleName)
	}

	beforeD := store.TransactionMetrics()
	dStarted := time.Now()
	dReceipt, err := UpdateAndInsertWithReceiptContext(
		context.Background(),
		store,
		[]UpdateMapper{transactionTestUserUpdate(primaryID, "pending-restart-d")},
		[]InsertMapper{transactionTestUserInsert(dSideID, "pending-restart-d-side")},
	)
	dElapsed := time.Since(dStarted)
	if err != nil {
		t.Fatalf("first post-restart write D: %v", err)
	}
	dDelta := transactionMetricsDelta(store.TransactionMetrics(), beforeD)
	if dDelta.TransactionsStarted != 1 || dDelta.CommitsAttempted != 1 || dDelta.CommitsSucceeded != 1 ||
		dDelta.TypedConflicts != 0 {
		t.Fatalf("first post-restart D transaction metrics=%+v", dDelta)
	}
	dID := swarmionprotocol.ParseEventID(dReceipt.EventID)
	dRoot := swarmionprotocol.ParseRootHash(dReceipt.PublishedRootHash)
	afterD := testRuntimeSnapshot(t, store)
	dEvent := requirePendingRestartEvent(t, afterD, dID, dRoot, cRoot, cID)
	if dEvent.AuthorSeq != sequenceBeforeRestart+1 || afterD.LocalAuthorSeq != sequenceBeforeRestart+1 {
		t.Fatalf("D author sequence event=%d local=%d, want %d", dEvent.AuthorSeq, afterD.LocalAuthorSeq, sequenceBeforeRestart+1)
	}
	if transactionTestUserName(t, store, primaryID) != "pending-restart-d" ||
		transactionTestUserCount(t, store, bSideID) != 1 ||
		transactionTestUserCount(t, store, cSideID) != 1 ||
		transactionTestUserCount(t, store, dSideID) != 1 {
		t.Fatal("post-restart D did not preserve the complete B -> C -> D content chain")
	}
	t.Logf("pending-chain restart readiness open=%s first_write=%s rollbacks=0/1 sql_view_not_ready=0", restartElapsed, dElapsed)
}

func testRuntimeSnapshot(t testing.TB, store *DB) *swarmionprotocol.NodeState {
	t.Helper()
	snapshot, ok := store.RuntimeSnapshot()
	if !ok || snapshot == nil {
		t.Fatal("Swarmion runtime snapshot is unavailable")
	}
	return snapshot
}

func requirePendingRestartEvent(
	t *testing.T,
	snapshot *swarmionprotocol.NodeState,
	eventID swarmionprotocol.EventID,
	root swarmionprotocol.RootHash,
	transitionBase swarmionprotocol.RootHash,
	parent swarmionprotocol.EventID,
) *swarmionprotocol.Event {
	t.Helper()
	if snapshot == nil {
		t.Fatal("pending-chain snapshot is nil")
	}
	event := snapshot.Clock[eventID]
	if event == nil {
		t.Fatalf("pending event %s is absent from clock", eventID)
	}
	if event.ProllyRootHash != root || event.TransitionBaseRootHash != transitionBase {
		t.Fatalf("event %s roots result=%s base=%s, want %s/%s", eventID, event.ProllyRootHash, event.TransitionBaseRootHash, root, transitionBase)
	}
	parents := swarmionprotocol.EventIDSetToSortedSlice(event.ParentEventIDs)
	if parent.IsZero() {
		if len(parents) != 0 {
			t.Fatalf("event %s parents=%v, want none", eventID, parents)
		}
	} else if len(parents) != 1 || parents[0] != parent {
		t.Fatalf("event %s parents=%v, want sole parent %s", eventID, parents, parent)
	}
	return event
}
