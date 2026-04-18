"use client";

import { useEffect, useState } from "react";
import Map, { Marker, NavigationControl } from "react-map-gl/maplibre";
import "maplibre-gl/dist/maplibre-gl.css";

const MN_BBOX = { minLat: 43.4994, minLng: -97.2390, maxLat: 49.3845, maxLng: -89.4919 };
const MN_CENTER = { lat: 46.4419, lng: -93.3641, zoom: 5.2 };

type Sighting = {
  id: string;
  lat: number;
  lng: number;
  label?: string;
  category: string;
  observed_at: string;
  decay_weight: number;
  confirms: number;
  denies: number;
};

export function SightingsMap() {
  const [sightings, setSightings] = useState<Sighting[]>([]);

  useEffect(() => {
    const controller = new AbortController();
    const url = new URL("/v1/sightings", process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080");
    url.searchParams.set("min_lat", String(MN_BBOX.minLat));
    url.searchParams.set("max_lat", String(MN_BBOX.maxLat));
    url.searchParams.set("min_lng", String(MN_BBOX.minLng));
    url.searchParams.set("max_lng", String(MN_BBOX.maxLng));
    fetch(url, { signal: controller.signal, cache: "no-store" })
      .then((r) => (r.ok ? r.json() : []))
      .then(setSightings)
      .catch(() => {});
    return () => controller.abort();
  }, []);

  return (
    <div className="map-wrap">
      <Map
        initialViewState={{ latitude: MN_CENTER.lat, longitude: MN_CENTER.lng, zoom: MN_CENTER.zoom }}
        // OSM raster tiles via MapLibre demo style. Replace with self-hosted
        // MapTiler / Protomaps before public launch (avoids leaking viewports
        // to a third party).
        mapStyle="https://demotiles.maplibre.org/style.json"
        // Floor on zoom: never tighter than the 200m grid is meaningful at.
        maxZoom={15}
      >
        <NavigationControl position="top-right" />
        {sightings.map((s) => (
          <Marker key={s.id} latitude={s.lat} longitude={s.lng}>
            <div
              title={`${s.category} · ${new Date(s.observed_at).toLocaleString()}`}
              style={{
                width: 12,
                height: 12,
                borderRadius: 6,
                background: `rgba(251, 191, 36, ${Math.max(0.3, s.decay_weight)})`,
                border: "1px solid #0b1220",
              }}
            />
          </Marker>
        ))}
      </Map>
    </div>
  );
}
