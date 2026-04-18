package decay

import (
	"testing"
	"time"
)

func TestVisibilityHorizon(t *testing.T) {
	now := time.Now()
	cases := []struct {
		ageHrs   float64
		visible  bool
	}{
		{0, true},
		{1, true},
		{HorizonHours - 0.01, true},
		{HorizonHours, false},
		{HorizonHours + 1, false},
		{24, false},
	}
	for _, c := range cases {
		obs := now.Add(-time.Duration(c.ageHrs * float64(time.Hour)))
		if got := Visible(obs, now); got != c.visible {
			t.Errorf("age=%vh: Visible=%v, want %v", c.ageHrs, got, c.visible)
		}
	}
}

func TestWeightDecreasesMonotonically(t *testing.T) {
	now := time.Now()
	prev := 1.1
	for h := 0.0; h < HorizonHours; h += 0.5 {
		w := Weight(now.Add(-time.Duration(h*float64(time.Hour))), now)
		if w > prev {
			t.Errorf("weight increased at h=%v: %v > %v", h, w, prev)
		}
		prev = w
	}
}
