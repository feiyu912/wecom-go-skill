package wecom

import (
	"testing"
	"time"
)

func TestParseTimeToEpochRFC3339(t *testing.T) {
	got, err := ParseTimeToEpoch("2026-04-14T15:00:00+08:00")
	if err != nil {
		t.Fatalf("ParseTimeToEpoch returned error: %v", err)
	}

	want := time.Date(2026, 4, 14, 15, 0, 0, 0, time.FixedZone("CST", 8*3600)).Unix()
	if got != want {
		t.Fatalf("unexpected epoch: got %d want %d", got, want)
	}
}

func TestDefaultMeetingStartRoundsToNextHour(t *testing.T) {
	now := time.Date(2026, 5, 7, 9, 23, 10, 0, time.Local)
	got := DefaultMeetingStart(now)
	want := time.Date(2026, 5, 7, 10, 0, 0, 0, time.Local).Unix()
	if got != want {
		t.Fatalf("unexpected default meeting start: got %d want %d", got, want)
	}
}
