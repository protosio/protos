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

func TestBuildCommitGraphLinearHistory(t *testing.T) {
	t.Parallel()

	graph := BuildCommitGraph([]CommitView{
		{Commit: Commit{Hash: "c", ParentHashes: []string{"b"}}},
		{Commit: Commit{Hash: "b", ParentHashes: []string{"a"}}},
		{Commit: Commit{Hash: "a"}},
	})

	if graph.LaneCount != 1 {
		t.Fatalf("LaneCount = %d, want 1", graph.LaneCount)
	}
	if len(graph.Items) != 3 {
		t.Fatalf("Items length = %d, want 3", len(graph.Items))
	}
	for i, item := range graph.Items {
		if item.Row != i {
			t.Fatalf("Items[%d].Row = %d, want %d", i, item.Row, i)
		}
		if item.Lane != 0 {
			t.Fatalf("Items[%d].Lane = %d, want 0", i, item.Lane)
		}
	}
	assertGraphRelation(t, graph.Items[0], "b", 0, 0, 1, true)
	assertGraphRelation(t, graph.Items[1], "a", 0, 0, 2, true)
}

func TestBuildCommitGraphMergeHistory(t *testing.T) {
	t.Parallel()

	graph := BuildCommitGraph([]CommitView{
		{Commit: Commit{Hash: "merge", ParentHashes: []string{"main", "feature"}}},
		{Commit: Commit{Hash: "main", ParentHashes: []string{"base"}}},
		{Commit: Commit{Hash: "feature", ParentHashes: []string{"feature-base"}}},
		{Commit: Commit{Hash: "feature-base", ParentHashes: []string{"base"}}},
		{Commit: Commit{Hash: "base"}},
	})

	if graph.LaneCount != 2 {
		t.Fatalf("LaneCount = %d, want 2", graph.LaneCount)
	}
	wantLanes := []int{0, 0, 1, 1, 0}
	for i, want := range wantLanes {
		if graph.Items[i].Lane != want {
			t.Fatalf("Items[%d].Lane = %d, want %d", i, graph.Items[i].Lane, want)
		}
	}
	assertGraphRelation(t, graph.Items[0], "main", 0, 0, 1, true)
	assertGraphRelation(t, graph.Items[0], "feature", 0, 1, 2, true)
	assertGraphRelation(t, graph.Items[3], "base", 1, 0, 4, true)
}

func TestBuildCommitGraphKeepsTentativeAndFinalizedLanes(t *testing.T) {
	t.Parallel()

	graph := BuildCommitGraph([]CommitView{
		{Commit: Commit{Hash: "tentative-new", ParentHashes: []string{"tentative-old"}}, States: []string{CommitStateTentative}},
		{Commit: Commit{Hash: "tentative-old", ParentHashes: []string{"shared"}}, States: []string{CommitStateTentative}},
		{Commit: Commit{Hash: "finalized-new", ParentHashes: []string{"shared"}}, States: []string{CommitStateFinalized}},
		{Commit: Commit{Hash: "shared", ParentHashes: []string{"base"}}, States: []string{CommitStateFinalized}},
		{Commit: Commit{Hash: "base"}, States: []string{CommitStateFinalized}},
	})

	if graph.LaneCount != 2 {
		t.Fatalf("LaneCount = %d, want 2", graph.LaneCount)
	}
	wantLanes := []int{0, 0, 1, 0, 0}
	for i, want := range wantLanes {
		if graph.Items[i].Lane != want {
			t.Fatalf("Items[%d].Lane = %d, want %d", i, graph.Items[i].Lane, want)
		}
	}
	assertGraphRelation(t, graph.Items[2], "shared", 1, 0, 3, true)
}

func TestBuildCommitGraphRecordsHiddenParent(t *testing.T) {
	t.Parallel()

	graph := BuildCommitGraph([]CommitView{
		{Commit: Commit{Hash: "tentative", ParentHashes: []string{"hidden-base"}}},
	})

	if len(graph.Items) != 1 {
		t.Fatalf("Items length = %d, want 1", len(graph.Items))
	}
	assertGraphRelation(t, graph.Items[0], "hidden-base", 0, 0, -1, false)
}

func assertGraphRelation(t *testing.T, item CommitGraphItem, parentHash string, fromLane int, toLane int, parentRow int, visible bool) {
	t.Helper()

	for _, relation := range item.Relations {
		if relation.ParentHash != parentHash {
			continue
		}
		if relation.FromLane != fromLane || relation.ToLane != toLane || relation.ParentRow != parentRow || relation.Visible != visible {
			t.Fatalf("relation to %q = %#v, want from=%d to=%d parentRow=%d visible=%t", parentHash, relation, fromLane, toLane, parentRow, visible)
		}
		return
	}
	t.Fatalf("relation to %q not found in %#v", parentHash, item.Relations)
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
