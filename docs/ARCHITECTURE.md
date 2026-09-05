# Architecture

## Rule

The CLI is the product. MCP tools are 1:1 wrappers around CLI commands. No MCP-only behavior.

Default stdout is JSON. `--human` is the only prose path. Errors are JSON on stderr and a non-zero exit:

```json
{"ok":false,"error":"ydotool not running","hint":"omarchy pkg add ydotool && systemctl --user start ydotool.service"}
```

## Why files, not base64

MCP clients truncate tool results. Grok's cap is about 20k bytes (`GROK_MAX_MCP_OUTPUT_BYTES`). A screenshot as base64 dies there.

`kage see` writes a PNG and returns `path`, `width`, `height`. The agent reads the file.

## Stack

Go 1.22+. One binary.

Call compositor tools. Do not reimplement Wayland screenshot protocols in v1.

| Job | Tool |
|---|---|
| Windows, monitors, workspaces, focus | `hyprctl -j monitors`, `clients`, `activewindow`, `workspaces` |
| Screenshot | `grim -o <monitor>` or `grim -g "x,y,w,h"` from client geometry |
| Clipboard | `wl-copy` / `wl-paste` |
| Keyboard to focused window | `wtype` |
| Global pointer + keys | `ydotool` against `ydotoold` if present. Arch unit is `ydotool.service` (`ExecStart=/usr/bin/ydotoold`). `systemctl --user start ydotoold` fails: no such unit. |
| Window actions | `hyprctl dispatch focuswindow address:...` and friends |
| AT-SPI widget tree | Optional after v1. Skip if the app has no tree |

`kage doctor` checks `WAYLAND_DISPLAY`, `HYPRLAND_INSTANCE_SIGNATURE`, grim, hyprctl, wtype, ydotool. Missing packages get an `omarchy pkg add` hint.

xdotool is not a fallback. It misses on Wayland.

## Layout (planned)

```
cmd/kage/main.go
internal/hypr/      # hyprctl JSON
internal/capture/   # grim
internal/input/     # wtype + ydotool
internal/see/       # snapshot + optional annotate
internal/mcp/       # stdio server
internal/config/
internal/doctor/
skill/SKILL.md      # canonical copy; kage install copies it to the client's skill dir
```

## Paths

| What | Where |
|---|---|
| Config | `~/.config/kage/config.toml` (`default_client = "grok"`) |
| Snapshots | `$XDG_RUNTIME_DIR/kage/` (fallback `/tmp/kage-$UID/`), mode 0700 |
| Logs | `~/.local/state/kage/kage.log` |
| Install | `go build -o kage ./cmd/kage` into `~/.local/bin` |

AUR / `omarchy pkg` is not v1.

## Safety

Observe is default. It needs no extra permission.

Input (`click`, `type`, `press`, `hotkey`) is gated. One of:

- `--yes` on the command
- `KAGE_ALLOW_INPUT=1`
- `allow_input = true` in config

Refuse with a clear error if none of those are set.

Never log keystrokes. Never dump the clipboard unless asked. Never click without a target (element id, query, or `x,y`).

Do not run interactive pickers. `slurp`, `omarchy screenshot`, and `omarchy capture screenshot region` wait for a human and hang the agent.

## Coordinates

Compositor logical pixels, the same space grim and hyprctl use. If monitor scale is 1.5, say so in JSON. Click coords stay in that space.

`--annotate` draws 1-based ids on each mapped client's rect. Those ids match `windows[].id`.

## Input backends

| Need | Backend |
|---|---|
| Focus a window | `hyprctl dispatch` |
| Type into focused window | `wtype` |
| Click at global `x,y` | `ydotool` |
| Hyprland chords (`SUPER+Q`) | `hyprctl dispatch sendshortcut` when it works, else ydotool |

v1 "element id" is the annotated **window** id from the last see snapshot, not a widget inside the app. AT-SPI widgets come later.

## Non-goals (v1)

- Menu-bar or overlay GUI
- `kage agent "..."` LLM loop (the calling agent owns the model)
- X11 / xdotool as the primary path
- Browser CDP (separate tool)
- Screen recording (Omarchy already has `omarchy screenrecord`)
- Perfect accessibility trees
- Homebrew, npm wrapper, Mac notarization

## After v1

The human "circle this and talk" UI is an Omarchy Quickshell plugin (`kage.ask`), not a window inside the kage binary and not `kage agent`. Spec: [ASK.md](ASK.md).
