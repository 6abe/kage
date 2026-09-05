# Kage

![Kage](assets/kage.jpg)

A Peekaboo-style desktop observer for **Omarchy Linux** (Arch + Hyprland + Wayland). Any coding agent that can run a CLI or speak MCP can drive it. Default install target is **Grok Build** (`grok`).

Kage is Japanese for shadow. The binary is `kage`. Observe first. Act only when allowed.

This repo is the spec. Code comes later.

## What it does

A coding agent can already run a shell and read a PNG. It cannot list Hyprland windows, grab a monitor without hanging on `slurp`, or click/type in a GUI app.

Kage is that missing loop:

1. `kage see` writes a screenshot and JSON (monitors, windows, focused client, path).
2. The agent reads the PNG.
3. `kage click` / `type` / `press` act on that snapshot.
4. `kage see` again to verify.

MCP is a thin stdio wrapper around the same CLI. Kage does not ship an LLM loop and does not pick a model. `kage install` wires the calling agent. With no args, that agent is Grok.

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

## Status

Documentation only. No binary yet.

## Docs

| File | Contents |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | CLI-first shape, stack, paths, safety |
| [docs/COMMANDS.md](docs/COMMANDS.md) | v1 command and JSON contracts |
| [docs/CLIENTS.md](docs/CLIENTS.md) | MCP, skill, `kage install` (default Grok) |

[AGENTS.md](AGENTS.md) is standing orders for implementing from this spec.
