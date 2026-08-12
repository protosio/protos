package db

import "sync/atomic"

// TransactionMetricsSnapshot is a fixed-cardinality snapshot of the backend's
// Swarmion-backed SQL transaction boundary. Counters are process-local
// diagnostics only; they are not recovery state.
type TransactionMetricsSnapshot struct {
	TransactionsStarted                  uint64
	CommitsAttempted                     uint64
	CommitsSucceeded                     uint64
	CommitsFailed                        uint64
	NoopCommitOutcomes                   uint64
	RollbacksAttempted                   uint64
	RollbacksSucceeded                   uint64
	RollbacksFailed                      uint64
	RollbacksApplyPhase                  uint64
	RollbacksBeforeCommitPhase           uint64
	RollbacksPanicPhase                  uint64
	RollbacksApplyFailure                uint64
	RollbacksContextCanceled             uint64
	RollbacksContextDeadline             uint64
	RollbacksSQLViewNotReady             uint64
	RollbacksPanic                       uint64
	TypedConflicts                       uint64
	OperationReceiptsFoundAfterCommitErr uint64
	UncertainEventReceiptsAfterCommitErr uint64
	SQLViewNotReadyOutcomes              uint64
	StaleWriteContextOutcomes            uint64
	ProjectionTooWideOutcomes            uint64
	OperationTransactionsAttempted       uint64
	OperationTransactionsExecuted        uint64
	OperationTransactionsAlreadyAccepted uint64
	OperationTransactionsNoChange        uint64
	OperationTransactionsFailed          uint64
	OperationWorkspaceDirtyOutcomes      uint64
	// OperationTransactionLifecycleOpaqueFailures counts custom/injected helper
	// errors that do not preserve Swarmion's OperationTransactionError, plus the
	// database/sql "rollback already done" result where the driver outcome is
	// unknowable. Other SDK failures expose begin/execute/rollback/commit/receipt
	// phases and are recorded exactly. When this counter is non-zero,
	// TransactionsStarted and Rollbacks* remain exact observed lower bounds but
	// cannot be used as a complete operation-transaction rollback denominator.
	OperationTransactionLifecycleOpaqueFailures uint64
}

type transactionRollbackReason uint8

const (
	transactionRollbackApplyFailure transactionRollbackReason = iota
	transactionRollbackContextCanceled
	transactionRollbackContextDeadline
	transactionRollbackSQLViewNotReady
	transactionRollbackPanic
)

type transactionRollbackPhase uint8

const (
	transactionRollbackPhaseApply transactionRollbackPhase = iota
	transactionRollbackPhaseBeforeCommit
	transactionRollbackPhasePanic
)

type transactionMetrics struct {
	transactionsStarted                         atomic.Uint64
	commitsAttempted                            atomic.Uint64
	commitsSucceeded                            atomic.Uint64
	commitsFailed                               atomic.Uint64
	noopCommitOutcomes                          atomic.Uint64
	rollbacksAttempted                          atomic.Uint64
	rollbacksSucceeded                          atomic.Uint64
	rollbacksFailed                             atomic.Uint64
	rollbacksApplyPhase                         atomic.Uint64
	rollbacksBeforeCommitPhase                  atomic.Uint64
	rollbacksPanicPhase                         atomic.Uint64
	rollbacksApplyFailure                       atomic.Uint64
	rollbacksContextCanceled                    atomic.Uint64
	rollbacksContextDeadline                    atomic.Uint64
	rollbacksSQLViewNotReady                    atomic.Uint64
	rollbacksPanic                              atomic.Uint64
	typedConflicts                              atomic.Uint64
	operationReceiptsFoundAfterCommitErr        atomic.Uint64
	uncertainEventReceiptsAfterCommitErr        atomic.Uint64
	sqlViewNotReadyOutcomes                     atomic.Uint64
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

func (m *transactionMetrics) recordRollback(phase transactionRollbackPhase, reason transactionRollbackReason, err error) {
	if m == nil {
		return
	}
	m.rollbacksAttempted.Add(1)
	switch phase {
	case transactionRollbackPhaseBeforeCommit:
		m.rollbacksBeforeCommitPhase.Add(1)
	case transactionRollbackPhasePanic:
		m.rollbacksPanicPhase.Add(1)
	default:
		m.rollbacksApplyPhase.Add(1)
	}
	switch reason {
	case transactionRollbackContextCanceled:
		m.rollbacksContextCanceled.Add(1)
	case transactionRollbackContextDeadline:
		m.rollbacksContextDeadline.Add(1)
	case transactionRollbackSQLViewNotReady:
		m.rollbacksSQLViewNotReady.Add(1)
	case transactionRollbackPanic:
		m.rollbacksPanic.Add(1)
	default:
		m.rollbacksApplyFailure.Add(1)
	}
	if err == nil {
		m.rollbacksSucceeded.Add(1)
	} else {
		m.rollbacksFailed.Add(1)
	}
}

func (m *transactionMetrics) snapshot() TransactionMetricsSnapshot {
	if m == nil {
		return TransactionMetricsSnapshot{}
	}
	return TransactionMetricsSnapshot{
		TransactionsStarted:                         m.transactionsStarted.Load(),
		CommitsAttempted:                            m.commitsAttempted.Load(),
		CommitsSucceeded:                            m.commitsSucceeded.Load(),
		CommitsFailed:                               m.commitsFailed.Load(),
		NoopCommitOutcomes:                          m.noopCommitOutcomes.Load(),
		RollbacksAttempted:                          m.rollbacksAttempted.Load(),
		RollbacksSucceeded:                          m.rollbacksSucceeded.Load(),
		RollbacksFailed:                             m.rollbacksFailed.Load(),
		RollbacksApplyPhase:                         m.rollbacksApplyPhase.Load(),
		RollbacksBeforeCommitPhase:                  m.rollbacksBeforeCommitPhase.Load(),
		RollbacksPanicPhase:                         m.rollbacksPanicPhase.Load(),
		RollbacksApplyFailure:                       m.rollbacksApplyFailure.Load(),
		RollbacksContextCanceled:                    m.rollbacksContextCanceled.Load(),
		RollbacksContextDeadline:                    m.rollbacksContextDeadline.Load(),
		RollbacksSQLViewNotReady:                    m.rollbacksSQLViewNotReady.Load(),
		RollbacksPanic:                              m.rollbacksPanic.Load(),
		TypedConflicts:                              m.typedConflicts.Load(),
		OperationReceiptsFoundAfterCommitErr:        m.operationReceiptsFoundAfterCommitErr.Load(),
		UncertainEventReceiptsAfterCommitErr:        m.uncertainEventReceiptsAfterCommitErr.Load(),
		SQLViewNotReadyOutcomes:                     m.sqlViewNotReadyOutcomes.Load(),
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
