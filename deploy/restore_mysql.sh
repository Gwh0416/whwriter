#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: deploy/restore_mysql.sh /path/to/backup.sql.gz" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

require_env MYSQL_ROOT_PASSWORD

backup_file="$1"
if [[ ! -f "${backup_file}" ]]; then
  echo "Backup file not found: ${backup_file}" >&2
  exit 1
fi

echo "Restoring ${backup_file} into database ${MYSQL_DATABASE}"
echo "This will overwrite current data. Press Ctrl+C within 5 seconds to abort."
sleep 5

if [[ "${backup_file}" == *.gz ]]; then
  gzip -dc "${backup_file}" | compose_mysql exec -T mysql mysql -uroot -p"${MYSQL_ROOT_PASSWORD}" "${MYSQL_DATABASE}"
else
  compose_mysql exec -T mysql mysql -uroot -p"${MYSQL_ROOT_PASSWORD}" "${MYSQL_DATABASE}" < "${backup_file}"
fi

echo "Restore finished."
