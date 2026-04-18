// Command api serves Fuzzbuster's public REST endpoints. The endpoint set is
// intentionally narrow: list sightings within a viewport, list resources in
// a region, list events, submit a sighting, vote on a sighting. All other
// service-to-service calls go over gRPC (defined in proto/).
//
// Safety: this binary never returns rows from `sightings` directly; it reads
// from the `sightings_public` view, which enforces the 8h decay horizon at
// the data layer.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fuzzbuster/fuzzbuster/pkg/decay"
	"github.com/fuzzbuster/fuzzbuster/pkg/geo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://fuzzbuster:fuzzbuster@localhost:5432/fuzzbuster?sslmode=disable"
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logger.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP) // we use it for rate-limit only; never logged
	r.Use(middleware.Timeout(15 * time.Second))
	r.Use(noStoreHeader)

	h := &handlers{pool: pool, log: logger}

	r.Get("/healthz", h.health)
	r.Route("/v1", func(r chi.Router) {
		r.Get("/sightings", h.listSightings)
		r.Post("/sightings", h.createSighting)
		r.Post("/sightings/{id}/vote", h.voteSighting)
		r.Get("/resources", h.listResources)
		r.Get("/events", h.listEvents)
	})

	addr := ":8080"
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("api listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

// noStoreHeader prevents intermediaries from caching API responses, since
// even short caches could reveal historical sighting state past the decay
// horizon.
func noStoreHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

type handlers struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func (h *handlers) health(w http.ResponseWriter, r *http.Request) {
	if err := h.pool.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "down"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type sightingDTO struct {
	ID            string    `json:"id"`
	Lat           float64   `json:"lat"`
	Lng           float64   `json:"lng"`
	Label         string    `json:"label,omitempty"`
	ObservedAt    time.Time `json:"observed_at"`
	Category      string    `json:"category"`
	NotesScrubbed string    `json:"notes_scrubbed,omitempty"`
	Confirms      int       `json:"confirms"`
	Denies        int       `json:"denies"`
	DecayWeight   float64   `json:"decay_weight"`
	SourceURL     string    `json:"source_url,omitempty"`
}

func (h *handlers) listSightings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	bbox := geo.MinnesotaBBox
	parseFloat := func(k string, dflt float64) float64 {
		if v := q.Get(k); v != "" {
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				return f
			}
		}
		return dflt
	}
	minLat := parseFloat("min_lat", bbox.MinLat)
	maxLat := parseFloat("max_lat", bbox.MaxLat)
	minLng := parseFloat("min_lng", bbox.MinLng)
	maxLng := parseFloat("max_lng", bbox.MaxLng)

	rows, err := h.pool.Query(r.Context(), `
		SELECT s.id,
		       ST_Y(location::geometry), ST_X(location::geometry),
		       COALESCE(label, ''),
		       observed_at, category::text, notes_scrubbed,
		       confirms, denies, decay_weight,
		       COALESCE(src.url, '')
		FROM sightings_public s
		LEFT JOIN sources src ON src.id = s.source_id
		WHERE ST_Intersects(
		  location,
		  ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography
		)
		ORDER BY observed_at DESC
		LIMIT 500`,
		minLng, minLat, maxLng, maxLat,
	)
	if err != nil {
		h.log.Error("list sightings", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	out := make([]sightingDTO, 0, 64)
	for rows.Next() {
		var s sightingDTO
		if err := rows.Scan(&s.ID, &s.Lat, &s.Lng, &s.Label,
			&s.ObservedAt, &s.Category, &s.NotesScrubbed,
			&s.Confirms, &s.Denies, &s.DecayWeight, &s.SourceURL); err != nil {
			h.log.Error("scan sighting", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out = append(out, s)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) createSighting(w http.ResponseWriter, r *http.Request) {
	// TODO(week-5): wire to ModerationService.Scrub then insert with snapped
	// coords. Returning 501 until then so the public surface is honest about
	// what does and does not exist.
	_ = decay.HorizonHours
	http.Error(w, "submission flow not implemented in skeleton", http.StatusNotImplemented)
}

func (h *handlers) voteSighting(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "vote flow not implemented in skeleton", http.StatusNotImplemented)
}

func (h *handlers) listResources(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "MN"
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, kind::text, name, description, languages, url, phone, email, hours,
		       COALESCE(street, ''), COALESCE(city, ''), COALESCE(region, ''),
		       COALESCE(postal_code, ''), COALESCE(country, '')
		FROM resources_public
		WHERE region = $1
		ORDER BY name`,
		region,
	)
	if err != nil {
		h.log.Error("list resources", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type resourceDTO struct {
		ID          string   `json:"id"`
		Kind        string   `json:"kind"`
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Languages   []string `json:"languages"`
		URL         string   `json:"url,omitempty"`
		Phone       string   `json:"phone,omitempty"`
		Email       string   `json:"email,omitempty"`
		Hours       string   `json:"hours,omitempty"`
		Street      string   `json:"street,omitempty"`
		City        string   `json:"city,omitempty"`
		Region      string   `json:"region,omitempty"`
		PostalCode  string   `json:"postal_code,omitempty"`
		Country     string   `json:"country,omitempty"`
	}

	out := make([]resourceDTO, 0, 32)
	for rows.Next() {
		var d resourceDTO
		if err := rows.Scan(&d.ID, &d.Kind, &d.Name, &d.Description, &d.Languages,
			&d.URL, &d.Phone, &d.Email, &d.Hours,
			&d.Street, &d.City, &d.Region, &d.PostalCode, &d.Country); err != nil {
			h.log.Error("scan resource", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) listEvents(w http.ResponseWriter, r *http.Request) {
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "MN"
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT id, title, description, languages, starts_at, ends_at, url,
		       COALESCE(street, ''), COALESCE(city, ''), region
		FROM events
		WHERE region = $1 AND status = 'published' AND starts_at >= NOW() - INTERVAL '6 hours'
		ORDER BY starts_at ASC
		LIMIT 200`,
		region,
	)
	if err != nil {
		h.log.Error("list events", "err", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type eventDTO struct {
		ID          string     `json:"id"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		Languages   []string   `json:"languages"`
		StartsAt    time.Time  `json:"starts_at"`
		EndsAt      *time.Time `json:"ends_at,omitempty"`
		URL         string     `json:"url,omitempty"`
		Street      string     `json:"street,omitempty"`
		City        string     `json:"city,omitempty"`
		Region      string     `json:"region,omitempty"`
	}

	out := make([]eventDTO, 0, 32)
	for rows.Next() {
		var d eventDTO
		if err := rows.Scan(&d.ID, &d.Title, &d.Description, &d.Languages,
			&d.StartsAt, &d.EndsAt, &d.URL,
			&d.Street, &d.City, &d.Region); err != nil {
			h.log.Error("scan event", "err", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		out = append(out, d)
	}
	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
