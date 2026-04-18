package main

import "testing"

func TestMNRegex(t *testing.T) {
	hits := []string{
		"ICE arrests in Minneapolis overnight",
		"Sheriff's office in Hennepin County issues statement",
		"Report from Saint Paul, MN",
		"St. Paul community rallies",
		"immigrant families in the Twin Cities",
		"Duluth legal aid clinic announced",
	}
	misses := []string{
		"ICE arrests in Chicago overnight",
		"immigration policy update from Washington",
		"Boston mayor on sanctuary city status",
	}
	for _, h := range hits {
		if !mnRegex.MatchString(h) {
			t.Errorf("expected MN hit: %q", h)
		}
	}
	for _, m := range misses {
		if mnRegex.MatchString(m) {
			t.Errorf("expected MN miss: %q", m)
		}
	}
}

func TestStripHTML(t *testing.T) {
	in := `<p>Hello <a href="https://example.com">world</a></p>`
	out := stripHTML(in)
	if out != "Hello world" {
		t.Errorf("stripHTML = %q, want %q", out, "Hello world")
	}
}
