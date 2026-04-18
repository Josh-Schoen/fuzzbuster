# Fuzzbuster

Community-safety platform that aggregates public reports of ICE activity and surfaces legal-aid, Know-Your-Rights, mutual-aid, and rapid-response resources for affected individuals and families.

## What this is

- An installable Progressive Web App (Next.js 15 + React + TypeScript) that works on any modern browser, with a state-scoped map, anonymous submission, and embedded resource directory.
- A backend of small Go services connected by gRPC + protobuf, with a separate Node + Playwright + Stagehand + Claude Agent SDK service that drives LLM browser agents to aggregate events from public sources.
- Local-first LLM inference via Ollama (Llama 3.1 8B for classify/summarize, `nomic-embed-text` for dedup, `qwen2.5:7b` for ES↔EN), with Anthropic Claude Haiku 4.5 / Sonnet 4.5 as a low-volume escalation tier.

## What this is **not**

- Not a vigilante tool. Never publishes ICE agent PII (names, faces, plates).
- Not a real-time operational interference tool. All sightings time-decay (visibility weight `exp(-age_hours/6)`, auto-hidden after 8 hours).
- Not a replacement for trained rapid-response networks (UWD MigraWatch, Mijente, NIJC, CHIRLA, Make the Road, etc.). It complements them.
- Not a place to hold donations for individuals. Family-aid donations route to established bond/mutual-aid funds (Envision Freedom Fund, Freedom for Immigrants, RAICES, local rapid-response networks).

See [`docs/safety-policy.md`](docs/safety-policy.md) for the full safety contract that every component is built to honor.

## Status

Pre-v0 skeleton. **No public deployment until** fiscal sponsor is approved and ACLU/EFF legal review is on file. See [`docs/legal-review-checklist.md`](docs/legal-review-checklist.md).

## Repo layout

```
fuzzbuster/
├── docs/              # safety-policy, legal-review-checklist, data-sources
├── proto/             # source-of-truth protobuf schemas (Buf-managed)
├── apps/
│   ├── web/           # Next.js 15 PWA (React + TS)
│   ├── api/           # Go + chi public REST + gRPC server
│   ├── orchestrator/  # Go + Asynq cron, scrape job dispatch, dedup
│   ├── llm-gateway/   # Go: Ollama + Anthropic SDK, escalation policy
│   ├── moderation/    # Go: PII scrub, decay, crowd-verify
│   └── scraper/       # Node + TS + Playwright + Stagehand + Claude Agent SDK
├── pkg/               # shared Go libraries
├── infra/             # docker-compose, hetzner cloud-init
└── .github/workflows/ # buf lint, go test, eslint, playwright e2e
```

## Local development

Prerequisites: Docker, Go 1.23+, Node 20+, pnpm, [Buf CLI](https://buf.build/docs/installation).

```sh
# Bring up Postgres+PostGIS+pgvector, Redis, Nominatim, Ollama
docker compose -f infra/docker-compose.yml up -d

# Generate Go + TS code from proto/
buf generate proto

# Backend
cd apps/api && go run ./...

# Frontend
cd apps/web && pnpm install && pnpm dev

# Scraper
cd apps/scraper && pnpm install && pnpm dev
```

## Contributing

Read [`docs/safety-policy.md`](docs/safety-policy.md) **before** opening a PR. Any change that affects what is collected, retained, displayed, or scrubbed requires a safety-review tag and a second reviewer.

## License

TBD (likely AGPL-3.0). No production use until license is set.
