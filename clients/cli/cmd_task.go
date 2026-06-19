package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
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
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "instance",
					Usage: "Read tasks from a remote instance admin API",
				},
			},
			Action: func(c *cli.Context) error {
				return listTasks(c.String("instance"))
			},
		},
		{
			Name:      "info",
			ArgsUsage: "<task id>",
			Usage:     "Display task details and events",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "instance",
					Usage: "Read the task from a remote instance admin API",
				},
			},
			Action: func(c *cli.Context) error {
				id := c.Args().Get(0)
				if id == "" {
					return showSubcommandHelp(c)
				}
				return taskInfo(id, c.String("instance"))
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

func listTasks(instance string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetTasks(ctx, &pbApic.GetTasksRequest{MaxResults: 200, Instance: instance})
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

func taskInfo(id string, instance string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("task id is empty")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := client.GetTask(ctx, &pbApic.GetTaskRequest{Id: id, IncludeEvents: true, Instance: instance})
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
		message := event.GetMessage()
		if details := taskDetailsSummary(event.GetDetailsJson(), ""); details != "" {
			message = strings.TrimSpace(message + " " + details)
		}
		fmt.Fprintf(w, "\n %s\t%d%%\t%s\t%s\t", event.GetStatus(), event.GetProgress(), event.GetCreatedAt(), message)
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

	printer := newTaskProgressPrinter(jsonl)
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
		if err := printer.print(resp); err != nil {
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
			printer.printFinal(latest)
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

type taskProgressPrinter struct {
	jsonl   bool
	started time.Time
	task    *pbApic.Task
	seen    map[string]struct{}
	upload  *taskUploadProgress
}

type taskUploadProgress struct {
	bytes uint64
	at    time.Time
}

func newTaskProgressPrinter(jsonl bool) *taskProgressPrinter {
	return &taskProgressPrinter{
		jsonl:   jsonl,
		started: time.Now(),
		seen:    map[string]struct{}{},
	}
}

func (p *taskProgressPrinter) print(resp *pbApic.WatchTaskResponse) error {
	if p.jsonl {
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
		p.task = task
		line := fmt.Sprintf(
			"watching %s (%s)\n%s\n",
			task.GetId(),
			task.GetStream(),
			p.formatProgress("snapshot", task.GetStatus(), task.GetProgress(), task.GetMessage(), "", task.GetUpdatedAt(), true),
		)
		return p.printOnce("snapshot:"+task.GetId()+":"+task.GetUpdatedAt()+":"+task.GetMessage(), line)
	}
	if update := resp.GetUpdate(); update != nil {
		source := "live"
		if update.GetDurable() {
			source = "saved"
		}
		line := p.formatProgress(source, update.GetStatus(), update.GetProgress(), update.GetMessage(), update.GetDetailsJson(), update.GetCreatedAt(), update.GetDurable())
		key := fmt.Sprintf("update:%d:%s:%d:%s:%t", resp.GetSequence(), update.GetStatus(), update.GetProgress(), update.GetMessage(), update.GetDurable())
		return p.printOnce(key, line+"\n")
	}
	return nil
}

func (p *taskProgressPrinter) printFinal(task *pbApic.Task) {
	if p.jsonl || task == nil {
		return
	}
	elapsed := time.Since(p.started).Round(time.Second)
	message := taskListMessage(task.GetMessage(), task.GetErrorMessage())
	if message == "" {
		message = task.GetStatus()
	}
	fmt.Printf("finished %s in %s: %s\n", task.GetId(), elapsed, message)
	if result := taskResultSummary(task.GetResultJson()); result != "" {
		fmt.Printf("result: %s\n", result)
	}
}

func (p *taskProgressPrinter) printOnce(key string, line string) error {
	if _, found := p.seen[key]; found {
		return nil
	}
	p.seen[key] = struct{}{}
	_, err := fmt.Fprint(os.Stdout, line)
	return err
}

func (p *taskProgressPrinter) formatProgress(source string, status string, progress int32, message string, detailsJSON string, at string, durable bool) string {
	elapsed := time.Since(p.started).Round(time.Second)
	if elapsed < 0 {
		elapsed = 0
	}
	details := taskDetailsSummary(detailsJSON, p.uploadRate(detailsJSON, durable))
	if details != "" {
		details = "  " + details
	}
	if message = strings.TrimSpace(message); message == "" {
		message = status
	}
	timestamp := taskClockLabel(at)
	if timestamp != "" {
		timestamp = " " + timestamp
	}
	return fmt.Sprintf("[%s%s] %-5s %-9s %3d%%  %s%s", elapsed, timestamp, source, status, progress, message, details)
}

func (p *taskProgressPrinter) uploadRate(detailsJSON string, durable bool) string {
	if durable || strings.TrimSpace(detailsJSON) == "" {
		return ""
	}
	details, err := decodeTaskJSONMap(detailsJSON)
	if err != nil {
		return ""
	}
	bytes, ok := numericDetail(details, "bytes_uploaded")
	if !ok {
		return ""
	}
	now := time.Now()
	defer func() {
		p.upload = &taskUploadProgress{bytes: bytes, at: now}
	}()
	if p.upload == nil || bytes <= p.upload.bytes {
		return ""
	}
	elapsed := now.Sub(p.upload.at)
	if elapsed <= 0 {
		return ""
	}
	rate := float64(bytes-p.upload.bytes) / elapsed.Seconds()
	if rate <= 0 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return ""
	}
	return formatByteCount(uint64(rate)) + "/s"
}

func taskDetailsSummary(detailsJSON string, rate string) string {
	detailsJSON = strings.TrimSpace(detailsJSON)
	if detailsJSON == "" || detailsJSON == "{}" {
		return ""
	}
	details, err := decodeTaskJSONMap(detailsJSON)
	if err != nil {
		return detailsJSON
	}
	var parts []string
	if uploaded, ok := numericDetail(details, "bytes_uploaded"); ok {
		if total, ok := numericDetail(details, "archive_size_bytes"); ok && total > 0 {
			parts = append(parts, fmt.Sprintf("%s/%s", formatByteCount(uploaded), formatByteCount(total)))
		} else {
			parts = append(parts, formatByteCount(uploaded))
		}
	}
	if rate != "" {
		parts = append(parts, rate)
	}
	if imageID, ok := stringDetail(details, "image_id"); ok {
		parts = append(parts, "image_id="+imageID)
	}
	if digest, ok := stringDetail(details, "target_digest"); ok {
		parts = append(parts, "digest="+digest)
	}
	if percent, ok := numericDetail(details, "percent"); ok && percent <= 100 {
		parts = append(parts, fmt.Sprintf("%d%%", percent))
	}
	if len(parts) > 0 {
		return "(" + strings.Join(parts, ", ") + ")"
	}
	compact, err := json.Marshal(details)
	if err != nil {
		return detailsJSON
	}
	return string(compact)
}

func taskResultSummary(resultJSON string) string {
	resultJSON = strings.TrimSpace(resultJSON)
	if resultJSON == "" || resultJSON == "{}" {
		return ""
	}
	result, err := decodeTaskJSONMap(resultJSON)
	if err != nil {
		return resultJSON
	}
	var parts []string
	for _, key := range []string{"image_id", "image_ref", "target_digest", "platform", "instance"} {
		if value, ok := stringDetail(result, key); ok {
			parts = append(parts, key+"="+value)
		}
	}
	if bytes, ok := numericDetail(result, "bytes_uploaded"); ok {
		parts = append(parts, "bytes_uploaded="+formatByteCount(bytes))
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	compact, err := json.Marshal(result)
	if err != nil {
		return resultJSON
	}
	return string(compact)
}

func numericDetail(values map[string]any, key string) (uint64, bool) {
	value, found := values[key]
	if !found {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		if typed < 0 {
			return 0, false
		}
		return uint64(typed), true
	case int:
		if typed < 0 {
			return 0, false
		}
		return uint64(typed), true
	case uint64:
		return typed, true
	case json.Number:
		parsed, err := typed.Int64()
		if err != nil || parsed < 0 {
			return 0, false
		}
		return uint64(parsed), true
	default:
		return 0, false
	}
}

func decodeTaskJSONMap(value string) (map[string]any, error) {
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.UseNumber()
	var out map[string]any
	if err := decoder.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func stringDetail(values map[string]any, key string) (string, bool) {
	value, found := values[key]
	if !found {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	return text, text != ""
}

func taskClockLabel(value string) string {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	local := parsed.Local()
	return local.Format("15:04:05")
}

func formatByteCount(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	value := float64(bytes)
	for _, suffix := range []string{"KiB", "MiB", "GiB", "TiB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f%s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1fPiB", value/unit)
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
