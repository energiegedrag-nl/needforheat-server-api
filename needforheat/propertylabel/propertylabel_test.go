package propertylabel

import (
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	got, err := PropertyLabel{PropertyName: " heat_tr_building_cond__W_K_1 ", Language: "nl_nl", Label: " Warmteverlies "}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	want := PropertyLabel{PropertyName: "heat_tr_building_cond__W_K_1", Language: "nl-NL", Label: "Warmteverlies"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if got, _ := (PropertyLabel{PropertyName: "x"}).Normalize(); got.Language != "nl-NL" {
		t.Errorf("empty language: got %q, want nl-NL", got.Language)
	}
	for _, bad := range []PropertyLabel{
		{PropertyName: "", Label: "a"},
		{PropertyName: strings.Repeat("p", MaxPropertyName+1), Label: "a"},
		{PropertyName: "x", Label: strings.Repeat("é", MaxLabel+1)},
		{PropertyName: "x", Language: "dutch", Label: "a"},
	} {
		if _, err := bad.Normalize(); err == nil {
			t.Errorf("%+v: want an error", bad)
		}
	}
}
