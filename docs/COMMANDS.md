# Commands (v1)

All commands print JSON unless `--human` is set.

## Catalog

```
kage doctor
kage see [--monitor NAME|--all] [--window ADDRESS|CLASS|TITLE] [--annotate] [--path FILE] [--max-width N]
kage windows
kage monitors
kage focus --window ADDRESS|CLASS|TITLE
kage click --at X,Y [--button left|right|middle] [--window ADDRESS]
kage click --on ID_OR_QUERY [--snapshot ID]
kage type "text" [--window ...] [--clear]
kage press Return|Tab|Escape|BackSpace|space|Left|Right|Up|Down
kage hotkey "SUPER+Q" | "CTRL+C" | "ALT+F4"
kage dispatch <hyprctl dispatch args...>
kage mcp
kage install [grok|claude|cursor|codex]    # no arg → default_client (grok)
kage uninstall [grok|claude|cursor|codex]
```

## `kage doctor`

Prints a table: compositor, `WAYLAND_DISPLAY`, grim, hyprctl, wtype, ydotool, wl-copy, capture probe (actually runs grim), input backend, `default_client`, and whether that client's skill + MCP entry are present. Extra rows for other clients if their config dirs exist.

Exit codes:

| Code | Meaning |
|---|---|
| 0 | Capture works |
| 1 | Capture fails |
| 3 | Capture works, input unavailable (see-only is still useful) |

## `kage see`

Core command.

1. Read hyprctl JSON for monitors, clients, active window.
2. Capture PNG. Default is the focused monitor. `--window` uses that client's geometry.
3. Downscale only if `--max-width` is set (default 1920 on the long edge so vision stays cheap).
4. Optional `--annotate`: red box + number per mapped client.
5. Write PNG, print JSON.

```json
{
  "ok": true,
  "snapshot_id": "kage_20260904_181203_a1b2",
  "path": "/run/user/1000/kage/kage_20260904_181203_a1b2.png",
  "width": 1920,
  "height": 1080,
  "monitor": {"name":"DP-1","x":0,"y":0,"width":2560,"height":1440,"scale":1.5},
  "focused": {
    "address": "0x123",
    "class": "google-chrome",
    "title": "...",
    "at": [100, 80],
    "size": [1400, 900],
    "workspace": 1,
    "pid": 4321
  },
  "windows": [
    {
      "id": 1,
      "address": "0x123",
      "class": "...",
      "title": "...",
      "at": [100, 80],
      "size": [1400, 900],
      "workspace": 1,
      "monitor": "DP-1",
      "floating": false,
      "mapped": true,
      "focus": true
    }
  ],
  "coordinate_space": "global_compositor_pixels"
}
```

## Window matchers

Exact `address` (`0x...`), or first match of `class` (case-insensitive), or substring `title`. Ambiguous matches return the list and exit 2. Never pick silently.

## `kage click`

Exactly one of `--at X,Y` or `--on ID`.

`--at` is global compositor coords. Needs ydotool. If ydotoold is down, error with an install hint (`systemctl --user start ydotool.service` — not `ydotoold`). Do not fake it.

`--on N` is the annotated window id from the last see snapshot (or `--snapshot ID`). Focus that window, then click its center.

## `kage type` / `press` / `hotkey`

Focus the target window, then `wtype`. Fail if wtype is missing.

`kage hotkey SUPER+K` prefers `hyprctl dispatch sendshortcut` when Hyprland accepts it.

Input commands require the [allow gate](ARCHITECTURE.md#safety).

## `kage dispatch`

JSON-wrapped `hyprctl dispatch`. For moves, close, fullscreen, workspace. Not a substitute for click/type.

## `kage install`

No argument uses `default_client` from config, which is `grok` unless changed. See [CLIENTS.md](CLIENTS.md) for what each provider gets (skill path + MCP registration).
