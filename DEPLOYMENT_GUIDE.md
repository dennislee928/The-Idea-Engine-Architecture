# Deployment Guide

This project can be deployed as two separate services:

- `backend`: Go API, ingestion scheduler, LLM analysis, SSE stream
- `frontend`: Next.js dashboard that talks to the backend over `NEXT_PUBLIC_API_BASE_URL`

## Recommended split

### Backend

Use [Dockerfile.backend](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/Dockerfile.backend).
Automate with [scripts/deploy-koyeb-backend.sh](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/scripts/deploy-koyeb-backend.sh).
Template env file: [.env.koyeb.backend.example](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/.env.koyeb.backend.example).

Best for:
- Koyeb
- Back4App Containers
- Hugging Face Spaces Docker

Required environment variables for a realistic deployment:
- `DATABASE_URL`
- `REDIS_URL`
- `KAFKA_BROKER`
- `LLM_PROVIDER`
- `EMBEDDING_PROVIDER`

For a lightweight demo deployment:
- `LLM_PROVIDER=mock`
- `EMBEDDING_PROVIDER=mock`
- `INTELLIGENCE_WORKERS=1`
- `INGESTION_INTERVAL=30m`
- `INGESTION_BATCH_SIZE=10`

### Frontend

Use [Dockerfile.frontend](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/Dockerfile.frontend).
Automate with [scripts/deploy-hf-space-frontend.sh](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/scripts/deploy-hf-space-frontend.sh).
Template env file: [.env.hf.space.example](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/.env.hf.space.example).

Required environment variable:
- `NEXT_PUBLIC_API_BASE_URL`

## Platform suggestions

### Koyeb

Recommended role:
- public backend API

Why:
- GitHub or Docker-based deploys fit the Go API well
- free web service is enough for a lightweight demo

Suggested setup:
1. Deploy the backend with `Dockerfile.backend`
2. Set region to the same geography as your main audience
3. Start with `mock` analyzer and embedder if you want a zero-quota public demo
4. Trigger ingestion with GitHub Actions hitting `/internal/ingest/run`

Important caution:
- Koyeb’s free Postgres tier is limited and is better treated as a demo database, not the permanent analytics store for this project

### Hugging Face Spaces

Recommended role:
- public interactive demo

Why:
- Docker Spaces can run arbitrary containers
- ideal for showcasing the dashboard or a simplified semantic search demo

Suggested setup:
1. Deploy the frontend with `Dockerfile.frontend`
2. Point it at a remote backend via `NEXT_PUBLIC_API_BASE_URL`
3. If you deploy the backend here instead, expect cold starts after inactivity

Important caution:
- free `cpu-basic` Spaces sleep after inactivity, so this is better for demos than for ingestion or always-on APIs

### Serv00

Recommended role:
- SSH-based compatibility testing
- cron-driven collectors

Why:
- strong fit for shell access and low-level runtime validation
- useful for running CLI or ingestion probes in a constrained environment

Suggested setup:
1. Use it to test lightweight collectors or CLI-style workflows
2. Use cron there to fetch data or hit your backend ingestion endpoint

Important caution:
- this project’s full Kafka + Redis + Postgres + SSE stack is heavier than what I’d target on Serv00 for a long-lived public service

### Back4App Containers

Recommended role:
- minimal backend demo

Why:
- supports Docker directly
- simple GitHub-connected preview deployments

Suggested setup:
1. Deploy the backend only
2. Keep worker count low
3. Use `mock` modes unless you are sure the memory budget is enough

Important caution:
- 256 MB RAM is tight for anything beyond a stripped-down API demo

### Alwaysdata

Recommended role:
- private staging or lightweight internal MVP

Why:
- built-in services like PostgreSQL, Redis, SSH, and cron reduce moving parts

Suggested setup:
1. Use Alwaysdata when you want one provider for app + DB + cache in a very small environment
2. Run the frontend separately or keep the backend as the main deployed service

Important caution:
- their free plan has meaningful restrictions, including limits around commercial use and custom websites, so I would not choose it for the public commercial version of this project

### Deta Space

Recommended role:
- not recommended for this architecture

Why:
- this project depends on a long-running Go service, SSE, and background-style ingestion
- Deta’s model is a poor fit for that operating profile

## My recommended free-stack combinations

### Best overall demo

- Backend API: Koyeb
- Frontend demo: Hugging Face Spaces
- Ingestion trigger: GitHub Actions cron

### Best for cheap internal testing

- Backend MVP: Alwaysdata
- Frontend: Hugging Face Spaces or local
- Compatibility tests: Serv00

### Best for minimal complexity

- Backend only: Koyeb
- Frontend served locally during development

## What I would avoid on free tiers

- Running Kafka as a separate public service on these platforms
- Treating free databases as the long-term source of truth for a production SaaS
- Hosting the entire stack in one tiny container unless everything is in demo mode
