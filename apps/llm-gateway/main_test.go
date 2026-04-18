package main

import "testing"

func TestParseClassifierJSON(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		bucket string
	}{
		{"bare", `{"bucket":"sighting","confidence":0.9,"reasoning":"ok"}`, "sighting"},
		{"code-fenced", "```json\n{\"bucket\":\"resource_candidate\",\"confidence\":0.8,\"reasoning\":\"\"}\n```", "resource_candidate"},
		{"unknown bucket coerced", `{"bucket":"wat","confidence":0.4,"reasoning":"\u200bbad"}`, "needs_human"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := parseClassifierJSON(c.in)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if r.Bucket != c.bucket {
				t.Errorf("bucket = %q, want %q", r.Bucket, c.bucket)
			}
		})
	}
}

func TestParseClassifierJSONClampsConfidence(t *testing.T) {
	r, err := parseClassifierJSON(`{"bucket":"sighting","confidence":1.7,"reasoning":""}`)
	if err != nil {
		t.Fatal(err)
	}
	if r.Confidence != 1 {
		t.Errorf("confidence = %v, want 1", r.Confidence)
	}
}
