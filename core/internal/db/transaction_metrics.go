package db

import "sync/atomic"

// TransactionMetricsSnapshot is a fixed-cardinality snapshot of the backend's
// Swarmion-backed SQL transaction boundary. Counters are process-local
// diagnostics only; they are not recovery state.
type TransactionMetricsSnapshot struct {
	TransactionsStarted                  uint64
	CommitsAttempted                     uint64
	CommitsSucceeded                     uint64
	NoopCommitOutcomes                   uint64
	TypedConflicts                       uint64
	OperationReceiptsFoundAfterCommitErr uint64
	UncertainEventReceiptsAfterCommitErr uint64
	StaleWriteContextOutcomes            uint64
	ProjectionTooWideOutcomes            uint64
	OperationTransactionsAttempted       uint64
	OperationTransactionsExecuted        uint64
	OperationTransactionsAlreadyAccepted uint64
	OperationTransactionsNoChange        uint64
	OperationTransactionsFailed          uint64
	OperationWorkspaceDirtyOutcomes      uint64
	// OperationTransactionLifecycleOpaqueFailures counts malformed or
	// unclassified results at the public Execute/PublicationOutcome boundary.
	// Such results never grant replay authority.
	OperationTransactionLifecycleOpaqueFailures uint64
}

type transactionMetrics struct {
	transactionsStarted                         atomic.Uint64
	commitsAttempted                            atomic.Uint64
	commitsSucceeded                            atomic.Uint64
	noopCommitOutcomes                          atomic.Uint64
	typedConflicts                              atomic.Uint64
	operationReceiptsFoundAfterCommitErr        atomic.Uint64
	uncertainEventReceiptsAfterCommitErr        atomic.Uint64
	staleWriteContextOutcomes                   atomic.Uint64
	projectionTooWideOutcomes                   atomic.Uint64
	operationTransactionsAttempted              atomic.Uint64
	operationTransactionsExecuted               atomic.Uint64
	operationTransactionsAlreadyAccepted        atomic.Uint64
	operationTransactionsNoChange               atomic.Uint64
	operationTransactionsFailed                 atomic.Uint64
	operationWorkspaceDirtyOutcomes             atomic.Uint64
	operationTransactionLifecycleOpaqueFailures atomic.Uint64
}

func (m *transactionMetrics) snapshot() TransactionMetricsSnapshot {
	if m == nil {
		return TransactionMetricsSnapshot{}
	}
	return TransactionMetricsSnapshot{
		TransactionsStarted:                         m.transactionsStarted.Load(),
		CommitsAttempted:                            m.commitsAttempted.Load(),
		CommitsSucceeded:                            m.commitsSucceeded.Load(),
		NoopCommitOutcomes:                          m.noopCommitOutcomes.Load(),
		TypedConflicts:                              m.typedConflicts.Load(),
		OperationReceiptsFoundAfterCommitErr:        m.operationReceiptsFoundAfterCommitErr.Load(),
		UncertainEventReceiptsAfterCommitErr:        m.uncertainEventReceiptsAfterCommitErr.Load(),
		StaleWriteContextOutcomes:                   m.staleWriteContextOutcomes.Load(),
		ProjectionTooWideOutcomes:                   m.projectionTooWideOutcomes.Load(),
		OperationTransactionsAttempted:              m.operationTransactionsAttempted.Load(),
		OperationTransactionsExecuted:               m.operationTransactionsExecuted.Load(),
		OperationTransactionsAlreadyAccepted:        m.operationTransactionsAlreadyAccepted.Load(),
		OperationTransactionsNoChange:               m.operationTransactionsNoChange.Load(),
		OperationTransactionsFailed:                 m.operationTransactionsFailed.Load(),
		OperationWorkspaceDirtyOutcomes:             m.operationWorkspaceDirtyOutcomes.Load(),
		OperationTransactionLifecycleOpaqueFailures: m.operationTransactionLifecycleOpaqueFailures.Load(),
	}
}

// TransactionMetrics returns a race-safe point-in-time counter snapshot.
func (db *DB) TransactionMetrics() TransactionMetricsSnapshot {
	if db == nil {
		return TransactionMetricsSnapshot{}
	}
	return db.transactionMetrics.snapshot()
}
