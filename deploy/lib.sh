#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${SCRIPT_DIR}/.env"

if [[ -f "${ENV_FILE}" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
  set +a
else
  echo "Missing ${ENV_FILE}. Copy deploy/env.example to deploy/.env and update secrets." >&2
  exit 1
fi

APP_DIR="${APP_DIR:-/opt/whwriter}"
DATA_DIR="${DATA_DIR:-/opt/whwriter-data}"
BACKUP_DIR="${BACKUP_DIR:-/opt/whwriter-backups}"
DOMAIN="${DOMAIN:-_}"
APP_PORT="${APP_PORT:-8080}"
SQLITE_PATH="${SQLITE_PATH:-${DATA_DIR}/whwriter.db}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Missing command: $1" >&2
    exit 1
  fi
}

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required env: ${name}" >&2
    exit 1
  fi
}

render_template() {
  local input="$1"
  local output="$2"
  python3 - "$input" "$output" <<'PY'
import os
import sys
from pathlib import Path

src = Path(sys.argv[1]).read_text()
for key, value in os.environ.items():
    src = src.replace("${" + key + "}", value)
Path(sys.argv[2]).write_text(src)
PY
}

write_backend_config() {
  mkdir -p "$(dirname "${SQLITE_PATH}")"
  cat > "${PROJECT_DIR}/backend/config.yaml" <<EOF
app:
  mode: prod
  host: "127.0.0.1"
  port: ${APP_PORT}

sqlite:
  path: "${SQLITE_PATH}"

llm:
  default_timeout_seconds: ${LLM_DEFAULT_TIMEOUT_SECONDS:-300}
  planner_timeout_seconds: ${LLM_PLANNER_TIMEOUT_SECONDS:-300}
  writer_timeout_seconds: ${LLM_WRITER_TIMEOUT_SECONDS:-600}
  settler_timeout_seconds: ${LLM_SETTLER_TIMEOUT_SECONDS:-300}
  auditor_timeout_seconds: ${LLM_AUDITOR_TIMEOUT_SECONDS:-300}
  reviser_timeout_seconds: ${LLM_REVISER_TIMEOUT_SECONDS:-300}
EOF
}

build_backend() {
  echo "Building backend..."
  (cd "${PROJECT_DIR}/backend" && go mod download && go build -o whwriter ./cmd/whwriter)
}

build_frontend() {
  echo "Building frontend..."
  (cd "${PROJECT_DIR}/frontend" && npm ci && npm run build)
}
