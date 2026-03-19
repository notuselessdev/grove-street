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
  <a href="https://github.com/notuselessdev/grove-street/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue" alt="License"></a>
  <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue" alt="Platform">
</p>

<p align="center">
  <img src="notification-demo.gif" width="700" alt="Grove Street notification demo" />
</p>

---

Grove Street plays Carl Johnson voice lines when your AI agent finishes work, hits an error, needs input, or runs low on context. Native notifications with CJ's face pop up on screen. Multiple notifications stack and reflow when dismissed.

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
| Session starts | *"It's CJ."* | You open a new session |
| Task complete | *"Ay, ain't this fun?"* | Agent finishes work |
| Error | *"Not this crap again."* | Something breaks |
| Needs input | *"What you smoking?"* | Agent needs your approval or answer |
| Context low | *"Dude, I ain't got time for this."* | Context window running out |
| Spam | *"Dude, please shut up."* | You're sending messages too fast |

## Sound Categories

Each category has multiple voice lines that play randomly:

**`session_start`** — New session begins
- "Ah shit, here we go again."
- "Grove Street. Home."
- "Hello."
- "It's Carl. Carl Johnson."
- "It's CJ."
- "It's Carl."
- "Who's this?"
- "Yeah, why not."
- "I'm just a street criminal, what can I say."
- "Just cause I look fine, I ain't a bitch."

**`task_complete`** — Work finished
- "Ay, ain't this fun?"
- "Cool."
- "Do you know who I am, fool?"
- "I appreciate that, I like yours too."
- "It's been nice talking to you."
- "Okay."
- "Real funny."
- "Thanks."
- "Very intelligent."
- "Yeah."
- "You having fun yet, fool?"
- "Respect+"

**`task_error`** — Something went wrong
- "I don't need any of this shit."
- "I'm losing my will to live here."
- "Inbreeding makes you dumb, huh?"
- "Not this crap again."
- "Oops."
- "That's complete and utter bullshit."
- "Well, you sound like a moron."
- "You a professional moron?"
- "You had a bad week, huh?"
- "You idiot."
- "You moron."
- "You pathetic."

**`input_required`** — Needs your input
- "Aw, what's wrong?"
- "Can you do me a favour?"
- "Go away."
- "I ain't saying yeah and I ain't saying no."
- "So you think I'm a punk, do you?"
- "What you smoking?"
- "You better be drunk."
- "You OK?"

**`resource_limit`** — Running low on context
- "Dude, I ain't got time for this."
- "I'll see you later, man."
- "I'm sorry, man."
- "It's nothing personal, I'm just a criminal."
- "Oh, you obviously don't know who I am."
- "You could run or get a beating. Easy choice, huh?"
- "You picked the wrong fool to jack, homie."

**`user_spam`** — Calm down
- "Coffee with you? No thank you."
- "Come on dude, don't make me laugh."
- "Dude, please shut up."
- "Get the fuck out of here."
- "Go away."
- "Hey, get out of here."
- "I don't give a fuck."
- "Like I give a fuck what you think."
- "My God, you boring me. Go away."
- "No thank you."
- "Please go away."
- "Uh, na."
- "We don't want your services around here no more."
- "Whatever, dude."
- "Whatever you do, don't ever call me again."
- "You wanna get slapped, fool?"

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
grove-street stop               # Disable notifications (CJ takes a break)
grove-street resume             # Re-enable notifications (CJ comes back)
grove-street fix                # Validate and repair hook registrations
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
make dev        # build binary + compile/sign macOS overlay
./grove-street setup
```

Or manually:

```sh
go build -o grove-street ./cmd/grove-street
./grove-street setup
```

## Disclaimer

Grove Street is a fan project and is not affiliated with or endorsed by Rockstar Games or Take-Two Interactive. GTA San Andreas and all related characters, voice lines, and assets are property of their respective owners. This project is intended for personal, non-commercial use.

## License

MIT — applies to the source code only. Audio assets from GTA San Andreas are the property of Rockstar Games / Take-Two Interactive and are not covered by this license.

---

*"All you had to do was follow the damn train, CJ!"*
