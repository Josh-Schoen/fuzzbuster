// Package decay implements the time-decay rules for sightings described in
// docs/safety-policy.md. The DB view sightings_public enforces the same
// rules at the storage layer; this package is for application-side checks
// and for generating the visibility weight that the client uses for sorting
// and dimming.
package decay

import (
	"math"
	"time"
)

// HalfLifeHours is the e-folding time of the sighting visibility weight.
const HalfLifeHours = 6.0

// HorizonHours is the hard cutoff after which a sighting is invisible.
const HorizonHours = 8.0

// Weight returns the visibility weight in [0, 1] for a sighting observed at
// observedAt, evaluated at now.
func Weight(observedAt, now time.Time) float64 {
	ageHrs := now.Sub(observedAt).Hours()
	if ageHrs < 0 {
		return 1.0
	}
	if ageHrs >= HorizonHours {
		return 0.0
	}
	return math.Exp(-ageHrs / HalfLifeHours)
}

// Visible returns true if the sighting should be returned by the public API.
func Visible(observedAt, now time.Time) bool {
	return Weight(observedAt, now) > 0
}
