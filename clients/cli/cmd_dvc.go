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

	if instanceID == "" {
		// Retrieve local commits from the database
		resp, err := client.GetLocalCommits(ctx, &pbApic.GetLocalCommitsRequest{})
		if err != nil {
			return fmt.Errorf("failed to retrieve local commits: %w", err)
		}

		commits = resp.Commits
	} else {
		// Retrieve commits from a remote machine
		resp, err := client.GetRemoteCommits(ctx, &pbApic.GetRemoteCommitsRequest{Remote: instanceID})
		if err != nil {
			return fmt.Errorf("failed to retrieve commits from instance '%s': %w", instanceID, err)
		}

		commits = resp.Commits
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
