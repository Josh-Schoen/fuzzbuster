// Package geo implements the safety-critical coarse-location snapping
// described in docs/safety-policy.md.
//
// Sighting coordinates MUST be snapped to a ~200m grid before they are
// persisted or transmitted. The original precision is discarded — never
// stored-then-redacted.
package geo

import "math"

// CoarseGridMeters is the side length of the snap grid for sightings.
const CoarseGridMeters = 200.0

// Coord is a (lat, lng) pair in degrees, WGS84.
type Coord struct {
	Lat float64
	Lng float64
}

// SnapToCoarseGrid rounds (lat, lng) to the nearest ~200m grid cell center.
// The transformation is one-way: callers MUST overwrite their original
// variables and discard the precise input.
//
// Implementation: convert to a local equirectangular projection scaled by
// CoarseGridMeters, round, convert back. Accurate to within a few meters
// for any latitude between ±60°, which covers all of CONUS including MN.
func SnapToCoarseGrid(c Coord) Coord {
	const metersPerDegLat = 111_320.0 // ≈ at 0°; varies <0.6% across CONUS
	metersPerDegLng := metersPerDegLat * math.Cos(c.Lat*math.Pi/180.0)
	if metersPerDegLng < 1 {
		metersPerDegLng = 1 // degenerate near the poles; shouldn't happen
	}

	cellsLat := math.Round(c.Lat * metersPerDegLat / CoarseGridMeters)
	cellsLng := math.Round(c.Lng * metersPerDegLng / CoarseGridMeters)

	return Coord{
		Lat: cellsLat * CoarseGridMeters / metersPerDegLat,
		Lng: cellsLng * CoarseGridMeters / metersPerDegLng,
	}
}

// MinnesotaBBox is the launch-state viewport. Used as a default by the API
// and the PWA when no viewport is supplied.
var MinnesotaBBox = struct {
	MinLat, MinLng, MaxLat, MaxLng float64
}{
	MinLat: 43.4994,
	MinLng: -97.2390,
	MaxLat: 49.3845,
	MaxLng: -89.4919,
}

// InMinnesotaBBox returns true if c falls inside the MN bounding box. Used
// as a coarse pre-filter on scraped events; the PostGIS query is the source
// of truth.
func InMinnesotaBBox(c Coord) bool {
	return c.Lat >= MinnesotaBBox.MinLat && c.Lat <= MinnesotaBBox.MaxLat &&
		c.Lng >= MinnesotaBBox.MinLng && c.Lng <= MinnesotaBBox.MaxLng
}
