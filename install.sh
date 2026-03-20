#!/usr/bin/env bash
set -euo pipefail

# Grove Street Installer
# "Ah shit, here we go again."

REPO="notuselessdev/grove-street"
INSTALL_DIR="${GROVE_STREET_DIR:-$HOME/.grove-street}"
BIN_DIR="/usr/local/bin"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[CJ]${NC} $1"; }
ok()    { echo -e "${GREEN}[CJ]${NC} $1"; }
warn()  { echo -e "${YELLOW}[CJ]${NC} $1"; }
error() { echo -e "${RED}[CJ]${NC} $1" >&2; }

detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux*)  OS="linux" ;;
        darwin*) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
}

get_latest_version() {
    if command -v curl &>/dev/null; then
        curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/'
    elif command -v wget &>/dev/null; then
        wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v?([^"]+)".*/\1/'
    else
        error "Neither curl nor wget found."
        exit 1
    fi
}

download_binary() {
    local version="$1"
    local url="https://github.com/$REPO/releases/download/v${version}/grove-street_${OS}_${ARCH}"
    [ "$OS" = "windows" ] && url="${url}.exe"

    info "Downloading grove-street v${version} for ${OS}/${ARCH}..." >&2

    local tmp
    tmp="$(mktemp)"
    if command -v curl &>/dev/null; then
        curl -fsSL "$url" -o "$tmp"
    else
        wget -qO "$tmp" "$url"
    fi

    chmod +x "$tmp"
    echo "$tmp"
}

install_sounds() {
    local sounds_dir="$INSTALL_DIR/sounds"
    info "Installing CJ voice lines..."

    local categories=("session_start" "task_complete" "task_error" "input_required" "resource_limit" "user_spam")
    for cat in "${categories[@]}"; do
        mkdir -p "$sounds_dir/$cat"
    done

    local version="$1"
    local sounds_url="https://github.com/$REPO/releases/download/v${version}/sounds.tar.gz"
    local tmp
    tmp="$(mktemp)"

    if command -v curl &>/dev/null; then
        if curl -fsSL "$sounds_url" -o "$tmp" 2>/dev/null; then
            tar -xzf "$tmp" -C "$sounds_dir"
            rm -f "$tmp"
            ok "Sound pack installed."
            return
        fi
    fi

    warn "Could not download sound pack. Add your own .wav/.mp3 files to: $sounds_dir/<category>/"
    warn "Categories: ${categories[*]}"
}

write_config() {
    local config_path="$INSTALL_DIR/config.json"
    if [ ! -f "$config_path" ]; then
        cat > "$config_path" <<'EOF'
{
  "enabled": true,
  "volume": 0.8,
  "auto_update": true
}
EOF
        ok "Default config written to $config_path"
    fi
}

register_hooks() {
    local settings_dir="$HOME/.claude"
    local settings_path="$settings_dir/settings.json"
    local bin_path="$BIN_DIR/grove-street"

    mkdir -p "$settings_dir"

    if [ -f "$settings_path" ]; then
        if command -v python3 &>/dev/null; then
            python3 - "$settings_path" "$bin_path" <<'PYEOF'
import json, sys
path, bin = sys.argv[1], sys.argv[2]
try:
    with open(path) as f:
        settings = json.load(f)
except:
    settings = {}

hook_cmd = f"{bin} hook"
hooks = settings.get("hooks", {})
for event in ["Stop", "Notification", "SubagentStop", "PreCompact"]:
    existing = hooks.get(event, [])
    filtered = []
    for h in existing:
        cmd = h.get("command", "")
        nested = h.get("hooks", [])
        nested_cmds = [n.get("command", "") for n in nested if isinstance(n, dict)]
        if "grove-street" not in cmd and not any("grove-street" in c for c in nested_cmds):
            filtered.append(h)
    filtered.append({"matcher": "", "hooks": [{"type": "command", "command": hook_cmd}]})
    hooks[event] = filtered
settings["hooks"] = hooks
with open(path, "w") as f:
    json.dump(settings, f, indent=2)
PYEOF
        else
            warn "python3 not found — run 'grove-street setup' after install to register hooks."
            return
        fi
    else
        cat > "$settings_path" <<JSONEOF
{
  "hooks": {
    "Stop": [{"matcher": "", "hooks": [{"type": "command", "command": "$bin_path hook"}]}],
    "Notification": [{"matcher": "", "hooks": [{"type": "command", "command": "$bin_path hook"}]}],
    "SubagentStop": [{"matcher": "", "hooks": [{"type": "command", "command": "$bin_path hook"}]}],
    "PreCompact": [{"matcher": "", "hooks": [{"type": "command", "command": "$bin_path hook"}]}]
  }
}
JSONEOF
    fi

    ok "Hooks registered in Claude Code."
}

setup_auto_update() {
    if command -v crontab &>/dev/null; then
        local existing
        existing="$(crontab -l 2>/dev/null || true)"
        if echo "$existing" | grep -q "grove-street update"; then
            return
        fi
        (echo "$existing"; echo "0 12 * * * $BIN_DIR/grove-street update >/dev/null 2>&1") | crontab -
        ok "Auto-update cron job installed (daily at noon)."
    fi
}

main() {
    echo ""
    echo -e "${GREEN} ██████╗ ██████╗  ██████╗ ██╗   ██╗███████╗${NC}"
    echo -e "${GREEN}██╔════╝ ██╔══██╗██╔═══██╗██║   ██║██╔════╝${NC}"
    echo -e "${GREEN}██║  ███╗██████╔╝██║   ██║██║   ██║█████╗  ${NC}"
    echo -e "${GREEN}██║   ██║██╔══██╗██║   ██║╚██╗ ██╔╝██╔══╝  ${NC}"
    echo -e "${GREEN}╚██████╔╝██║  ██║╚██████╔╝ ╚████╔╝ ███████╗${NC}"
    echo -e "${GREEN} ╚═════╝ ╚═╝  ╚═╝ ╚═════╝   ╚═══╝  ╚══════╝${NC}"
    echo -e "        ${CYAN}███████╗████████╗██████╗ ███████╗███████╗████████╗${NC}"
    echo -e "        ${CYAN}██╔════╝╚══██╔══╝██╔══██╗██╔════╝██╔════╝╚══██╔══╝${NC}"
    echo -e "        ${CYAN}███████╗   ██║   ██████╔╝█████╗  █████╗     ██║   ${NC}"
    echo -e "        ${CYAN}╚════██║   ██║   ██╔══██╗██╔══╝  ██╔══╝     ██║   ${NC}"
    echo -e "        ${CYAN}███████║   ██║   ██║  ██║███████╗███████╗   ██║   ${NC}"
    echo -e "        ${CYAN}╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚══════╝   ╚═╝   ${NC}"
    echo ""
    echo -e "  ${YELLOW}\"Ah shit, here we go again.\"${NC} — CJ"
    echo ""

    detect_platform
    info "Detected: ${OS}/${ARCH}"

    local version
    version="$(get_latest_version)"
    if [ -z "$version" ]; then
        error "Could not determine latest version."
        exit 1
    fi
    info "Latest version: v${version}"

    local tmp_bin
    tmp_bin="$(download_binary "$version")"

    mkdir -p "$INSTALL_DIR"
    if [ -w "$BIN_DIR" ]; then
        mv "$tmp_bin" "$BIN_DIR/grove-street"
    else
        info "Need sudo to install to $BIN_DIR"
        sudo mv "$tmp_bin" "$BIN_DIR/grove-street"
        sudo chmod +x "$BIN_DIR/grove-street"
    fi
    ok "Binary installed to $BIN_DIR/grove-street"

    install_sounds "$version"
    write_config
    register_hooks
    setup_auto_update

    echo ""
    ok "Installation complete!"
    echo ""
    echo -e "  ${CYAN}Grove Street. Home. CJ is watching your terminal now.${NC}"
    echo ""
    echo "  Commands:"
    echo "    grove-street play session_start   # Test a sound"
    echo "    grove-street list                  # See all sounds"
    echo "    grove-street update                # Check for updates"
    echo "    grove-street help                  # Full help"
    echo ""
    echo -e "  ${YELLOW}-------------------------------------------------------${NC}"
    echo ""
    echo -e "  ${GREEN}🎮 Enjoying Grove Street? Help CJ spread the word!${NC}"
    echo ""
    echo -e "  ${CYAN}⭐ Give us a star:${NC} https://github.com/notuselessdev/grove-street"
    echo -e "  ${CYAN}🐦 Share on X:${NC}     https://x.com/intent/tweet?text=Just%20installed%20Grove%20Street%20%E2%80%94%20CJ%20from%20GTA%20San%20Andreas%20now%20narrates%20my%20AI%20coding%20agent%20%F0%9F%8E%AE%F0%9F%94%8A&url=https://github.com/notuselessdev/grove-street"
    echo -e "  ${CYAN}🌐 Website:${NC}        https://notuseless.dev"
    echo -e "  ${CYAN}👤 Follow the dev:${NC} https://x.com/notuselessdev"
    echo -e "  ${CYAN}🐛 Report issues:${NC}  https://github.com/notuselessdev/grove-street/issues"
    echo ""
    echo -e "  ${YELLOW}\"Respect is everything, CJ.\" 🫡${NC}"
    echo ""
}

main "$@"
