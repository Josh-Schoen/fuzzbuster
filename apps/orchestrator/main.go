// Command orchestrator schedules scraping jobs (Asynq cron), dispatches
// them to Go static scrapers or (in week 3-4) the Node browser-agent
// service, fans resulting RawItems into the LLM gateway, and writes
// Sightings / Resources / Events back to Postgres.
//
// v1 (this slice) implements:
//   - RSS fan-out via gofeed (see scrape_rss.go)
//   - Classify step that calls llm-gateway and promotes raw_items into
//     sightings / resources / events based on the classifier bucket
//   - Decay sweep that flips old sightings to `expired`
//
// Still TODO (week 3-4): gRPC to apps/scraper for browser-agent jobs;
// embedding-based dedup across raw_items before promotion.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	taskScrapeRSS   = "fuzzbuster:scrape:rss"
	taskScrapeAgent = "fuzzbuster:scrape:browser_agent"
	taskClassifyRaw = "fuzzbuster:llm:classify"
	taskDecaySweep  = "fuzzbuster:decay:sweep"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	redisAddr := envOr("REDIS_ADDR", "localhost:6379")
	dsn := envOr("DATABASE_URL", "postgres://fuzzbuster:fuzzbuster@localhost:5432/fuzzbuster?sslmode=disable")
	llmGatewayURL := envOr("LLM_GATEWAY_URL", "http://localhost:8090")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("connect db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	llm := &llmClient{base: llmGatewayURL, httpc: &http.Client{Timeout: 90 * time.Second}}

	// Cron schedule. Cadences match docs/data-sources.md.
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr}, nil)
	mustRegister := func(spec, taskType string, payload []byte) {
		if _, err := scheduler.Register(spec, asynq.NewTask(taskType, payload)); err != nil {
			log.Error("register schedule", "spec", spec, "type", taskType, "err", err)
			os.Exit(1)
		}
	}

	mustRegister("@every 15m", taskScrapeRSS, []byte(`{"producer":"rss-gdelt","region":"US-MN"}`))
	mustRegister("@every 1h", taskScrapeRSS, []byte(`{"producer":"rss-dhs","region":"US-MN"}`))
	mustRegister("@every 1h", taskScrapeRSS, []byte(`{"producer":"rss-ice","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeRSS, []byte(`{"producer":"rss-google-news","region":"US-MN"}`))

	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-mprnews","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-startribune","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-sahan","region":"US-MN"}`))
	mustRegister("@every 5m", taskScrapeAgent, []byte(`{"producer":"bluesky-search","region":"US-MN"}`))

	// Classify runs every minute: grab the oldest unprocessed raw_items and
	// send them through the llm gateway.
	mustRegister("@every 1m", taskClassifyRaw, nil)

	// Decay sweep every 5m — belt-and-suspenders to the view-level filter.
	mustRegister("@every 5m", taskDecaySweep, nil)

	go func() {
		if err := scheduler.Run(); err != nil {
			log.Error("scheduler", "err", err)
			os.Exit(1)
		}
	}()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 8,
			Queues: map[string]int{
				"default": 3,
				"scrape":  5,
				"llm":     2,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(taskScrapeRSS, handleScrapeRSSImpl(pool, log))
	mux.HandleFunc(taskScrapeAgent, handleScrapeAgentStub(log))
	mux.HandleFunc(taskClassifyRaw, handleClassifyImpl(pool, llm, log))
	mux.HandleFunc(taskDecaySweep, handleDecaySweepImpl(pool, log))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		log.Info("shutting down")
		srv.Shutdown()
		cancel()
	}()

	log.Info("orchestrator running", "llm_gateway", llmGatewayURL)
	if err := srv.Run(mux); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}

// ----------------------------------------------------------------------
// Browser-agent stub — real gRPC client lands when the Node service is up.
// ----------------------------------------------------------------------

func handleScrapeAgentStub(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("browser-agent (stub)", "payload", string(t.Payload()))
		return nil
	}
}

// ----------------------------------------------------------------------
// Classify handler: pull unclassified raw_items, call llm-gateway, write
// back the bucket + confidence, and promote to sighting/resource/event.
// ----------------------------------------------------------------------

type rawItem struct {
	ID       string
	SourceID string
	Text     string
	Language string
}

func handleClassifyImpl(pool *pgxpool.Pool, llm *llmClient, log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		// Batch up to 20 per tick — small enough to keep latency low,
		// large enough that we keep up with 4 RSS feeds @ 15-60m.
		rows, err := pool.Query(ctx, `
			SELECT id, source_id, text, language::text
			FROM raw_items
			WHERE processed_at IS NULL
			ORDER BY captured_at ASC
			LIMIT 20`)
		if err != nil {
			return err
		}
		var items []rawItem
		for rows.Next() {
			var it rawItem
			if err := rows.Scan(&it.ID, &it.SourceID, &it.Text, &it.Language); err != nil {
				rows.Close()
				return err
			}
			items = append(items, it)
		}
		rows.Close()

		classified := 0
		for _, it := range items {
			resp, err := llm.Classify(ctx, it.Text, it.Language)
			if err != nil {
				log.Warn("classify call failed; leaving raw_item for retry", "id", it.ID, "err", err)
				continue
			}
			if err := applyClassification(ctx, pool, it, resp, log); err != nil {
				log.Error("apply classification", "id", it.ID, "err", err)
				continue
			}
			classified++
		}
		if classified > 0 {
			log.Info("classify tick", "items", classified)
		}
		return nil
	}
}

func applyClassification(ctx context.Context, pool *pgxpool.Pool, it rawItem, r classifyResult, log *slog.Logger) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE raw_items
		SET classified_bucket = $1,
		    classifier_confidence = $2,
		    classifier_model = $3,
		    processed_at = NOW()
		WHERE id = $4`,
		r.Bucket, r.Confidence, r.Model, it.ID); err != nil {
		return err
	}

	// Sightings promoted from RSS are always `pending` — they don't publish
	// until a moderator confirms, because we can't geocode a public article
	// tight enough for the 200m grid from headline + blurb alone. The
	// moderator dashboard (week 5) hooks here.
	switch r.Bucket {
	case "sighting":
		// We deliberately do NOT insert into sightings from a news-only source
		// here: without a coordinate we can't satisfy the NOT NULL location
		// column, and without a fresh human observation we shouldn't claim
		// it IS a sighting. The moderator view sees these in a "needs geo"
		// queue — tracked for week 5.
		log.Info("classified as sighting (pending moderator geocode)", "raw_id", it.ID)
	case "resource_candidate":
		log.Info("classified as resource candidate (awaiting moderator)", "raw_id", it.ID)
	case "event_candidate":
		log.Info("classified as event candidate (awaiting moderator)", "raw_id", it.ID)
	case "discard":
		// no-op
	case "needs_human":
		log.Info("needs human", "raw_id", it.ID)
	}

	return tx.Commit(ctx)
}

// ----------------------------------------------------------------------
// Decay sweep: flip sightings older than the horizon to `expired` and emit
// a moderation_decisions row. The public view already hides them; this
// sweep is for the internal archive and for the admin-dashboard counts.
// ----------------------------------------------------------------------

func handleDecaySweepImpl(pool *pgxpool.Pool, log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		rows, err := pool.Query(ctx, `
			UPDATE sightings
			SET status = 'expired', updated_at = NOW()
			WHERE status = 'published'
			  AND observed_at < NOW() - INTERVAL '8 hours'
			RETURNING id`)
		if err != nil {
			return err
		}
		defer rows.Close()
		ids := make([]string, 0, 32)
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			ids = append(ids, id)
		}
		for _, id := range ids {
			if _, err := pool.Exec(ctx, `
				INSERT INTO moderation_decisions (subject_id, subject_kind, action, source, reason)
				VALUES ($1, 'sighting', 'hide', 'decay', 'age >= 8h')`, id); err != nil {
				return err
			}
		}
		if len(ids) > 0 {
			log.Info("decay swept", "count", len(ids))
		}
		return nil
	}
}

// ----------------------------------------------------------------------
// llm-gateway HTTP client
// ----------------------------------------------------------------------

type llmClient struct {
	base  string
	httpc *http.Client
}

type classifyResult struct {
	Bucket     string  `json:"bucket"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
	Model      string  `json:"model"`
}

func (c *llmClient) Classify(ctx context.Context, text, lang string) (classifyResult, error) {
	body, _ := json.Marshal(map[string]string{"text": text, "language": lang})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/v1/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return classifyResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return classifyResult{}, errors.New("llm-gateway: " + resp.Status + ": " + string(msg))
	}
	var out classifyResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return classifyResult{}, err
	}
	return out, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
