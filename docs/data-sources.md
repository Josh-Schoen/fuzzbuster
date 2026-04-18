# Data Sources

Sources Fuzzbuster aggregates from. Public / partner-friendly first; never scrape closed platforms; respect `robots.txt` and rate limits everywhere.

## v1: Minnesota launch

### Static feeds (Go RSS scrapers)

| Source | URL / Endpoint | Cadence | Notes |
| --- | --- | --- | --- |
| GDELT 2.0 | `gdeltproject.org/data.html` | 15 min | Filter to `themes` containing `IMMIGRATION`, location ∈ Minnesota bbox. Free, geocoded. |
| DHS press releases | `dhs.gov/news.xml` | 1 hr | Filter to ICE/ERO/HSI mentions of Minnesota. |
| ICE press releases | `ice.gov/news/all.rss` (verify endpoint) | 1 hr | National; filter to MN ROs (St. Paul AOR). |
| Google News RSS | `news.google.com/rss/search?q=ICE+Minnesota` | 30 min | Brittle; back up with TLD-restricted Bing News. |
| TRAC Immigration | `trac.syr.edu` | weekly | Detention + court statistics; pull aggregates only. |
| EOIR statistics | `justice.gov/eoir/statistics-and-publications` | weekly | Court-load statistics. |
| CourtListener | `courtlistener.com/api/rest/v3/` | daily | Immigration court filings (D. Minn., 8th Cir.). |

### Browser-agent sources (Node + Playwright + Stagehand + Claude Agent SDK)

Spin up a headless Chromium only when JS rendering, login, search, or anti-bot defenses make a static fetch impractical.

| Source | Skill name | Cadence | Notes |
| --- | --- | --- | --- |
| Bluesky firehose / search | `bluesky-search` | continuous | Use `@atproto/api` Jetstream where possible; agent only for complex search queries. Hashtags: `#ICE`, `#MNimmigration`, `#TwinCities`, etc. |
| Mastodon (mastodon.social, hachyderm.io, social.coop) | `mastodon-tag-search` | 5 min | Streaming API + agent-driven tag search. |
| Reddit (`r/Minneapolis`, `r/StPaul`, `r/Minnesota`, `r/immigration`) | `reddit-subreddit-watch` | 15 min | Paid Reddit API; agent narrows to relevant posts; never scrape user PII. |
| Star Tribune | `news-startribune` | 30 min | Search-page navigation + article extraction. |
| MPR News | `news-mprnews` | 30 min | Search-page navigation + article extraction. |
| Sahan Journal | `news-sahan` | 30 min | Strong immigration coverage in MN. |
| MinnPost | `news-minnpost` | 30 min | |
| Minnesota Reformer | `news-mnreformer` | 30 min | |
| Local TV (KARE 11, WCCO, KSTP, FOX 9) | `news-local-tv` (one Skill, multiple sites) | 30 min | Often has MN-specific raid coverage. |
| St. Paul AOR ICE field-office page | `ice-field-office-stpaul` | daily | Public press / community announcements. |
| Sheriff's office press pages (Hennepin, Ramsey, Anoka, Dakota, Washington) | `mn-sheriff-press` | daily | 287(g) and detainer-policy news. |

### Partner / hotline feeds (require partnership; never scrape)

| Partner | Integration model | Status |
| --- | --- | --- |
| ILCM (ilcm.org) | Verified-resource directory entry; staff-curated alerts via signed webhook | not started |
| The Advocates for Human Rights (theadvocatesforhumanrights.org) | Verified-resource directory entry | not started |
| Mid-Minnesota Legal Aid (mylegalaid.org) | Verified-resource directory entry | not started |
| Unidos MN (unidosmn.org) | Verified-resource entry; potential rapid-response feed | not started |
| COPAL (copalmn.org) | Verified-resource entry; ES content review | not started |
| Navigate MN (navigatemn.org) | Verified-resource entry | not started |
| CLUES (clues.org) | Verified-resource entry | not started |
| CTUL (ctul.net) | Verified-resource entry | not started |
| Twin Cities Rapid Response Network | Signed webhook for verified sightings | not started |
| MIRAC | Coordination only; no data feed | not started |
| UWD MigraWatch (1-844-363-1423) | National hotline; cross-list as last-resort verifier | not started |

### Legal-aid directory seeds

- Immigration Advocates Network (`immigrationadvocates.org/nonprofit/legaldirectory`) — pull MN entries with attribution.
- LawHelpMN (`lawhelpmn.org`) — pull immigration-relevant entries with attribution.
- CLINIC affiliate directory (`cliniclegal.org/directory`) — pull MN entries.
- ABA Pro Bono (`americanbar.org/groups/probono_public_service/projects_awards/immigration_pro_bono`).

### Know-Your-Rights content (link, do not rehost without permission)

- ACLU KYR for Immigrants (`aclu.org/know-your-rights/immigrants-rights`) — EN + ES + more.
- ILRC red cards (`ilrc.org/red-cards`) — print-ready in 19 languages.
- NILC (`nilc.org`) — policy + KYR.
- Informed Immigrant (`informedimmigrant.com`).

## Rules for adding a new source

1. Source must be either (a) public and `robots.txt`-permitted, or (b) covered by an explicit partnership with written consent.
2. Add an entry to this file before adding the Skill / scraper.
3. Every scraped record must persist its source URL.
4. The Skill or scraper must include a fixture-based test in CI.
5. Rate limits must be at least 5x more conservative than the source's published limit, or 1 req/30s if no limit is published.
6. Any source that requires solving a CAPTCHA is **out**: that signals the operator does not consent to programmatic access.
