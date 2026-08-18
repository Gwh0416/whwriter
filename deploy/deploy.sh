#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

require_cmd go
require_cmd npm
require_cmd python3

cd "${PROJECT_DIR}"

echo "==> Pull latest code if this is a git checkout"
if [[ -d .git ]]; then
  git pull
fi

echo "==> Backup SQLite before deploy"
"${SCRIPT_DIR}/backup_sqlite.sh"

echo "==> Rewrite backend config from deploy/.env"
write_backend_config

build_backend
build_frontend

echo "==> Restart backend"
sudo systemctl restart whwriter

echo "==> Reload Nginx"
if command -v nginx >/dev/null 2>&1; then
  sudo nginx -t
  sudo systemctl reload nginx
fi

echo "==> Health check"
sleep 2
curl -f "http://127.0.0.1:${APP_PORT}/health"
echo
echo "Deploy finished."
