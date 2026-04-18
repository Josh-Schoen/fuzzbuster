package geo

import (
	"math"
	"testing"
)

func TestSnapToCoarseGridMovesLessThanGridDiagonal(t *testing.T) {
	// Snap must move every input by less than the grid's half-diagonal
	// (otherwise it would be snapping to a non-nearest cell).
	const halfDiagonal = CoarseGridMeters * math.Sqrt2 / 2
	const metersPerDegLat = 111_320.0

	cases := []Coord{
		{Lat: 44.9778, Lng: -93.2650}, // Minneapolis
		{Lat: 44.9537, Lng: -93.0900}, // St. Paul
		{Lat: 46.7867, Lng: -92.1005}, // Duluth
		{Lat: 45.5579, Lng: -94.1632}, // St. Cloud
		{Lat: 43.6155, Lng: -116.2023}, // Boise (well outside MN, sanity)
	}
	for _, c := range cases {
		s := SnapToCoarseGrid(c)
		mppLng := metersPerDegLat * math.Cos(c.Lat*math.Pi/180.0)
		dy := (c.Lat - s.Lat) * metersPerDegLat
		dx := (c.Lng - s.Lng) * mppLng
		moved := math.Hypot(dx, dy)
		if moved > halfDiagonal+1e-3 {
			t.Errorf("snap moved %.2fm (>%.2fm) for %+v -> %+v", moved, halfDiagonal, c, s)
		}
	}
}

func TestSnapIsIdempotent(t *testing.T) {
	c := Coord{Lat: 44.9778, Lng: -93.2650}
	s1 := SnapToCoarseGrid(c)
	s2 := SnapToCoarseGrid(s1)
	if s1 != s2 {
		t.Errorf("snap not idempotent: %+v -> %+v", s1, s2)
	}
}

func TestInMinnesotaBBox(t *testing.T) {
	if !InMinnesotaBBox(Coord{Lat: 44.9778, Lng: -93.2650}) {
		t.Error("Minneapolis should be in MN bbox")
	}
	if InMinnesotaBBox(Coord{Lat: 41.8781, Lng: -87.6298}) {
		t.Error("Chicago should not be in MN bbox")
	}
}
