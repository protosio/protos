package db

import (
	"errors"
	"testing"
)

func TestRetryableSwarmionBootstrapError(t *testing.T) {
	retryable := errors.New("failed to open swarmion runtime: fetch checkpoint history from bootstrap state: fetch checkpoint history root=abc commits=1: sync_checkpoint_history abc: no connected providers")
	if !retryableSwarmionBootstrapError(retryable) {
		t.Fatalf("retryableSwarmionBootstrapError(%q) = false, want true", retryable.Error())
	}

	tests := []error{
		nil,
		errors.New("failed to open swarmion runtime: fetch checkpoint history from bootstrap state: checksum mismatch"),
		errors.New("failed to open swarmion runtime: sync_checkpoint_history abc: no connected providers"),
		errors.New("failed to create swarmion transport: listen tcp: bind: address already in use"),
	}
	for _, err := range tests {
		if retryableSwarmionBootstrapError(err) {
			t.Fatalf("retryableSwarmionBootstrapError(%v) = true, want false", err)
		}
	}
}
