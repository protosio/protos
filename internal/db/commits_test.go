package db

import "testing"

func TestCombineCommitBranchesKeepsUnfinalizedTentativeFirst(t *testing.T) {
	t.Parallel()

	finalized := []Commit{
		{Hash: "finalized-new", Committer: "alice", Message: "new finalized"},
		{Hash: "shared", Committer: "alice", Message: "now finalized"},
		{Hash: "finalized-old", Committer: "bob", Message: "old finalized"},
	}
	tentative := []Commit{
		{Hash: "tentative-new", Committer: "carol", Message: "new tentative"},
		{Hash: "shared", Committer: "carol", Message: "tentative copy"},
		{Hash: "tentative-old", Committer: "dave", Message: "old tentative"},
		{Hash: "base", Committer: "swarmion-sync", Message: tentativeBaseCommitMessage},
		{Hash: "old-published", Committer: "erin", Message: "published old tentative"},
	}

	got := CombineCommitBranches(finalized, tentative)
	wantHashes := []string{
		"tentative-new",
		"tentative-old",
		"finalized-new",
		"shared",
		"finalized-old",
	}
	wantStates := [][]string{
		{CommitStateTentative},
		{CommitStateTentative},
		{CommitStateFinalized},
		{CommitStateFinalized},
		{CommitStateFinalized},
	}

	if len(got) != len(wantHashes) {
		t.Fatalf("combined length = %d, want %d: %#v", len(got), len(wantHashes), got)
	}
	for i := range wantHashes {
		if got[i].Hash != wantHashes[i] {
			t.Fatalf("combined[%d].Hash = %q, want %q", i, got[i].Hash, wantHashes[i])
		}
		if !sameStrings(got[i].States, wantStates[i]) {
			t.Fatalf("combined[%d].States = %#v, want %#v", i, got[i].States, wantStates[i])
		}
	}
}

func TestCombineCommitBranchesHidesPublishedTentativeHistory(t *testing.T) {
	t.Parallel()

	finalized := []Commit{
		{Hash: "finalized-new", Committer: "alice", Message: "new finalized"},
		{Hash: "finalized-old", Committer: "bob", Message: "old finalized"},
	}
	tentative := []Commit{
		{Hash: "base", Committer: "swarmion-sync", Message: tentativeBaseCommitMessage},
		{Hash: "already-published", Committer: "carol", Message: "published tentative"},
	}

	got := CombineCommitBranches(finalized, tentative)
	wantHashes := []string{"finalized-new", "finalized-old"}

	if len(got) != len(wantHashes) {
		t.Fatalf("combined length = %d, want %d: %#v", len(got), len(wantHashes), got)
	}
	for i := range wantHashes {
		if got[i].Hash != wantHashes[i] {
			t.Fatalf("combined[%d].Hash = %q, want %q", i, got[i].Hash, wantHashes[i])
		}
		if !sameStrings(got[i].States, []string{CommitStateFinalized}) {
			t.Fatalf("combined[%d].States = %#v, want finalized", i, got[i].States)
		}
	}
}

func TestCombineCommitBranchesCanHideTentativeWhenBranchesMatch(t *testing.T) {
	t.Parallel()

	finalized := []Commit{
		{Hash: "finalized-new", Committer: "alice", Message: "new finalized"},
	}
	tentative := []Commit{
		{Hash: "tentative-new", Committer: "carol", Message: "new tentative"},
	}

	got := combineCommitBranches(finalized, tentative, false)
	wantHashes := []string{"finalized-new"}

	if len(got) != len(wantHashes) {
		t.Fatalf("combined length = %d, want %d: %#v", len(got), len(wantHashes), got)
	}
	for i := range wantHashes {
		if got[i].Hash != wantHashes[i] {
			t.Fatalf("combined[%d].Hash = %q, want %q", i, got[i].Hash, wantHashes[i])
		}
		if !sameStrings(got[i].States, []string{CommitStateFinalized}) {
			t.Fatalf("combined[%d].States = %#v, want finalized", i, got[i].States)
		}
	}
}

func sameStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
