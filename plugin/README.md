# kage.ask

Omarchy Quickshell overlay for kage. Summon it, `kage see` grabs the focused monitor, and the PNG stays on screen. Esc hides; `keepLoaded` leaves the plugin mounted.

This repo is the kage CLI, so the plugin lives in `plugin/` rather than at the git root. Do not add a GUI to the `kage` binary.

## Install

Copy or symlink into the third-party plugin dir, then rescan and enable:

```bash
mkdir -p ~/.config/omarchy/plugins
ln -sfn /path/to/kage/plugin ~/.config/omarchy/plugins/kage.ask
# or: cp -a /path/to/kage/plugin ~/.config/omarchy/plugins/kage.ask
omarchy-shell shell rescanPlugins
omarchy plugin enable kage.ask
```

`omarchy plugin add` wants `manifest.json` at a git root. Until this plugin is its own repo, install by copy or symlink.

## Summon

Summon, do not toggle. Toggle would close the overlay when you meant to recapture.

```bash
omarchy-shell shell summon kage.ask '{"capture":"monitor"}'
omarchy-shell shell hide kage.ask
```

Esc hides the overlay. The service stays loaded. Send (button or Ctrl+Enter) creates or resumes a Grok session; the reply streams into the overlay. Voice does not auto-send.

## Bind

Do not rebind `PRINT`, `SUPER+PRINT`, or `SUPER+CTRL+PRINT`. Suggested chord in `~/.config/hypr/bindings.lua`:

```lua
o.bind("SUPER + SHIFT + A", "Ask kage",
  "omarchy-shell shell summon kage.ask '{\"capture\":\"monitor\"}'")
```

Unbind first if that chord is already mapped.

## Capture

The overlay shells out to `kage see --path …`. It does not call grim, slurp, or `omarchy screenshot`. Snapshots land under `$XDG_RUNTIME_DIR/kage/ask/` at mode 0700.

Send writes `prompt.txt`, then runs `grok --prompt-json --output-format streaming-json` with `--session-id` (first turn) or `--resume` (later). cwd is `$HOME`. Ask mode uses `--permission-mode dontAsk` and denies kage click/type/press/hotkey. The current session UUID is `~/.config/kage/ask-session` (mode 0600). Screenshots stay files; nothing is base64.
