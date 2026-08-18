#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib.sh"

mkdir -p "${BACKUP_DIR}"
backup_file="${BACKUP_DIR}/whwriter_$(date +%F_%H%M%S).db"

if [[ ! -f "${SQLITE_PATH}" ]]; then
  echo "SQLite database not found: ${SQLITE_PATH}" >&2
  exit 1
fi

echo "Backing up SQLite database to ${backup_file}"
cp "${SQLITE_PATH}" "${backup_file}"

echo "Backup finished: ${backup_file}"
