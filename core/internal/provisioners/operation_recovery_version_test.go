package provisioners

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/protosio/protos/internal/tasks"
)

func TestPreviewDeleteIdentityFailsClosedAtRecoveryVersionBoundary(t *testing.T) {
	legacy := []byte(`{
		"key":"0123456789abcdef0123456789abcdef",
		"intent_digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"author_peer_id":"legacy-author",
		"expected_invariant":{"kind":"instance_absent","instance_id":"legacy-instance"}
	}`)
	var identity instanceDeleteOperationIdentity
	if err := json.Unmarshal(legacy, &identity); err != nil {
		t.Fatalf("decode preview identity fixture: %v", err)
	}
	err := validateInstanceDeleteOperationIdentity(identity, "legacy-task", "legacy-instance", false)
	if err == nil || !strings.Contains(err.Error(), "preview key/digest-only state is not accepted") {
		t.Fatalf("preview identity error=%v, want explicit fresh-state boundary", err)
	}
}

func TestPreviewOperationEffectFactCannotBecomeRecoveryAuthority(t *testing.T) {
	legacyPayload := json.RawMessage(`{
		"operation_id":"legacy-task",
		"operation":"delete",
		"expected_invariant":{"kind":"instance_absent","instance_id":"legacy-instance"}
	}`)
	fact := tasks.OperationFact{
		ID:           "legacy-id",
		TaskID:       "legacy-task",
		Kind:         tasks.OperationFactKindEffect,
		OperationKey: "0123456789abcdef0123456789abcdef",
		IntentDigest: strings.Repeat("a", 64),
		AuthorPeerID: "legacy-author",
		SubjectType:  taskSubjectInstance,
		SubjectID:    "legacy-instance",
		Payload:      legacyPayload,
	}
	if _, err := instanceDeleteIdentityFromEffectFact(fact); err == nil ||
		!strings.Contains(err.Error(), "invalid instance delete effect fact") {
		t.Fatalf("preview fact error=%v, want fail-closed recovery boundary", err)
	}
}
