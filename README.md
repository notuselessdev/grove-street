# Grove Street

> "Ah shit, here we go again." — CJ

GTA San Andreas voice notifications for AI coding agents. Stop babysitting your terminal — let CJ watch it for you.

Grove Street plays Carl Johnson voice lines when your AI agent finishes work, hits an error, needs input, or runs low on context. Works with Claude Code, with more agents coming soon.

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

Grove Street hooks into Claude Code's event system. When your agent does something, CJ reacts:

| Event | CJ Says | When |
|-------|---------|------|
| Session starts | *"Grove Street. Home."* | You open a new Claude Code session |
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
│   ├── grove_street_home.wav
│   └── here_we_go_again.wav
├── task_complete/
│   ├── piece_of_cake.wav
│   └── easy.wav
├── task_error/
│   ├── follow_the_damn_train.wav
│   └── ah_shit.wav
├── input_required/
│   ├── what_you_want.wav
│   └── whats_the_plan.wav
├── resource_limit/
│   ├── gotta_get_outta_here.wav
│   └── runnin_out_of_time.wav
└── user_spam/
    ├── chill_chill.wav
    └── back_off.wav
```

Grove Street picks a random file from the matching category each time.

## Commands

```sh
grove-street play <category>   # Test a sound
grove-street list               # List all sounds
grove-street setup              # Register Claude Code hooks
grove-street update             # Check for updates
grove-street uninstall          # Remove hooks
grove-street version            # Print version
```

## Configuration

Edit `~/.grove-street/config.json`:

```json
{
  "enabled": true,
  "volume": 0.8,
  "auto_update": true
}
```

| Key | Default | Description |
|-----|---------|-------------|
| `enabled` | `true` | Master on/off switch |
| `volume` | `0.8` | Volume level (0.0 - 1.0) |
| `auto_update` | `true` | Check for updates daily |

## Auto-Updates

When `auto_update` is enabled (default), Grove Street checks for new releases daily via a cron job (macOS/Linux) or scheduled task (Windows). Updates download and apply automatically.

Manual update:

```sh
grove-street update
```

## Platform Support

| Platform | Audio Player | Notes |
|----------|-------------|-------|
| macOS | `afplay` (built-in) | Zero dependencies |
| Linux | PipeWire, PulseAudio, FFmpeg, mpv, or ALSA | Auto-detects best available player |
| Windows | PowerShell MediaPlayer | Uses WPF MediaPlayer API |

## Uninstall

```sh
# Homebrew
brew uninstall grove-street

# Shell script
curl -fsSL https://raw.githubusercontent.com/notuselessdev/grove-street/main/uninstall.sh | bash

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

## License

MIT

---

*"All you had to do was follow the damn train, CJ!"*
