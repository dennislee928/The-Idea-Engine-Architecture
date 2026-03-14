#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

KOYEB_APP="${KOYEB_APP:-idea-engine}"
KOYEB_SERVICE="${KOYEB_SERVICE:-idea-engine-api}"
KOYEB_GIT="${KOYEB_GIT:-}"
KOYEB_GIT_BRANCH="${KOYEB_GIT_BRANCH:-main}"
KOYEB_GIT_SHA="${KOYEB_GIT_SHA:-}"
KOYEB_DOCKERFILE="${KOYEB_DOCKERFILE:-Dockerfile.backend}"
KOYEB_REGION="${KOYEB_REGION:-was}"
KOYEB_INSTANCE_TYPE="${KOYEB_INSTANCE_TYPE:-nano}"
KOYEB_ENV_FILE="${KOYEB_ENV_FILE:-$ROOT_DIR/.env.koyeb.backend}"
KOYEB_PORT_SPEC="${KOYEB_PORT_SPEC:-8080:http}"
KOYEB_ROUTE_SPEC="${KOYEB_ROUTE_SPEC:-/:8080}"
KOYEB_HEALTHCHECK="${KOYEB_HEALTHCHECK:-8080:http:/healthz}"
KOYEB_HEALTHCHECK_GRACE="${KOYEB_HEALTHCHECK_GRACE:-8080=30}"
KOYEB_MIN_SCALE="${KOYEB_MIN_SCALE:-1}"
KOYEB_MAX_SCALE="${KOYEB_MAX_SCALE:-1}"
KOYEB_WAIT_TIMEOUT="${KOYEB_WAIT_TIMEOUT:-10m}"
KOYEB_LIGHT_SLEEP_DELAY="${KOYEB_LIGHT_SLEEP_DELAY:-}"
KOYEB_DEEP_SLEEP_DELAY="${KOYEB_DEEP_SLEEP_DELAY:-}"

usage() {
  cat <<'EOF'
Usage:
  KOYEB_GIT=github.com/<user>/<repo> ./scripts/deploy-koyeb-backend.sh

Required:
  KOYEB_GIT              GitHub repository in github.com/<owner>/<repo> form

Optional:
  KOYEB_APP              Default: idea-engine
  KOYEB_SERVICE          Default: idea-engine-api
  KOYEB_GIT_BRANCH       Default: main
  KOYEB_GIT_SHA          Optional commit SHA override
  KOYEB_ENV_FILE         Default: .env.koyeb.backend
  KOYEB_REGION           Default: was
  KOYEB_INSTANCE_TYPE    Default: nano
  KOYEB_PORT_SPEC        Default: 8080:http
  KOYEB_ROUTE_SPEC       Default: /:8080
  KOYEB_HEALTHCHECK      Default: 8080:http:/healthz
  KOYEB_WAIT_TIMEOUT     Default: 10m
  KOYEB_TOKEN            Optional Koyeb API token
EOF
}

fail() {
  echo "Error: $*" >&2
  exit 1
}

run_koyeb() {
  if [[ -n "${KOYEB_TOKEN:-}" ]]; then
    koyeb --token "${KOYEB_TOKEN}" "$@"
  else
    koyeb "$@"
  fi
}

normalize_github_remote() {
  local remote="$1"

  remote="${remote#ssh://git@github.com/}"
  remote="${remote#git@github.com:}"
  remote="${remote#https://github.com/}"
  remote="${remote#http://github.com/}"
  remote="${remote%.git}"

  if [[ "${remote}" == github.com/* ]]; then
    echo "${remote}"
  else
    echo "github.com/${remote}"
  fi
}

load_env_args() {
  ENV_ARGS=()

  [[ -f "${KOYEB_ENV_FILE}" ]] || fail "Missing env file: ${KOYEB_ENV_FILE}"

  while IFS= read -r line || [[ -n "${line}" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    [[ -z "${line}" ]] && continue
    [[ "${line}" == \#* ]] && continue
    ENV_ARGS+=(--env "${line}")
  done < "${KOYEB_ENV_FILE}"
}

build_deploy_args() {
  DEPLOY_ARGS=(
    --type web
    --git "${KOYEB_GIT}"
    --git-branch "${KOYEB_GIT_BRANCH}"
    --git-builder docker
    --git-docker-dockerfile "${KOYEB_DOCKERFILE}"
    --instance-type "${KOYEB_INSTANCE_TYPE}"
    --regions "${KOYEB_REGION}"
    --ports "${KOYEB_PORT_SPEC}"
    --routes "${KOYEB_ROUTE_SPEC}"
    --checks "${KOYEB_HEALTHCHECK}"
    --checks-grace-period "${KOYEB_HEALTHCHECK_GRACE}"
    --min-scale "${KOYEB_MIN_SCALE}"
    --max-scale "${KOYEB_MAX_SCALE}"
    --wait
    --wait-timeout "${KOYEB_WAIT_TIMEOUT}"
  )

  if [[ -n "${KOYEB_GIT_SHA}" ]]; then
    DEPLOY_ARGS+=(--git-sha "${KOYEB_GIT_SHA}")
  fi

  if [[ -n "${KOYEB_LIGHT_SLEEP_DELAY}" ]]; then
    DEPLOY_ARGS+=(--light-sleep-delay "${KOYEB_LIGHT_SLEEP_DELAY}")
  fi

  if [[ -n "${KOYEB_DEEP_SLEEP_DELAY}" ]]; then
    DEPLOY_ARGS+=(--deep-sleep-delay "${KOYEB_DEEP_SLEEP_DELAY}")
  fi
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
  fi

  command -v koyeb >/dev/null 2>&1 || fail "koyeb CLI not found. See docs/koyeb_runbook.md"

  if [[ -z "${KOYEB_GIT}" ]]; then
    if git -C "${ROOT_DIR}" config --get remote.origin.url >/dev/null 2>&1; then
      KOYEB_GIT="$(normalize_github_remote "$(git -C "${ROOT_DIR}" config --get remote.origin.url)")"
    else
      fail "KOYEB_GIT is required when origin remote is unavailable"
    fi
  fi

  load_env_args
  build_deploy_args

  echo "Deploying backend to Koyeb"
  echo "  app:      ${KOYEB_APP}"
  echo "  service:  ${KOYEB_SERVICE}"
  echo "  git:      ${KOYEB_GIT}"
  echo "  branch:   ${KOYEB_GIT_BRANCH}"
  echo "  dockerfile: ${KOYEB_DOCKERFILE}"

  if ! run_koyeb apps get "${KOYEB_APP}" >/dev/null 2>&1; then
    echo "Creating Koyeb app ${KOYEB_APP}"
    run_koyeb apps create "${KOYEB_APP}"
  fi

  if run_koyeb services get "${KOYEB_SERVICE}" --app "${KOYEB_APP}" >/dev/null 2>&1; then
    echo "Updating existing Koyeb service ${KOYEB_APP}/${KOYEB_SERVICE}"
    run_koyeb services update "${KOYEB_APP}/${KOYEB_SERVICE}" "${DEPLOY_ARGS[@]}" "${ENV_ARGS[@]}"
  else
    echo "Creating new Koyeb service ${KOYEB_SERVICE}"
    run_koyeb services create "${KOYEB_SERVICE}" --app "${KOYEB_APP}" "${DEPLOY_ARGS[@]}" "${ENV_ARGS[@]}"
  fi

  echo
  echo "Deployment complete. Current app summary:"
  run_koyeb apps get "${KOYEB_APP}"
  echo
  echo "Current service summary:"
  run_koyeb services get "${KOYEB_SERVICE}" --app "${KOYEB_APP}"
}

main "$@"
