.PHONY: help up down proto go-test web-dev scraper-dev pull-models

help:
	@echo "Targets:"
	@echo "  up           bring up postgres+postgis+pgvector, redis, ollama, nominatim"
	@echo "  down         tear it back down (data preserved)"
	@echo "  proto        regenerate Go + TS code from proto/"
	@echo "  go-test      run all Go tests across the workspace"
	@echo "  web-dev      run the Next.js PWA"
	@echo "  scraper-dev  run the Node + Playwright + Stagehand scraper service"
	@echo "  pull-models  pull the Ollama models we use"

up:
	docker compose -f infra/docker-compose.yml up -d

down:
	docker compose -f infra/docker-compose.yml down

proto:
	cd proto && buf generate

go-test:
	go test ./...

web-dev:
	cd apps/web && pnpm install && pnpm dev

scraper-dev:
	cd apps/scraper && pnpm install && pnpm dev

pull-models:
	docker exec -it $$(docker compose -f infra/docker-compose.yml ps -q ollama) ollama pull llama3.1:8b-instruct-q4_K_M
	docker exec -it $$(docker compose -f infra/docker-compose.yml ps -q ollama) ollama pull nomic-embed-text
	docker exec -it $$(docker compose -f infra/docker-compose.yml ps -q ollama) ollama pull qwen2.5:7b-instruct
