# Koyeb Token And Backend Env Setup

This guide shows you how to get:

- `KOYEB_TOKEN`
- a usable `.env.koyeb.backend`

The goal is to give you a backend environment that can actually boot on Koyeb with the current codebase.

## What you need to know first

This project currently has different connection requirements for each dependency:

- PostgreSQL: standard connection string works.
- Redis: URL-based connection works, including TLS URLs such as `rediss://...`.
- Kafka: the current code only accepts a plain broker address like `host:9092`.

Why this matters:

- [redis.go](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/redis.go#L15) uses `redis.ParseURL`, so `redis://` and `rediss://` are both compatible.
- [kafka.go](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/kafka.go#L15) uses `kafka.TCP(brokerURL)` and does not currently configure SASL or TLS.

That means:

- managed PostgreSQL is easy
- managed Redis is easy
- managed Kafka is usually **not** plug-and-play with the current code

If you want the fastest path to a working Koyeb deploy **today**, use:

- Koyeb for the backend app
- a separate VM or VPS you control for Postgres + Redis + Kafka

## 1. Get `KOYEB_TOKEN`

Follow this flow in Koyeb:

1. Sign in to Koyeb.
2. Open `https://app.koyeb.com/user/settings/api`.
3. Open the **API** section if it is not already selected.
4. Click **Create API token** or **Create API Access Token**.
5. Give it a clear name, for example `idea-engine-github-actions`.
6. Copy the token immediately and store it safely.

Important:

- You will not be able to view the full token again later.
- If you lose it, create a new one and rotate the old one out.

### Store it locally

```bash
export KOYEB_TOKEN='replace-me'
```

### Store it in GitHub Actions

Create a repository secret named:

```text
KOYEB_TOKEN
```

## 2. Choose how you will obtain `DATABASE_URL`, `REDIS_URL`, and `KAFKA_BROKER`

### Recommended with the current code: one infra VM

Because Kafka is currently plaintext-only in this codebase, the most reliable setup is:

- one Ubuntu VM, VPS, or cloud host you control
- run the repo's `docker-compose.yml` there
- point Koyeb at that host

This is the fastest way to create a **working** `.env.koyeb.backend`.

### Alternative hybrid setup

You can also mix providers:

- PostgreSQL: Neon
- Redis: Upstash Redis
- Kafka: your own VM running Kafka or Redpanda in plaintext mode

That works too, but Kafka still needs to come from infrastructure you control unless we add SASL/TLS support to the app first.

## 3. Fastest working route: run Postgres + Redis + Kafka on one VM

### 3.1 Provision a VM

Use any Linux machine with:

- Docker
- Docker Compose
- a public IP or DNS name

For a quick demo, a small Ubuntu VM is enough.

### 3.2 Copy this repo to the VM

On the VM:

```bash
git clone https://github.com/dennislee928/The-Idea-Engine-Architecture.git
cd The-Idea-Engine-Architecture
```

### 3.3 Update the Kafka advertised listener

Edit [docker-compose.yml](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/docker-compose.yml#L1).

Change:

```yaml
KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092
```

To your VM's real host or public IP:

```yaml
KAFKA_CFG_ADVERTISED_LISTENERS=PLAINTEXT://<your-vm-host>:9092
```

If you skip this, Koyeb will connect to Kafka and get told to use `localhost:9092`, which will fail.

### 3.4 Start the infra stack

On the VM:

```bash
docker compose up -d
```

This repo already defines:

- PostgreSQL with pgvector
- Redis
- Zookeeper
- Kafka

### 3.5 Open the required ports

Your Koyeb service needs network access to:

- `5432` for PostgreSQL
- `6379` for Redis
- `9092` for Kafka

For a quick demo, you can open these ports publicly.

For anything beyond a demo:

- restrict access with firewall rules
- prefer private networking or VPN
- do not leave plaintext Kafka exposed on the public internet long-term

## 4. Build a usable `.env.koyeb.backend`

Create `.env.koyeb.backend` in the repo root:

```dotenv
PORT=8080

DATABASE_URL=postgres://idea_admin:idea_password@<your-vm-host>:5432/idea_engine?sslmode=disable
REDIS_URL=redis://<your-vm-host>:6379/0
KAFKA_BROKER=<your-vm-host>:9092

KAFKA_TOPIC=raw-posts
KAFKA_GROUP_ID=idea-engine-llm-group
DEDUP_TTL=168h
INGESTION_INTERVAL=30m
INGESTION_BATCH_SIZE=10
INTELLIGENCE_WORKERS=1
INTERNAL_API_TOKEN=replace-me

LLM_PROVIDER=mock
EMBEDDING_PROVIDER=mock
EMBEDDING_DIMENSIONS=256

GEMINI_API_KEY=
GEMINI_MODEL=gemini-1.5-flash
GEMINI_EMBEDDING_MODEL=gemini-embedding-001
GROQ_API_KEY=
GROQ_MODEL=llama-3.1-8b-instant
GROQ_BASE_URL=https://api.groq.com/openai/v1/chat/completions

DCARD_FORUMS=softwareengineer,job
DCARD_KEYWORDS=求救,有人也這樣嗎,手動,好累,卡住,崩潰
REDDIT_CLIENT_ID=
REDDIT_CLIENT_SECRET=
REDDIT_USER_AGENT=idea-engine/0.1
REDDIT_SUBREDDITS=SmallBusiness,Excel,RemoteWork
REDDIT_KEYWORDS=manual,spreadsheet,frustrated,pain,workaround,too slow
APP_STORE_FEEDS=
APP_STORE_KEYWORDS=slow,expensive,subscription,too many features
TRANSCRIPT_FEED_URLS=
TRANSCRIPT_KEYWORDS=copy paste,manual,workflow,hack,spreadsheet
```

This is the simplest deployable starting point because:

- Postgres matches the repo's compose credentials
- Redis matches the repo's compose setup
- Kafka matches the app's current plaintext client behavior
- `mock` providers keep CPU and memory pressure down on Koyeb

## 5. Optional: use managed Postgres and Redis

If you do not want to self-host Postgres and Redis, you can split the setup:

- `DATABASE_URL`: Neon
- `REDIS_URL`: Upstash Redis
- `KAFKA_BROKER`: still from your own VM for now

### 5.1 Get `DATABASE_URL` from Neon

Recommended flow:

1. Create a Neon project.
2. Open the project dashboard.
3. Click **Connect**.
4. Choose the branch, database, and role.
5. Copy the generated Postgres connection string.

Use the pooled connection string if you expect more concurrent connections.

Example:

```dotenv
DATABASE_URL=postgresql://<user>:<password>@<host>/<db>?sslmode=require
```

### 5.2 Get `REDIS_URL` from Upstash Redis

Recommended flow:

1. Create an Upstash Redis database.
2. Open the database page.
3. Copy the connection details shown in the console.
4. Use the TLS connection string.

Example:

```dotenv
REDIS_URL=rediss://:<password>@<endpoint>:<port>
```

Because this project uses `redis.ParseURL`, `rediss://` is valid here.

### 5.3 Example hybrid `.env.koyeb.backend`

```dotenv
PORT=8080

DATABASE_URL=postgresql://<neon-user>:<neon-password>@<neon-host>/<db>?sslmode=require
REDIS_URL=rediss://:<upstash-password>@<upstash-endpoint>:<upstash-port>
KAFKA_BROKER=<your-vm-host>:9092

KAFKA_TOPIC=raw-posts
KAFKA_GROUP_ID=idea-engine-llm-group
DEDUP_TTL=168h
INGESTION_INTERVAL=30m
INGESTION_BATCH_SIZE=10
INTELLIGENCE_WORKERS=1
INTERNAL_API_TOKEN=replace-me

LLM_PROVIDER=mock
EMBEDDING_PROVIDER=mock
EMBEDDING_DIMENSIONS=256
```

## 6. Validate the environment before deploying to Koyeb

If you have local tools installed, test each dependency before deploying.

### PostgreSQL

```bash
psql "$DATABASE_URL" -c 'select 1;'
```

### Redis

```bash
redis-cli -u "$REDIS_URL" ping
```

Expected:

```text
PONG
```

### Kafka

If you have `kcat`:

```bash
kcat -b "$KAFKA_BROKER" -L
```

If Kafka metadata loads correctly, the broker is reachable.

## 7. Put the env file into GitHub Actions

The Koyeb GitHub Actions workflow expects a multiline secret named:

```text
KOYEB_ENV_FILE_CONTENTS
```

The easiest way to create it:

```bash
cat .env.koyeb.backend
```

Copy the full contents and paste them into the GitHub repository secret `KOYEB_ENV_FILE_CONTENTS`.

The workflow will recreate `.env.koyeb.backend` at runtime and deploy using:

- [scripts/deploy-koyeb-backend.sh](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/scripts/deploy-koyeb-backend.sh#L1)
- [.github/workflows/deploy-koyeb-backend.yml](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/.github/workflows/deploy-koyeb-backend.yml#L1)

## 8. Minimum checklist

You are ready to deploy when all of these are true:

- `KOYEB_TOKEN` exists
- `.env.koyeb.backend` exists
- `DATABASE_URL` is reachable from outside your laptop
- `REDIS_URL` is reachable from outside your laptop
- `KAFKA_BROKER` is reachable from outside your laptop
- Kafka advertises the VM host, not `localhost`
- you can hit those dependencies from a network other than your local machine

## 9. Production note

For production, I would not keep plaintext Kafka exposed publicly.

The cleaner long-term path is:

1. add SASL/TLS support to [kafka.go](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/kafka.go#L15)
2. switch Kafka to a managed provider
3. keep Postgres and Redis on managed services too

Until then, a self-hosted Kafka broker is the most honest "works with the current code" answer.

## Sources

- Koyeb API token page: https://app.koyeb.com/user/settings/api
- Koyeb CLI reference: https://www.koyeb.com/docs/build-and-deploy/cli/reference
- Koyeb environment variables: https://www.koyeb.com/docs/build-and-deploy/environment-variables
- Neon connection guide: https://neon.com/docs/get-started/connect-neon
- Upstash Redis client connection guide: https://upstash.com/docs/redis/howto/connectclient
