package provisioners

import (
	"fmt"
	"strings"

	"github.com/protosio/protos/internal/tasks"
)

// ErrInstanceLifecycleOwnerUnavailable means no immutable peer authority can
// be proven for an instance. Historical or ambiguous rows deliberately remain in
// this state until an explicit recovery protocol is introduced.
var ErrInstanceLifecycleOwnerUnavailable = fmt.Errorf("instance lifecycle owner unavailable")

// ErrInstanceLifecycleOwnerConflict means a peer or task attempted an
// imperative lifecycle operation for an instance owned by another peer.
var ErrInstanceLifecycleOwnerConflict = fmt.Errorf("instance lifecycle owner conflict")

func instanceLifecycleOwner(instance InstanceInfo) (string, error) {
	ownerPeerID := strings.TrimSpace(instance.LifecycleOwnerPeerID)
	if ownerPeerID == "" {
		return "", fmt.Errorf(
			"%w: instance=%s has no persisted lifecycle owner",
			ErrInstanceLifecycleOwnerUnavailable,
			strings.TrimSpace(instance.ID),
		)
	}
	return ownerPeerID, nil
}

func (cm *Manager) localLifecycleExecutorPeerID() string {
	if cm == nil {
		return ""
	}
	if cm.tasks != nil {
		if peerID := strings.TrimSpace(cm.tasks.ExecutorPeerID()); peerID != "" {
			return peerID
		}
	}
	if cm.db != nil {
		if status, ok := cm.db.SwarmionStatus(); ok {
			return strings.TrimSpace(status.PeerID)
		}
	}
	return ""
}

func (cm *Manager) assertInstanceLifecycleExecutor(instance InstanceInfo, taskOwnerPeerID string) error {
	ownerPeerID, err := instanceLifecycleOwner(instance)
	if err != nil {
		return err
	}
	taskOwnerPeerID = strings.TrimSpace(taskOwnerPeerID)
	if taskOwnerPeerID != "" && taskOwnerPeerID != ownerPeerID {
		return fmt.Errorf(
			"%w: instance=%s persisted_owner=%s task_owner=%s",
			ErrInstanceLifecycleOwnerConflict,
			instance.ID,
			ownerPeerID,
			taskOwnerPeerID,
		)
	}
	localPeerID := cm.localLifecycleExecutorPeerID()
	if localPeerID == "" || localPeerID != ownerPeerID {
		return fmt.Errorf(
			"%w: instance=%s persisted_owner=%s local_executor=%s",
			ErrInstanceLifecycleOwnerConflict,
			instance.ID,
			ownerPeerID,
			localPeerID,
		)
	}
	if cm != nil && cm.provisionerMutationDisabled {
		return fmt.Errorf(
			"%w: instance=%s executor=%s lacks the provision capability",
			ErrInstanceLifecycleOwnerUnavailable,
			instance.ID,
			localPeerID,
		)
	}
	if cm != nil && cm.db != nil {
		if status, ok := cm.db.SwarmionStatus(); !ok || strings.TrimSpace(status.PeerID) != ownerPeerID {
			statusPeerID := ""
			if ok {
				statusPeerID = strings.TrimSpace(status.PeerID)
			}
			return fmt.Errorf(
				"%w: instance=%s persisted_owner=%s database_author=%s",
				ErrInstanceLifecycleOwnerConflict,
				instance.ID,
				ownerPeerID,
				statusPeerID,
			)
		}
	}
	return nil
}

func (cm *Manager) lifecycleTaskOwner(instance InstanceInfo) (string, error) {
	return instanceLifecycleOwner(instance)
}

func instanceTaskOwner(record tasks.Record) string {
	return strings.TrimSpace(record.OwnerPeerID)
}
