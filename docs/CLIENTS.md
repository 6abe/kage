# Agent clients

Kage does not call an LLM. The agent that invoked it does. The interface is the CLI plus MCP stdio. Any client that can run a command or speak MCP can drive it.

Default install target is **Grok Build** (`grok`). Other clients are first-class, not afterthoughts.

```toml
# ~/.config/kage/config.toml
default_client = "grok"   # grok | claude | cursor | codex
```

`kage install` with no args uses `default_client`.

## MCP (every client)

`kage mcp` speaks MCP over stdio. Tools mirror the CLI. Keep the list small:

- `kage_doctor`
- `kage_see` (monitor, window, annotate, max_width)
- `kage_windows`
- `kage_focus`
- `kage_click` (at xor on)
- `kage_type`
- `kage_press`
- `kage_hotkey`

Each tool description must tell the model: call `kage_see` first, read the PNG at `path`, then click using coordinates or window id from that snapshot. Do not guess coordinates.

Tools return CLI JSON text only. Do not attach an image content block. JSON `path` is enough. The skill tells the agent to read the file.

Do not return screenshot bytes in the MCP payload. Some clients truncate hard (Grok at ~20k bytes via `GROK_MAX_MCP_OUTPUT_BYTES`). Files survive that. Base64 does not.

Point any MCP client at:

```
kage mcp
```

## Skill (every client)

One canonical `skill/SKILL.md` in this repo. Provider-neutral wording: "the agent", not "Grok". Install copies it to that client's skill directory.

Frontmatter:

```yaml
---
name: kage
description: See and control the Omarchy/Hyprland desktop. Use when the user asks what is on screen, to click/type in a GUI app, to verify a visual change, to screenshot a window, or to drive a browser/app that has no CLI. Linux/Wayland only.
---
```

Rules the skill must state:

- Prefer the `kage` CLI via the shell if MCP is missing.
- Never run `omarchy screenshot`, `omarchy capture screenshot region`, or `slurp`.
- Loop: `kage see --annotate` → read the PNG → decide → click/type/press → `kage see` again.
- After any mutation, see again. Do not assume the click worked.
- If `kage doctor` fails, tell the user which package to add. Do not improvise with scrot or import.
- If input is refused, ask the user to allow it (`allow_input` or `--yes`).
- "What does this app look like?" is see only.
- Multi-monitor: see the monitor that has the focused window unless the user names one.

## `kage install`

```
kage install              # default_client, which is grok unless config says otherwise
kage install grok
kage install claude
kage install cursor
kage install codex
kage uninstall [client]   # default_client if omitted
```

Each installer copies the skill and registers MCP for that client.

### Grok Build (default)

1. Copy skill to `~/.grok/skills/kage/`
2. `grok mcp add kage -- kage mcp` (or write `~/.grok/config.toml` if `grok mcp` is missing)

```toml
[mcp_servers.kage]
command = "kage"
args = ["mcp"]
enabled = true
```

Inside the TUI, `/mcps` then `r` refreshes after config edits.

### Claude Code

1. Copy skill to `~/.claude/skills/kage/`
2. `claude mcp add kage -- kage mcp`

### Cursor

1. Copy skill to `~/.cursor/skills/kage/`
2. Add stdio server `kage` → `kage mcp` in `~/.cursor/mcp.json`

Cursor's built-in Browser tool is a webview. Kage is the Wayland desktop. They do not replace each other.

### Codex CLI

1. Copy skill to `~/.codex/skills/kage/` (or the current Codex skills path if it moved)
2. Register `kage mcp` in Codex MCP config

## After install

Ask the agent:

```
what's on my screen?
```

It should `kage see --annotate`, read the PNG, and answer. Clicks stay off until input is allowed.
