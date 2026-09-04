# Agent notes

This repo is a spec. There is no binary yet.

## Before writing code

Read [README.md](README.md), then the doc that matches the work:

- Shape, stack, safety, non-goals → [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Flags and JSON → [docs/COMMANDS.md](docs/COMMANDS.md)
- MCP and client install → [docs/CLIENTS.md](docs/CLIENTS.md)

Probe this machine before inventing paths: `hyprctl -j monitors`, `which grim wtype ydotool`, `echo $HYPRLAND_INSTANCE_SIGNATURE`.

## Hard rules

CLI is the source of truth. MCP wraps it.

Screenshots are files. Never base64 in JSON or MCP.

No `slurp`. No `omarchy screenshot`. Capture with grim and hyprctl geometry.

Input is gated. Observe is not.

xdotool is not a Wayland fallback.

## Implement in this order

1. `hypr` package + `kage windows` / `kage monitors` / `kage doctor` (no grim yet).
2. `kage see` writes PNG + JSON. Run it on this machine.
3. `--annotate` boxes + ids.
4. `kage focus`, `type`, `press` via wtype. Gate with allow_input.
5. `kage click --at` via ydotool if present; otherwise error with install hint.
6. `kage mcp` stdio.
7. `kage install` (default grok) writes skill + mcp config. Other clients via `kage install claude|cursor|codex`.
8. Smoke: `kage see --annotate --path /tmp/kage-smoke.png` exists and is a PNG.

## Done (when we build)

- `go test ./...` passes.
- `kage doctor` on this Omarchy box reports capture ok.
- See-smoke PNG exists.
- `kage mcp` answers initialize + tools/list.
- `kage install` (no args) leaves the Grok skill + MCP entry on disk. `kage install claude` does the Claude equivalent.
