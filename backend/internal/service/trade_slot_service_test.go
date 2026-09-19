package service

import (
	"log/slog"
	"testing"
	"time"

	"github.com/lp/campus-market/internal/dto"
)

func TestTradeSlotBuildSlots(t *testing.T) {
	slotsSvc := NewTradeSlotService(newFakeSlotRepo(newFakeDB()), slog.Default())
	now := time.Now()

	t.Run("accepts future half-hour slots", func(t *testing.T) {
		future := now.Add(4 * time.Hour).Truncate(time.Hour)
		got, err := slotsSvc.BuildSlots([]dto.CreateSlotInput{{StartTime: future}}, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 || got[0].EndTime.Sub(got[0].StartTime) != 30*time.Minute {
			t.Fatalf("expected one 30-minute slot")
		}
	})

	t.Run("rejects quarter-past start", func(t *testing.T) {
		bad := now.Truncate(time.Hour).Add(3 * time.Hour).Add(15 * time.Minute)
		if _, err := slotsSvc.BuildSlots([]dto.CreateSlotInput{{StartTime: bad}}, now); err == nil {
			t.Fatalf("expected error for off-grid start")
		}
	})

	t.Run("rejects past start", func(t *testing.T) {
		past := now.Add(-2 * time.Hour).Truncate(time.Hour)
		if _, err := slotsSvc.BuildSlots([]dto.CreateSlotInput{{StartTime: past}}, now); err == nil {
			t.Fatalf("expected error for past start")
		}
	})

	t.Run("rejects duplicates", func(t *testing.T) {
		at := now.Add(5 * time.Hour).Truncate(time.Hour)
		if _, err := slotsSvc.BuildSlots([]dto.CreateSlotInput{{StartTime: at}, {StartTime: at}}, now); err == nil {
			t.Fatalf("expected error for duplicate slots")
		}
	})

	t.Run("rejects over cap", func(t *testing.T) {
		base := now.Add(24 * time.Hour).Truncate(time.Hour)
		in := make([]dto.CreateSlotInput, 0, 41)
		for i := 0; i < 41; i++ {
			in = append(in, dto.CreateSlotInput{StartTime: base.Add(time.Duration(i) * 30 * time.Minute)})
		}
		if _, err := slotsSvc.BuildSlots(in, now); err == nil {
			t.Fatalf("expected error when exceeding slot cap")
		}
	})

	t.Run("sorts slots ascending", func(t *testing.T) {
		a := now.Add(26 * time.Hour).Truncate(time.Hour)
		b := now.Add(24 * time.Hour).Truncate(time.Hour)
		got, err := slotsSvc.BuildSlots([]dto.CreateSlotInput{{StartTime: a}, {StartTime: b}}, now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got[0].StartTime.Before(got[1].StartTime) {
			t.Fatalf("slots must be sorted ascending")
		}
	})
}
