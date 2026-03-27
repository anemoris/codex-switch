package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadUsageSnapshotReturnsLatestTokenCountEvent(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sessions", "2026", "03", "27", "rollout-1.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}

	data := "" +
		"{\"timestamp\":\"2026-03-27T01:44:33.706Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":1.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":96.0,\"window_minutes\":10080},\"plan_type\":\"team\"}}}\n" +
		"{\"timestamp\":\"2026-03-27T01:44:35.743Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":7.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":2.0,\"window_minutes\":10080},\"plan_type\":\"plus\"}}}\n"
	if err := os.WriteFile(sessionPath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := ReadUsageSnapshot(dir)
	if !snapshot.Present {
		t.Fatal("expected usage snapshot to be present")
	}
	if snapshot.PrimaryUsedPercent != 7 || snapshot.SecondaryUsedPercent != 2 {
		t.Fatalf("unexpected snapshot percentages: %#v", snapshot)
	}
	if snapshot.PrimaryWindowMinutes != 300 || snapshot.SecondaryWindowMinutes != 10080 {
		t.Fatalf("unexpected snapshot windows: %#v", snapshot)
	}
	if snapshot.PlanType != "plus" {
		t.Fatalf("unexpected plan type: %#v", snapshot)
	}
}

func TestReadUsageSnapshotPrefersNewestEventAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	olderPath := filepath.Join(dir, "sessions", "2026", "03", "26", "rollout-older.jsonl")
	newerPath := filepath.Join(dir, "sessions", "2026", "03", "27", "rollout-newer.jsonl")
	for _, path := range []string{olderPath, newerPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	older := "{\"timestamp\":\"2026-03-26T23:59:59Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":3.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":1.0,\"window_minutes\":10080},\"plan_type\":\"plus\"}}}\n"
	newer := "{\"timestamp\":\"2026-03-27T00:00:01Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":5.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":2.0,\"window_minutes\":10080},\"plan_type\":\"pro\"}}}\n"
	if err := os.WriteFile(olderPath, []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newerPath, []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := ReadUsageSnapshot(dir)
	if snapshot.PrimaryUsedPercent != 5 || snapshot.SecondaryUsedPercent != 2 || snapshot.PlanType != "pro" {
		t.Fatalf("expected newest snapshot, got %#v", snapshot)
	}
}

func TestReadUsageSnapshotUsesLatestTimestampAcrossSameDayFiles(t *testing.T) {
	dir := t.TempDir()
	olderByTime := filepath.Join(dir, "sessions", "2026", "03", "27", "z-last-by-name.jsonl")
	newerByTime := filepath.Join(dir, "sessions", "2026", "03", "27", "a-first-by-name.jsonl")
	for _, path := range []string{olderByTime, newerByTime} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	older := "{\"timestamp\":\"2026-03-27T09:00:00Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":11.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":4.0,\"window_minutes\":10080},\"plan_type\":\"plus\"}}}\n"
	newer := "{\"timestamp\":\"2026-03-27T10:00:00Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":22.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":5.0,\"window_minutes\":10080},\"plan_type\":\"pro\"}}}\n"
	if err := os.WriteFile(olderByTime, []byte(older), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newerByTime, []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := ReadUsageSnapshot(dir)
	if snapshot.PrimaryUsedPercent != 22 || snapshot.SecondaryUsedPercent != 5 || snapshot.PlanType != "pro" {
		t.Fatalf("expected newest timestamp across files, got %#v", snapshot)
	}
}

func TestReadUsageSnapshotReturnsZeroValueWhenMissing(t *testing.T) {
	snapshot := ReadUsageSnapshot(t.TempDir())
	if snapshot.Present {
		t.Fatalf("expected missing usage snapshot, got %#v", snapshot)
	}
}

func TestReadUsageSnapshotIgnoresMalformedSessionData(t *testing.T) {
	dir := t.TempDir()
	sessionPath := filepath.Join(dir, "sessions", "2026", "03", "27", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}

	data := "" +
		"not-json\n" +
		"{\"timestamp\":\"bad-time\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":7.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":2.0,\"window_minutes\":10080},\"plan_type\":\"plus\"}}}\n"
	if err := os.WriteFile(sessionPath, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	snapshot := ReadUsageSnapshot(dir)
	if snapshot.Present {
		t.Fatalf("expected malformed data to degrade to unavailable, got %#v", snapshot)
	}
}
