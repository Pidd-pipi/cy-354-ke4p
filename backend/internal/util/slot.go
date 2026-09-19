package util

import (
	"errors"
	"time"
)

// ErrSlotTime is returned when a meetup slot timestamp cannot be parsed.
var ErrSlotTime = errors.New("invalid slot time")

// ParseSlotStart parses a meetup slot start timestamp. Frontend submits
// RFC3339 strings such as "2026-09-20T14:30:00+08:00".
func ParseSlotStart(raw string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, ErrSlotTime
	}
	return t, nil
}

// IsHalfHourAligned reports whether t is exactly on an :00 or :30 boundary.
func IsHalfHourAligned(t time.Time) bool {
	m, s := t.Minute(), t.Second()
	return m%30 == 0 && s == 0 && t.Nanosecond() == 0
}

// ValidateSlotTimes normalizes, de-duplicates and validates meetup slot
// timestamps: non-empty, within the allowed future window and half-hour
// aligned. Times are compared by instant; the returned slice is sorted
// chronologically.
func ValidateSlotTimes(raw []string, now time.Time, advanceDays, maxCount int) ([]time.Time, error) {
	seen := make(map[int64]struct{}, len(raw))
	out := make([]time.Time, 0, len(raw))
	latest := now.Add(time.Duration(advanceDays) * 24 * time.Hour)
	for _, s := range raw {
		if s == "" {
			return nil, ErrSlotTime
		}
		t, err := ParseSlotStart(s)
		if err != nil {
			return nil, err
		}
		if !t.After(now) {
			return nil, ErrSlotTime
		}
		if t.After(latest) {
			return nil, ErrSlotTime
		}
		if !IsHalfHourAligned(t) {
			return nil, ErrSlotTime
		}
		if _, ok := seen[t.Unix()]; ok {
			continue
		}
		seen[t.Unix()] = struct{}{}
		out = append(out, t)
	}
	if len(out) > maxCount {
		return nil, ErrSlotTime
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Before(out[i]) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}
