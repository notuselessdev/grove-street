#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

ok()    { echo -e "${GREEN}[CJ]${NC} $1"; }

BIN_DIR="/usr/local/bin"
INSTALL_DIR="${GROVE_STREET_DIR:-$HOME/.grove-street}"

if [ -f "$BIN_DIR/grove-street" ]; then
    if [ -w "$BIN_DIR" ]; then
        rm "$BIN_DIR/grove-street"
    else
        sudo rm "$BIN_DIR/grove-street"
    fi
    ok "Binary removed."
fi

if [ -d "$INSTALL_DIR" ]; then
    rm -rf "$INSTALL_DIR"
    ok "Data directory removed."
fi

SETTINGS="$HOME/.claude/settings.json"
if [ -f "$SETTINGS" ] && command -v python3 &>/dev/null; then
    python3 - "$SETTINGS" <<'PYEOF'
import json, sys
path = sys.argv[1]
try:
    with open(path) as f:
        settings = json.load(f)
except:
    sys.exit(0)

hooks = settings.get("hooks", {})
for event in list(hooks.keys()):
    hooks[event] = [h for h in hooks[event] if "grove-street" not in h.get("command", "")]
    if not hooks[event]:
        del hooks[event]
if hooks:
    settings["hooks"] = hooks
elif "hooks" in settings:
    del settings["hooks"]
with open(path, "w") as f:
    json.dump(settings, f, indent=2)
PYEOF
    ok "Hooks removed from Claude Code."
fi

if command -v crontab &>/dev/null; then
    crontab -l 2>/dev/null | grep -v "grove-street" | crontab - 2>/dev/null || true
    ok "Cron job removed."
fi

echo ""
ok "CJ has left the building. See you around, homie."
echo ""
