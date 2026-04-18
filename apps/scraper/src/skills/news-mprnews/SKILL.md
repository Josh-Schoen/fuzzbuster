# Skill: news-mprnews

Browses MPR News (`mprnews.org`) for recent immigration / ICE-related coverage in Minnesota and returns one `RawItem` per article. The Claude Agent SDK loads this skill when a `ScrapeJob` arrives with `producer = "news-mprnews"`.

## Inputs

- `region` — must be `"US-MN"` for v1; the agent will reject other regions.
- `params.lookback_hours` — optional, defaults to 24.

## Outputs

One `RawItem` per article matching:

- Published within `lookback_hours`.
- Headline or body mentions ICE, ERO, HSI, immigration enforcement, detention, deportation, raid, or any of the partner-org names listed in `docs/data-sources.md`.

For each article, set:

- `text` = headline + 60-word agent-written summary in EN (NOT the full article body — copyright caution).
- `source.url` = canonical article URL.
- `metadata.headline`, `metadata.published_at`, `metadata.byline` (only if the byline is the publication's, never a private individual).
- `language` = detected article language.
- `candidate_location` = first MN city / county the agent can extract; null if unclear.

## Hard rules (safety)

1. Always respect `mprnews.org`'s `robots.txt`. Abort the job if it disallows the search path.
2. Never log the user's IP (we are the scraper; no end-user IPs flow here, but we also don't log anything resembling reader analytics).
3. Never extract or persist the names, photos, or quotes attributed to private individuals impacted by ICE actions, even if the article itself does. Summary should describe the situation, not the person.
4. Never extract names, photos, badge numbers, or plate numbers of ICE personnel.
5. Rate-limit: 1 page load every 30 seconds, max 20 pages per job.
6. Do NOT solve a CAPTCHA. If one appears, abort the job and emit `error = "captcha_blocked"`.

## Verification

- `apps/scraper/src/skills/news-mprnews/run.test.ts` runs the agent against a fixture HTML snapshot in `fixtures/` to prove the output shape and PII-elision rules.
