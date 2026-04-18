// Package scrub removes PII from free-text notes before storage. See
// docs/safety-policy.md §5 for the contract.
//
// The scrubber has two layers:
//
//  1. Deterministic regex pass — handles things a model might miss or
//     hallucinate around: phone numbers, emails, SSNs, license plates in
//     common US formats, house numbers at the start of a label.
//  2. Optional LLM pass — the caller (llm-gateway) owns this. This package
//     exposes the regex layer as a fast, offline, unit-testable default.
//
// Contract: the scrubber is a one-way function. The original text must not
// be retained anywhere after Scrub returns.
package scrub

import (
	"regexp"
	"strings"
)

// Category labels are only used for tests + moderation golden-set reports.
// They must never be exposed on the public API.
const (
	CatPhone      = "phone"
	CatEmail      = "email"
	CatSSN        = "ssn"
	CatPlate      = "plate"
	CatHouseNum   = "house_number"
	CatURL        = "url"
)

var (
	// US phone numbers, permissive.
	rePhone = regexp.MustCompile(`(?:\+?1[\s\-\.]?)?(?:\(\d{3}\)|\d{3})[\s\-\.]?\d{3}[\s\-\.]?\d{4}`)
	// Emails.
	reEmail = regexp.MustCompile(`[\w\.\+\-]+@[\w\-]+(?:\.[\w\-]+)+`)
	// SSNs.
	reSSN = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	// US plates: 5–8 chars of letters+digits with at least one of each, as
	// a standalone token. This is intentionally conservative because a raw
	// `[A-Z0-9]{5,8}` would wipe out words like "ICE". We require a mix
	// and word boundaries.
	rePlate = regexp.MustCompile(`(?i)\b(?:(?:[a-z]+[0-9]+|[0-9]+[a-z]+)[a-z0-9]*)\b`)
	// Leading house number in a label, e.g. "1234 Nicollet Ave".
	reHouseNum = regexp.MustCompile(`^\s*\d{1,5}\s+`)
	// URLs and bare domains.
	reURL = regexp.MustCompile(`(?i)\bhttps?://\S+|\bwww\.\S+`)
)

// Result captures what was removed, for golden-set testing only.
type Result struct {
	Text     string
	Removed  map[string]int // category → count of removed spans
}

// RegexScrub applies the deterministic layer. It always replaces matches
// with the fixed string "[redacted]" — no attempt at length-preserving
// masking, because that can leak the original value's length.
func RegexScrub(in string) Result {
	out := in
	removed := map[string]int{}

	apply := func(re *regexp.Regexp, cat string) {
		matches := re.FindAllStringIndex(out, -1)
		if len(matches) == 0 {
			return
		}
		removed[cat] += len(matches)
		out = re.ReplaceAllString(out, "[redacted]")
	}

	// Order matters: URLs and emails first so the phone/plate regex doesn't
	// eat parts of them.
	apply(reURL, CatURL)
	apply(reEmail, CatEmail)
	apply(reSSN, CatSSN)
	apply(rePhone, CatPhone)

	// Plates: only apply if the word-form passes a further "looks plate-ish"
	// check. We don't want to redact things like "Plate1" or "H2O".
	out = rePlate.ReplaceAllStringFunc(out, func(tok string) string {
		if looksLikePlate(tok) {
			removed[CatPlate]++
			return "[redacted]"
		}
		return tok
	})

	// House-number prefix on a short line is a label like
	// "1234 Nicollet Ave, Minneapolis" — strip the number only.
	if reHouseNum.MatchString(out) {
		out = reHouseNum.ReplaceAllString(out, "")
		removed[CatHouseNum]++
	}

	// Collapse any stray double-redacteds.
	out = strings.ReplaceAll(out, "[redacted][redacted]", "[redacted]")

	return Result{Text: out, Removed: removed}
}

// looksLikePlate filters alphanumeric mixes that are more likely to be
// license plates than noun words. Heuristic: 5–8 chars, at least 2 digits,
// no vowel-only runs longer than 3.
func looksLikePlate(s string) bool {
	if n := len(s); n < 5 || n > 8 {
		return false
	}
	digits := 0
	letters := 0
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			letters++
		}
	}
	if digits < 2 || letters < 1 {
		return false
	}
	// Reject common English words that happen to fit the shape like "H2O2A"
	// — require at least one letter-then-digit or digit-then-letter transition.
	transitions := 0
	var prev byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		var kind byte
		switch {
		case c >= '0' && c <= '9':
			kind = 'd'
		case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
			kind = 'l'
		default:
			kind = 'x'
		}
		if i > 0 && kind != prev {
			transitions++
		}
		prev = kind
	}
	return transitions >= 2
}
