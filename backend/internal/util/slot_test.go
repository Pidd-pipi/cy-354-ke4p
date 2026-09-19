package util

import (
	"testing"
	"time"
)

func TestIsHalfHourAligned(t *testing.T) {
	base := time.Date(2026, 9, 20, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		add  time.Duration
		want bool
	}{
		{name: "on the hour", add: 0, want: true},
		{name: "half past", add: 30 * time.Minute, want: true},
		{name: "quarter past", add: 15 * time.Minute, want: false},
		{name: "ten to", add: 50 * time.Minute, want: false},
		{name: "with seconds", add: time.Second, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsHalfHourAligned(base.Add(tt.add)); got != tt.want {
				t.Fatalf("IsHalfHourAligned(%v) = %v, want %v", base.Add(tt.add), got, tt.want)
			}
		})
	}
}

func TestValidateSlotTimes(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	mk := func(day, hour, minute int) string {
		return time.Date(2026, 9, day, hour, minute, 0, 0, time.UTC).Format(time.RFC3339)
	}
	tests := []struct {
		name    string
		raw     []string
		wantErr bool
		wantN   int
	}{
		{name: "empty allowed", raw: nil, wantErr: false, wantN: 0},
		{name: "two unique sorted", raw: []string{mk(20, 14, 30), mk(20, 10, 0)}, wantErr: false, wantN: 2},
		{name: "duplicates deduped", raw: []string{mk(20, 10, 0), mk(20, 10, 0)}, wantErr: false, wantN: 1},
		{name: "past rejected", raw: []string{mk(18, 10, 0)}, wantErr: true},
		{name: "unaligned rejected", raw: []string{mk(20, 10, 7)}, wantErr: true},
		{name: "garbage rejected", raw: []string{"not-a-time"}, wantErr: true},
		{name: "too far ahead rejected", raw: []string{time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateSlotTimes(tt.raw, now, 30, 30)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got %v", got)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tt.wantErr && len(got) != tt.wantN {
				t.Fatalf("expected %d slots, got %d", tt.wantN, len(got))
			}
		})
	}
}

func TestValidateSlotTimesSortsChronologically(t *testing.T) {
	now := time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC)
	mk := func(day, hour, minute int) string {
		return time.Date(2026, 9, day, hour, minute, 0, 0, time.UTC).Format(time.RFC3339)
	}
	got, err := ValidateSlotTimes([]string{mk(21, 9, 0), mk(20, 9, 0)}, now, 30, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got[0].Before(got[1]) {
		t.Fatalf("expected sorted slots, got %v then %v", got[0], got[1])
	}
}
