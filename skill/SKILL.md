---
name: kage
description: See and control the Omarchy/Hyprland desktop. Use when the user asks what is on screen, to click/type in a GUI app, to verify a visual change, to screenshot a window, or to drive a browser/app that has no CLI. Linux/Wayland only.
---

# Kage

Kage lets the agent see and (when allowed) control this Omarchy Hyprland/Wayland desktop. Observe first. Act only when input is allowed.

Prefer the `kage` CLI via the shell if MCP is missing.

## Never

- Never run `omarchy screenshot`.
- Never run `omarchy capture screenshot region` (it waits on slurp).
- Never run `slurp`.
- If `kage doctor` fails, tell the user which package to add. Do not improvise with scrot or import.

## Loop

1. `kage see --annotate`
2. Read the PNG at `path` from the JSON.
3. Decide.
4. `kage click` / `type` / `press` using coordinates or window ids from that snapshot.
5. `kage see` again.

After any mutation, see again. Do not assume the click worked.

"What does this app look like?" is see only.

Multi-monitor: see the monitor that has the focused window unless the user names one.

## Input

Click, type, press, and hotkey are gated. If input is refused, ask the user to allow it (`allow_input` or `--yes`). Do not retry in a loop.

If click fails with `ydotool not running`, start the Arch user unit: `systemctl --user start ydotool.service`. The daemon binary is `ydotoold`; the unit is not `ydotoold.service`.

## Commands

- `kage doctor`
- `kage see [--annotate] [--monitor NAME|--window Q|--all] [--path FILE]`
- `kage windows` / `kage monitors`
- `kage focus --window ADDRESS|CLASS|TITLE`
- `kage click --at X,Y` or `kage click --on ID`
- `kage type TEXT` / `kage press KEY` / `kage hotkey CHORD`

Coordinates are compositor logical pixels from the last see snapshot.
