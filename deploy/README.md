# whwriter Server Deploy

This deploy setup is for:

- Backend: Go binary managed by systemd
- Frontend: Vite static files served by Nginx
- Database: local SQLite file

## 1. Server Dependencies

Install these on the ECS server:

```bash
sudo apt update
sudo apt install -y nginx rsync curl git
```

Install Go and Node.js separately. Recommended:

- Go 1.23+
- Node.js 20+

## 2. Prepare Config

Put the project under `/opt/whwriter`:

```bash
sudo mkdir -p /opt/whwriter
sudo chown -R $USER:$USER /opt/whwriter
cd /opt/whwriter
git clone <your-repo-url> .
```

Create production env:

```bash
cp deploy/env.example deploy/.env
vim deploy/.env
```

Must change at least:

- `DOMAIN`

## 3. First Install

```bash
cd /opt/whwriter
bash deploy/install.sh
```

The script will:

- Generate `backend/config.yaml`
- Build backend binary
- Build frontend static files
- Install `whwriter.service`
- Install Nginx site
- Check `/health`

## 4. Update Service

```bash
cd /opt/whwriter
bash deploy/deploy.sh
```

The script will:

- `git pull`
- Backup SQLite database
- Rebuild backend
- Rebuild frontend
- Restart backend
- Reload Nginx
- Check `/health`

## 5. Backup SQLite

```bash
cd /opt/whwriter
bash deploy/backup_sqlite.sh
```

Backups are written to `BACKUP_DIR`, default:

```text
/opt/whwriter-backups
```

## 6. Restore SQLite

```bash
cd /opt/whwriter
bash deploy/restore_sqlite.sh /opt/whwriter-backups/whwriter_YYYY-MM-DD_HHMMSS.db
```

## 7. Useful Commands

Backend logs:

```bash
sudo journalctl -u whwriter -f
```

Backend status:

```bash
sudo systemctl status whwriter --no-pager
```

## 8. Security Notes

- ECS security group should open only `22`, `80`, and `443`.
- Do not commit `deploy/.env`.
- Keep production-only paths and timeout settings in `deploy/.env`.
- Keep `SQLITE_PATH` under `DATA_DIR` and back it up before deploys.
