#!/usr/bin/env bash
set -euo pipefail

# First installation only.
# After this, normal updates should be:
#   git pull
#   sudo systemctl restart spark-whatsapp-module

APP_NAME="spark-whatsapp-module"
APP_DIR="/opt/${APP_NAME}"
ENV_FILE="/etc/${APP_NAME}.env"
SERVICE_FILE="/etc/systemd/system/${APP_NAME}.service"

if [[ "${EUID}" -ne 0 ]]; then
  echo "Please run as root: sudo bash scripts/install-vps.sh"
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Installing Go toolchain dependencies..."
  apt-get update
  apt-get install -y golang git build-essential sqlite3 rsync
else
  apt-get update
  apt-get install -y git build-essential sqlite3 rsync
fi

mkdir -p "${APP_DIR}"
rsync -a --delete \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude 'spark-whatsapp-module.db' \
  --exclude 'sessions.db' \
  ./ "${APP_DIR}/"

if [[ ! -f "${ENV_FILE}" ]]; then
  cat > "${ENV_FILE}" <<'EOF'
SQLITE_PATH=/opt/spark-whatsapp-module/spark-whatsapp-module.db
POSTGRES_URI=
HTTP_ADDRESS=:8080
WHATSAPP_SUBSCRIBE_WORD=subscribe
WHATSAPP_UNSUBSCRIBE_WORD=unsubscribe
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_IDLE_TIME=5m
DB_CONN_MAX_LIFETIME=30m
EOF
  chmod 600 "${ENV_FILE}"
fi

cd "${APP_DIR}"
mkdir -p bin
if [[ ! -f "${APP_DIR}/.env" ]]; then
  cp "${ENV_FILE}" "${APP_DIR}/.env"
  chmod 600 "${APP_DIR}/.env"
fi
go run ./cmd/setup
go build -o ./bin/app ./cmd/spark-whatsapp-module

cat > "${SERVICE_FILE}" <<EOF
[Unit]
Description=spark-whatsapp-module
After=network.target

[Service]
Type=simple
WorkingDirectory=${APP_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${APP_DIR}/bin/app
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable "${APP_NAME}"
systemctl restart "${APP_NAME}"

echo
echo "Installed ${APP_NAME}."
echo "Check status with:"
echo "  sudo systemctl status ${APP_NAME}"
echo "Edit config with:"
echo "  sudo nano ${ENV_FILE}"
echo
echo "First install only. Future updates:"
echo "  git pull"
echo "  sudo systemctl restart ${APP_NAME}"
