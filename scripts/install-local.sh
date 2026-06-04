#!/bin/sh
# LLMHub local binary installer with full VPS setup
# Usage:
#   ./scripts/install-local.sh /path/to/llmhub-linux-amd64  # install specific binary
#   ./scripts/install-local.sh                              # install script-local ./llmhub, ./llmhub, or select from dist/
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
ENV_FILE="${ENV_FILE:-${INSTALL_DIR}/.env}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CADDY_DOMAIN="${CADDY_DOMAIN:-}"
CADDYFILE_PATH="${CADDYFILE_PATH:-/etc/caddy/Caddyfile}"
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)

find_local_config_example() {
    selected_dir=$(dirname -- "$SELECTED")

    for cfg in \
        "config.example.yaml" \
        "${SCRIPT_DIR}/config.example.yaml" \
        "${selected_dir}/config.example.yaml"; do
        if [ -f "$cfg" ]; then
            printf '%s\n' "$cfg"
            return 0
        fi
    done

    return 1
}

find_local_env_file() {
    selected_dir=$(dirname -- "$SELECTED")

    for env in \
        ".env" \
        "${SCRIPT_DIR}/.env" \
        "${selected_dir}/.env"; do
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

    if [ ! -f "$env_file" ]; then
        return 1
    fi

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

    # Interactive selection from repo-root dist/.
    DIST_DIR="dist"

    if [ ! -f "go.mod" ]; then
        echo "error: no llmhub binary was found next to install-local.sh or in the current directory" >&2
        echo "Place llmhub, install-local.sh, and optionally config.example.yaml in the same directory." >&2
        echo "Or run this script from the repository root with dist/ artifacts, or pass a binary path." >&2
        exit 1
    fi

    if [ ! -d "$DIST_DIR" ]; then
        echo "error: $DIST_DIR/ not found" >&2
        echo "Run one of:" >&2
        echo "  make build             # creates ./llmhub" >&2
        echo "  make release-snapshot  # creates dist/ artifacts" >&2
        echo "Or place llmhub and install-local.sh in the same directory." >&2
        echo "Or provide a direct path: $0 /path/to/binary" >&2
        exit 1
    fi

    BINARIES=$(find "$DIST_DIR" -type f -name "${BINARY}-linux-*" ! -name "*.txt" ! -name "*.json" 2>/dev/null | sort || true)

    if [ -z "$BINARIES" ]; then
        echo "error: no Linux binaries found in $DIST_DIR/" >&2
        echo "Run one of:" >&2
        echo "  make build             # creates ./llmhub" >&2
        echo "  make release-snapshot  # creates dist/ artifacts" >&2
        echo "Or place llmhub and install-local.sh in the same directory." >&2
        exit 1
    fi

    BINARY_COUNT=$(echo "$BINARIES" | wc -l)

    if [ "$BINARY_COUNT" -eq 1 ]; then
        SELECTED="$BINARIES"
        echo "==> Found 1 binary: $(basename "$SELECTED")"
        return 0
    fi

    echo "==> Found $BINARY_COUNT Linux binaries in $DIST_DIR/:"
    echo ""

    i=1
    echo "$BINARIES" | while IFS= read -r binary; do
        size=$(du -h "$binary" | cut -f1)
        mtime=$(stat -c '%y' "$binary" 2>/dev/null | cut -d' ' -f1 || stat -f '%Sm' -t '%Y-%m-%d' "$binary" 2>/dev/null || echo "unknown")
        printf "%2d) %-40s  %8s  %s\n" "$i" "$(basename "$binary")" "$size" "$mtime"
        i=$((i + 1))
    done

    echo ""
    printf "Select binary to install [1-%d]: " "$BINARY_COUNT"
    read -r choice

    if ! echo "$choice" | grep -Eq '^[0-9]+$' || [ "$choice" -lt 1 ] || [ "$choice" -gt "$BINARY_COUNT" ]; then
        echo "error: invalid selection" >&2
        exit 1
    fi

    SELECTED=$(echo "$BINARIES" | sed -n "${choice}p")
}

render_system_config() {
    src="$1"
    awk -v host="$DEFAULT_HOST" -v port="$DEFAULT_PORT" -v auth_dir="${DATA_DIR}/auths" '
        BEGIN { saw_host = 0; saw_port = 0 }
        /^[[:space:]]*host:[[:space:]]*/ {
            print "host: \"" host "\""
            saw_host = 1
            next
        }
        /^[[:space:]]*port:[[:space:]]*/ {
            print "port: " port
            saw_port = 1
            next
        }
        /^[[:space:]]*auth-dir:[[:space:]]*/ {
            print "auth-dir: \"" auth_dir "\""
            next
        }
        { print }
        END {
            if (!saw_host) {
                print "host: \"" host "\""
            }
            if (!saw_port) {
                print "port: " port
            }
        }
    ' "$src"
}

install_rendered_config() {
    src="$1"
    dst="$2"
    TMP_RENDERED="$(mktemp)"
    render_system_config "$src" >"$TMP_RENDERED"
    install -m 0640 -o root -g "$SERVICE_GROUP" "$TMP_RENDERED" "$dst"
    rm -f "$TMP_RENDERED"
}

ensure_env_bind() {
    env_file="$1"
    TMP_ENV_BIND="$(mktemp)"
    awk -v host="$DEFAULT_HOST" -v port="$DEFAULT_PORT" -v writable_path="$DATA_DIR" '
        BEGIN { wrote_host = 0; wrote_port = 0; wrote_writable_path = 0 }
        /^[[:space:]]*WRITABLE_PATH=/ && !wrote_writable_path {
            print "WRITABLE_PATH=" writable_path
            wrote_writable_path = 1
            next
        }
        /^[[:space:]]*LLMHUB_HOST=/ && !wrote_host {
            print "LLMHUB_HOST=" host
            wrote_host = 1
            next
        }
        /^[[:space:]]*LLMHUB_PORT=/ && !wrote_port {
            print "LLMHUB_PORT=" port
            wrote_port = 1
            next
        }
        { print }
        END {
            if (!wrote_writable_path) {
                print "WRITABLE_PATH=" writable_path
            }
            if (!wrote_host) {
                print "LLMHUB_HOST=" host
            }
            if (!wrote_port) {
                print "LLMHUB_PORT=" port
            }
        }
    ' "$env_file" >"$TMP_ENV_BIND"
    install -m 0640 -o root -g "$SERVICE_GROUP" "$TMP_ENV_BIND" "$env_file"
    rm -f "$TMP_ENV_BIND"
}

prompt_postgres_runtime() {
    env_file="$1"

    if [ ! -t 0 ]; then
        return 0
    fi

    if [ -n "${PGSTORE_DSN:-}" ]; then
        return 0
    fi

    current_dsn="$(read_env_value "$env_file" PGSTORE_DSN 2>/dev/null || true)"
    current_schema="$(read_env_value "$env_file" PGSTORE_SCHEMA 2>/dev/null || true)"
    current_retention="$(read_env_value "$env_file" PGSTORE_USAGE_RETENTION_SECONDS 2>/dev/null || true)"

    if [ -n "$current_dsn" ]; then
        printf "Postgres runtime DSN found in %s. Update it now? [y/N]: " "$env_file"
    else
        printf "Configure Postgres runtime storage now? [y/N]: "
    fi
    read -r postgres_choice

    case "$postgres_choice" in
        [yY]|[yY][eE][sS])
            ;;
        *)
            return 0
            ;;
    esac

    while :; do
        if [ -n "$current_dsn" ]; then
            printf "PGSTORE_DSN [%s]: " "$current_dsn"
        else
            printf "PGSTORE_DSN: "
        fi
        read -r prompted_dsn
        if [ -z "$prompted_dsn" ]; then
            prompted_dsn="$current_dsn"
        fi
        if [ -n "$prompted_dsn" ]; then
            PROMPTED_PGSTORE_DSN="$prompted_dsn"
            break
        fi
        echo "error: PGSTORE_DSN cannot be empty when Postgres runtime is enabled" >&2
    done

    schema_default="${current_schema:-llmhub}"
    printf "PGSTORE_SCHEMA [%s]: " "$schema_default"
    read -r prompted_schema
    if [ -z "$prompted_schema" ]; then
        prompted_schema="$schema_default"
    fi
    PROMPTED_PGSTORE_SCHEMA="$prompted_schema"

    retention_default="${current_retention:-60}"
    while :; do
        printf "PGSTORE_USAGE_RETENTION_SECONDS [%s]: " "$retention_default"
        read -r prompted_retention
        if [ -z "$prompted_retention" ]; then
            prompted_retention="$retention_default"
        fi
        if echo "$prompted_retention" | grep -Eq '^[0-9]+$'; then
            PROMPTED_PGSTORE_USAGE_RETENTION_SECONDS="$prompted_retention"
            break
        fi
        echo "error: PGSTORE_USAGE_RETENTION_SECONDS must be a non-negative integer" >&2
    done
}

ensure_env_postgres() {
    env_file="$1"
    pg_dsn="${PGSTORE_DSN:-${PROMPTED_PGSTORE_DSN:-}}"

    if [ -z "$pg_dsn" ]; then
        return 0
    fi

    pg_schema="${PGSTORE_SCHEMA:-${PROMPTED_PGSTORE_SCHEMA:-}}"
    if [ -z "$pg_schema" ]; then
        pg_schema="llmhub"
    fi

    pg_retention="${PGSTORE_USAGE_RETENTION_SECONDS:-${PROMPTED_PGSTORE_USAGE_RETENTION_SECONDS:-}}"
    if [ -z "$pg_retention" ]; then
        pg_retention="60"
    fi

    TMP_ENV_POSTGRES="$(mktemp)"
    awk -v dsn="$pg_dsn" -v schema="$pg_schema" -v retention="$pg_retention" '
        BEGIN {
            wrote_dsn = 0
            wrote_schema = 0
            wrote_retention = 0
        }
        /^[[:space:]]*PGSTORE_DSN=/ && !wrote_dsn {
            print "PGSTORE_DSN=" dsn
            wrote_dsn = 1
            next
        }
        /^[[:space:]]*PGSTORE_SCHEMA=/ && !wrote_schema {
            print "PGSTORE_SCHEMA=" schema
            wrote_schema = 1
            next
        }
        /^[[:space:]]*PGSTORE_USAGE_RETENTION_SECONDS=/ && !wrote_retention {
            print "PGSTORE_USAGE_RETENTION_SECONDS=" retention
            wrote_retention = 1
            next
        }
        { print }
        END {
            if (!wrote_dsn) {
                print "PGSTORE_DSN=" dsn
            }
            if (!wrote_schema) {
                print "PGSTORE_SCHEMA=" schema
            }
            if (!wrote_retention) {
                print "PGSTORE_USAGE_RETENTION_SECONDS=" retention
            }
        }
    ' "$env_file" >"$TMP_ENV_POSTGRES"
    install -m 0640 -o root -g "$SERVICE_GROUP" "$TMP_ENV_POSTGRES" "$env_file"
    rm -f "$TMP_ENV_POSTGRES"
}

config_port() {
    cfg="${CONFIG_DIR}/config.yaml"
    if [ ! -f "$cfg" ]; then
        return 1
    fi
    awk '
        /^[[:space:]]*#/ { next }
        /^[[:space:]]*port:[[:space:]]*/ {
            value=$0
            sub(/^[[:space:]]*port:[[:space:]]*/, "", value)
            sub(/[[:space:]]*#.*/, "", value)
            gsub(/["'\''[:space:]]/, "", value)
            if (value ~ /^[0-9]+$/) {
                print value
                exit 0
            }
        }
    ' "$cfg"
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
            if (port ~ /^[0-9]+$/) {
                print port
                exit 0
            }
        }
    '
}

resolved_port() {
    port="$(service_port | head -n 1)"
    if [ -n "$port" ]; then
        printf '%s\n' "$port"
        return 0
    fi
    port="$(config_port | head -n 1)"
    if [ -n "$port" ]; then
        printf '%s\n' "$port"
        return 0
    fi
    printf '%s\n' "$DEFAULT_PORT"
}

allow_firewall_port() {
    port="$1"
    if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -qi '^Status:[[:space:]]*active'; then
        ufw allow "${port}/tcp" >/dev/null
        echo "    firewall: allowed ${port}/tcp via ufw"
    fi
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
            active|activating|reloading)
                ;;
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

valid_domain() {
    domain="$1"
    if [ -z "$domain" ]; then
        return 0
    fi

    printf '%s\n' "$domain" | grep -Eq \
        '^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)+$'
}

prompt_domain() {
    if [ -n "$CADDY_DOMAIN" ]; then
        if ! valid_domain "$CADDY_DOMAIN"; then
            echo "error: invalid CADDY_DOMAIN: $CADDY_DOMAIN" >&2
            exit 1
        fi
        return 0
    fi

    if [ -t 0 ]; then
        printf "Domain for Caddy HTTPS (leave blank to skip): "
        read -r CADDY_DOMAIN
        if [ -n "$CADDY_DOMAIN" ] && ! valid_domain "$CADDY_DOMAIN"; then
            echo "error: invalid domain: $CADDY_DOMAIN" >&2
            exit 1
        fi
    fi
}

ensure_debian_apt() {
    if ! command -v apt-get >/dev/null 2>&1 || [ ! -f /etc/debian_version ]; then
        echo "error: automatic Caddy setup currently supports Debian/Ubuntu only" >&2
        exit 1
    fi
}

install_caddy_if_needed() {
    ensure_debian_apt

    if command -v caddy >/dev/null 2>&1; then
        echo "    caddy: $(caddy version | head -n 1)"
        return 0
    fi

    echo "==> Installing Caddy for domain mode"
    apt-get update -qq
    apt-get install -y -qq debian-keyring debian-archive-keyring apt-transport-https curl gnupg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' \
        | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' \
        -o /etc/apt/sources.list.d/caddy-stable.list
    apt-get update -qq
    apt-get install -y -qq caddy
    echo "    caddy: installed"
}

configure_caddy() {
    domain="$1"
    upstream_port="$2"
    caddy_dir=$(dirname -- "$CADDYFILE_PATH")
    backup_file=""

    install_caddy_if_needed
    install -d -m 0755 "$caddy_dir"

    if [ -f "$CADDYFILE_PATH" ]; then
        backup_file="${CADDYFILE_PATH}.bak.$(date +%s)"
        cp -p "$CADDYFILE_PATH" "$backup_file"
        echo "    caddy backup: $backup_file"
    fi

    cat >"$CADDYFILE_PATH" <<EOF
$domain {
    encode gzip
    reverse_proxy 127.0.0.1:$upstream_port
}
EOF

    if ! caddy validate --config "$CADDYFILE_PATH" >/dev/null 2>&1; then
        if [ -n "$backup_file" ] && [ -f "$backup_file" ]; then
            cp -p "$backup_file" "$CADDYFILE_PATH"
        else
            rm -f "$CADDYFILE_PATH"
        fi
        echo "error: generated Caddyfile is invalid" >&2
        exit 1
    fi

    systemctl daemon-reload
    systemctl enable caddy >/dev/null
    systemctl restart caddy

    if ! systemctl is-active --quiet caddy; then
        echo "error: caddy failed to start" >&2
        systemctl status caddy --no-pager >&2 || true
        journalctl -u caddy -n 80 --no-pager >&2 || true
        exit 1
    fi

    allow_firewall_port 80
    allow_firewall_port 443

    if command -v curl >/dev/null 2>&1; then
        if curl -fsSI -H "Host: $domain" "http://127.0.0.1/" >/dev/null 2>&1; then
            echo "    caddy: local proxy check passed"
        else
            echo "warning: local Caddy proxy check failed; confirm DNS points to this VPS and inspect caddy logs" >&2
        fi
    fi
}


# Require root
if [ "$(id -u)" -ne 0 ]; then
    echo "error: run this script with sudo" >&2
    exit 1
fi

# Detect architecture for validation
MACHINE="$(uname -m)"
case "$MACHINE" in
    x86_64)        EXPECTED_ARCH="amd64" ;;
    aarch64|arm64) EXPECTED_ARCH="arm64" ;;
    *)
        echo "error: unsupported architecture: $MACHINE" >&2
        exit 1
        ;;
esac

# Function to validate binary
validate_binary() {
    bin_path="$1"

    if [ ! -f "$bin_path" ]; then
        echo "error: file not found: $bin_path" >&2
        return 1
    fi

    if [ ! -r "$bin_path" ]; then
        echo "error: file not readable: $bin_path" >&2
        return 1
    fi

    # Check if it is an executable binary when file(1) is available.
    if command -v file >/dev/null 2>&1; then
        if ! file "$bin_path" | grep -qE 'ELF.*(executable|shared object)'; then
            echo "error: not a valid Linux executable: $bin_path" >&2
            return 1
        fi
    else
        echo "warning: file(1) not found; skipping ELF validation" >&2
    fi

    # Warn if architecture mismatch can be inferred from the filename.
    bin_name="$(basename "$bin_path")"
    if echo "$bin_name" | grep -q "$EXPECTED_ARCH"; then
        return 0
    elif echo "$bin_name" | grep -qE 'amd64|arm64'; then
        echo "warning: binary architecture may not match system ($MACHINE)" >&2
        printf "Continue anyway? [y/N] "
        read -r confirm
        case "$confirm" in
            [yY]*) return 0 ;;
            *) return 1 ;;
        esac
    fi

    return 0
}

# Determine source binary
if [ $# -eq 1 ]; then
    # Direct path provided
    SELECTED="$1"
    echo "==> Installing from: $SELECTED"
    validate_binary "$SELECTED" || exit 1
elif [ $# -gt 1 ]; then
    echo "error: usage: $0 [path/to/llmhub-linux-${EXPECTED_ARCH}]" >&2
    exit 1
else
    find_source_binary
    validate_binary "$SELECTED" || exit 1
fi

prompt_domain

echo "==> Installing llmhub to $INSTALL_DIR"

# Install binary
install -d -m 0755 "$INSTALL_DIR"
install -m 0755 "$SELECTED" "${INSTALL_DIR}/${BINARY}"
echo "    binary: ${INSTALL_DIR}/${BINARY}"

# Create system group and user (idempotent)
if ! getent group "$SERVICE_GROUP" >/dev/null 2>&1; then
    groupadd --system "$SERVICE_GROUP"
    echo "    group: created $SERVICE_GROUP"
else
    echo "    group: $SERVICE_GROUP (existing)"
fi

if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    useradd --system --gid "$SERVICE_GROUP" --home "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
    echo "    user: created $SERVICE_USER"
else
    echo "    user: $SERVICE_USER (existing)"
fi

# Create runtime directories (idempotent)
mkdir -p "$CONFIG_DIR" "${DATA_DIR}/auths" "$LOG_DIR"
chown -R "${SERVICE_USER}:${SERVICE_GROUP}" "$DATA_DIR" "$LOG_DIR"
chmod 755 "$CONFIG_DIR"
chmod 750 "$DATA_DIR" "${DATA_DIR}/auths"
echo "    directories: $CONFIG_DIR, $DATA_DIR, $LOG_DIR"

# Seed config if not already present
if [ ! -f "${CONFIG_DIR}/config.yaml" ]; then
    if ! TMP_CFG=$(find_local_config_example); then
        echo "error: ${CONFIG_DIR}/config.yaml does not exist and no local config.example.yaml was found" >&2
        echo "Place llmhub, install-local.sh, and optionally config.example.yaml in the same directory." >&2
        echo "Or place config.example.yaml in the current directory or next to the selected binary." >&2
        exit 1
    fi

    echo "    config source: $TMP_CFG"

    install_rendered_config "$TMP_CFG" "${CONFIG_DIR}/config.yaml"

    echo "    config: ${CONFIG_DIR}/config.yaml (seeded from local example - edit before use)"
else
    CURRENT_PORT="$(config_port | head -n 1)"
    if [ "$CURRENT_PORT" != "$DEFAULT_PORT" ]; then
        install_rendered_config "${CONFIG_DIR}/config.yaml" "${CONFIG_DIR}/config.yaml"
        echo "    config: ${CONFIG_DIR}/config.yaml (existing - updated port to ${DEFAULT_PORT})"
    else
        echo "    config: ${CONFIG_DIR}/config.yaml (existing - port ${DEFAULT_PORT})"
    fi
fi

# Write optional environment file for secrets/store backends. By default this
# lives next to the installed binary, so the service loads the binary-adjacent
# .env file operators commonly update during deploys.
if [ ! -f "$ENV_FILE" ]; then
    if TMP_ENV=$(find_local_env_file); then
        install -m 0640 -o root -g "$SERVICE_GROUP" "$TMP_ENV" "$ENV_FILE"
        echo "    environment: $ENV_FILE (seeded from $TMP_ENV)"
    else
        cat >"$ENV_FILE" <<ENV
# Optional environment for llmhub systemd service.
# Examples:
# HOME_JWT=
# PGSTORE_DSN=
# OBJECTSTORE_ENDPOINT=
# GITSTORE_GIT_URL=
WRITABLE_PATH=${DATA_DIR}
LLMHUB_HOST=${DEFAULT_HOST}
LLMHUB_PORT=${DEFAULT_PORT}
ENV
        chmod 0640 "$ENV_FILE"
        chown root:"$SERVICE_GROUP" "$ENV_FILE"
        echo "    environment: $ENV_FILE (created)"
    fi
else
    echo "    environment: $ENV_FILE (existing - left unchanged)"
fi
prompt_postgres_runtime "$ENV_FILE"
ensure_env_bind "$ENV_FILE"
ensure_env_postgres "$ENV_FILE"
echo "    environment: $ENV_FILE (writable path ${DATA_DIR}, host public, port ${DEFAULT_PORT})"
if read_env_value "$ENV_FILE" PGSTORE_DSN >/dev/null 2>&1; then
    echo "    postgres: enabled via $ENV_FILE"
fi

# Write systemd unit
cat >"$UNIT_FILE" <<UNIT
[Unit]
Description=LLMHub proxy server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${CONFIG_DIR}
Environment=HOME=${DATA_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStart=${INSTALL_DIR}/${BINARY} -config ${CONFIG_DIR}/config.yaml
Restart=on-failure
RestartSec=5s
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=${DATA_DIR} ${LOG_DIR} ${CONFIG_DIR}

[Install]
WantedBy=multi-user.target
UNIT

echo "    systemd: $UNIT_FILE"

# Enable and restart so upgrades pick up the newly installed binary.
systemctl daemon-reload
systemctl enable "${SERVICE_NAME}.service"
systemctl restart "${SERVICE_NAME}.service"
echo ""
SERVER_PORT="$(wait_for_service_port)"
systemctl status "${SERVICE_NAME}.service" --no-pager || true

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "SERVER_IP")"
if [ -z "$SERVER_PORT" ]; then
    SERVER_PORT="$(resolved_port)"
fi

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
echo "    Edit ${CONFIG_DIR}/config.yaml to configure providers and accounts."
echo "    Optional env: $ENV_FILE"
echo "    Logs: journalctl -u ${SERVICE_NAME} -f"
if [ -n "$CADDY_DOMAIN" ]; then
    echo "    Caddy logs: journalctl -u caddy -f"
fi
