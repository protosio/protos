package db

import (
	"context"
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
	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return true, err
	}
	defer rows.Close()

	var count int
	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return true, err
		}
	}
	if err := rows.Err(); err != nil {
		return true, err
	}
	return count > 0, nil
}
