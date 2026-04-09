# pihole-tui

A beautiful TUI (Terminal User Interface) for managing [Pi-hole](https://pi-hole.net/) remotely from your terminal.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and the Pi-hole v6 API.

## Features

- **Live Dashboard** — real-time stats with auto-refresh (queries, blocked %, clients)
- **Top Domains / Top Blocked** — bar charts of most queried and blocked domains
- **Query Log** — recent DNS queries with status (OK/blocked)
- **DNS Records** — view, add, and manage local DNS entries
- **Deny/Allow Lists** — manage blocked and allowed domains
- **Blocking Control** — enable/disable blocking (30s, 5m, indefinitely)
- **Gravity Update** — trigger blocklist updates

## Install

### Homebrew

```bash
brew install rodrigodealer/tap/pihole-tui
```

### Go

```bash
go install github.com/rodrigodealer/pihole-tui/cmd/pihole-tui@latest
```

### Binary

Download from [Releases](https://github.com/rodrigodealer/pihole-tui/releases).

## Configuration

Create `~/.config/pihole-tui/config.json`:

```json
{
  "host": "http://192.168.0.70",
  "password": "your-pihole-password"
}
```

## Usage

```bash
pihole-tui
```

### Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `↑/↓` or `k/j` | Navigate menu |
| `enter` | Select item |
| `r` | Refresh current view |
| `a` | Add domain (in list views) |
| `esc` | Back / Quit |
| `q` | Quit |

## Requirements

- Pi-hole v6+ with API enabled
- Network access to Pi-hole host

## License

MIT
