# Fuzzbuster Safety Policy

This document is the safety contract that every component of Fuzzbuster is built to honor. Any code change that weakens any of these guarantees requires explicit review and a second approver.

## 1. What we collect

- **From public users:** an anonymous report consisting of (a) a coarse location (rounded to ~200m), (b) a time window in the recent past, (c) a category, (d) a source URL or partner-hotline token, and (e) optional free-text notes that pass through a PII scrubber before storage.
- **From scrapers:** publicly available content from RSS, public APIs, and public web pages. We respect `robots.txt`, rate-limit aggressively, and store source URLs for every record.
- **From moderators:** decisions, with moderator identity scoped to the moderation service only.

## 2. What we never collect or publish

- Names, faces, voice, license plates, badge numbers, or any other PII of ICE agents or other law-enforcement personnel.
- Names, faces, addresses, or other PII of private individuals affected by ICE actions, except when explicitly published by them or their authorized counsel/family.
- IP addresses of public users (we do not log them at the application layer; reverse-proxy logs are disabled or scrubbed within 24h).
- Email addresses, phone numbers, or device identifiers from public users.
- Anything that would allow a future operator (or a subpoena) to deanonymize a reporter.

## 3. Time decay

- Every sighting's display weight = `exp(-age_hours / 6)`.
- Sightings auto-hide after 8 hours.
- Resources, events, and legal-aid entries do not decay; they are reviewed quarterly for accuracy.
- The historical archive is internal-only, used for moderation training and aggregate reporting; it is never queryable by location + time at the API.

## 4. Geo coarsening

- Sighting locations are rounded to ~200m before storage. The original precision is discarded, not stored-then-redacted.
- Resource locations (legal-aid offices, hotline desks, KYR distribution sites) may be precise because they are public.
- The map UI never zooms tighter than the coarsening grid.

## 5. PII scrubber

- All free-text notes pass through an LLM-driven scrubber before storage.
- The scrubber removes: personal names, plate numbers, exact street addresses (any house number is dropped), phone numbers, vehicle descriptions tied to a person, and physical descriptions of individuals.
- The scrubber preserves: agency descriptions ("ICE", "ERO", "marked van"), location landmarks ("near 47th & Western"), counts ("3 vehicles"), and timing.
- Scrubber output is logged for moderation review against a golden test set; target leak rate on the golden set is 0.

## 6. Crowd verification

- Every public report has confirm/deny buttons.
- Auto-hide threshold: `deny:confirm > 2:1` with N ≥ 5.
- Confirm/deny actions are anonymous and rate-limited per device.
- Flagged reports go to the moderator queue.

## 7. Rate limits

- Public submission: 5 reports/hour/device.
- Public confirm/deny: 60 actions/hour/device.
- Public read API: 600 requests/hour/IP (IP not logged beyond the reverse proxy's in-memory counter).

## 8. Escalation to hosted LLMs

- Local Ollama inference is the default for all user-submitted content.
- Hosted Claude API is used only for: ambiguous classifier outputs (confidence < 0.7), translations of multilingual or slang-heavy notes, and moderator-assist on flagged items.
- User-submitted notes sent to Claude are scrubbed first (PII scrubber runs locally before any external API call) and content-hashed for cache-only retention; we do not retain prompts or responses beyond 24 hours.

## 9. Operational security

- Tor / VPN / residential / mobile traffic is treated identically. No "browser fingerprinting" or anti-fraud heuristics that would discriminate against privacy-preserving users.
- HTTPS only, HSTS preloaded.
- Database backups are encrypted at rest with a key held by the fiscal sponsor's counsel, not the operator.
- Source code is open. Any private operational secret (API keys, partner tokens) lives in Hashicorp Vault or the equivalent and rotates quarterly.

## 10. What we do when pressured

- Any government request for user data is forwarded immediately to the fiscal sponsor's counsel and posted to a public transparency report within 30 days (or as soon as legally permitted).
- We will not implement back-doors. If forced to, we shut the service down and post the reason.
- The codebase, schema, and documentation are public so that the community can fork and re-host if the canonical instance is taken down.

## 11. Out of scope

- We do not facilitate, encourage, or coordinate physical confrontation with law-enforcement personnel.
- We do not provide concealment, transit, or harboring services. We link to legal aid; we do not advise on evading lawful process.
- We do not allow private messaging between users on the platform.

## 12. Reporting a safety bug

If you find a way to deanonymize a reporter, leak agent or private-individual PII, bypass time decay, or otherwise violate this policy: **do not file a public issue.** Email `security@fuzzbuster.example` (placeholder until fiscal sponsor is in place) with details. We will publish a fix and a CVE-style advisory.
