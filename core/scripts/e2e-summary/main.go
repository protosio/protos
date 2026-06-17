package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
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
	ID               string `json:"id"`
	Stream           string `json:"stream"`
	SubjectType      string `json:"subject_type"`
	SubjectID        string `json:"subject_id"`
	Status           string `json:"status"`
	Message          string `json:"message"`
	ErrorMessage     string `json:"error_message"`
	Progress         int32  `json:"progress"`
	Provider         string `json:"provider"`
	Location         string `json:"location"`
	ImageName        string `json:"image_name"`
	ImageID          string `json:"image_id"`
	Instance         string `json:"instance"`
	ImageRef         string `json:"image_ref"`
	TargetDigest     string `json:"target_digest"`
	Platform         string `json:"platform"`
	BytesUploaded    uint64 `json:"bytes_uploaded"`
	ArchiveSizeBytes uint64 `json:"archive_size_bytes"`
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
		fmt.Printf("- %s %s provider=%s location=%s status=%s\n", item.ResourceType, item.Name, item.Provider, item.Location, item.Status)
		if item.Error != "" {
			fmt.Printf("  error=%s\n", item.Error)
		}
	}
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
