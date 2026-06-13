#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

require_cmd docker
require_cmd go
require_cmd npm
require_cmd python3
require_env MYSQL_ROOT_PASSWORD
require_env MYSQL_PASSWORD
require_env JWT_SECRET

echo "==> Prepare directories"
sudo mkdir -p "${APP_DIR}" "${DATA_DIR}/mysql" "${BACKUP_DIR}"

if [[ "${PROJECT_DIR}" != "${APP_DIR}" ]]; then
  echo "==> Sync project to ${APP_DIR}"
  sudo rsync -a --delete \
    --exclude '.git' \
    --exclude 'backend/whwriter' \
    --exclude 'frontend/dist' \
    --exclude 'mysql_data' \
    "${PROJECT_DIR}/" "${APP_DIR}/"
  sudo chown -R "${USER}:${USER}" "${APP_DIR}"
  echo "==> Continue install from ${APP_DIR}"
  exec "${APP_DIR}/deploy/install.sh"
fi

cd "${PROJECT_DIR}"

echo "==> Write backend config"
write_backend_config

echo "==> Start Docker MySQL"
compose_mysql up -d
wait_mysql

build_backend
build_frontend

echo "==> Install systemd service"
render_template "${SCRIPT_DIR}/whwriter.service.template" /tmp/whwriter.service
sudo mv /tmp/whwriter.service /etc/systemd/system/whwriter.service
sudo systemctl daemon-reload
sudo systemctl enable whwriter
sudo systemctl restart whwriter

echo "==> Install Nginx site"
if command -v nginx >/dev/null 2>&1; then
  render_template "${SCRIPT_DIR}/nginx.whwriter.conf.template" /tmp/whwriter.nginx.conf
  if [[ -d /etc/nginx/sites-available ]]; then
    sudo mv /tmp/whwriter.nginx.conf /etc/nginx/sites-available/whwriter
    sudo ln -sf /etc/nginx/sites-available/whwriter /etc/nginx/sites-enabled/whwriter
  else
    sudo mv /tmp/whwriter.nginx.conf /etc/nginx/conf.d/whwriter.conf
  fi
  sudo nginx -t
  sudo systemctl reload nginx
else
  echo "Nginx is not installed. Install it and run this script again, or configure reverse proxy manually."
fi

echo "==> Health check"
sleep 2
curl -f "http://127.0.0.1:${APP_PORT}/health"
echo
echo "Install finished. Visit http://${DOMAIN}/"
