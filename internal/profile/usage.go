package profile

import (
	"bufio"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type UsageSnapshot struct {
	Present                bool
	PrimaryUsedPercent     int
	SecondaryUsedPercent   int
	PrimaryWindowMinutes   int
	SecondaryWindowMinutes int
	PlanType               string
}

// ReadUsageSnapshot reads the most recent token_count snapshot from the local
// Codex session logs under CODEX_HOME. Any unreadable or malformed session data
// is treated as unavailable so callers can surface a degraded but stable view.
func ReadUsageSnapshot(codexHome string) UsageSnapshot {
	sessionsDir := filepath.Join(codexHome, "sessions")
	if _, err := os.Stat(sessionsDir); err != nil {
		return UsageSnapshot{}
	}

	var latest UsageSnapshot
	var latestTimestamp time.Time
	_ = filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return nil
		}
		snapshot, ts, ok := readLatestSnapshotFromFile(path)
		if ok && (!latest.Present || ts.After(latestTimestamp)) {
			latest = snapshot
			latestTimestamp = ts
		}
		return nil
	})
	return latest
}

func readLatestSnapshotFromFile(path string) (UsageSnapshot, time.Time, bool) {
	file, err := os.Open(path)
	if err != nil {
		return UsageSnapshot{}, time.Time{}, false
	}
	defer file.Close()

	var latest UsageSnapshot
	var latestTimestamp time.Time

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) > 0 {
			snapshot, ts, ok := parseUsageSnapshotLine(line)
			if ok && (!latest.Present || ts.After(latestTimestamp)) {
				latest = snapshot
				latestTimestamp = ts
			}
		}
		if readErr != nil {
			break
		}
	}
	return latest, latestTimestamp, latest.Present
}

func parseUsageSnapshotLine(line []byte) (UsageSnapshot, time.Time, bool) {
	line = []byte(strings.TrimSpace(string(line)))
	if len(line) == 0 {
		return UsageSnapshot{}, time.Time{}, false
	}

	var entry struct {
		Timestamp string          `json:"timestamp"`
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &entry); err != nil || entry.Type != "event_msg" {
		return UsageSnapshot{}, time.Time{}, false
	}

	var payload struct {
		Type       string `json:"type"`
		RateLimits struct {
			Primary struct {
				UsedPercent  float64 `json:"used_percent"`
				WindowMinute int     `json:"window_minutes"`
			} `json:"primary"`
			Secondary struct {
				UsedPercent  float64 `json:"used_percent"`
				WindowMinute int     `json:"window_minutes"`
			} `json:"secondary"`
			PlanType string `json:"plan_type"`
		} `json:"rate_limits"`
	}
	if err := json.Unmarshal(entry.Payload, &payload); err != nil || payload.Type != "token_count" {
		return UsageSnapshot{}, time.Time{}, false
	}

	ts, err := time.Parse(time.RFC3339Nano, entry.Timestamp)
	if err != nil {
		return UsageSnapshot{}, time.Time{}, false
	}

	return UsageSnapshot{
		Present:                true,
		PrimaryUsedPercent:     int(math.Round(payload.RateLimits.Primary.UsedPercent)),
		SecondaryUsedPercent:   int(math.Round(payload.RateLimits.Secondary.UsedPercent)),
		PrimaryWindowMinutes:   payload.RateLimits.Primary.WindowMinute,
		SecondaryWindowMinutes: payload.RateLimits.Secondary.WindowMinute,
		PlanType:               payload.RateLimits.PlanType,
	}, ts, true
}
