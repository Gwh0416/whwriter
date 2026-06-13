#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

require_env MYSQL_ROOT_PASSWORD

mkdir -p "${BACKUP_DIR}"
backup_file="${BACKUP_DIR}/whwriter_$(date +%F_%H%M%S).sql.gz"

echo "Backing up MySQL to ${backup_file}"
compose_mysql exec -T mysql mysqldump \
  -uroot \
  -p"${MYSQL_ROOT_PASSWORD}" \
  --single-transaction \
  --routines \
  --triggers \
  "${MYSQL_DATABASE}" | gzip > "${backup_file}"

echo "Backup finished: ${backup_file}"
