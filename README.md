# The Idea Engine Architecture

The Idea Engine is a Go + Next.js system that monitors community complaints, product reviews, and transcript feeds, then turns them into structured SaaS opportunities with a live dashboard.

## What is implemented

### Backend MVP
- Go ingestion pipeline with configurable sources.
- Redis-based deduplication with TTL.
- Kafka queue between ingestion and intelligence workers.
- PostgreSQL storage for structured insights.
- Recurring pain-point clustering with `cluster_key` / `cluster_label`.
- pgvector-ready embeddings with semantic retrieval APIs.
- Analyzer abstraction with `gemini`, `groq`, and `mock` providers.
- SSE endpoint for a real-time pain-point stream.

### Source adapters
- Dcard forum scraper with keyword filtering and detail fetch.
- Reddit scraper using OAuth client credentials.
- App Store / Atom feed scraper for review-style feeds.
- Transcript feed scraper for JSON feeds produced by Cloudflare Workers or other collectors.

### Frontend MVP
- Next.js app router structure.
- Tailwind dashboard with:
  - snapshot metrics
  - featured high-signal ideas
  - live SSE stream
  - source and keyword badges

## Implementation plan used

1. Stabilize infrastructure and configuration.
2. Normalize every source into one shared `ScrapedPost` model.
3. Push deduplicated items into Kafka.
4. Analyze with an LLM worker pool and persist to Postgres.
5. Expose `insights`, `stats`, and `stream` APIs.
6. Build a live Next.js dashboard on top of those APIs.
7. Add `.env` template and a GitHub Actions cron trigger.

## Project structure

- `main.go`: service bootstrap and graceful shutdown.
- `engine.go`: ingestion scheduler and intelligence workers.
- `config.go`: environment-driven runtime config.
- `dcard.go`, `reddit.go`, `appstore.go`, `transcript.go`: ingestion adapters.
- `llm.go`: analyzer providers and prompt logic.
- `postgres.go`: schema migration and query layer.
- `server.go`: Gin routes, stats API, SSE, internal ingestion trigger.
- `app/page.tsx`: live dashboard.
- `DEPLOYMENT_GUIDE.md`: free-platform deployment strategy and Docker entrypoints.
- `docs/koyeb_runbook.md`: Koyeb backend deployment runbook.
- `docs/koyeb_token_and_env_setup.md`: how to get `KOYEB_TOKEN` and a usable backend env file.
- `docs/huggingface_space_runbook.md`: Hugging Face Space frontend deployment runbook.
- `koyeb/`: repo-side deployment manifest templates for Koyeb.
- `scripts/`: deploy helpers for Koyeb and Hugging Face.
- `.github/workflows/ingestion-cron.yml`: scheduled remote ingestion trigger.
- `.github/workflows/deploy-koyeb-backend.yml`: GitHub Actions deploy to Koyeb.
- `.github/workflows/deploy-hf-space-frontend.yml`: GitHub Actions deploy to Hugging Face Space.

## Quick start

### 1. Start dependencies

```bash
docker compose up -d
```

### 2. Configure environment

```bash
cp .env.example .env
```

Important variables:
- `LLM_PROVIDER=mock|gemini|groq`
- `EMBEDDING_PROVIDER=mock|gemini`
- `EMBEDDING_DIMENSIONS=256`
- `GEMINI_API_KEY` or `GROQ_API_KEY`
- `GEMINI_EMBEDDING_MODEL=gemini-embedding-001`
- `REDDIT_CLIENT_ID` and `REDDIT_CLIENT_SECRET` for Reddit ingestion
- `APP_STORE_FEEDS` for review feeds
- `TRANSCRIPT_FEED_URLS` for Cloudflare Worker transcript feeds
- `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`

### 3. Run the backend

```bash
go run .
```

### 4. Run the frontend

```bash
npm install
npm run dev
```

Frontend: `http://localhost:3000`  
Backend API: `http://localhost:8080`

## APIs

- `GET /healthz`
- `GET /api/insights?limit=50`
- `GET /api/insights/:id/similar?limit=8`
- `GET /api/stats`
- `GET /api/trends?limit=12&window_hours=168`
- `GET /api/search?q=manual+spreadsheet&limit=8`
- `GET /api/stream`
- `POST /internal/ingest/run`

`/internal/ingest/run` can be protected with `INTERNAL_API_TOKEN`. The included GitHub Actions workflow can call this endpoint on a schedule.

## Notes and next steps

- Current clustering is heuristic and keyword-aware; semantic retrieval now uses pgvector-backed embeddings, and the next upgrade would be higher-quality embedding models plus indexed ANN search.
- `mock` embeddings use a deterministic hashing strategy, while Gemini embeddings use the official `embedContent` API.
- Transcript collection is intentionally feed-based so you can plug in a Cloudflare Worker later.
- The `mock` analyzer is useful for local development when you do not want to burn LLM quota.
- Current frontend uses SSE instead of polling for the live stream, with periodic snapshot refresh for stats.
