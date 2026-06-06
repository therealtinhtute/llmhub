#!/bin/sh
# LLMHub local binary installer with full VPS setup
# Usage:
#   ./scripts/install-local.sh /path/to/llmhub-linux-amd64
#   ./scripts/install-local.sh
set -eu

BINARY="llmhub"
DEFAULT_HOST="${DEFAULT_HOST:-0.0.0.0}"
DEFAULT_PORT="${DEFAULT_PORT:-9090}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/llmhub}"
DATA_DIR="${DATA_DIR:-/var/lib/llmhub}"
LOG_DIR="${LOG_DIR:-/var/log/llmhub}"
SERVICE_USER="${SERVICE_USER:-llmhub}"
SERVICE_GROUP="${SERVICE_GROUP:-$SERVICE_USER}"
SERVICE_NAME="${SERVICE_NAME:-llmhub}"
ENV_FILE="${ENV_FILE:-${CONFIG_DIR}/llmhub.env}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CADDY_DOMAIN="${CADDY_DOMAIN:-}"
CADDYFILE_PATH="${CADDYFILE_PATH:-/etc/caddy/Caddyfile}"
SERVICE_RESTART="${SERVICE_RESTART:-always}"
SERVICE_RESTART_SEC="${SERVICE_RESTART_SEC:-3s}"
SERVICE_START_LIMIT_INTERVAL="${SERVICE_START_LIMIT_INTERVAL:-0}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)

find_local_env_file() {
    selected_dir=$(dirname -- "$SELECTED")
    for env in ".env" "${SCRIPT_DIR}/.env" "${selected_dir}/.env"; do
        if [ -f "$env" ]; then
            printf '%s\n' "$env"
            return 0
        fi
    done
    return 1
}

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

find_source_binary() {
    if [ -f "${SCRIPT_DIR}/${BINARY}" ]; then
        SELECTED="${SCRIPT_DIR}/${BINARY}"
        echo "==> Found script-local binary: $SELECTED"
        return 0
    fi
    if [ -f "./${BINARY}" ]; then
        SELECTED="./${BINARY}"
        echo "==> Found local binary: $SELECTED"
        return 0
    fi
    if [ ! -f "go.mod" ] || [ ! -d "dist" ]; then
        echo "error: no llmhub binary was found next to install-local.sh or in the current directory" >&2
        echo "Place llmhub and install-local.sh in the same directory, or pass a binary path." >&2
        exit 1
    fi
    BINARIES=$(find "dist" -type f -name "${BINARY}-linux-*" ! -name "*.txt" ! -name "*.json" 2>/dev/null | sort || true)
    if [ -z "$BINARIES" ]; then
        echo "error: no Linux binaries found in dist/" >&2
        exit 1
    fi
    BINARY_COUNT=$(echo "$BINARIES" | wc -l)
    if [ "$BINARY_COUNT" -eq 1 ]; then
        SELECTED="$BINARIES"
        echo "==> Found 1 binary: $(basename "$SELECTED")"
        return 0
    fi
    echo "==> Found $BINARY_COUNT Linux binaries in dist/:"
    i=1
    echo "$BINARIES" | while IFS= read -r binary; do
        size=$(du -h "$binary" | cut -f1)
        printf "%2d) %-40s  %8s\n" "$i" "$(basename "$binary")" "$size"
        i=$((i + 1))
    done
    printf "Select binary to install [1-%d]: " "$BINARY_COUNT"
    read -r choice
    SELECTED=$(echo "$BINARIES" | sed -n "${choice}p")
}

validate_binary() {
    bin_path="$1"
    [ -f "$bin_path" ] || { echo "error: file not found: $bin_path" >&2; return 1; }
    [ -r "$bin_path" ] || { echo "error: file not readable: $bin_path" >&2; return 1; }
    if command -v file >/dev/null 2>&1; then
        file "$bin_path" | grep -qE 'ELF.*(executable|shared object)' || {
            echo "error: not a valid Linux executable: $bin_path" >&2
            return 1
        }
    fi
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
    echo "Press Enter on an empty first line to skip when Postgres is already initialized."
    echo "Finish with a line containing only END."
    tmp_yaml="$(mktemp)"
    while IFS= read -r line; do
        if [ -z "$line" ] && [ ! -s "$tmp_yaml" ]; then
            rm -f "$tmp_yaml"
            return 0
        fi
        [ "$line" = "END" ] && break
        printf '%s\n' "$line" >>"$tmp_yaml"
    done
    if [ ! -s "$tmp_yaml" ]; then
        rm -f "$tmp_yaml"
        return 0
    fi
    PROMPTED_INIT_CONFIG_B64="$(base64 <"$tmp_yaml" | tr -d '\n')"
    rm -f "$tmp_yaml"
}

ensure_env_runtime() {
    env_file="$1"
    current_dsn="$(read_env_value "$env_file" PGSTORE_DSN 2>/dev/null || true)"
    current_schema="$(read_env_value "$env_file" PGSTORE_SCHEMA 2>/dev/null || true)"
    current_retention="$(read_env_value "$env_file" PGSTORE_USAGE_RETENTION_SECONDS 2>/dev/null || true)"
    current_b64="$(read_env_value "$env_file" LLMHUB_INIT_CONFIG_B64 2>/dev/null || true)"
    current_yaml="$(read_env_value "$env_file" LLMHUB_INIT_CONFIG_YAML 2>/dev/null || true)"
    pg_dsn="${PGSTORE_DSN:-${PROMPTED_PGSTORE_DSN:-$current_dsn}}"
    pg_schema="${PGSTORE_SCHEMA:-${PROMPTED_PGSTORE_SCHEMA:-${current_schema:-llmhub}}}"
    pg_retention="${PGSTORE_USAGE_RETENTION_SECONDS:-${PROMPTED_PGSTORE_USAGE_RETENTION_SECONDS:-${current_retention:-60}}}"
    init_b64="${LLMHUB_INIT_CONFIG_B64:-${PROMPTED_INIT_CONFIG_B64:-$current_b64}}"
    init_yaml="${LLMHUB_INIT_CONFIG_YAML:-$current_yaml}"
    if [ -z "$pg_dsn" ]; then
        echo "error: PGSTORE_DSN is required" >&2
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

valid_domain() {
    domain="$1"
    [ -z "$domain" ] && return 0
    printf '%s\n' "$domain" | grep -Eq '^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$'
}

prompt_domain() {
    if [ -n "$CADDY_DOMAIN" ]; then
        valid_domain "$CADDY_DOMAIN" || { echo "error: invalid CADDY_DOMAIN: $CADDY_DOMAIN" >&2; exit 1; }
        return 0
    fi
    if [ -t 0 ]; then
        printf "Domain for Caddy HTTPS (leave blank to skip): "
        read -r CADDY_DOMAIN
        [ -z "$CADDY_DOMAIN" ] || valid_domain "$CADDY_DOMAIN" || { echo "error: invalid domain: $CADDY_DOMAIN" >&2; exit 1; }
    fi
}

allow_firewall_port() {
    port="$1"
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -qi '^Status:[[:space:]]*active'; then
        ufw allow "${port}/tcp" >/dev/null
        echo "    firewall: allowed ${port}/tcp via ufw"
    fi
}

install_caddy_if_needed() {
    if command -v caddy >/dev/null 2>&1; then
        echo "    caddy: $(caddy version | head -n 1)"
        return 0
    fi
    if ! command -v apt-get >/dev/null 2>&1 || [ ! -f /etc/debian_version ]; then
        echo "error: automatic Caddy setup currently supports Debian/Ubuntu only" >&2
        exit 1
    fi
    apt-get update -qq
    apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https curl gnupg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' -o /etc/apt/sources.list.d/caddy-stable.list
    apt-get update -qq
    apt-get install -y -qq caddy
}

configure_caddy() {
    domain="$1"
    upstream_port="$2"
    install_caddy_if_needed
    install -d -m 0755 "$(dirname -- "$CADDYFILE_PATH")"
    cat >"$CADDYFILE_PATH" <<EOF
$domain {
    encode gzip
    reverse_proxy 127.0.0.1:$upstream_port
}
EOF
    caddy validate --config "$CADDYFILE_PATH" >/dev/null 2>&1 || { echo "error: generated Caddyfile is invalid" >&2; exit 1; }
    systemctl daemon-reload
    systemctl enable caddy >/dev/null
    systemctl restart caddy
    allow_firewall_port 80
    allow_firewall_port 443
}

service_port() {
    if ! command -v systemctl >/dev/null 2>&1 || ! command -v ss >/dev/null 2>&1; then
        return 1
    fi
    pid=$(systemctl show -p MainPID --value "${SERVICE_NAME}.service" 2>/dev/null || true)
    case "$pid" in
        ''|0|*[!0-9]*) return 1 ;;
    esac
    ss -ltnp 2>/dev/null | awk -v pid="$pid" '
        index($0, "pid=" pid ",") > 0 {
            split($4, parts, ":")
            port=parts[length(parts)]
            if (port ~ /^[0-9]+$/) { print port; exit 0 }
        }
    '
}

wait_for_service_port() {
    attempts=0
    while [ "$attempts" -lt 30 ]; do
        state="$(systemctl is-active "${SERVICE_NAME}.service" 2>/dev/null || true)"
        port="$(service_port | head -n 1)"
        if [ -n "$port" ]; then
            printf '%s\n' "$port"
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

if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script with sudo" >&2
    exit 1
fi

if [ $# -eq 1 ]; then
    SELECTED="$1"
    echo "==> Installing from: $SELECTED"
elif [ $# -gt 1 ]; then
    echo "error: usage: $0 [path/to/llmhub-linux-amd64]" >&2
    exit 1
else
    find_source_binary
fi
validate_binary "$SELECTED" || exit 1
prompt_domain

echo "==> Installing llmhub to $INSTALL_DIR"
install -d -m 0755 "$INSTALL_DIR" "$CONFIG_DIR" "$DATA_DIR" "$LOG_DIR"
install -m 0755 "$SELECTED" "${INSTALL_DIR}/${BINARY}"
echo "    binary: ${INSTALL_DIR}/${BINARY}"

if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
    groupadd --system "$SERVICE_GROUP"
fi
if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --gid "$SERVICE_GROUP" --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi
chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "$DATA_DIR" "$LOG_DIR"
chmod 750 "$DATA_DIR"
echo "    directories: $CONFIG_DIR, $DATA_DIR, $LOG_DIR"

if [ ! -f "$ENV_FILE" ]; then
    if tmp_env=$(find_local_env_file); then
        install -m 0640 -o root -g "$SERVICE_GROUP" "$tmp_env" "$ENV_FILE"
        echo "    environment: $ENV_FILE (seeded from $tmp_env)"
    else
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
        echo "    environment: $ENV_FILE (created)"
    fi
else
    echo "    environment: $ENV_FILE (existing - left unchanged)"
fi

prompt_postgres_runtime "$ENV_FILE"
prompt_init_config "$ENV_FILE"
ensure_env_runtime "$ENV_FILE"
echo "    environment: $ENV_FILE"

cat >"$UNIT_FILE" <<UNIT
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target
StartLimitIntervalSec=${SERVICE_START_LIMIT_INTERVAL}

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${DATA_DIR}
Environment=HOME=${DATA_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStartPre=${INSTALL_DIR}/${BINARY} init-db-from-env -env-file ${ENV_FILE}
ExecStart=${INSTALL_DIR}/${BINARY}
Restart=${SERVICE_RESTART}
RestartSec=${SERVICE_RESTART_SEC}
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=${DATA_DIR} ${LOG_DIR} ${CONFIG_DIR}

[Install]
WantedBy=multi-user.target
UNIT

echo "    systemd: $UNIT_FILE"
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"
echo ""
SERVER_PORT="$(wait_for_service_port)"
systemctl status "${SERVICE_NAME}.service" --no-pager || true

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "SERVER_IP")"
if [ -n "$CADDY_DOMAIN" ]; then
    configure_caddy "$CADDY_DOMAIN" "$SERVER_PORT"
else
    allow_firewall_port "$SERVER_PORT"
fi
echo ""
echo "==> llmhub is running"
if [ -n "$CADDY_DOMAIN" ]; then
    echo "    API endpoint:     https://${CADDY_DOMAIN}/v1"
    echo "    Management panel: https://${CADDY_DOMAIN}/management.html"
    echo "    Direct app port:  http://${SERVER_IP}:${SERVER_PORT}"
    echo "    DNS note:         point ${CADDY_DOMAIN} to ${SERVER_IP}; HTTPS becomes live after DNS propagation"
else
    echo "    API endpoint:     http://${SERVER_IP}:${SERVER_PORT}/v1"
    echo "    Management panel: http://${SERVER_IP}:${SERVER_PORT}/management.html"
fi
echo ""
echo "    Bootstrap env: $ENV_FILE"
echo "    Logs: journalctl -u ${SERVICE_NAME} -f"
if [ -n "$CADDY_DOMAIN" ]; then
    echo "    Caddy logs: journalctl -u caddy -f"
fi
