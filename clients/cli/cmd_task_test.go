package main

import "testing"

func TestTaskDetailsSummaryFormatsTransferDetails(t *testing.T) {
	summary := taskDetailsSummary(`{"bytes_uploaded":5242880,"archive_size_bytes":10485760,"percent":50}`, "2.0MiB/s")
	want := "(5.0MiB/10.0MiB, 2.0MiB/s, 50%)"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}

func TestTaskResultSummaryPrefersKnownFields(t *testing.T) {
	summary := taskResultSummary(`{"image_ref":"docker.io/protosio/probe:latest","target_digest":"sha256:abc","bytes_uploaded":6741473}`)
	want := "image_ref=docker.io/protosio/probe:latest target_digest=sha256:abc bytes_uploaded=6.4MiB"
	if summary != want {
		t.Fatalf("summary = %q, want %q", summary, want)
	}
}
