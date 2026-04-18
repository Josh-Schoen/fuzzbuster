// RSS scraping. One handler, many sources; the task payload names the
// producer so we know which feed URL + filter to use.
//
// The filter is intentionally coarse here — we include any item that
// mentions Minnesota or a major MN city. The classifier (next step in the
// pipeline) is what actually decides sighting vs. resource vs. discard.
//
// Rules:
//   * Strip HTML from description before storage (but don't strip links —
//     we keep the source URL as a separate column).
//   * Never store the author field: Google News RSS in particular can
//     smuggle in private-individual names.
//   * Dedup: skip if a raw_item already exists for this source URL.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmcdole/gofeed"
)

type rssJobPayload struct {
	Producer string `json:"producer"`
	Region   string `json:"region"`
}

var rssSources = map[string]string{
	"rss-dhs":         "https://www.dhs.gov/news.xml",
	"rss-ice":         "https://www.ice.gov/rss/news",
	"rss-google-news": "https://news.google.com/rss/search?q=ICE+Minnesota+immigration&hl=en-US&gl=US&ceid=US:en",
	// GDELT has a specific event-feed URL scheme; configured as a separate
	// producer in week 3-4. Placeholder:
	"rss-gdelt": "https://api.gdeltproject.org/api/v2/events/events?query=sourcecountry:usa%20AND%20%28ICE%20OR%20immigration%29%20AND%20Minnesota&format=rss",
}

// MN-relevance filter: very simple, case-insensitive substring match. The
// classifier does the real work; this just keeps raw_items small.
var mnRegex = regexp.MustCompile(`(?i)\b(minnesota|mn\b|minneapolis|saint\s+paul|st\.?\s*paul|duluth|rochester|st\.?\s*cloud|mankato|moorhead|worthington|willmar|bloomington|hennepin|ramsey|twin\s+cities)\b`)

var htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	return strings.TrimSpace(htmlTagRegex.ReplaceAllString(s, ""))
}

func handleScrapeRSSImpl(pool *pgxpool.Pool, log *slog.Logger) asynq.HandlerFunc {
	return func(ctx context.Context, t *asynq.Task) error {
		var p rssJobPayload
		if err := json.Unmarshal(t.Payload(), &p); err != nil {
			return err
		}
		url, ok := rssSources[p.Producer]
		if !ok {
			return errors.New("unknown rss producer: " + p.Producer)
		}

		parser := gofeed.NewParser()
		parser.UserAgent = "fuzzbuster-bot/0.1 (+https://fuzzbuster.example/robots)"
		fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		feed, err := parser.ParseURLWithContext(url, fetchCtx)
		if err != nil {
			log.Warn("rss parse failed", "producer", p.Producer, "err", err)
			return err
		}

		inserted := 0
		for _, it := range feed.Items {
			if it == nil || it.Link == "" {
				continue
			}
			title := stripHTML(it.Title)
			desc := stripHTML(it.Description)
			combined := title + "\n" + desc
			if !mnRegex.MatchString(combined) {
				continue
			}

			// Dedup by source URL before inserting.
			var exists bool
			if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM sources WHERE url = $1)`, it.Link).Scan(&exists); err != nil {
				log.Error("dedup query", "err", err)
				continue
			}
			if exists {
				continue
			}

			publishedAt := time.Now()
			if it.PublishedParsed != nil {
				publishedAt = *it.PublishedParsed
			} else if it.UpdatedParsed != nil {
				publishedAt = *it.UpdatedParsed
			}

			tx, err := pool.Begin(ctx)
			if err != nil {
				log.Error("begin tx", "err", err)
				continue
			}
			var sourceID string
			err = tx.QueryRow(ctx, `
				INSERT INTO sources (kind, url, producer, fetched_at)
				VALUES ('rss', $1, $2, $3)
				RETURNING id
			`, it.Link, p.Producer, publishedAt).Scan(&sourceID)
			if err != nil {
				log.Error("insert source", "err", err)
				_ = tx.Rollback(ctx)
				continue
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO raw_items (source_id, text, metadata, language, captured_at)
				VALUES ($1, $2, $3::jsonb, 'en', $4)
			`, sourceID, combined, `{"headline":"`+jsonEscape(title)+`"}`, publishedAt)
			if err != nil {
				log.Error("insert raw_item", "err", err)
				_ = tx.Rollback(ctx)
				continue
			}

			if err := tx.Commit(ctx); err != nil {
				log.Error("commit", "err", err)
				continue
			}
			inserted++
		}

		log.Info("rss scraped", "producer", p.Producer, "feed_items", len(feed.Items), "inserted", inserted)
		return nil
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	// Strip the surrounding quotes added by json.Marshal.
	if len(b) >= 2 {
		return string(b[1 : len(b)-1])
	}
	return ""
}
