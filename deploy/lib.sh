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
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_DATABASE="${MYSQL_DATABASE:-whwriter}"
MYSQL_USER="${MYSQL_USER:-whwriter}"

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

compose_mysql() {
  docker compose --env-file "${ENV_FILE}" -f "${SCRIPT_DIR}/docker-compose.mysql.yml" "$@"
}

wait_mysql() {
  echo "Waiting for MySQL container to become healthy..."
  for _ in $(seq 1 60); do
    if compose_mysql exec -T mysql mysqladmin ping -h 127.0.0.1 -uroot -p"${MYSQL_ROOT_PASSWORD}" --silent >/dev/null 2>&1; then
      echo "MySQL is ready."
      return 0
    fi
    sleep 2
  done
  echo "MySQL did not become ready in time." >&2
  compose_mysql logs mysql >&2 || true
  exit 1
}

write_backend_config() {
  require_env MYSQL_PASSWORD
  require_env JWT_SECRET
  cat > "${PROJECT_DIR}/backend/config.yaml" <<EOF
app:
  mode: prod
  host: "127.0.0.1"
  port: ${APP_PORT}

mysql:
  host: "127.0.0.1"
  port: ${MYSQL_PORT}
  user: "${MYSQL_USER}"
  password: "${MYSQL_PASSWORD}"
  database: "${MYSQL_DATABASE}"
  charset: utf8mb4

jwt:
  secret: "${JWT_SECRET}"

admin:
  email: "${ADMIN_EMAIL:-admin@example.com}"
  username: "${ADMIN_USERNAME:-admin}"
  password: "${ADMIN_PASSWORD:-ChangeAdminPassword123}"

smtp:
  host: "${SMTP_HOST:-}"
  port: ${SMTP_PORT:-587}
  user: "${SMTP_USER:-}"
  password: "${SMTP_PASSWORD:-}"
  from: "${SMTP_FROM:-}"

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
