// Command orchestrator schedules scraping jobs (Asynq cron), dispatches them
// to either Go static scrapers or the Node browser-agent service over gRPC,
// fans the resulting RawItems into the LLM gateway, and writes Sightings /
// Resources / Events back to Postgres.
//
// This file is the v0 skeleton: it wires the cron schedule and an empty
// task handler. Real scrapers and the gRPC client land in week 3-4.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/hibiken/asynq"
)

const (
	taskScrapeRSS    = "fuzzbuster:scrape:rss"
	taskScrapeAgent  = "fuzzbuster:scrape:browser_agent"
	taskClassifyRaw  = "fuzzbuster:llm:classify"
	taskDecaySweep   = "fuzzbuster:decay:sweep"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Cron schedule. Cadences match docs/data-sources.md.
	scheduler := asynq.NewScheduler(asynq.RedisClientOpt{Addr: redisAddr}, nil)
	mustRegister := func(spec, taskType string, payload []byte) {
		if _, err := scheduler.Register(spec, asynq.NewTask(taskType, payload)); err != nil {
			logger.Error("register schedule", "spec", spec, "type", taskType, "err", err)
			os.Exit(1)
		}
	}

	// RSS jobs (Go-side static scrapers, not yet implemented).
	mustRegister("@every 15m", taskScrapeRSS, []byte(`{"producer":"rss-gdelt","region":"US-MN"}`))
	mustRegister("@every 1h", taskScrapeRSS, []byte(`{"producer":"rss-dhs","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeRSS, []byte(`{"producer":"rss-google-news","region":"US-MN"}`))

	// Browser-agent jobs (dispatched to Node ScraperService over gRPC).
	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-mprnews","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-startribune","region":"US-MN"}`))
	mustRegister("@every 30m", taskScrapeAgent, []byte(`{"producer":"news-sahan","region":"US-MN"}`))
	mustRegister("@every 5m",  taskScrapeAgent, []byte(`{"producer":"bluesky-search","region":"US-MN"}`))

	// Periodic decay sweep that flips published → expired and emits a
	// moderation_decisions row. Belt-and-suspenders to the view-level filter.
	mustRegister("@every 5m", taskDecaySweep, nil)

	go func() {
		if err := scheduler.Run(); err != nil {
			logger.Error("scheduler", "err", err)
			os.Exit(1)
		}
	}()

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 8,
			Queues: map[string]int{
				"default":  3,
				"scrape":   5,
				"llm":      2,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(taskScrapeRSS, handleScrapeRSS(logger))
	mux.HandleFunc(taskScrapeAgent, handleScrapeAgent(logger))
	mux.HandleFunc(taskClassifyRaw, handleClassify(logger))
	mux.HandleFunc(taskDecaySweep, handleDecaySweep(logger))

	logger.Info("orchestrator running")
	if err := srv.Run(mux); err != nil {
		logger.Error("server", "err", err)
		os.Exit(1)
	}
}

// All four handlers are stubs in the skeleton. They log and return nil so
// the scheduler proves out end-to-end before real implementations land.

func handleScrapeRSS(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("scrape rss (stub)", "payload", string(t.Payload()))
		return nil
	}
}

func handleScrapeAgent(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("scrape browser-agent (stub)", "payload", string(t.Payload()))
		// TODO: gRPC call into apps/scraper ScraperService.RunJob.
		return nil
	}
}

func handleClassify(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("classify raw item (stub)", "payload", string(t.Payload()))
		// TODO: gRPC call into apps/llm-gateway ModerationService.Classify.
		return nil
	}
}

func handleDecaySweep(log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		log.Info("decay sweep (stub)")
		// TODO: UPDATE sightings SET status='expired'
		// WHERE status='published' AND observed_at < NOW() - INTERVAL '8 hours';
		return nil
	}
}
