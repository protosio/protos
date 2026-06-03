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
