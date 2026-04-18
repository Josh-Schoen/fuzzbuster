package scrub

import (
	"strings"
	"testing"
)

// Golden set for the regex layer. Each case is (input, categories that MUST
// be removed, substrings that MUST remain visible). The LLM layer runs
// separately and has its own fixture.
func TestRegexScrubGoldenSet(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		removed []string
		keeps   []string
	}{
		{
			name:    "phone E164",
			in:      "Call me at +1 612-555-0131",
			removed: []string{CatPhone},
			keeps:   []string{"Call me at"},
		},
		{
			name:    "phone parens",
			in:      "ICE tip line (651) 555-1212",
			removed: []string{CatPhone},
			keeps:   []string{"ICE tip line"},
		},
		{
			name:    "email",
			in:      "Contact john.doe@example.org for details",
			removed: []string{CatEmail},
			keeps:   []string{"Contact", "for details"},
		},
		{
			name:    "SSN",
			in:      "They asked for 123-45-6789",
			removed: []string{CatSSN},
			keeps:   []string{"They asked for"},
		},
		{
			name:    "plate",
			in:      "White van, plate ABC1234 heading west",
			removed: []string{CatPlate},
			keeps:   []string{"White van, plate", "heading west"},
		},
		{
			name:    "URL",
			in:      "Source: https://www.mprnews.org/story/123",
			removed: []string{CatURL},
			keeps:   []string{"Source:"},
		},
		{
			name:    "house number leading label",
			in:      "1234 Nicollet Ave",
			removed: []string{CatHouseNum},
			keeps:   []string{"Nicollet Ave"},
		},
		{
			name:    "keeps ICE token",
			in:      "ICE vehicles seen on Cedar Ave S",
			removed: nil,
			keeps:   []string{"ICE", "Cedar Ave"},
		},
		{
			name:    "keeps intersection label",
			in:      "near 47th & Chicago in Minneapolis",
			removed: nil,
			keeps:   []string{"47th & Chicago", "Minneapolis"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := RegexScrub(c.in)
			for _, cat := range c.removed {
				if r.Removed[cat] == 0 {
					t.Errorf("expected category %q to be removed; got %v", cat, r.Removed)
				}
			}
			for _, keep := range c.keeps {
				if !strings.Contains(r.Text, keep) {
					t.Errorf("expected %q to remain in %q", keep, r.Text)
				}
			}
		})
	}
}

// Zero-leak property: on the fixtures we DO expect scrubbing, the output
// must not contain the sensitive token. Extends the golden set above.
func TestRegexScrubNoLeak(t *testing.T) {
	cases := []struct{ in, sensitive string }{
		{"Call me at +1 612-555-0131", "612-555-0131"},
		{"Contact john.doe@example.org please", "john.doe@example.org"},
		{"They asked for 123-45-6789", "123-45-6789"},
		{"plate ABC1234 visible", "ABC1234"},
		{"1234 Nicollet Ave", "1234"},
		{"visit www.example.org for details", "www.example.org"},
	}
	for _, c := range cases {
		r := RegexScrub(c.in)
		if strings.Contains(r.Text, c.sensitive) {
			t.Errorf("scrubbed text %q still contains %q", r.Text, c.sensitive)
		}
	}
}
