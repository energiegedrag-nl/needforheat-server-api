package displaytime

import (
	"testing"
	"time"
)

func TestAmsterdam(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		now        time.Time
		offset     int
		nextChange time.Time
		nextOffset int
	}{
		// Winter: next change is summer time on Sunday 28 March 2027, 01:00 UTC.
		{time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC), 3600, time.Date(2027, 3, 28, 1, 0, 0, 0, time.UTC), 7200},
		// Summer: next change is winter time on Sunday 25 October 2026, 01:00 UTC.
		{time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC), 7200, time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC), 3600},
		// Exactly at a change the new offset applies.
		{time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC), 3600, time.Date(2027, 3, 28, 1, 0, 0, 0, time.UTC), 7200},
		// Far ahead (beyond the explicit transitions in tzdata, from the zone's rule).
		{time.Date(2040, 1, 15, 0, 0, 0, 0, time.UTC), 3600, time.Date(2040, 3, 25, 1, 0, 0, 0, time.UTC), 7200},
	}
	for _, c := range cases {
		got := At(loc, c.now)
		if got.UTCOffset != c.offset || got.NextChange != c.nextChange.Unix() || got.NextUTCOffset == nil || *got.NextUTCOffset != c.nextOffset {
			next := -1
			if got.NextUTCOffset != nil {
				next = *got.NextUTCOffset
			}
			t.Errorf("At(%v) = %d, %v, %d; want %d, %v, %d", c.now, got.UTCOffset,
				time.Unix(got.NextChange, 0).UTC(), next, c.offset, c.nextChange, c.nextOffset)
		}
	}
}

func TestNoTransitions(t *testing.T) {
	got := At(time.UTC, time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC))
	if got.UTCOffset != 0 || got.NextChange != 0 || got.NextUTCOffset != nil {
		t.Errorf("UTC: got %+v", got)
	}
}
