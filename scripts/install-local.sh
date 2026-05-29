#!/bin/sh
# LLMHub local binary installer with full VPS setup
# Usage:
#   ./scripts/install-local.sh /path/to/llmhub-linux-amd64  # install specific binary
#   ./scripts/install-local.sh                              # install ./llmhub, or select from dist/
set -eu

BINARY="llmhub"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="${CONFIG_DIR:-/etc/llmhub}"
DATA_DIR="${DATA_DIR:-/var/lib/llmhub}"
LOG_DIR="${LOG_DIR:-/var/log/llmhub}"
SERVICE_USER="${SERVICE_USER:-llmhub}"
SERVICE_GROUP="${SERVICE_GROUP:-$SERVICE_USER}"
SERVICE_NAME="${SERVICE_NAME:-llmhub}"
ENV_FILE="${ENV_FILE:-/etc/default/llmhub}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
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
    # Prefer the standard local build output when it exists.
    if [ -f "./${BINARY}" ]; then
        SELECTED="./${BINARY}"
        echo "==> Found local binary: $SELECTED"
        validate_binary "$SELECTED" || exit 1
    else
        # Interactive selection from dist/
        DIST_DIR="dist"

        # Check if running from repo root
        if [ ! -f "go.mod" ]; then
            echo "error: run this script from the repository root or pass a binary path" >&2
            exit 1
        fi

        # Ensure dist/ exists
        if [ ! -d "$DIST_DIR" ]; then
            echo "error: $DIST_DIR/ not found" >&2
            echo "Run one of:" >&2
            echo "  make build             # creates ./llmhub" >&2
            echo "  make release-snapshot  # creates dist/ artifacts" >&2
            echo "Or provide a direct path: $0 /path/to/binary" >&2
            exit 1
        fi

        # Find available binaries
        BINARIES=$(find "$DIST_DIR" -type f -name "${BINARY}-linux-*" ! -name "*.txt" ! -name "*.json" 2>/dev/null | sort || true)

        if [ -z "$BINARIES" ]; then
            echo "error: no Linux binaries found in $DIST_DIR/" >&2
            echo "Run one of:" >&2
            echo "  make build             # creates ./llmhub" >&2
            echo "  make release-snapshot  # creates dist/ artifacts" >&2
            exit 1
        fi

        # Count binaries
        BINARY_COUNT=$(echo "$BINARIES" | wc -l)

        if [ "$BINARY_COUNT" -eq 1 ]; then
            # Only one binary, use it directly
            SELECTED="$BINARIES"
            echo "==> Found 1 binary: $(basename "$SELECTED")"
        else
            # Multiple binaries, show selection menu
            echo "==> Found $BINARY_COUNT Linux binaries in $DIST_DIR/:"
            echo ""

            # Build menu with file info
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

            # Validate choice
            if ! echo "$choice" | grep -Eq '^[0-9]+$' || [ "$choice" -lt 1 ] || [ "$choice" -gt "$BINARY_COUNT" ]; then
                echo "error: invalid selection" >&2
                exit 1
            fi

            # Get selected binary
            SELECTED=$(echo "$BINARIES" | sed -n "${choice}p")
        fi

        validate_binary "$SELECTED" || exit 1
    fi
fi

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
        echo "Place config.example.yaml in the current directory, next to install-local.sh, or next to the binary." >&2
        exit 1
    fi

    echo "    config source: $TMP_CFG"

    # Adjust auth-dir path for system installation
    TMP_RENDERED="$(mktemp)"
    sed "s|auth-dir:.*|auth-dir: \"${DATA_DIR}/auths\"|" "$TMP_CFG" >"$TMP_RENDERED"
    install -m 0640 -o root -g "$SERVICE_GROUP" "$TMP_RENDERED" "${CONFIG_DIR}/config.yaml"
    rm -f "$TMP_RENDERED"

    echo "    config: ${CONFIG_DIR}/config.yaml (seeded from local example - edit before use)"
else
    echo "    config: ${CONFIG_DIR}/config.yaml (existing - left unchanged)"
fi

# Write optional environment file for secrets/store backends.
if [ ! -f "$ENV_FILE" ]; then
    cat >"$ENV_FILE" <<ENV
# Optional environment for llmhub systemd service.
# Examples:
# HOME_JWT=
# PGSTORE_DSN=
# OBJECTSTORE_ENDPOINT=
# GITSTORE_GIT_URL=
WRITABLE_PATH=${DATA_DIR}
ENV
    chmod 0640 "$ENV_FILE"
    chown root:"$SERVICE_GROUP" "$ENV_FILE"
    echo "    environment: $ENV_FILE (created)"
else
    echo "    environment: $ENV_FILE (existing - left unchanged)"
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
systemctl status "${SERVICE_NAME}.service" --no-pager || true

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}' || echo "SERVER_IP")"
echo ""
echo "==> llmhub is running"
echo "    API endpoint:     http://${SERVER_IP}:8317/v1"
echo "    Management panel: http://${SERVER_IP}:8317/management.html"
echo ""
echo "    Edit ${CONFIG_DIR}/config.yaml to configure providers and accounts."
echo "    Optional env: $ENV_FILE"
echo "    Logs: journalctl -u ${SERVICE_NAME} -f"
