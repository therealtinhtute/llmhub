#!/bin/sh
# LLMHub one-command VPS installer (Linux only)
# Usage: curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/scripts/install.sh | sudo sh
set -eu

REPO="therealtinhtute/llmhub"
BINARY="llmhub"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/llmhub"
DATA_DIR="/var/lib/llmhub"
LOG_DIR="/var/log/llmhub"
SERVICE_USER="llmhub"

# Require root
if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script with sudo" >&2
    exit 1
fi

# Detect architecture
MACHINE="$(uname -m)"
case "$MACHINE" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "error: unsupported architecture: $MACHINE" >&2
        exit 1
        ;;
esac

# Resolve version (env override or latest)
TAG="${LLMHUB_VERSION:-${VERSION:-}}"
if [ -z "$TAG" ]; then
    LATEST_URL="$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
        "https://github.com/${REPO}/releases/latest")"
    TAG="${LATEST_URL##*/}"
fi

echo "==> Installing llmhub ${TAG} (linux/${ARCH})"

# Download and install binary
ASSET="${BINARY}-linux-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
TMP_BIN="$(mktemp)"
curl -fL --progress-bar "$DOWNLOAD_URL" -o "$TMP_BIN"
install -m 0755 "$TMP_BIN" "${INSTALL_DIR}/${BINARY}"
rm -f "$TMP_BIN"
echo "    binary: ${INSTALL_DIR}/${BINARY}"

# Create system user (idempotent)
if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

# Create runtime directories (idempotent)
mkdir -p "$CONFIG_DIR" "${DATA_DIR}/auths" "$LOG_DIR"
chown -R "${SERVICE_USER}:${SERVICE_USER}" "$DATA_DIR" "$LOG_DIR"
chmod 750 "$DATA_DIR" "${DATA_DIR}/auths"

# Seed config if not already present
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    TMP_CFG="$(mktemp)"
    curl -fsSL \
        "https://raw.githubusercontent.com/${REPO}/master/config.example.yaml" \
        -o "$TMP_CFG"
    sed -i 's|auth-dir:.*|auth-dir: "/var/lib/llmhub/auths"|' "$TMP_CFG"
    install -m 0640 -o root -g "$SERVICE_USER" "$TMP_CFG" "${CONFIG_DIR}/config.yaml"
    rm -f "$TMP_CFG"
    echo "    config: ${CONFIG_DIR}/config.yaml (seeded from example — edit before use)"
else
    echo "    config: ${CONFIG_DIR}/config.yaml (existing — left unchanged)"
fi

# Write systemd unit
cat >/etc/systemd/system/llmhub.service <<UNIT
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${CONFIG_DIR}
ExecStart=${INSTALL_DIR}/${BINARY} -config ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
UNIT

# Enable and start
systemctl daemon-reload
systemctl enable --now llmhub
echo ""
systemctl status llmhub --no-pager || true

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "SERVER_IP")"
echo ""
echo "==> llmhub ${TAG} is running"
echo "    API endpoint:     http://${SERVER_IP}:8317/v1"
echo "    Management panel: http://${SERVER_IP}:8317/management.html"
echo ""
echo "    Edit ${CONFIG_DIR}/config.yaml to configure providers and accounts."
echo "    Logs: journalctl -u llmhub -f"
