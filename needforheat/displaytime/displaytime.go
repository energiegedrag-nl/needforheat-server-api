// Package displaytime tells devices the offset of local time from UTC, so they can show the time to people
// without a time zone database. Devices keep their clock and all timestamps in UTC.
package displaytime

import (
	"time"
	_ "time/tzdata" // The distroless image has no zoneinfo; embed it (about 450 kB).
)

// Offset is the local time offset now and, if the zone has transitions, the next change.
// A device uses UTCOffset until NextChange and NextUTCOffset after it.
type Offset struct {
	UTCOffset     int   `json:"utc_offset_s"`
	NextChange    int64 `json:"next_offset_change,omitempty"` // Unix time (UTC) of the next change; absent if none.
	NextUTCOffset *int  `json:"next_utc_offset_s,omitempty"`  // Offset after NextChange; absent if none.
}

// At returns the offset of loc at now.
func At(loc *time.Location, now time.Time) Offset {
	local := now.In(loc)
	_, offset := local.Zone()
	result := Offset{UTCOffset: offset}

	// Zone periods can end without a change of offset (Go splits periods that come from the zone's rule at the
	// start of a year), so walk on until the offset really changes. Ten periods is far more than any real zone needs.
	end := local
	for range 10 {
		_, end = end.ZoneBounds()
		if end.IsZero() {
			return result
		}
		end = end.In(loc)
		if _, next := end.Zone(); next != offset {
			result.NextChange = end.Unix()
			result.NextUTCOffset = &next
			return result
		}
	}
	return result
}
