package inventory

import (
	"errors"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	for in, want := range map[string]string{
		"aa:bb:cc:dd:ee:ff": "AA:BB:CC:DD:EE:FF",
		"AA-BB-CC-DD-EE-FF": "AA:BB:CC:DD:EE:FF",
		"aabbccddeeff":      "AA:BB:CC:DD:EE:FF",
	} {
		got, err := NormalizeMAC(in)
		if err != nil || got != want {
			t.Errorf("NormalizeMAC(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "aa:bb:cc:dd:ee", "gg:bb:cc:dd:ee:ff", "aa:bb:cc:dd:ee:ff:00"} {
		if _, err := NormalizeMAC(in); !errors.Is(err, ErrInvalidMAC) {
			t.Errorf("NormalizeMAC(%q) accepted", in)
		}
	}
}

func TestNameMatchesMAC(t *testing.T) {
	if !NameMatchesMAC("1A2B-DDEEFF", "AA:BB:CC:DD:EE:FF") {
		t.Error("matching name rejected")
	}
	if NameMatchesMAC("1A2B-DDEEF0", "AA:BB:CC:DD:EE:FF") || NameMatchesMAC("1a2b-ddeeff", "AA:BB:CC:DD:EE:FF") {
		t.Error("wrong name accepted")
	}
}

func TestNext(t *testing.T) {
	cases := []struct {
		state, event string
		force        bool
		want         string
		err          error
	}{
		{"", EventFlash, false, Flashed, nil},
		{Erased, EventFlash, false, Flashed, nil},
		{Assigned, EventFlash, false, "", ErrStillAssigned},
		{Shipped, EventFlash, true, Flashed, nil},
		{Flashed, EventAssign, false, Assigned, nil},
		{Assigned, EventAssign, false, Assigned, nil},
		{Shipped, EventAssign, false, "", ErrInvalidTransition},
		{Assigned, EventShip, false, Shipped, nil},
		{Flashed, EventShip, false, "", ErrInvalidTransition},
		{Shipped, EventReturn, false, Returned, nil},
		{Returned, EventErase, false, Erased, nil},
		{"", EventErase, false, "", ErrInvalidTransition},
		{Shipped, EventNote, false, Shipped, nil},
		{Flashed, "steal", false, "", ErrUnknownEvent},
	}
	for _, c := range cases {
		got, err := Next(c.state, c.event, c.force)
		if got != c.want || !errors.Is(err, c.err) {
			t.Errorf("Next(%q, %q, %v) = %q, %v; want %q, %v", c.state, c.event, c.force, got, err, c.want, c.err)
		}
	}
}
