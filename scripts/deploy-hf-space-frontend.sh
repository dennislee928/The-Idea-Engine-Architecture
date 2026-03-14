#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

HF_ENV_FILE="${HF_ENV_FILE:-$ROOT_DIR/.env.hf.space}"
HF_USERNAME="${HF_USERNAME:-}"
HF_SPACE_NAME="${HF_SPACE_NAME:-}"
HF_TOKEN="${HF_TOKEN:-}"
HF_SPACE_BRANCH="${HF_SPACE_BRANCH:-main}"
HF_SPACE_TITLE="${HF_SPACE_TITLE:-The Idea Engine}"
HF_SPACE_COLOR_FROM="${HF_SPACE_COLOR_FROM:-green}"
HF_SPACE_COLOR_TO="${HF_SPACE_COLOR_TO:-blue}"
NEXT_PUBLIC_API_BASE_URL="${NEXT_PUBLIC_API_BASE_URL:-}"

usage() {
  cat <<'EOF'
Usage:
  HF_USERNAME=<user> HF_SPACE_NAME=<space> HF_TOKEN=<token> NEXT_PUBLIC_API_BASE_URL=https://api.example.com \
    ./scripts/deploy-hf-space-frontend.sh

Required:
  HF_USERNAME                 Hugging Face username or org
  HF_SPACE_NAME               Hugging Face Space name
  HF_TOKEN                    Hugging Face write token
  NEXT_PUBLIC_API_BASE_URL    Public backend API base URL

Optional:
  HF_SPACE_BRANCH             Default: main
  HF_SPACE_TITLE              Default: The Idea Engine
  HF_SPACE_COLOR_FROM         Default: green
  HF_SPACE_COLOR_TO           Default: blue
EOF
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || fail "Missing required command: $1"
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  require_cmd git
  require_cmd mktemp

  if [[ -f "${HF_ENV_FILE}" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "${HF_ENV_FILE}"
    set +a
  fi

  [[ -n "${HF_USERNAME}" ]] || fail "HF_USERNAME is required"
  [[ -n "${HF_SPACE_NAME}" ]] || fail "HF_SPACE_NAME is required"
  [[ -n "${HF_TOKEN}" ]] || fail "HF_TOKEN is required"
  [[ -n "${NEXT_PUBLIC_API_BASE_URL}" ]] || fail "NEXT_PUBLIC_API_BASE_URL is required"

  local workdir
  workdir="$(mktemp -d)"
  trap 'rm -rf "${workdir}"' EXIT

  mkdir -p "${workdir}/app" "${workdir}/public"

  cp -R "${ROOT_DIR}/app/." "${workdir}/app/"
  if [[ -d "${ROOT_DIR}/public" ]]; then
    cp -R "${ROOT_DIR}/public/." "${workdir}/public/" 2>/dev/null || true
  fi

  cp "${ROOT_DIR}/package.json" "${workdir}/package.json"
  cp "${ROOT_DIR}/package-lock.json" "${workdir}/package-lock.json"
  cp "${ROOT_DIR}/tsconfig.json" "${workdir}/tsconfig.json"
  cp "${ROOT_DIR}/next-env.d.ts" "${workdir}/next-env.d.ts"
  cp "${ROOT_DIR}/postcss.config.js" "${workdir}/postcss.config.js"
  cp "${ROOT_DIR}/tailwind.config.ts" "${workdir}/tailwind.config.ts"

  cat > "${workdir}/.env.production" <<EOF
NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL}
EOF

  cat > "${workdir}/Dockerfile" <<'EOF'
FROM node:20-alpine AS deps

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

FROM node:20-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY package.json package-lock.json tsconfig.json next-env.d.ts postcss.config.js tailwind.config.ts ./
COPY .env.production ./.env.production
COPY app ./app
COPY public ./public

ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000

COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/package-lock.json ./package-lock.json
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/app ./app
COPY --from=builder /app/public ./public

EXPOSE 3000

CMD ["npm", "run", "start"]
EOF

  cat > "${workdir}/README.md" <<EOF
---
title: "${HF_SPACE_TITLE}"
colorFrom: ${HF_SPACE_COLOR_FROM}
colorTo: ${HF_SPACE_COLOR_TO}
sdk: docker
app_port: 3000
pinned: false
---

# ${HF_SPACE_TITLE}

Generated deployment context for the Idea Engine frontend demo.
EOF

  (
    cd "${workdir}"
    git init >/dev/null 2>&1
    git checkout -B "${HF_SPACE_BRANCH}" >/dev/null 2>&1
    git config user.name "Idea Engine Deploy Bot"
    git config user.email "deploybot@example.com"
    git add .
    git commit -m "Deploy frontend to Hugging Face Space" >/dev/null 2>&1
    git push --force "https://${HF_USERNAME}:${HF_TOKEN}@huggingface.co/spaces/${HF_USERNAME}/${HF_SPACE_NAME}" "${HF_SPACE_BRANCH}"
  )

  echo "Hugging Face Space deployment pushed successfully."
  echo "Open: https://huggingface.co/spaces/${HF_USERNAME}/${HF_SPACE_NAME}"
}

main "$@"
