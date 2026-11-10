package cockpit

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"":          0,
		"P1D":       24 * time.Hour,
		"PT12H":     12 * time.Hour,
		"P1DT6H":    30 * time.Hour,
		"PT90M":     90 * time.Minute,
		"P2W":       14 * 24 * time.Hour,
		"P1M":       30 * 24 * time.Hour,
		"PT1H30M5S": time.Hour + 30*time.Minute + 5*time.Second,
	}
	for in, want := range cases {
		got, err := ParseDuration(in)
		if err != nil || got != want {
			t.Errorf("ParseDuration(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"P", "PT", "1D", "P1H", "PxD", "P1.5D"} {
		if _, err := ParseDuration(bad); err == nil {
			t.Errorf("ParseDuration(%q): want error", bad)
		}
	}
}

func at(s string) *time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return &t
}

func TestStateOfSource(t *testing.T) {
	now := *at("2026-11-09T12:00:00Z")
	day := 24 * time.Hour
	cases := []struct {
		name              string
		activated, latest *time.Time
		threshold         time.Duration
		want              string
	}{
		{"not connected", nil, nil, day, SourceMissing},
		{"just activated", at("2026-11-09T10:00:00Z"), nil, day, SourceWaiting},
		{"activated long ago, nothing received", at("2026-11-07T10:00:00Z"), nil, day, SourceLate},
		{"recent upload", at("2026-11-01T10:00:00Z"), at("2026-11-09T11:50:00Z"), day, SourceOK},
		{"old upload", at("2026-11-01T10:00:00Z"), at("2026-11-08T11:00:00Z"), day, SourceLate},
		{"old upload, no threshold", at("2026-11-01T10:00:00Z"), at("2026-10-01T11:00:00Z"), 0, SourceOK},
	}
	for _, c := range cases {
		if got := StateOfSource(c.activated, c.latest, c.threshold, now); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}

func TestStateOfAccount(t *testing.T) {
	act := at("2026-11-01T10:00:00Z")
	if got := StateOfAccount(nil, nil); got != AccountInvited {
		t.Errorf("not activated: got %s", got)
	}
	if got := StateOfAccount(act, []SourceStatus{{State: SourceOK}, {State: SourceWaiting}}); got != AccountActive {
		t.Errorf("all fine: got %s", got)
	}
	if got := StateOfAccount(act, []SourceStatus{{State: SourceOK}, {State: SourceMissing}}); got != AccountIncomplete {
		t.Errorf("missing source: got %s", got)
	}
	if got := StateOfAccount(act, []SourceStatus{{State: SourceMissing}, {State: SourceLate}}); got != AccountAttention {
		t.Errorf("late source: got %s", got)
	}
}

func TestStateOfCampaign(t *testing.T) {
	now := *at("2026-11-09T12:00:00Z")
	if got := StateOfCampaign(nil, nil, now); got != CampaignPlanned {
		t.Errorf("no times: got %s", got)
	}
	if got := StateOfCampaign(at("2026-12-01T00:00:00Z"), nil, now); got != CampaignPlanned {
		t.Errorf("future start: got %s", got)
	}
	if got := StateOfCampaign(at("2026-10-01T00:00:00Z"), at("2027-04-01T00:00:00Z"), now); got != CampaignRunning {
		t.Errorf("running: got %s", got)
	}
	if got := StateOfCampaign(at("2026-01-01T00:00:00Z"), at("2026-04-01T00:00:00Z"), now); got != CampaignEnded {
		t.Errorf("ended: got %s", got)
	}
}
