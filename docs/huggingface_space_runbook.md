# Hugging Face Space Deployment Runbook

This runbook deploys the **Next.js frontend demo** of The Idea Engine to a Hugging Face Docker Space.

## Scope

- Service: frontend demo
- Space type: Docker Space
- Built from: generated frontend-only deployment context
- Recommended for: public demo dashboard

## Before you start

You need:

1. A Hugging Face account.
2. A Docker Space already created in the Hugging Face UI.
3. A Hugging Face access token with write permission.
4. The public backend URL that the frontend should call.

Official references:

- Docker Spaces use `sdk: docker` in the Space `README.md`.
- You can set the public port with `app_port`.
- Spaces are Git repositories and can be synced by pushing to `https://huggingface.co/spaces/<user>/<space>`.

## 1. Create the Docker Space once

In the Hugging Face UI:

1. Click **New Space**.
2. Choose your account or org.
3. Set a Space name.
4. Choose **Docker** as the SDK.
5. Create the Space.

You only need to do this once.

## 2. Decide the backend API URL

This frontend needs a public backend URL, for example:

```bash
https://idea-engine-api.koyeb.app
```

The deployment script will bake this value into the frontend build as:

```bash
NEXT_PUBLIC_API_BASE_URL
```

## 3. Export the deploy variables

Set these in your shell:

```bash
export HF_USERNAME=<your_hf_username>
export HF_SPACE_NAME=<your_space_name>
export HF_TOKEN=<your_hf_write_token>
export NEXT_PUBLIC_API_BASE_URL=https://<your-backend-host>
```

Or start from the example file:

```bash
cp .env.hf.space.example .env.hf.space
```

Optional display metadata:

```bash
export HF_SPACE_TITLE="The Idea Engine"
export HF_SPACE_COLOR_FROM=green
export HF_SPACE_COLOR_TO=blue
```

## 4. Run the deploy script

```bash
./scripts/deploy-hf-space-frontend.sh
```

What the script does:

1. Creates a temporary frontend-only deploy context.
2. Generates a Docker Space `README.md` with `sdk: docker` and `app_port: 3000`.
3. Writes `.env.production` with `NEXT_PUBLIC_API_BASE_URL`.
4. Commits that generated context to a temporary git repo.
5. Force-pushes it to your Hugging Face Space repo.

## 5. Validate the Space

Open:

```text
https://huggingface.co/spaces/<HF_USERNAME>/<HF_SPACE_NAME>
```

When the Space finishes building, verify:

1. The page loads.
2. The dashboard can fetch `/api/insights` from your backend.
3. The semantic explorer works.

## 6. Update flow

Each time you change the frontend:

```bash
export NEXT_PUBLIC_API_BASE_URL=https://<your-backend-host>
./scripts/deploy-hf-space-frontend.sh
```

Because this script pushes a generated deployment context, it uses `--force`.

## 7. Common issues

### Space builds but the UI points to localhost

This means `NEXT_PUBLIC_API_BASE_URL` was not exported before deployment.

Fix:

```bash
export NEXT_PUBLIC_API_BASE_URL=https://<your-backend-host>
./scripts/deploy-hf-space-frontend.sh
```

### Space is slow to open

Free Spaces can sleep after inactivity. The first request after sleeping may take time while the container wakes up.

### SSE stream does not connect

Check that:

- the backend URL is correct
- the backend allows public access
- your backend host supports the `/api/stream` endpoint

## 8. GitHub Actions sync

Hugging Face documents a Git-based sync pattern:

```bash
git remote add space https://huggingface.co/spaces/HF_USERNAME/SPACE_NAME
git push --force space main
```

This repo already includes a workflow template you can use as-is:

- [.github/workflows/deploy-hf-space-frontend.yml](/Users/dennis_leedennis_lee/Documents/GitHub/The%20Idea%20Engine%20Architecture/.github/workflows/deploy-hf-space-frontend.yml)

Required GitHub secrets:

- `HF_USERNAME`
- `HF_SPACE_NAME`
- `HF_TOKEN`
- `NEXT_PUBLIC_API_BASE_URL`

Optional GitHub repository variables:

- `HF_SPACE_TITLE`
- `HF_SPACE_COLOR_FROM`
- `HF_SPACE_COLOR_TO`
