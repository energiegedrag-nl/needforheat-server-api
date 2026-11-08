package energyquerytype

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Rounding rules per property name, applied by the app before an energy query result leaves the
// phone. Results computed from public data (e.g. a heat performance signature from the address)
// are quasi-identifiers; a campaign chooses the rules beforehand, for example with AnonyMate's
// rounding analysis. Stored as JSON text, for example:
//
//	{
//	  "heat_tr_building_cond__W_K_1": {"step": 25, "min": 50, "max": 600},
//	  "th_mass_building__Wh_K_1": {"bins": [{"lt": 10000, "value": 7500}, {"lt": 20000, "value": 15000}, {"value": 25000}]}
//	}
//
// A step rule rounds to a multiple of step, after clamping to [min, max] (optional), so the tails
// end up in one value each. A bins rule maps a value to the first bin whose "lt" (exclusive upper
// bound) it is below and uploads that bin's representative "value"; the last bin has no "lt".
type RoundingRule struct {
	Step *float64 `json:"step,omitempty"`
	Min  *float64 `json:"min,omitempty"`
	Max  *float64 `json:"max,omitempty"`
	Bins []Bin    `json:"bins,omitempty"`
}

// A Bin of a bins rule.
type Bin struct {
	Lt    *float64 `json:"lt,omitempty"`
	Value float64  `json:"value"`
}

// ErrInvalidRounding is returned for rounding JSON that cannot be used.
var ErrInvalidRounding = errors.New("invalid rounding")

// ValidateRounding checks rounding JSON (empty is allowed: the app then uses its defaults).
func ValidateRounding(text string) error {
	if text == "" {
		return nil
	}
	var rules map[string]RoundingRule
	if err := json.Unmarshal([]byte(text), &rules); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRounding, err)
	}
	for name, r := range rules {
		hasStep := r.Step != nil
		hasBins := len(r.Bins) > 0
		switch {
		case hasStep == hasBins:
			return fmt.Errorf("%w: %s needs either step or bins", ErrInvalidRounding, name)
		case hasStep && *r.Step <= 0:
			return fmt.Errorf("%w: %s step must be positive", ErrInvalidRounding, name)
		case hasStep && r.Min != nil && r.Max != nil && *r.Min > *r.Max:
			return fmt.Errorf("%w: %s min is above max", ErrInvalidRounding, name)
		case hasBins:
			for i, b := range r.Bins {
				last := i == len(r.Bins)-1
				if last != (b.Lt == nil) {
					return fmt.Errorf("%w: %s: only the last bin has no lt", ErrInvalidRounding, name)
				}
				if i > 0 && !last && *b.Lt <= *r.Bins[i-1].Lt {
					return fmt.Errorf("%w: %s: bin bounds must increase", ErrInvalidRounding, name)
				}
			}
		}
	}
	return nil
}
