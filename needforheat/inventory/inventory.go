// Package inventory holds the device inventory of the NeedForHeat Cockpit flash station (energiegedrag-cockpit docs/ontwerp.md, § 6): every
// ESP32 that was flashed in the lab, keyed by its MAC address, with its life cycle and history.
//
// Life cycle: flashed → assigned (to a participant account) → shipped → returned → erased → flashed again.
// A returned device keeps its account: after a repair it can be shipped to the same participant again. To hand it
// to another participant it is released: the device row the app activated is archived (renamed to
// <name>~<yyyymmdd-hhmmss>, released_at set; its uploads stay with the previous participant), so the name is free
// for the next account, and the device is in stock (flashed) again.
// Whether a device is in use follows from the device table (a device with the same name was activated by the app);
// the inventory does not duplicate that.
package inventory

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Device states.
const (
	Flashed  = "flashed"  // firmware written in the lab, in stock
	Assigned = "assigned" // reserved for a participant account
	Shipped  = "shipped"  // handed out or sent to the participant
	Returned = "returned" // back in the lab, not erased yet
	Erased   = "erased"   // flash erased; ready to be flashed again
)

// Events. Each one is stored in the history of a device.
const (
	EventFlash  = "flash"
	EventAssign = "assign"
	EventShip   = "ship"
	EventReturn = "return"
	EventErase  = "erase"
	EventNote   = "note"
	// EventRelease archives the activated device row and puts the device in stock for another participant.
	EventRelease = "release"
)

var (
	ErrInvalidMAC        = errors.New("invalid MAC address")
	ErrUnknownEvent      = errors.New("unknown event")
	ErrInvalidTransition = errors.New("event not allowed in this state")
	// ErrStillAssigned is returned when a device that is assigned, shipped or returned (still linked to its
	// participant) is flashed without force.
	ErrStillAssigned = errors.New("device is still assigned to a participant")
	// ErrNoAccount is returned when a returned device without an account is shipped (nobody to send it back to).
	ErrNoAccount = errors.New("device has no account to send it back to")
)

// Device is one physical device in the inventory.
type Device struct {
	ID              uint    `json:"id"`
	MAC             string  `json:"mac"`
	Chip            string  `json:"chip"`
	DeviceType      string  `json:"device_type"`
	Name            string  `json:"name"` // name in the provisioning QR code, e.g. 1A2B-3C4D5E; matches device.name after activation
	Firmware        string  `json:"firmware"`
	FirmwareVersion string  `json:"firmware_version"`
	State           string  `json:"state"`
	AccountID       *uint   `json:"account_id"`
	Note            string  `json:"note"`
	CreatedAt       int64   `json:"created_at"`
	UpdatedAt       int64   `json:"updated_at"`
	InUse           *InUse  `json:"in_use"`           // set when the app activated a device with this name
	Events          []Event `json:"events,omitempty"` // only when one device is requested
}

// InUse tells which account activated the device, according to the device table.
type InUse struct {
	AccountID   uint   `json:"account_id"`
	CampaignID  uint   `json:"campaign_id"`
	ActivatedAt *int64 `json:"activated_at"`
}

// Event is one step in the history of a device.
type Event struct {
	ID        uint   `json:"id"`
	At        int64  `json:"at"`
	Event     string `json:"event"`
	State     string `json:"state"` // state after the event
	AccountID *uint  `json:"account_id"`
	Details   string `json:"details"`
}

// Flash is what the flash station reports after flashing and verifying a device.
type Flash struct {
	MAC             string `json:"mac"`
	Chip            string `json:"chip"`
	DeviceType      string `json:"device_type"`
	Name            string `json:"name"`
	Firmware        string `json:"firmware"`
	FirmwareVersion string `json:"firmware_version"`
	Force           bool   `json:"force"` // flash a device that is still assigned, shipped or returned
	// Release: the device goes to another participant. The device row the app activated is archived, the account
	// link is removed. Implies Force.
	Release bool `json:"release"`
	// KeepAccount: repaired for the same participant. The account link stays and the device is returned (in the
	// lab), ready to be shipped back; the participant pairs it again with the app. Implies Force.
	KeepAccount bool `json:"keep_account"`
}

// Change is an event on an existing device.
type Change struct {
	Event     string `json:"event"`
	AccountID *uint  `json:"account_id"` // for assign
	Details   string `json:"details"`
}

var macPattern = regexp.MustCompile(`^[0-9A-F]{2}(:[0-9A-F]{2}){5}$`)

// NormalizeMAC returns the MAC address as AA:BB:CC:DD:EE:FF. It accepts ':', '-' or no separators, any case.
func NormalizeMAC(mac string) (string, error) {
	s := strings.ToUpper(strings.NewReplacer(":", "", "-", "", " ", "").Replace(mac))
	if len(s) != 12 {
		return "", ErrInvalidMAC
	}
	out := fmt.Sprintf("%s:%s:%s:%s:%s:%s", s[0:2], s[2:4], s[4:6], s[6:8], s[8:10], s[10:12])
	if !macPattern.MatchString(out) {
		return "", ErrInvalidMAC
	}
	return out, nil
}

var namePattern = regexp.MustCompile(`^[0-9A-F]{4}-[0-9A-F]{6}$`)

// ValidName reports whether name looks like a device service name (%04X-%02X%02X%02X: device type hash and the
// last three bytes of the MAC address).
func ValidName(name string) bool {
	return namePattern.MatchString(name)
}

// NameMatchesMAC reports whether the last six hex digits of the name are the last three bytes of the MAC address.
func NameMatchesMAC(name, mac string) bool {
	return ValidName(name) && len(mac) == 17 && name[5:] == strings.ReplaceAll(mac[9:], ":", "")
}

// Next returns the state after event, or an error if the event is not allowed in state. An empty state means the
// device is not in the inventory yet.
func Next(state, event string, force bool) (string, error) {
	switch event {
	case EventFlash:
		if KeepsAccount(state) && !force {
			return "", ErrStillAssigned
		}
		return Flashed, nil
	case EventAssign:
		if state == Flashed || state == Assigned {
			return Assigned, nil
		}
	case EventShip:
		// From returned: back to the same participant after a repair.
		if state == Assigned || state == Returned {
			return Shipped, nil
		}
	case EventReturn:
		if state == Assigned || state == Shipped {
			return Returned, nil
		}
	case EventErase:
		if state != "" {
			return Erased, nil
		}
	case EventRelease:
		// Only for a device in the lab: a shipped device is returned first.
		switch state {
		case Flashed, Assigned, Returned:
			return Flashed, nil
		case Erased:
			return Erased, nil
		}
	case EventNote:
		if state != "" {
			return state, nil
		}
	default:
		return "", ErrUnknownEvent
	}
	return "", ErrInvalidTransition
}

// KeepsAccount reports whether a device in this state stays linked to its account.
func KeepsAccount(state string) bool {
	return state == Assigned || state == Shipped || state == Returned
}

// ArchivedName is the name of a released device row: the original name plus the moment of release, so the
// original name is free again and several releases of the same device stay apart.
func ArchivedName(name string, at time.Time) string {
	return name + "~" + at.UTC().Format("20060102-150405")
}
