# rlp User Guide

## Overview

rlp is a one-line terminal radio player. It is designed to run in a single tmux or Zellij pane and stay out of the way while streaming internet radio via `mpv`.

## Setup

### tmux pane (recommended)

Add a dedicated 1-line pane to your tmux layout and run `rlp` inside it. When you press `Space`, rlp opens a floating `display-popup` for station selection without leaving the pane.

### Zellij

Same as tmux — rlp detects the `ZELLIJ` environment variable and opens a floating pane via `zellij run --floating`.

### Standalone terminal

rlp also works outside tmux/Zellij. Pressing `Space` opens the station selector inline using the alternate screen buffer.

---

## Player Screen

The player screen occupies a single line:

```
[rlp] Station Name [XX]  |  Track Title
```

`[XX]` is the two-letter country abbreviation. The track title (ICY metadata) appears when the stream provides it.

| State | Display |
|-------|---------|
| No station selected | `[rlp] [space] select/search` |
| Connecting | Station name animates with a color gradient for ~4 seconds |
| Playing | Station name + track title with animated gradient |
| Stream ended / unreachable | Status message for 4 seconds |

### Volume Control

Adjusts the system default sink volume via `pactl`:

| Key | Action |
|-----|--------|
| `k` or `↑` | +5% |
| `j` or `↓` | −5% |

The current volume level is shown briefly as a status message.

### Quitting

Press `q` on the player screen. This kills `mpv` and exits rlp.

---

## Station Selector

Open with `Space`. The selector has four search modes:

| Key | Mode | Source |
|-----|------|--------|
| `1` | Genre | Local list (22 categories) |
| `2` | Country | radio-browser API (cached 24h) |
| `3` | Language | radio-browser API (cached 24h) |
| `/` | Name search | radio-browser API (live query) |

Switch modes with the number keys or `←` / `→` to cycle through them.

### Navigating a list (Genre / Country / Language)

- **Type** to filter in real-time — all characters including `j`, `k` go to the filter input
- **`↑` / `↓`** to move the cursor through the list
- **`Enter`** to load stations for the selected item
- **`Esc`** to close the selector

### Name search

- Type a station name and press **`Enter`** to search the API
- **`Esc`** to cancel

### List caching

Country and Language lists are cached to `~/.cache/rlp/` as JSON files with a 24-hour TTL. The second time you open either list, it loads instantly from disk.

---

## Station List

After selecting a genre / country / language / name query, up to 50 stations are shown ordered by popularity.

Each line shows the station name on the left and the country abbreviation on the right.

| Key | Action |
|-----|--------|
| `j` `l` / `↓` | Move down |
| `k` `h` / `↑` | Move up |
| `Enter` | Play selected station |
| `Esc` / `Backspace` | Back to search screen |

---

## Playback

rlp delegates audio to `mpv`:

- Launched with `--no-video --no-terminal --really-quiet`
- Runs in its own process session (`setsid`) so it survives popup close
- Stops automatically when the main rlp process exits
- If a stream becomes unreachable, `mpv` exits and rlp shows `stream ended or unreachable`

### State persistence

The last-played station is saved to `~/.cache/rlp/current.json`. The `mpv` process ID is tracked in `~/.cache/rlp/mpv.pid`. On restart, if `mpv` is still running, rlp displays the station as active.
