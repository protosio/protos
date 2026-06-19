package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const (
	CommitStateFinalized = "finalized"
	CommitStateTentative = "tentative"

	tentativeBaseCommitMessage = "[swarmion] sync tentative base"
)

type CommitView struct {
	Commit
	States []string
}

func ExtractCommitSignerPublicKey(message string) string {
	for _, line := range strings.Split(message, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "swarmion.writer.public_key" {
			continue
		}
		return strings.TrimSpace(value)
	}
	return ""
}

func (db *DB) GetCombinedCommits(finalizedBranch string, tentativeBranch string) ([]CommitView, error) {
	finalizedBranch = normalizeCommitBranch(finalizedBranch)
	finalizedCommits, err := db.GetCommits(finalizedBranch)
	if err != nil {
		return nil, err
	}
	var tentativeCommits []Commit
	if tentativeBranch != "" {
		tentativeBranch = normalizeCommitBranch(tentativeBranch)
		includeTentative := true
		if hasDiff, err := db.branchesHaveDataDiff(finalizedBranch, tentativeBranch); err == nil && !hasDiff {
			includeTentative = false
		}
		if includeTentative {
			if commits, err := db.GetCommits(tentativeBranch); err == nil {
				tentativeCommits = commits
			}
		}
	}
	return CombineCommitBranches(finalizedCommits, tentativeCommits), nil
}

func CombineCommitBranches(finalizedCommits []Commit, tentativeCommits []Commit) []CommitView {
	return combineCommitBranches(finalizedCommits, tentativeCommits, true)
}

func combineCommitBranches(finalizedCommits []Commit, tentativeCommits []Commit, includeTentative bool) []CommitView {
	finalizedHashes := make(map[string]struct{}, len(finalizedCommits))
	for _, commit := range finalizedCommits {
		if commit.Hash == "" {
			continue
		}
		finalizedHashes[commit.Hash] = struct{}{}
	}

	combined := make([]CommitView, 0, len(finalizedCommits)+len(tentativeCommits))
	if includeTentative {
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

func normalizeCommitBranch(branch string) string {
	if strings.TrimSpace(branch) == "" {
		return "main"
	}
	return branch
}

func (db *DB) branchesHaveDataDiff(baseBranch string, compareBranch string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM dolt_diff_summary('%s', '%s');",
		escapeSQL(baseBranch),
		escapeSQL(compareBranch),
	)

	var count int
	if err := db.ReadRows(context.Background(), query, nil, func(rows *sql.Rows) error {
		if rows.Next() {
			if err := rows.Scan(&count); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return true, err
	}
	return count > 0, nil
}
