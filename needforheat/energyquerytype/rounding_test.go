package energyquerytype

import "testing"

func TestValidateRounding(t *testing.T) {
	ok := []string{
		``,
		`{}`,
		`{"heat_tr_building_cond__W_K_1":{"step":25}}`,
		`{"heat_tr_building_cond__W_K_1":{"step":25,"min":50,"max":600}}`,
		`{"aperture_inf__cm2":{"bins":[{"lt":50,"value":25},{"lt":150,"value":100},{"value":200}]}}`,
		`{"x":{"bins":[{"value":1}]}}`,
	}
	for _, s := range ok {
		if err := ValidateRounding(s); err != nil {
			t.Errorf("%s: unexpected error %v", s, err)
		}
	}
	bad := []string{
		`not json`,
		`{"x":{}}`,
		`{"x":{"step":0}}`,
		`{"x":{"step":5,"bins":[{"value":1}]}}`,
		`{"x":{"step":5,"min":10,"max":1}}`,
		`{"x":{"bins":[{"lt":5,"value":1}]}}`,
		`{"x":{"bins":[{"value":1},{"value":2}]}}`,
		`{"x":{"bins":[{"lt":5,"value":1},{"lt":3,"value":2},{"value":9}]}}`,
	}
	for _, s := range bad {
		if err := ValidateRounding(s); err == nil {
			t.Errorf("%s: expected an error", s)
		}
	}
}
