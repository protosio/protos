package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type runSummary struct {
	StartedAt    string            `json:"started_at"`
	FinishedAt   string            `json:"finished_at"`
	Status       string            `json:"status"`
	Error        string            `json:"error"`
	WorkDir      string            `json:"work_dir"`
	Suffix       string            `json:"suffix"`
	ImageName    string            `json:"image_name"`
	AppImage     string            `json:"app_image"`
	Providers    []summaryProvider `json:"providers"`
	Images       []summaryImage    `json:"images"`
	Instances    []summaryInstance `json:"instances"`
	Tasks        []summaryTask     `json:"tasks"`
	ImageSources []imageSource     `json:"image_sources"`
	Cleanup      []summaryCleanup  `json:"cleanup"`
}

type summaryProvider struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

type summaryImage struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Location string `json:"location"`
	TaskID   string `json:"task_id"`
	Status   string `json:"status"`
}

type summaryInstance struct {
	Name     string `json:"name"`
	ID       string `json:"id"`
	PeerID   string `json:"peer_id"`
	Provider string `json:"provider"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

type summaryTask struct {
	ID               string              `json:"id"`
	Stream           string              `json:"stream"`
	SubjectType      string              `json:"subject_type"`
	SubjectID        string              `json:"subject_id"`
	Status           string              `json:"status"`
	Message          string              `json:"message"`
	ErrorMessage     string              `json:"error_message"`
	Progress         int32               `json:"progress"`
	Provider         string              `json:"provider"`
	Location         string              `json:"location"`
	ImageName        string              `json:"image_name"`
	ImageID          string              `json:"image_id"`
	Instance         string              `json:"instance"`
	ImageRef         string              `json:"image_ref"`
	TargetDigest     string              `json:"target_digest"`
	Platform         string              `json:"platform"`
	BytesUploaded    uint64              `json:"bytes_uploaded"`
	ArchiveSizeBytes uint64              `json:"archive_size_bytes"`
	CreatedAt        string              `json:"created_at"`
	StartedAt        string              `json:"started_at"`
	FinishedAt       string              `json:"finished_at"`
	Events           []summaryTaskEvent  `json:"events"`
	Updates          []summaryTaskUpdate `json:"updates"`
}

type summaryTaskEvent struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Progress    int32  `json:"progress"`
	DetailsJSON string `json:"details_json"`
	CreatedAt   string `json:"created_at"`
}

type summaryTaskUpdate struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	Progress    int32  `json:"progress"`
	DetailsJSON string `json:"details_json"`
	CreatedAt   string `json:"created_at"`
	Durable     bool   `json:"durable"`
}

type imageSource struct {
	Phase        string `json:"phase"`
	Instance     string `json:"instance"`
	ImageRef     string `json:"image_ref"`
	Found        bool   `json:"found"`
	HasContent   bool   `json:"has_content"`
	Source       string `json:"source"`
	TargetDigest string `json:"target_digest"`
	Platform     string `json:"platform"`
	Descriptors  int    `json:"descriptors"`
	Error        string `json:"error"`
}

type summaryCleanup struct {
	ResourceType string `json:"resource_type"`
	Name         string `json:"name"`
	ID           string `json:"id"`
	Provider     string `json:"provider"`
	Location     string `json:"location"`
	Status       string `json:"status"`
	Error        string `json:"error"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at"`
	DurationMs   int64  `json:"duration_ms"`
}

func main() {
	summaryPath := flag.String("summary", ".tmp/mixed-cloud-e2e-summary.json", "mixed-cloud e2e summary artifact")
	flag.Parse()

	data, err := os.ReadFile(*summaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read summary %s: %v\n", *summaryPath, err)
		os.Exit(1)
	}
	var summary runSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		fmt.Fprintf(os.Stderr, "decode summary %s: %v\n", *summaryPath, err)
		os.Exit(1)
	}

	fmt.Printf("summary %s status=%s suffix=%s image=%s app_image=%s\n", *summaryPath, summary.Status, summary.Suffix, summary.ImageName, summary.AppImage)
	if summary.Error != "" {
		fmt.Printf("run error: %s\n", summary.Error)
	}
	printProviders(summary.Providers)
	printImages(summary.Images)
	printInstances(summary.Instances)
	printTasks(summary.Tasks)
	printImageSources(summary.ImageSources)
	printCleanup(summary.Cleanup)

	failures := validateSummary(summary)
	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "mixed-cloud summary check failed:")
		for _, failure := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", failure)
		}
		os.Exit(1)
	}
	fmt.Println("mixed-cloud summary ok")
}

func printProviders(items []summaryProvider) {
	if len(items) == 0 {
		fmt.Println("providers: none")
		return
	}
	fmt.Println("providers:")
	for _, item := range items {
		fmt.Printf("- %s type=%s location=%s status=%s\n", item.Name, item.Type, item.Location, item.Status)
	}
}

func printImages(items []summaryImage) {
	if len(items) == 0 {
		fmt.Println("images: none")
		return
	}
	fmt.Println("images:")
	for _, item := range items {
		fmt.Printf("- %s provider=%s location=%s id=%s status=%s task=%s\n", item.Name, item.Provider, item.Location, item.ID, item.Status, item.TaskID)
	}
}

func printInstances(items []summaryInstance) {
	if len(items) == 0 {
		fmt.Println("instances: none")
		return
	}
	fmt.Println("instances:")
	for _, item := range items {
		fmt.Printf("- %s provider=%s location=%s id=%s peer=%s status=%s\n", item.Name, item.Provider, item.Location, item.ID, item.PeerID, item.Status)
	}
}

func printTasks(items []summaryTask) {
	if len(items) == 0 {
		fmt.Println("tasks: none")
		return
	}
	fmt.Println("tasks:")
	for _, item := range items {
		label := firstNonEmpty(item.ImageName, item.ImageRef, item.SubjectID)
		location := strings.Trim(strings.Join(nonEmpty(item.Provider, item.Location, item.Instance), "/"), "/")
		fmt.Printf("- %s stream=%s status=%s progress=%d subject=%s label=%s location=%s\n", item.ID, item.Stream, item.Status, item.Progress, item.SubjectID, label, location)
		if duration := taskDuration(item); duration != "" {
			fmt.Printf("  duration=%s\n", duration)
		}
		if timing := taskTimingSummary(item); timing != "" {
			fmt.Printf("  timing=%s\n", timing)
		}
		if item.ErrorMessage != "" {
			fmt.Printf("  error=%s\n", item.ErrorMessage)
		}
	}
}

func printImageSources(items []imageSource) {
	if len(items) == 0 {
		fmt.Println("image sources: none")
		return
	}
	fmt.Println("image sources:")
	for _, item := range items {
		fmt.Printf("- phase=%s instance=%s image=%s found=%t content=%t source=%s digest=%s blobs=%d\n", item.Phase, item.Instance, item.ImageRef, item.Found, item.HasContent, item.Source, item.TargetDigest, item.Descriptors)
		if item.Error != "" {
			fmt.Printf("  error=%s\n", item.Error)
		}
	}
}

func printCleanup(items []summaryCleanup) {
	if len(items) == 0 {
		fmt.Println("cleanup: none")
		return
	}
	fmt.Println("cleanup:")
	for _, item := range items {
		duration := ""
		if item.DurationMs > 0 {
			duration = fmt.Sprintf(" duration=%s", formatDurationMs(item.DurationMs))
		}
		fmt.Printf("- %s %s provider=%s location=%s status=%s%s\n", item.ResourceType, item.Name, item.Provider, item.Location, item.Status, duration)
		if item.Error != "" {
			fmt.Printf("  error=%s\n", item.Error)
		}
	}
}

func formatDurationMs(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	seconds := ms / 1000
	remainMs := ms % 1000
	if seconds < 60 {
		if remainMs == 0 {
			return fmt.Sprintf("%ds", seconds)
		}
		return fmt.Sprintf("%d.%03ds", seconds, remainMs)
	}
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

func taskDuration(item summaryTask) string {
	startedAt, ok := parseSummaryTime(item.StartedAt)
	if !ok {
		return ""
	}
	finishedAt, ok := parseSummaryTime(item.FinishedAt)
	if !ok || finishedAt.Before(startedAt) {
		return ""
	}
	return formatDuration(finishedAt.Sub(startedAt))
}

func taskTimingSummary(item summaryTask) string {
	var parts []string
	if transfer := taskTransferSummary(item); transfer != "" {
		parts = append(parts, transfer)
	}
	switch item.Stream {
	case "provisioners.image.upload":
		parts = append(parts, phaseDuration(item, "setup", item.StartedAt, eventTime(item, "uploading image")))
		parts = append(parts, phaseDuration(item, "provider_phase", eventTime(item, "uploading image"), eventTime(item, "image uploaded")))
		parts = append(parts, phaseDuration(item, "finalize", eventTime(item, "image uploaded"), item.FinishedAt))
	case "instances.image_archive.upload":
		parts = append(parts, phaseDuration(item, "connect", item.StartedAt, eventTime(item, "uploading image archive")))
		parts = append(parts, phaseDuration(item, "import", eventTime(item, "importing image archive"), eventTime(item, "image archive loaded")))
		parts = append(parts, phaseDuration(item, "finalize", eventTime(item, "image archive loaded"), item.FinishedAt))
	}
	parts = nonEmpty(parts...)
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

func phaseDuration(item summaryTask, label string, startValue string, endValue string) string {
	startedAt, ok := parseSummaryTime(startValue)
	if !ok {
		return ""
	}
	finishedAt, ok := parseSummaryTime(endValue)
	if !ok || finishedAt.Before(startedAt) {
		return ""
	}
	return fmt.Sprintf("%s=%s", label, formatDuration(finishedAt.Sub(startedAt)))
}

type taskProgressPoint struct {
	at             time.Time
	bytesUploaded  uint64
	totalBytes     uint64
	bytesPerSecond uint64
}

func taskTransferSummary(item summaryTask) string {
	points := taskProgressPoints(item)
	if len(points) == 0 {
		return ""
	}
	sort.SliceStable(points, func(i, j int) bool {
		return points[i].at.Before(points[j].at)
	})
	first := points[0]
	last := points[0]
	for _, point := range points {
		if point.bytesUploaded < first.bytesUploaded || (point.bytesUploaded == 0 && first.bytesUploaded != 0) {
			first = point
		}
		if point.bytesUploaded > last.bytesUploaded {
			last = point
		}
	}
	total := maxUint64(last.totalBytes, item.ArchiveSizeBytes, item.BytesUploaded)
	transferred := last.bytesUploaded
	if transferred == 0 {
		transferred = item.BytesUploaded
	}
	if transferred == 0 && total == 0 {
		return ""
	}
	duration := last.at.Sub(first.at)
	var parts []string
	if duration > 0 {
		parts = append(parts, "transfer="+formatDuration(duration))
	}
	if total > 0 {
		parts = append(parts, fmt.Sprintf("bytes=%s/%s", formatBytes(transferred), formatBytes(total)))
	} else {
		parts = append(parts, fmt.Sprintf("bytes=%s", formatBytes(transferred)))
	}
	throughput := last.bytesPerSecond
	if throughput == 0 && duration > 0 && last.bytesUploaded >= first.bytesUploaded {
		throughput = uint64(float64(last.bytesUploaded-first.bytesUploaded) / duration.Seconds())
	}
	if throughput > 0 {
		parts = append(parts, fmt.Sprintf("throughput=%s/s", formatBytes(throughput)))
	}
	return strings.Join(parts, " ")
}

func taskProgressPoints(item summaryTask) []taskProgressPoint {
	var points []taskProgressPoint
	for _, event := range item.Events {
		if point, ok := taskProgressPointFromDetails(event.CreatedAt, event.DetailsJSON); ok {
			points = append(points, point)
		}
	}
	for _, update := range item.Updates {
		if point, ok := taskProgressPointFromDetails(update.CreatedAt, update.DetailsJSON); ok {
			points = append(points, point)
		}
	}
	return points
}

func taskProgressPointFromDetails(createdAt string, detailsJSON string) (taskProgressPoint, bool) {
	at, ok := parseSummaryTime(createdAt)
	if !ok {
		return taskProgressPoint{}, false
	}
	bytesUploaded, hasBytes := detailUint64(detailsJSON, "bytes_uploaded")
	totalBytes, hasTotal := detailUint64(detailsJSON, "archive_size_bytes")
	bytesPerSecond, _ := detailUint64(detailsJSON, "bytes_per_second")
	if !hasBytes && !hasTotal {
		return taskProgressPoint{}, false
	}
	return taskProgressPoint{
		at:             at,
		bytesUploaded:  bytesUploaded,
		totalBytes:     totalBytes,
		bytesPerSecond: bytesPerSecond,
	}, true
}

func eventTime(item summaryTask, message string) string {
	for _, event := range item.Events {
		if event.Message == message && strings.TrimSpace(event.CreatedAt) != "" {
			return event.CreatedAt
		}
	}
	for _, update := range item.Updates {
		if update.Message == message && strings.TrimSpace(update.CreatedAt) != "" {
			return update.CreatedAt
		}
	}
	return ""
}

func parseSummaryTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed, err == nil
}

func formatDuration(duration time.Duration) string {
	if duration < 0 {
		return ""
	}
	if duration < time.Millisecond {
		return duration.String()
	}
	return duration.Round(time.Millisecond).String()
}

func formatBytes(bytes uint64) string {
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

func detailUint64(detailsJSON string, key string) (uint64, bool) {
	var details map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(detailsJSON)), &details); err != nil {
		return 0, false
	}
	raw, found := details[key]
	if !found {
		return 0, false
	}
	var value uint64
	if err := json.Unmarshal(raw, &value); err == nil {
		return value, true
	}
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil && number >= 0 {
		return uint64(number), true
	}
	return 0, false
}

func maxUint64(values ...uint64) uint64 {
	var max uint64
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func validateSummary(summary runSummary) []string {
	var failures []string
	if summary.Status != "passed" {
		failures = append(failures, fmt.Sprintf("run status is %q", summary.Status))
	}
	for _, task := range summary.Tasks {
		if task.Status != "succeeded" {
			failures = append(failures, fmt.Sprintf("task %s status is %q", task.ID, task.Status))
		}
		if task.Stream == "instances.image_archive.upload" && task.ArchiveSizeBytes > 0 && task.BytesUploaded != task.ArchiveSizeBytes {
			failures = append(failures, fmt.Sprintf("task %s uploaded %d of %d bytes", task.ID, task.BytesUploaded, task.ArchiveSizeBytes))
		}
	}
	for _, source := range summary.ImageSources {
		if source.Error != "" {
			failures = append(failures, fmt.Sprintf("image source %s/%s recorded error: %s", source.Instance, source.ImageRef, source.Error))
			continue
		}
		if !source.Found || !source.HasContent {
			failures = append(failures, fmt.Sprintf("image source %s/%s found=%t has_content=%t", source.Instance, source.ImageRef, source.Found, source.HasContent))
		}
	}
	for _, cleanup := range summary.Cleanup {
		if cleanup.Status == "failed" {
			failures = append(failures, fmt.Sprintf("cleanup %s %s failed: %s", cleanup.ResourceType, cleanup.Name, cleanup.Error))
		}
	}
	sort.Strings(failures)
	return failures
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}
