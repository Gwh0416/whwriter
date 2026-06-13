# whwriter Server Deploy

This deploy setup is for:

- Backend: Go binary managed by systemd
- Frontend: Vite static files served by Nginx
- MySQL: Docker container on the server

## 1. Server Dependencies

Install these on the ECS server:

```bash
sudo apt update
sudo apt install -y nginx docker.io docker-compose-plugin rsync curl git
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

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `JWT_SECRET`
- `ADMIN_PASSWORD`
- `DOMAIN`

## 3. First Install

```bash
cd /opt/whwriter
bash deploy/install.sh
```

The script will:

- Start Docker MySQL with `deploy/docker-compose.mysql.yml`
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
- Backup Docker MySQL
- Rebuild backend
- Rebuild frontend
- Restart backend
- Reload Nginx
- Check `/health`

## 5. Backup MySQL

```bash
cd /opt/whwriter
bash deploy/backup_mysql.sh
```

Backups are written to `BACKUP_DIR`, default:

```text
/opt/whwriter-backups
```

## 6. Restore MySQL

```bash
cd /opt/whwriter
bash deploy/restore_mysql.sh /opt/whwriter-backups/whwriter_YYYY-MM-DD_HHMMSS.sql.gz
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

MySQL logs:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.mysql.yml logs -f mysql
```

Stop MySQL:

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.mysql.yml down
```

## 8. Security Notes

- MySQL is bound to `127.0.0.1:${MYSQL_PORT}` only, not public network.
- ECS security group should open only `22`, `80`, and `443`.
- Do not commit `deploy/.env`.
- Change `backend/config.yaml` secrets on the server only.
