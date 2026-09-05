# kage.ask

Omarchy Quickshell overlay for kage. Summon it, `kage see` grabs the focused monitor (or the focused window with `{"capture":"window"}`), and the PNG stays on screen. Esc hides; `keepLoaded` leaves the plugin mounted. Recapture grabs again in the same session.

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

Summon, do not toggle. Toggle would close the overlay when you meant to recapture. A second summon while the overlay is already open grabs again.

```bash
omarchy-shell shell summon kage.ask '{"capture":"monitor"}'
omarchy-shell shell summon kage.ask '{"capture":"window"}'
omarchy-shell shell summon kage.ask '{"fresh":true,"capture":"monitor"}'
omarchy-shell shell call kage.ask grab '{"capture":"monitor"}'
omarchy-shell shell call kage.ask grab '{"capture":"window"}'
omarchy-shell shell hide kage.ask
```

`{"capture":"window"}` runs `kage see --window` on the focused client. `grab` recaptures while the overlay is open: the mic starts again, the new PNG is the next user image, and the same Grok session continues. Old images stay on disk and in the thread.

`{"fresh":true}` starts a new Grok session id, writes it to `~/.config/kage/ask-session`, and leaves the previous thread on disk. It does not delete the old session.

Esc hides the overlay. The service stays loaded. Send (button or Ctrl+Enter) creates or resumes a Grok session; the reply streams into the overlay. Voice does not auto-send.

## Bind

Do not rebind `PRINT`, `SUPER+PRINT`, or `SUPER+CTRL+PRINT`. Suggested chords in `~/.config/hypr/bindings.lua`:

```lua
o.bind("SUPER + SHIFT + A", "Ask kage",
  "omarchy-shell shell summon kage.ask '{\"capture\":\"monitor\"}'")
o.bind("SUPER + SHIFT + W", "Ask kage (window)",
  "omarchy-shell shell summon kage.ask '{\"capture\":\"window\"}'")
o.bind("SUPER + SHIFT + N", "Ask kage (new)",
  "omarchy-shell shell summon kage.ask '{\"fresh\":true,\"capture\":\"monitor\"}'")
```

Unbind first if that chord is already mapped.

## Capture

The overlay shells out to `kage see --path …` (and `kage see --window ADDRESS --path …` for a focused-window grab). It does not call grim, slurp, or `omarchy screenshot`. Snapshots land under `$XDG_RUNTIME_DIR/kage/ask/` at mode 0700. Recapture writes `raw-2.png` (and later numbered files) so earlier grabs are not overwritten.

Send writes `prompt.txt`, then runs `grok --prompt-json --output-format streaming-json` with `--session-id` (first turn) or `--resume` (later). cwd is `$HOME`. Ask is the default: `--permission-mode dontAsk` and denies kage click/type/press/hotkey. Do allows those tools only when kage's input gate is already open (`allow_input = true` in `~/.config/kage/config.toml` or `KAGE_ALLOW_INPUT=1`). Missing gate shows an error in the overlay; the plugin does not pass `--yes` to bypass it. After a successful kage input tool in Do, the overlay recaptures and flashes `updated`. The current session UUID is `~/.config/kage/ask-session` (mode 0600). Screenshots stay files; nothing is base64.
