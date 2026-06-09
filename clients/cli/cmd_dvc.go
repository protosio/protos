package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"

	"github.com/urfave/cli/v2"
)

var cmdDvc *cli.Command = &cli.Command{
	Name:  "dvc",
	Usage: "Distributed version control",
	Subcommands: []*cli.Command{
		{
			Name:      "log",
			Usage:     "Retrieve local or instance commits",
			ArgsUsage: "[instance]",
			Action: func(c *cli.Context) error {
				instance := c.Args().Get(0)
				return getCommits(instance)
			},
		},
	},
}

func getCommits(instanceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var commits []*pbApic.Commit
	var graph *pbApic.CommitGraph

	if instanceID == "" {
		// Retrieve local commits from the database
		resp, err := client.GetLocalCommits(ctx, &pbApic.GetLocalCommitsRequest{})
		if err != nil {
			return fmt.Errorf("failed to retrieve local commits: %w", err)
		}

		commits = resp.Commits
		graph = resp.Graph
	} else {
		// Retrieve commits from a remote machine
		resp, err := client.GetRemoteCommits(ctx, &pbApic.GetRemoteCommitsRequest{Remote: instanceID})
		if err != nil {
			return fmt.Errorf("failed to retrieve commits from instance '%s': %w", instanceID, err)
		}

		commits = resp.Commits
		graph = resp.Graph
	}

	if graph != nil && len(graph.Items) > 0 {
		return printCommitGraph(graph)
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)

	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "State", "Hash", "Committer", "Message")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "-----", "---------", "---------", "-------")
	for _, commit := range commits {
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", commitStateLabel(commit), commit.Hash, commit.Committer, commit.Message)
	}
	fmt.Fprint(w, "\n")

	return nil
}

func commitStateLabel(commit *pbApic.Commit) string {
	if commit == nil || len(commit.States) == 0 {
		return "unknown"
	}
	return strings.Join(commit.States, ",")
}

func printCommitGraph(graph *pbApic.CommitGraph) error {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	laneCount := commitGraphLaneCount(graph)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t", "Graph", "Hash", "State", "Committer", "Date", "Message")
	fmt.Fprintf(w, "\n%s\t%s\t%s\t%s\t%s\t%s\t", "-----", "----", "-----", "---------", "----", "-------")
	for _, item := range graph.Items {
		if item == nil || item.Commit == nil {
			continue
		}
		commit := item.Commit
		fmt.Fprintf(
			w,
			"\n%s\t%s\t%s\t%s\t%s\t%s\t",
			commitGraphItemPrefix(item, laneCount),
			shortCommitHash(commit.Hash),
			commitGraphAnnotation(commit),
			commit.Committer,
			commitDateLabel(commit),
			commit.Message,
		)
		if relationPrefix := commitGraphRelationPrefix(item, laneCount); relationPrefix != "" {
			fmt.Fprintf(w, "\n%s\t\t\t\t\t\t", relationPrefix)
		}
	}
	fmt.Fprint(w, "\n")
	return nil
}

func commitGraphLaneCount(graph *pbApic.CommitGraph) int {
	laneCount := int(graph.GetLaneCount())
	for _, item := range graph.GetItems() {
		if item == nil {
			continue
		}
		laneCount = max(laneCount, int(item.GetLane())+1)
		for _, relation := range item.GetRelations() {
			if relation == nil {
				continue
			}
			laneCount = max(laneCount, int(relation.GetFromLane())+1)
			laneCount = max(laneCount, int(relation.GetToLane())+1)
		}
	}
	if laneCount < 1 {
		return 1
	}
	return laneCount
}

func commitGraphItemPrefix(item *pbApic.CommitGraphItem, laneCount int) string {
	active := make(map[int32]struct{}, len(item.GetActiveLanes()))
	for _, lane := range item.GetActiveLanes() {
		active[lane] = struct{}{}
	}

	var b strings.Builder
	for lane := 0; lane < laneCount; lane++ {
		if lane > 0 {
			b.WriteByte(' ')
		}
		switch {
		case int32(lane) == item.GetLane():
			b.WriteByte('*')
		case hasLane(active, int32(lane)):
			b.WriteByte('|')
		default:
			b.WriteByte(' ')
		}
	}
	return strings.TrimRight(b.String(), " ")
}

func commitGraphRelationPrefix(item *pbApic.CommitGraphItem, laneCount int) string {
	width := laneCount*2 - 1
	if width < 1 {
		width = 1
	}
	chars := make([]rune, width)
	for i := range chars {
		chars[i] = ' '
	}
	for _, lane := range item.GetActiveLanes() {
		pos := int(lane) * 2
		if pos >= 0 && pos < len(chars) {
			chars[pos] = '|'
		}
	}

	drawn := false
	for _, relation := range item.GetRelations() {
		if relation == nil || !relation.GetVisible() || relation.GetFromLane() == relation.GetToLane() {
			continue
		}
		from := int(relation.GetFromLane()) * 2
		to := int(relation.GetToLane()) * 2
		if from < 0 || to < 0 || from >= len(chars) || to >= len(chars) {
			continue
		}
		start, end := from, to
		if start > end {
			start, end = end, start
		}
		for pos := start; pos <= end; pos++ {
			if pos%2 == 0 {
				if chars[pos] == ' ' {
					chars[pos] = '|'
				}
			} else {
				chars[pos] = '-'
			}
		}
		if to > from {
			chars[from] = '\\'
		} else {
			chars[from] = '/'
		}
		chars[to] = '|'
		drawn = true
	}
	if !drawn {
		return ""
	}
	return strings.TrimRight(string(chars), " ")
}

func hasLane(active map[int32]struct{}, lane int32) bool {
	_, ok := active[lane]
	return ok
}

func shortCommitHash(hash string) string {
	const length = 12
	if len(hash) <= length {
		return hash
	}
	return hash[:length]
}

func commitGraphAnnotation(commit *pbApic.Commit) string {
	annotations := []string{commitStateLabel(commit)}
	for _, ref := range commit.GetRefs() {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		annotations = append(annotations, ref)
	}
	return strings.Join(annotations, ",")
}

func commitDateLabel(commit *pbApic.Commit) string {
	if commit.GetDateUnix() <= 0 {
		return ""
	}
	return time.Unix(commit.GetDateUnix(), 0).Local().Format("2006-01-02 15:04")
}
