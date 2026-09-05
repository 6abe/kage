# Kage

![Kage](assets/kage.jpg)

A desktop observer for **Omarchy Linux** (Arch + Hyprland + Wayland). Any coding agent that can run a CLI or speak MCP can drive this session. Default install target is **Grok Build** (`grok`).

Kage is Japanese for shadow. The binary is `kage`. Observe first. Act only when allowed.

## Loop

1. `kage see --annotate` writes a PNG and JSON (monitors, windows, focused client, path).
2. The agent reads the PNG.
3. `kage click` / `type` / `press` act on that snapshot.
4. `kage see` again to verify.

MCP is a thin stdio wrapper around the same CLI. Kage does not call an LLM and does not pick a model. `kage install` with no args wires Grok. Claude, Cursor, and Codex are first-class too.

JSON on stdout by default. `--human` is the only prose path. Screenshots stay files. Never base64.

## Install

```bash
go build -o ~/.local/bin/kage ./cmd/kage
kage doctor
kage install            # Grok skill + MCP (`kage mcp`)
kage install claude     # Claude equivalent
```

Needs `grim`, `hyprctl`, and `wtype` on PATH. Click needs `ydotool`; on Arch the user unit is `ydotool.service` (it runs `ydotoold`):

```bash
omarchy pkg add ydotool
systemctl --user start ydotool.service
```

Input (`click`, `type`, `press`, `hotkey`) is gated. One of `--yes`, `KAGE_ALLOW_INPUT=1`, or `allow_input = true` in `~/.config/kage/config.toml`. Observe commands are not gated.

## Why not Omarchy's screenshot command

Omarchy already has capture:

```bash
omarchy screenshot                                          # interactive (hangs an agent)
omarchy capture screenshot fullscreen save                  # focused monitor, prints a path
```

`fullscreen save` is the only agent-safe Omarchy grab. It does not list windows, annotate them, or click. Kage uses `grim` + `hyprctl` for the structured half. It does not call `omarchy screenshot` or `slurp`.

## Why not built-in computer use

| Tool | What it actually drives |
|---|---|
| Grok Build CLI | Shell, files, pasted images. No desktop tools. |
| Grok Bot | A **cloud** desktop, not this Hyprland session. |
| Cursor local Agent | In-app **browser** pane. |
| Cursor Cloud / worker | Isolated VM, or Linux **X11** worker. Not Wayland. |
| Claude Code / Codex | Shell + MCP. No Hyprland driver unless you add one. |

None of those click Firefox on this box. Kage is the Hyprland driver. Point Grok, Claude, Cursor, or Codex at the same binary.

Cousin on macOS: [Peekaboo](https://github.com/openclaw/Peekaboo). Same loop, different compositor.

## Docs

| File | Contents |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | CLI-first shape, stack, paths, safety |
| [docs/COMMANDS.md](docs/COMMANDS.md) | v1 command and JSON contracts |
| [docs/CLIENTS.md](docs/CLIENTS.md) | MCP, skill, `kage install` (default Grok) |
| [docs/ASK.md](docs/ASK.md) | Post-v1 Quickshell overlay: circle the screen and talk to Grok |

[AGENTS.md](AGENTS.md) is standing orders for this tree.
