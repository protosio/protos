package db

const (
	CommitStateFinalized = "finalized"
	CommitStateTentative = "tentative"

	tentativeBaseCommitMessage = "[swarmion] sync tentative base"
)

type CommitView struct {
	Commit
	States []string
}

func (db *DB) GetCombinedCommits(finalizedBranch string, tentativeBranch string) ([]CommitView, error) {
	finalizedCommits, err := db.GetCommits(finalizedBranch)
	if err != nil {
		return nil, err
	}
	var tentativeCommits []Commit
	if tentativeBranch != "" {
		if commits, err := db.GetCommits(tentativeBranch); err == nil {
			tentativeCommits = commits
		}
	}
	return CombineCommitBranches(finalizedCommits, tentativeCommits), nil
}

func CombineCommitBranches(finalizedCommits []Commit, tentativeCommits []Commit) []CommitView {
	finalizedHashes := make(map[string]struct{}, len(finalizedCommits))
	for _, commit := range finalizedCommits {
		if commit.Hash == "" {
			continue
		}
		finalizedHashes[commit.Hash] = struct{}{}
	}

	combined := make([]CommitView, 0, len(finalizedCommits)+len(tentativeCommits))
	for _, commit := range tentativeCommits {
		if commit.Message == tentativeBaseCommitMessage {
			break
		}
		if commit.Hash == "" {
			continue
		}
		if _, finalized := finalizedHashes[commit.Hash]; finalized {
			continue
		}
		combined = append(combined, CommitView{
			Commit: commit,
			States: []string{CommitStateTentative},
		})
	}
	for _, commit := range finalizedCommits {
		if commit.Hash == "" {
			continue
		}
		combined = append(combined, CommitView{
			Commit: commit,
			States: []string{CommitStateFinalized},
		})
	}
	return combined
}
