<p align="center">
  <img src="assets/icon.png" width="150" alt="CJ from GTA San Andreas" />
</p>

<h1 align="center">Grove Street</h1>

<p align="center">
  <em>"Ah shit, here we go again." — CJ</em>
</p>

<p align="center">
  GTA San Andreas voice notifications for AI coding agents.<br/>
  Stop babysitting your terminal — let CJ watch it for you.
</p>

<p align="center">
  <a href="https://github.com/notuselessdev/grove-street/releases/latest"><img src="https://img.shields.io/github/v/release/notuselessdev/grove-street" alt="Release"></a>
  <a href="https://github.com/notuselessdev/grove-street/blob/main/LICENSE"><img src="https://img.shields.io/github/license/notuselessdev/grove-street" alt="License"></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
</p>

---

Grove Street plays Carl Johnson voice lines when your AI agent finishes work, hits an error, needs input, or runs low on context. Native macOS notifications with CJ's face pop up on screen — click to focus your terminal. Multiple notifications stack and reflow when dismissed.

**Supported IDEs:** Claude Code, Cursor, Windsurf, GitHub Copilot, Kiro

## Install

### Homebrew (macOS & Linux)

```sh
brew install notuselessdev/tap/grove-street
```

### Shell script (macOS & Linux)

```sh
curl -fsSL https://raw.githubusercontent.com/notuselessdev/grove-street/main/install.sh | bash
```

### PowerShell (Windows)

```powershell
irm https://raw.githubusercontent.com/notuselessdev/grove-street/main/install.ps1 | iex
```

### From source

```sh
go install github.com/notuselessdev/grove-street/cmd/grove-street@latest
grove-street setup
```

## How It Works

Grove Street hooks into your AI coding agent's event system. When your agent does something, CJ reacts with a voice line and a native notification:

| Event | CJ Says | When |
|-------|---------|------|
| Session starts | *"Grove Street. Home."* | You open a new session |
| Task complete | *"Piece of cake!"* | Agent finishes work |
| Error | *"All you had to do was follow the damn train, CJ!"* | Something breaks |
| Needs input | *"So what's the plan?"* | Agent needs your approval or answer |
| Context low | *"I gotta get outta here!"* | Context window running out |
| Spam | *"Chill, chill!"* | You're sending messages too fast |

## Sound Categories

Each category has multiple voice lines that play randomly:

**`session_start`** — New session begins
- "Grove Street. Home. At least it was before I fucked everything up."
- "Ah shit, here we go again."
- "Let's roll."
- "Time to put in work."
- "You picked the wrong house, fool!"

**`task_complete`** — Work finished
- "Piece of cake!"
- "Easy!"
- "Show me the money!"
- "Nice!"
- "Look at me now!"

**`task_error`** — Something went wrong
- "Ah shit, here we go again."
- "All you had to do was follow the damn train, CJ!"
- "I hate gravity!"
- "Shit, this happens too often!"

**`input_required`** — Needs your input
- "What you want?"
- "So what's the plan?"
- "What we gonna do now?"
- "Say something!"

**`resource_limit`** — Running low on context
- "I gotta get outta here!"
- "We runnin' out of time!"
- "We on fire!"
- "I can smell burning!"

**`user_spam`** — Calm down
- "Chill, chill!"
- "Man, chill out!"
- "Hey, relax!"
- "Back off!"

## Adding Your Own Sounds

Drop `.wav` or `.mp3` files into the category folders:

```
~/.grove-street/sounds/
├── session_start/
├── task_complete/
├── task_error/
├── input_required/
├── resource_limit/
└── user_spam/
```

Grove Street picks a random file from the matching category each time.

## Commands

```sh
grove-street play <category>   # Test a sound
grove-street list               # List all sounds
grove-street setup              # Register hooks for detected IDEs
grove-street update             # Check for updates
grove-street uninstall          # Remove hooks from all IDEs
grove-street version            # Print version
```

## Configuration

Edit `~/.grove-street/config.json`:

```json
{
  "enabled": true,
  "volume": 0.8,
  "auto_update": true,
  "notifications": true,
  "notification_position": "top-right",
  "notification_duration_seconds": 7
}
```

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Master on/off switch |
| `volume` | `0.8` | Volume level (0.0 - 1.0) |
| `auto_update` | `true` | Check for updates daily |
| `notifications` | `true` | Show native overlay notifications |
| `notification_position` | `"top-right"` | Position: `top-left`, `top-center`, `top-right`, `bottom-left`, `bottom-center`, `bottom-right`, `center` |
| `notification_duration_seconds` | `7` | How long notifications stay on screen |

## Platform Support

| Platform | Audio Player | Notifications |
|----------|-------------|---------------|
| macOS | `afplay` (built-in) | Native overlay (Swift/Cocoa) |
| Linux | PipeWire, PulseAudio, FFmpeg, mpv, or ALSA | Native overlay (Python/GTK3) |
| Windows | PowerShell MediaPlayer | Native overlay (PowerShell/WPF) |

## Uninstall

```sh
# Homebrew (auto-removes hooks from all IDEs)
brew uninstall grove-street

# Manual
grove-street uninstall
rm -rf ~/.grove-street
```

## Building from Source

```sh
git clone https://github.com/notuselessdev/grove-street.git
cd grove-street
go build -o grove-street ./cmd/grove-street
./grove-street setup
```

## Disclaimer

Grove Street is a fan project and is not affiliated with or endorsed by Rockstar Games or Take-Two Interactive. GTA San Andreas and all related characters, voice lines, and assets are property of their respective owners. This project is intended for personal, non-commercial use.

## License

MIT — applies to the source code only. Audio assets from GTA San Andreas are the property of Rockstar Games / Take-Two Interactive and are not covered by this license.

---

*"All you had to do was follow the damn train, CJ!"*
