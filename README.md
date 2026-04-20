# rlp

A minimal one-line terminal radio player powered by [radio-browser.info](https://www.radio-browser.info/).

```
- NHK World Radio Japan @ Japan  128kbps
```

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea). Designed to live in a single tmux or Zellij pane.

## Features

- Search stations by **Genre**, **Country**, **Language**, or **Name**
- Live stream via `mpv`
- Volume control with `j` / `k` / cursor keys
- Connecting spinner and stream failure detection
- Country/Language list caching (24h TTL)
- tmux / Zellij `display-popup` integration

## Requirements

- `mpv`
- `pactl` (PipeWire / PulseAudio)
- tmux or Zellij (recommended)

## Install

```bash
go install github.com/256x/rlp@latest
```

Or build from source:

```bash
git clone https://github.com/256x/rlp
cd rlp
go build -o ~/.local/bin/rlp .
```

## Usage

```bash
rlp
```

Press `Space` to open the station selector. In tmux or Zellij, the selector opens as a floating popup centered on screen. In a standalone terminal, it opens inline.

## Key Bindings

**Player:**

| Key | Action |
|-----|--------|
| `Space` | Open station selector |
| `k` / `↑` | Volume +5% |
| `j` / `↓` | Volume −5% |
| `q` | Quit (stops playback) |

**Station selector:**

| Key | Action |
|-----|--------|
| `1` | Genre list |
| `2` | Country list |
| `3` | Language list |
| `/` | Name search |
| `←` / `→` | Cycle through modes |
| `↑` / `↓` | Navigate list |
| Type | Filter current list |
| `Enter` | Confirm selection |
| `Esc` | Go back / close |

**Station list:**

| Key | Action |
|-----|--------|
| `j` `k` `h` `l` / `↑` `↓` | Navigate |
| `Enter` | Play selected station |
| `Esc` / `Backspace` | Back to search |

## License

MIT
