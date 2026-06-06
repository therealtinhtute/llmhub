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
SERVICE_GROUP="llmhub"
SERVICE_NAME="llmhub"
ENV_FILE="${CONFIG_DIR}/llmhub.env"
DEFAULT_HOST="${DEFAULT_HOST:-0.0.0.0}"
DEFAULT_PORT="${DEFAULT_PORT:-8317}"

read_env_value() {
    env_file="$1"
    key="$2"
    [ -f "$env_file" ] || return 1
    awk -v key="$key" '
        /^[[:space:]]*#/ { next }
        index($0, key "=") == 1 {
            print substr($0, length(key) + 2)
            exit 0
        }
    ' "$env_file"
}

prompt_postgres_runtime() {
    env_file="$1"
    if [ ! -t 0 ]; then
        return 0
    fi

    current_dsn="$(read_env_value "$env_file" PGSTORE_DSN 2>/dev/null || true)"
    current_schema="$(read_env_value "$env_file" PGSTORE_SCHEMA 2>/dev/null || true)"
    current_retention="$(read_env_value "$env_file" PGSTORE_USAGE_RETENTION_SECONDS 2>/dev/null || true)"

    while :; do
        if [ -n "$current_dsn" ]; then
            printf "PGSTORE_DSN [%s]: " "$current_dsn"
        else
            printf "PGSTORE_DSN: "
        fi
        read -r prompted_dsn
        [ -n "$prompted_dsn" ] || prompted_dsn="$current_dsn"
        if [ -n "$prompted_dsn" ]; then
            PROMPTED_PGSTORE_DSN="$prompted_dsn"
            break
        fi
        echo "error: PGSTORE_DSN cannot be empty" >&2
    done

    schema_default="${current_schema:-llmhub}"
    printf "PGSTORE_SCHEMA [%s]: " "$schema_default"
    read -r prompted_schema
    [ -n "$prompted_schema" ] || prompted_schema="$schema_default"
    PROMPTED_PGSTORE_SCHEMA="$prompted_schema"

    retention_default="${current_retention:-60}"
    while :; do
        printf "PGSTORE_USAGE_RETENTION_SECONDS [%s]: " "$retention_default"
        read -r prompted_retention
        [ -n "$prompted_retention" ] || prompted_retention="$retention_default"
        if echo "$prompted_retention" | grep -Eq '^[0-9]+$'; then
            PROMPTED_PGSTORE_USAGE_RETENTION_SECONDS="$prompted_retention"
            break
        fi
        echo "error: PGSTORE_USAGE_RETENTION_SECONDS must be a non-negative integer" >&2
    done
}

prompt_init_config() {
    env_file="$1"
    current_b64="$(read_env_value "$env_file" LLMHUB_INIT_CONFIG_B64 2>/dev/null || true)"
    current_yaml="$(read_env_value "$env_file" LLMHUB_INIT_CONFIG_YAML 2>/dev/null || true)"
    if [ -n "$current_b64" ] || [ -n "$current_yaml" ]; then
        return 0
    fi
    if [ ! -t 0 ]; then
        echo "error: missing LLMHUB_INIT_CONFIG_B64/LLMHUB_INIT_CONFIG_YAML in $env_file for first boot" >&2
        exit 1
    fi

    echo ""
    echo "Paste initial LLMHub config YAML for database bootstrap."
    echo "Finish with a line containing only END."
    tmp_yaml="$(mktemp)"
    while IFS= read -r line; do
        [ "$line" = "END" ] && break
        printf '%s\n' "$line" >>"$tmp_yaml"
    done
    if [ ! -s "$tmp_yaml" ]; then
        rm -f "$tmp_yaml"
        echo "error: initial config YAML cannot be empty" >&2
        exit 1
    fi
    PROMPTED_INIT_CONFIG_B64="$(base64 <"$tmp_yaml" | tr -d '\n')"
    rm -f "$tmp_yaml"
}

ensure_env_runtime() {
    env_file="$1"
    pg_dsn="${PGSTORE_DSN:-${PROMPTED_PGSTORE_DSN:-}}"
    pg_schema="${PGSTORE_SCHEMA:-${PROMPTED_PGSTORE_SCHEMA:-llmhub}}"
    pg_retention="${PGSTORE_USAGE_RETENTION_SECONDS:-${PROMPTED_PGSTORE_USAGE_RETENTION_SECONDS:-60}}"
    init_b64="${LLMHUB_INIT_CONFIG_B64:-${PROMPTED_INIT_CONFIG_B64:-}}"
    init_yaml="${LLMHUB_INIT_CONFIG_YAML:-}"

    if [ -z "$pg_dsn" ]; then
        echo "error: PGSTORE_DSN is required" >&2
        exit 1
    fi
    if [ -z "$init_b64" ] && [ -z "$init_yaml" ]; then
        echo "error: missing LLMHUB_INIT_CONFIG_B64/LLMHUB_INIT_CONFIG_YAML for first boot" >&2
        exit 1
    fi

    tmp_env="$(mktemp)"
    awk \
        -v host="$DEFAULT_HOST" \
        -v port="$DEFAULT_PORT" \
        -v dsn="$pg_dsn" \
        -v schema="$pg_schema" \
        -v retention="$pg_retention" \
        -v init_b64="$init_b64" \
        -v init_yaml="$init_yaml" '
        BEGIN {
            wrote_host = 0
            wrote_port = 0
            wrote_dsn = 0
            wrote_schema = 0
            wrote_retention = 0
            wrote_b64 = 0
            wrote_yaml = 0
        }
        /^[[:space:]]*LLMHUB_HOST=/ && !wrote_host { print "LLMHUB_HOST=" host; wrote_host = 1; next }
        /^[[:space:]]*LLMHUB_PORT=/ && !wrote_port { print "LLMHUB_PORT=" port; wrote_port = 1; next }
        /^[[:space:]]*PGSTORE_DSN=/ && !wrote_dsn { print "PGSTORE_DSN=" dsn; wrote_dsn = 1; next }
        /^[[:space:]]*PGSTORE_SCHEMA=/ && !wrote_schema { print "PGSTORE_SCHEMA=" schema; wrote_schema = 1; next }
        /^[[:space:]]*PGSTORE_USAGE_RETENTION_SECONDS=/ && !wrote_retention { print "PGSTORE_USAGE_RETENTION_SECONDS=" retention; wrote_retention = 1; next }
        /^[[:space:]]*LLMHUB_INIT_CONFIG_B64=/ && !wrote_b64 {
            if (init_b64 != "") { print "LLMHUB_INIT_CONFIG_B64=" init_b64 }
            wrote_b64 = 1
            next
        }
        /^[[:space:]]*LLMHUB_INIT_CONFIG_YAML=/ && !wrote_yaml {
            if (init_yaml != "") { print "LLMHUB_INIT_CONFIG_YAML=" init_yaml }
            wrote_yaml = 1
            next
        }
        { print }
        END {
            if (!wrote_host) print "LLMHUB_HOST=" host
            if (!wrote_port) print "LLMHUB_PORT=" port
            if (!wrote_dsn) print "PGSTORE_DSN=" dsn
            if (!wrote_schema) print "PGSTORE_SCHEMA=" schema
            if (!wrote_retention) print "PGSTORE_USAGE_RETENTION_SECONDS=" retention
            if (!wrote_b64 && init_b64 != "") print "LLMHUB_INIT_CONFIG_B64=" init_b64
            if (!wrote_yaml && init_yaml != "") print "LLMHUB_INIT_CONFIG_YAML=" init_yaml
        }
    ' "$env_file" >"$tmp_env"
    install -m 0640 -o root -g "$SERVICE_GROUP" "$tmp_env" "$env_file"
    rm -f "$tmp_env"
}

wait_for_service_port() {
    attempts=0
    while [ "$attempts" -lt 30 ]; do
        state="$(systemctl is-active "${SERVICE_NAME}.service" 2>/dev/null || true)"
        if ss -ltnp 2>/dev/null | grep -q ":${DEFAULT_PORT} "; then
            printf '%s\n' "$DEFAULT_PORT"
            return 0
        fi
        case "$state" in
            active|activating|reloading) ;;
            *)
                echo "error: ${SERVICE_NAME}.service is not active after restart (state: ${state:-unknown})" >&2
                systemctl status "${SERVICE_NAME}.service" --no-pager >&2 || true
                journalctl -u "${SERVICE_NAME}.service" -n 80 --no-pager >&2 || true
                return 1
                ;;
        esac
        sleep 1
        attempts=$((attempts + 1))
    done

    echo "error: ${SERVICE_NAME}.service is active but no listening TCP port was detected" >&2
    systemctl status "${SERVICE_NAME}.service" --no-pager >&2 || true
    journalctl -u "${SERVICE_NAME}.service" -n 80 --no-pager >&2 || true
    return 1
}

allow_firewall_port() {
    port="$1"
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -qi '^Status:[[:space:]]*active'; then
        ufw allow "${port}/tcp" >/dev/null
        echo "    firewall: allowed ${port}/tcp via ufw"
    fi
}

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script with sudo" >&2
    exit 1
fi

MACHINE="$(uname -m)"
case "$MACHINE" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *)
        echo "error: unsupported architecture: $MACHINE" >&2
        exit 1
        ;;
esac

TAG="${LLMHUB_VERSION:-${VERSION:-}}"
if [ -z "$TAG" ]; then
    LATEST_URL="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
    TAG="${LATEST_URL##*/}"
fi

echo "==> Installing llmhub ${TAG} (linux/${ARCH})"

ASSET="${BINARY}-linux-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
TMP_BIN="$(mktemp)"
curl -fL --progress-bar "$DOWNLOAD_URL" -o "$TMP_BIN"
install -d -m 0755 "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
install -m 0755 "$TMP_BIN" "${INSTALL_DIR}/${BINARY}"
rm -f "$TMP_BIN"
echo "    binary: ${INSTALL_DIR}/${BINARY}"

if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
    groupadd --system "$SERVICE_GROUP"
fi
if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --gid "$SERVICE_GROUP" --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "$DATA_DIR" "$LOG_DIR"
chmod 750 "$DATA_DIR"

if [ ! -f "$ENV_FILE" ]; then
    cat >"$ENV_FILE" <<ENV
# Bootstrap env for llmhub.
LLMHUB_HOST=${DEFAULT_HOST}
LLMHUB_PORT=${DEFAULT_PORT}
PGSTORE_DSN=
PGSTORE_SCHEMA=llmhub
PGSTORE_USAGE_RETENTION_SECONDS=60
# First boot only. Keep one of these set until init succeeds.
# LLMHUB_INIT_CONFIG_B64=
# LLMHUB_INIT_CONFIG_YAML=
ENV
    chmod 0640 "$ENV_FILE"
    chown root:"$SERVICE_GROUP" "$ENV_FILE"
fi

prompt_postgres_runtime "$ENV_FILE"
prompt_init_config "$ENV_FILE"
ensure_env_runtime "$ENV_FILE"
echo "    environment: $ENV_FILE"

cat >/etc/systemd/system/llmhub.service <<UNIT
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${DATA_DIR}
Environment=HOME=${DATA_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStartPre=${INSTALL_DIR}/${BINARY} init-db-from-env -env-file ${ENV_FILE}
ExecStart=${INSTALL_DIR}/${BINARY}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable --now llmhub
echo ""
SERVER_PORT="$(wait_for_service_port)"
systemctl status llmhub --no-pager || true
allow_firewall_port "$SERVER_PORT"

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "SERVER_IP")"
echo ""
echo "==> llmhub ${TAG} is running"
echo "    API endpoint:     http://${SERVER_IP}:${SERVER_PORT}/v1"
echo "    Management panel: http://${SERVER_IP}:${SERVER_PORT}/management.html"
echo ""
echo "    Bootstrap env: ${ENV_FILE}"
echo "    Logs: journalctl -u llmhub -f"
