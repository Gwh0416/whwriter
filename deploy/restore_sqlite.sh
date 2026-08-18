#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: deploy/restore_sqlite.sh /path/to/backup.db" >&2
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

backup_file="$1"
if [[ ! -f "${backup_file}" ]]; then
  echo "Backup file not found: ${backup_file}" >&2
  exit 1
fi

echo "Restoring ${backup_file} into SQLite database ${SQLITE_PATH}"
echo "This will overwrite current data. Press Ctrl+C within 5 seconds to abort."
sleep 5

mkdir -p "$(dirname "${SQLITE_PATH}")"
cp "${backup_file}" "${SQLITE_PATH}"

echo "Restore finished."
