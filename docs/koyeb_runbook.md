# Koyeb Deployment Runbook

This runbook deploys the **Go backend** of The Idea Engine to Koyeb using a Git-driven Docker build.

## Scope

- Service: backend API
- Dockerfile: [Dockerfile.backend](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/Dockerfile.backend)
- Recommended for: public demo API or lightweight MVP backend

## Before you start

You need:

1. A Koyeb account.
2. A GitHub repository containing this project.
3. The Koyeb CLI installed.
4. A backend environment file ready for Koyeb.

Official references:

- Koyeb CLI install: `brew install koyeb/tap/koyeb` or `curl -fsSL https://raw.githubusercontent.com/koyeb/koyeb-cli/master/install.sh | sh`
- Git-driven Docker deployment is supported by `koyeb services create ... --git-builder docker`

## 1. Prepare the repository

Push this project to GitHub first.

Example:

```bash
git remote add origin git@github.com:<your-user>/<your-repo>.git
git push -u origin main
```

The deploy script expects a GitHub-style repository reference such as:

```bash
github.com/<your-user>/<your-repo>
```

## 2. Prepare the Koyeb environment file

Create a file named `.env.koyeb.backend` in the repo root.

You can start from:

```bash
cp .env.koyeb.backend.example .env.koyeb.backend
```

Minimal demo example:

```dotenv
PORT=8080
DATABASE_URL=postgres://...
REDIS_URL=redis://...
KAFKA_BROKER=...
KAFKA_TOPIC=raw-posts
KAFKA_GROUP_ID=idea-engine-llm-group
INTERNAL_API_TOKEN=replace-me
LLM_PROVIDER=mock
EMBEDDING_PROVIDER=mock
INTELLIGENCE_WORKERS=1
INGESTION_INTERVAL=30m
INGESTION_BATCH_SIZE=10
```

If you want real model calls:

```dotenv
LLM_PROVIDER=gemini
EMBEDDING_PROVIDER=gemini
GEMINI_API_KEY=...
GEMINI_MODEL=gemini-1.5-flash
GEMINI_EMBEDDING_MODEL=gemini-embedding-001
```

## 3. Install and authenticate the Koyeb CLI

Install:

```bash
brew install koyeb/tap/koyeb
```

Or:

```bash
curl -fsSL https://raw.githubusercontent.com/koyeb/koyeb-cli/master/install.sh | sh
export PATH="$HOME/.koyeb/bin:$PATH"
```

Login interactively:

```bash
koyeb login
```

Or use an API token:

```bash
export KOYEB_TOKEN=your_token_here
```

## 4. Run the deploy script

Set the required deployment metadata:

```bash
export KOYEB_APP=idea-engine
export KOYEB_SERVICE=idea-engine-api
export KOYEB_GIT=github.com/<your-user>/<your-repo>
export KOYEB_GIT_BRANCH=main
```

Optional but useful:

```bash
export KOYEB_REGION=fra
export KOYEB_INSTANCE_TYPE=nano
export KOYEB_ENV_FILE=.env.koyeb.backend
```

Repo-side manifest templates:

- [koyeb/app.yaml](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/koyeb/app.yaml)
- [koyeb/service.backend.manifest.yaml](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/koyeb/service.backend.manifest.yaml)

These are project templates and documentation artifacts. Koyeb’s current official docs emphasize CLI and control-panel deployment rather than a repo-native `app.yaml`.

Deploy:

```bash
./scripts/deploy-koyeb-backend.sh
```

What the script does:

1. Ensures the Koyeb app exists.
2. Creates the backend service if missing.
3. Otherwise updates the existing service.
4. Uses `Dockerfile.backend` for the build.
5. Waits for deployment to finish.

## 5. Validate the deployment

Check the app and service:

```bash
koyeb apps get "$KOYEB_APP"
koyeb services get "$KOYEB_SERVICE" --app "$KOYEB_APP"
```

Then call the health endpoint using the generated `*.koyeb.app` URL:

```bash
curl https://<your-app-subdomain>.koyeb.app/healthz
```

Expected result:

```json
{"status":"ok","time":"..."}
```

## 6. Update flow

After code changes:

1. Push the branch to GitHub.
2. Re-run the same script.

```bash
git push origin main
./scripts/deploy-koyeb-backend.sh
```

## 7. Recommended Koyeb settings for this project

For a free or demo-friendly deployment:

- `LLM_PROVIDER=mock`
- `EMBEDDING_PROVIDER=mock`
- `INTELLIGENCE_WORKERS=1`
- `INGESTION_BATCH_SIZE=10`
- `INGESTION_INTERVAL=30m`

That keeps memory and CPU pressure much lower.

## 8. Common issues

### Build succeeds but app is unhealthy

Check that:

- `PORT=8080` is consistent with the service port.
- `/healthz` responds correctly.
- your database, Redis, and Kafka endpoints are reachable from Koyeb.

### Service deploys but ingestion fails

This usually means one of:

- invalid `DATABASE_URL`
- invalid `REDIS_URL`
- invalid `KAFKA_BROKER`
- LLM keys missing when not in `mock` mode

### Koyeb cannot build the repo

Make sure:

- the repo is accessible to Koyeb
- `KOYEB_GIT` is in `github.com/owner/repo` form
- `Dockerfile.backend` exists at the repo root
