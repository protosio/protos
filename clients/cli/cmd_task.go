package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"
)

var cmdTask *cli.Command = &cli.Command{
	Name:    "task",
	Aliases: []string{"tasks"},
	Usage:   "Inspect background tasks",
	Subcommands: []*cli.Command{
		{
			Name:  "ls",
			Usage: "List tasks",
			Action: func(c *cli.Context) error {
				return listTasks()
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<task id>",
			Usage:     "Display task details and events",
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				if id == "" {
					return showSubcommandHelp(c)
				}
				return taskInfo(id)
			},
		},
		{
			Name:      "follow",
			ArgsUsage: "<task id>",
			Usage:     "Stream task progress until the task reaches a terminal state",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "jsonl",
					Usage: "Write WatchTask responses as JSON lines",
				},
			},
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				if id == "" {
					return showSubcommandHelp(c)
				}
				_, err := followTaskUntilTerminal(context.Background(), id, c.Bool("jsonl"))
				return err
			},
		},
	},
}

func listTasks() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetTasks(ctx, &pbApic.GetTasksRequest{MaxResults: 200})
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t", "ID", "Stream", "Subject", "Subject ID", "Status", "Progress", "Updated", "Message")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t", "--", "------", "-------", "----------", "------", "--------", "-------", "-------")
	for _, task := range resp.GetTasks() {
		message := taskListMessage(task.GetMessage(), task.GetErrorMessage())
		fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t%s\t%d%%\t%s\t%s\t",
			task.GetId(),
			task.GetStream(),
			task.GetSubjectType(),
			task.GetSubjectId(),
			task.GetStatus(),
			task.GetProgress(),
			task.GetUpdatedAt(),
			message,
		)
	}
	fmt.Fprint(w, "\n")
	return nil
}

func taskListMessage(message string, errorMessage string) string {
	message = strings.TrimSpace(message)
	errorMessage = strings.TrimSpace(errorMessage)
	if errorMessage == "" {
		return message
	}
	if message == "" || message == "failed" {
		return errorMessage
	}
	return message + ": " + errorMessage
}

func taskInfo(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("task id is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetTask(ctx, &pbApic.GetTaskRequest{Id: id, IncludeEvents: true})
	if err != nil {
		return fmt.Errorf("failed to retrieve task '%s': %w", id, err)
	}
	task := resp.GetTask()
	if task == nil || task.GetId() == "" {
		return fmt.Errorf("task '%s' not found", id)
	}
	fmt.Printf("ID: %s\n", task.GetId())
	fmt.Printf("Stream: %s\n", task.GetStream())
	fmt.Printf("Subject: %s %s\n", task.GetSubjectType(), task.GetSubjectId())
	fmt.Printf("Status: %s\n", task.GetStatus())
	fmt.Printf("Progress: %d%%\n", task.GetProgress())
	fmt.Printf("Attempts: %d/%d\n", task.GetAttempts(), task.GetMaxAttempts())
	fmt.Printf("Title: %s\n", task.GetTitle())
	fmt.Printf("Message: %s\n", task.GetMessage())
	if task.GetErrorMessage() != "" {
		fmt.Printf("Error: %s\n", task.GetErrorMessage())
	}
	fmt.Printf("Created: %s\n", task.GetCreatedAt())
	fmt.Printf("Updated: %s\n", task.GetUpdatedAt())
	if task.GetStartedAt() != "" {
		fmt.Printf("Started: %s\n", task.GetStartedAt())
	}
	if task.GetFinishedAt() != "" {
		fmt.Printf("Finished: %s\n", task.GetFinishedAt())
	}
	if task.GetPayloadJson() != "" {
		fmt.Printf("Payload: %s\n", task.GetPayloadJson())
	}
	if task.GetResultJson() != "" {
		fmt.Printf("Result: %s\n", task.GetResultJson())
	}

	fmt.Println("Events:")
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t", "Status", "Progress", "Created", "Message")
	fmt.Fprintf(w, "\n %s\t%s\t%s\t%s\t", "------", "--------", "-------", "-------")
	for _, event := range resp.GetEvents() {
		fmt.Fprintf(w, "\n %s\t%d%%\t%s\t%s\t", event.GetStatus(), event.GetProgress(), event.GetCreatedAt(), event.GetMessage())
	}
	fmt.Fprint(w, "\n")
	return nil
}

func followTaskUntilTerminal(ctx context.Context, id string, jsonl bool) (*pbApic.Task, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("task id is empty")
	}
	stream, err := client.WatchTask(ctx, &pbApic.WatchTaskRequest{
		Id:                  id,
		IncludeSnapshot:     true,
		IncludeEvents:       false,
		HeartbeatIntervalMs: 5000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to watch task '%s': %w", id, err)
	}

	var latest *pbApic.Task
	var latestStatus string
	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return fetchTaskAfterWatch(ctx, id, latestStatus)
		}
		if err != nil {
			return nil, fmt.Errorf("watch task '%s': %w", id, err)
		}
		if err := printTaskWatchResponse(resp, jsonl); err != nil {
			return nil, err
		}
		if task := resp.GetTask(); task != nil && task.GetId() != "" {
			latest = task
			latestStatus = task.GetStatus()
		}
		if update := resp.GetUpdate(); update != nil && update.GetStatus() != "" {
			latestStatus = update.GetStatus()
		}
		if taskStatusTerminal(latestStatus) {
			finalTask, err := fetchTaskAfterWatch(ctx, id, latestStatus)
			if err != nil {
				return nil, err
			}
			if finalTask != nil {
				latest = finalTask
			}
			if taskStatusFailed(latestStatus) {
				message := latestStatus
				if latest != nil {
					message = taskListMessage(latest.GetMessage(), latest.GetErrorMessage())
				}
				return latest, fmt.Errorf("task '%s' %s", id, message)
			}
			return latest, nil
		}
	}
}

func fetchTaskAfterWatch(ctx context.Context, id string, latestStatus string) (*pbApic.Task, error) {
	resp, err := client.GetTask(ctx, &pbApic.GetTaskRequest{Id: id, IncludeEvents: false})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve task '%s': %w", id, err)
	}
	task := resp.GetTask()
	if task == nil || task.GetId() == "" {
		return nil, fmt.Errorf("task '%s' not found", id)
	}
	if taskStatusFailed(task.GetStatus()) {
		return task, fmt.Errorf("task '%s' %s", id, taskListMessage(task.GetMessage(), task.GetErrorMessage()))
	}
	if latestStatus != "" && !taskStatusTerminal(task.GetStatus()) {
		return task, fmt.Errorf("task '%s' stream ended before terminal state; last status %s", id, latestStatus)
	}
	return task, nil
}

func printTaskWatchResponse(resp *pbApic.WatchTaskResponse, jsonl bool) error {
	if jsonl {
		raw, err := protojson.MarshalOptions{EmitUnpopulated: false}.Marshal(resp)
		if err != nil {
			return err
		}
		fmt.Println(string(raw))
		return nil
	}
	if resp.GetHeartbeat() {
		return nil
	}
	if task := resp.GetTask(); task != nil && task.GetId() != "" {
		fmt.Printf("task %s: %s %d%% %s\n", task.GetId(), task.GetStatus(), task.GetProgress(), task.GetMessage())
		return nil
	}
	if update := resp.GetUpdate(); update != nil {
		fmt.Printf("task %s: %s %d%% %s\n", update.GetTaskId(), update.GetStatus(), update.GetProgress(), update.GetMessage())
	}
	return nil
}

func taskStatusTerminal(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func taskStatusFailed(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "cancelled":
		return true
	default:
		return false
	}
}
