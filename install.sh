#!/bin/sh
# LLMHub One-Line Automated Installer & Updater for VPS / Linux / macOS
# Repository: https://github.com/therealtinhtute/llmhub
#
# Quick install:
#   curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/install.sh | bash
#
# Custom directory:
#   curl -fsSL https://raw.githubusercontent.com/therealtinhtute/llmhub/master/install.sh | LLMHUB_DIR=/opt/llmhub bash
set -eu

REPO="therealtinhtute/llmhub"
GITHUB_URL="https://github.com/${REPO}"
API_URL="https://api.github.com/repos/${REPO}"

# ANSI color codes
BOLD="\033[1m"
GREEN="\033[0;32m"
BLUE="\033[0;34m"
YELLOW="\033[0;33m"
RED="\033[0;31m"
RESET="\033[0m"

log_info() {
    printf "${BLUE}[INFO]${RESET} %s\n" "$1"
}

log_success() {
    printf "${GREEN}[OK]${RESET} %s\n" "$1"
}

log_warn() {
    printf "${YELLOW}[WARN]${RESET} %s\n" "$1"
}

log_error() {
    printf "${RED}[ERROR]${RESET} %s\n" "$1" >&2
}

banner() {
    printf "${BOLD}${BLUE}"
    cat << 'EOF'
    __    __    __  ___ __  __      __  
   / /   / /   /  |/  // / / /_  __/ /_ 
  / /   / /   / /|_/ // /_/ / / / / __ \
 / /___/ /___/ /  / // __  / /_/ / /_/ /
/_____/_____/_/  /_//_/ /_/\__,_/_.___/ 
EOF
    printf "${RESET}"
    printf "${BOLD}LLMHub Unified Gateway Installer & Updater${RESET}\n\n"
}

# Check command existence
has_cmd() {
    command -v "$1" >/dev/null 2>&1
}

# Download wrapper (curl or wget)
download_file() {
    url="$1"
    output="$2"
    if has_cmd curl; then
        curl -fsSL "$url" -o "$output"
    elif has_cmd wget; then
        wget -qO "$output" "$url"
    else
        log_error "Neither curl nor wget is installed."
        exit 1
    fi
}

download_stdout() {
    url="$1"
    if has_cmd curl; then
        curl -fsSL "$url"
    elif has_cmd wget; then
        wget -qO- "$url"
    else
        log_error "Neither curl nor wget is installed."
        exit 1
    fi
}

# Generate 32-byte base64 quota secret key
generate_quota_secret() {
    if has_cmd openssl; then
        openssl rand -base64 32 | tr -d '\n'
    else
        head -c 32 /dev/urandom | base64 | tr -d '\n'
    fi
}

# Generate random management password
generate_password() {
    if has_cmd openssl; then
        openssl rand -hex 16
    else
        head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'
    fi
}

read_env_value() {
    env_file="$1"
    key="$2"
    [ -f "$env_file" ] || return 1
    val=$(awk -v key="$key" '
        /^[[:space:]]*#/ { next }
        index($0, key "=") == 1 {
            print substr($0, length(key) + 2)
            exit 0
        }
    ' "$env_file")
    val=$(printf '%s' "$val" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
    case "$val" in
        \"*\") val="${val#\"}"; val="${val%\"}" ;;
        \'*\') val="${val#\'}"; val="${val%\'}" ;;
    esac
    printf '%s' "$val"
}

# Detect operating system
detect_os() {
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux*)  printf "linux" ;;
        darwin*) printf "darwin" ;;
        freebsd*) printf "freebsd" ;;
        *)
            log_error "Unsupported operating system: $os"
            exit 1
            ;;
    esac
}

# Detect architecture
detect_arch() {
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   printf "amd64" ;;
        aarch64|arm64)  printf "arm64" ;;
        armv7l|armv6l)  printf "armv7" ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

# Verify sha256 checksum
verify_checksum() {
    file="$1"
    checksum="$2"
    if has_cmd sha256sum; then
        actual="$(sha256sum "$file" | awk '{print $1}')"
    elif has_cmd shasum; then
        actual="$(shasum -a 256 "$file" | awk '{print $1}')"
    elif has_cmd openssl; then
        actual="$(openssl dgst -sha256 "$file" | awk '{print $NF}')"
    else
        log_warn "sha256sum/shasum not found; skipping binary checksum verification."
        return 0
    fi

    if [ "$actual" != "$checksum" ]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $checksum"
        log_error "Actual:   $actual"
        return 1
    fi
    return 0
}

# Find primary IP address
get_server_ip() {
    if has_cmd hostname; then
        ip="$(hostname -I 2>/dev/null | awk '{print $1}' || true)"
        [ -n "$ip" ] && printf '%s' "$ip" && return 0
    fi
    if has_cmd ip; then
        ip="$(ip -4 route get 1.1.1.1 2>/dev/null | awk '{print $7}' || true)"
        [ -n "$ip" ] && printf '%s' "$ip" && return 0
    fi
    printf "127.0.0.1"
}

main() {
    banner

    OS="$(detect_os)"
    ARCH="$(detect_arch)"
    log_info "Detected environment: ${OS}-${ARCH}"

    # Target directory setup (all-in-one directory on VPS)
    if [ -n "${LLMHUB_DIR:-}" ]; then
        TARGET_DIR="$LLMHUB_DIR"
    elif [ "$(id -u)" -eq 0 ]; then
        TARGET_DIR="/opt/llmhub"
    else
        TARGET_DIR="${HOME}/llmhub"
    fi

    DATA_DIR="${TARGET_DIR}/data"
    UPDATE_DIR="${DATA_DIR}/update"
    ENV_FILE="${TARGET_DIR}/.env"
    BIN_TARGET="${TARGET_DIR}/llmhub"

    log_info "Target directory: ${TARGET_DIR}"
    mkdir -p "${TARGET_DIR}" "${DATA_DIR}" "${UPDATE_DIR}"

    # Fetch latest release info
    ASSET_NAME="llmhub-${OS}-${ARCH}"
    CHECKSUM_NAME="checksums.txt"
    LATEST_RELEASE_URL="${GITHUB_URL}/releases/latest/download"

    TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

    log_info "Downloading latest release (${ASSET_NAME})..."
    TMP_BIN="${TMP_DIR}/${ASSET_NAME}"
    TMP_CHECKSUMS="${TMP_DIR}/${CHECKSUM_NAME}"

    download_file "${LATEST_RELEASE_URL}/${ASSET_NAME}" "$TMP_BIN"
    chmod +x "$TMP_BIN"

    # Verify checksum if checksums.txt is available
    if download_file "${LATEST_RELEASE_URL}/${CHECKSUM_NAME}" "$TMP_CHECKSUMS" 2>/dev/null; then
        EXPECTED_HASH="$(grep -F "$ASSET_NAME" "$TMP_CHECKSUMS" | awk '{print $1}' || true)"
        if [ -n "$EXPECTED_HASH" ]; then
            log_info "Verifying binary checksum..."
            verify_checksum "$TMP_BIN" "$EXPECTED_HASH"
            log_success "Checksum verified: $(printf '%.16s' "$EXPECTED_HASH")..."
        fi
    fi

    # Install binary
    IS_UPGRADE=0
    if [ -f "$BIN_TARGET" ]; then
        IS_UPGRADE=1
        log_info "Upgrading existing binary at ${BIN_TARGET}..."
        cp "$TMP_BIN" "${BIN_TARGET}.tmp"
        mv -f "${BIN_TARGET}.tmp" "$BIN_TARGET"
    else
        cp "$TMP_BIN" "$BIN_TARGET"
    fi
    chmod 755 "$BIN_TARGET"
    log_success "Binary installed to ${BIN_TARGET}"

    # Configure .env file
    if [ ! -f "$ENV_FILE" ]; then
        log_info "Creating default environment configuration: ${ENV_FILE}"

        MGMT_PASS="${MANAGEMENT_PASSWORD:-$(generate_password)}"
        QUOTA_KEY="${LLMHUB_QUOTA_SECRET_KEY_B64:-$(generate_quota_secret)}"
        PG_DSN="${PGSTORE_DSN:-}"
        PORT="${LLMHUB_PORT:-8317}"
        HOST="${LLMHUB_HOST:-0.0.0.0}"

        # Interactive Postgres prompt if terminal is attached and PGSTORE_DSN is unset
        if [ -z "$PG_DSN" ] && [ -r /dev/tty ] && [ -w /dev/tty ]; then
            printf "\nEnter PostgreSQL connection DSN (e.g. postgresql://user:pass@host:5432/llmhub) or press Enter to skip: " >/dev/tty
            read -r PG_DSN_INPUT </dev/tty || PG_DSN_INPUT=""
            PG_DSN="$(printf '%s' "$PG_DSN_INPUT" | tr -d ' \n')"
        fi
        cat > "$ENV_FILE" << EOF
# LLMHub Configuration
# Target Directory: ${TARGET_DIR}

LLMHUB_HOST=${HOST}
LLMHUB_PORT=${PORT}
MANAGEMENT_PASSWORD=${MGMT_PASS}

# PostgreSQL Token & Quota Store (required for quota alerts & multi-instance sync)
PGSTORE_DSN=${PG_DSN}
PGSTORE_SCHEMA=llmhub
PGSTORE_USAGE_RETENTION_SECONDS=60

# Quota Alert Telegram Encryption Key (AES-256-GCM 32-byte root key)
LLMHUB_QUOTA_SECRET_KEY_B64=${QUOTA_KEY}
LLMHUB_QUOTA_SECRET_KEY_ID=runtime

# Persistent data directory
DATA_DIR=${DATA_DIR}
EOF
        chmod 600 "$ENV_FILE"
        log_success "Created ${ENV_FILE} with pre-configured quota secret key and management password"
    else
        log_info "Preserving existing ${ENV_FILE}"

        # Ensure LLMHUB_QUOTA_SECRET_KEY_B64 exists in existing .env
        EXISTING_KEY="$(read_env_value "$ENV_FILE" LLMHUB_QUOTA_SECRET_KEY_B64 || true)"
        if [ -z "$EXISTING_KEY" ]; then
            EXISTING_KEY="$(read_env_value "$ENV_FILE" llmhub_quota_secret_key_b64 || true)"
        fi
        if [ -z "$EXISTING_KEY" ]; then
            NEW_KEY="$(generate_quota_secret)"
            printf "\n# Quota Alert Telegram Encryption Key (auto-generated)\nLLMHUB_QUOTA_SECRET_KEY_B64=%s\nLLMHUB_QUOTA_SECRET_KEY_ID=runtime\n" "$NEW_KEY" >> "$ENV_FILE"
            log_success "Added missing LLMHUB_QUOTA_SECRET_KEY_B64 to existing ${ENV_FILE}"
        fi

        # Ensure DATA_DIR is recorded in existing .env
        EXISTING_DATA_DIR="$(read_env_value "$ENV_FILE" DATA_DIR || true)"
        if [ -z "$EXISTING_DATA_DIR" ]; then
            printf "\nDATA_DIR=%s\n" "$DATA_DIR" >> "$ENV_FILE"
        fi
    fi

    # Initialize Postgres schema if DSN is set
    CONFIGURED_DSN="$(read_env_value "$ENV_FILE" PGSTORE_DSN || true)"
    if [ -n "$CONFIGURED_DSN" ]; then
        log_info "Initializing/migrating database schema..."
        if "$BIN_TARGET" init-db-from-env -env-file "$ENV_FILE" >/dev/null 2>&1; then
            log_success "Database schema initialized successfully"
        else
            log_warn "Could not connect to PostgreSQL with the configured DSN. Please verify PGSTORE_DSN in ${ENV_FILE}."
        fi
    fi

    # Setup systemd service if running on Linux with systemd and root/sudo access
    SERVICE_INSTALLED=0
    if [ "$OS" = "linux" ] && has_cmd systemctl && [ "$(id -u)" -eq 0 ]; then
        SERVICE_FILE="/etc/systemd/system/llmhub.service"
        log_info "Configuring systemd service: ${SERVICE_FILE}"

        cat > "$SERVICE_FILE" << EOF
[Unit]
Description=LLMHub Gateway Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${TARGET_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStartPre=+${BIN_TARGET} apply-staged-update
ExecStart=${BIN_TARGET}
Restart=always
RestartSec=3s
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

        systemctl daemon-reload
        systemctl enable llmhub.service >/dev/null 2>&1
        systemctl restart llmhub.service
        SERVICE_INSTALLED=1
        log_success "Systemd service 'llmhub' enabled and started"
    fi

    # Read final configuration for summary
    PORT="$(read_env_value "$ENV_FILE" LLMHUB_PORT || echo "8317")"
    MGMT_PASS="$(read_env_value "$ENV_FILE" MANAGEMENT_PASSWORD || echo "(check ${ENV_FILE})")"
    SERVER_IP="$(get_server_ip)"

    printf "\n"
    printf "${BOLD}${GREEN}======================================================${RESET}\n"
    if [ "$IS_UPGRADE" -eq 1 ]; then
        printf "${BOLD}${GREEN}   LLMHub Successfully Upgraded!                     ${RESET}\n"
    else
        printf "${BOLD}${GREEN}   LLMHub Successfully Installed!                    ${RESET}\n"
    fi
    printf "${BOLD}${GREEN}======================================================${RESET}\n\n"

    printf "  ${BOLD}Working Directory:${RESET}  %s\n" "${TARGET_DIR}"
    printf "  ${BOLD}Binary Location:${RESET}    %s\n" "${BIN_TARGET}"
    printf "  ${BOLD}Config / Env:${RESET}       %s\n" "${ENV_FILE}"
    printf "  ${BOLD}Management Web UI:${RESET}  http://%s:%s/management.html\n" "${SERVER_IP}" "${PORT}"
    printf "  ${BOLD}API Endpoint:${RESET}       http://%s:%s/v1\n" "${SERVER_IP}" "${PORT}"
    printf "  ${BOLD}Management Key:${RESET}     %s\n\n" "${MGMT_PASS}"

    if [ "$SERVICE_INSTALLED" -eq 1 ]; then
        printf "  ${BOLD}Service Commands:${RESET}\n"
        printf "    Check status:  systemctl status llmhub\n"
        printf "    View live log: journalctl -u llmhub -f\n"
        printf "    Restart:       systemctl restart llmhub\n\n"
    else
        printf "  ${BOLD}Manual Start Command:${RESET}\n"
        printf "    cd %s && ./llmhub\n\n" "${TARGET_DIR}"
    fi
}

main "$@"
