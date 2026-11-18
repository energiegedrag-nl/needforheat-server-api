package campaign

import "testing"

func TestNormalizeLanguage(t *testing.T) {
	ok := map[string]string{"": "nl-NL", "nl-NL": "nl-NL", "nl_nl": "nl-NL", "en-us": "en-US", "de": "de", "fry-NL": "fry-NL"}
	for in, want := range ok {
		got, err := NormalizeLanguage(in)
		if err != nil || got != want {
			t.Errorf("NormalizeLanguage(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"NL", "nl-NLD", "dutch", "nl-NL; DROP", "nl-", "-NL"} {
		if _, err := NormalizeLanguage(in); err == nil {
			t.Errorf("NormalizeLanguage(%q): want an error", in)
		}
	}
}
