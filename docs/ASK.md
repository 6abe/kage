# Ask (Quickshell plugin, post-v1)

A human overlay for talking about what is on screen. Not part of kage v1.

The kage binary stays a CLI. This lives inside `omarchy-shell` as a third-party Quickshell plugin, same host as the clipboard and emoji overlays. Capture is `kage see`. Speech is voxtype. The model is Grok. The overlay is the conversation.

v1 still does not grow a GUI or call an LLM. [ARCHITECTURE.md](ARCHITECTURE.md) non-goals stand.

## Rule

The screenshot is the first message, not the product. You hit a chord because something on screen is wrong. You circle it and talk. Grok answers in the overlay. You talk again. You do not bounce to a TUI, and you do not start a new session every time you say "no, the other one."

## Why a plugin, not a kage window

Omarchy already has a long-running Quickshell process. Summoning an overlay is an IPC call into that process, not a cold `quickshell -p`. Themes, layer-shell, and keybinds already work that way (`omarchy.clipboard`, `omarchy.emojis`, `omarchy.image-picker`).

A GTK/Qt app next to kage would miss all of that and fight the compositor. A `kage ask` TUI would hide the screen you are asking about.

## Loop

1. Chord. Overlay summons. `kage see` writes a PNG. Mic starts.
2. Draw on the image. Talk. Transcript lands in an editable composer.
3. Send. Grok sees the annotated PNG plus the text.
4. Reply streams under the image.
5. Follow-ups are voice or typed text against the same picture. Recapture or a new circle is another user message with a new image.
6. Esc hides. The session stays. The same chord shows the overlay again.

Preview before send is mandatory. Auto-send from a half sentence and a half-drawn scribble is the failure mode.

## Plugin shape

Install as a third-party plugin, not a first-party `omarchy.*` id.

```
plugin/                         # in this repo, until it earns its own git
  manifest.json
  Overlay.qml                   # kinds: overlay
  Service.qml                   # kinds: service
  BarWidget.qml                 # later; kinds: bar-widget
```

`omarchy plugin add` wants `manifest.json` at the git root. This repo is the Go CLI, so do not put the manifest at `/`. Ship the plugin from `plugin/`. Install by copy or symlink into `~/.config/omarchy/plugins/kage.ask/`, then:

```
omarchy-shell shell rescanPlugins
omarchy plugin enable kage.ask
```

A later split into its own git repo can use `omarchy plugin add <url>` without changing the id.

### Manifest

```json
{
  "schemaVersion": 1,
  "id": "kage.ask",
  "name": "Kage ask",
  "version": "0.1.0",
  "author": "kage",
  "license": "MIT",
  "description": "Circle something on screen and talk to Grok about it",
  "kinds": ["service", "overlay"],
  "keepLoaded": true,
  "entryPoints": {
    "service": "Service.qml",
    "overlay": "Overlay.qml"
  }
}
```

`keepLoaded: true` so hide does not kill the Grok process or the transcript. Same reason `omarchy.clipboard` and `omarchy.image-picker` stay mounted.

Kinds:

| Kind | Job |
|---|---|
| `overlay` | Fullscreen conversation. Image, marks, chat, composer, rec light. |
| `service` | Holds the Grok session, draft dir, and voxtype. Lives even when the overlay is hidden. |
| `bar-widget` | Later. Idle / recording / unread-reply badge. Not in the first cut. |

Overlay entry point follows the clipboard/emoji contract: `opened`, `open(payloadJson)`, `close()`, and hide via `shell.hide(id)`. Theme off `qs.Commons` menu tokens so it looks like the rest of the shell.

### IPC

The chord summons. It does not toggle. Toggle would close the conversation when you meant to recapture.

```
omarchy-shell shell summon kage.ask '{"capture":"monitor"}'
omarchy-shell shell summon kage.ask '{"capture":"window"}'
omarchy-shell shell summon kage.ask '{"fresh":true,"capture":"monitor"}'
omarchy-shell shell hide kage.ask
omarchy-shell shell call kage.ask grab '{"capture":"monitor"}'
```

| Payload | Effect |
|---|---|
| `{capture:"monitor"}` | Focused monitor. Same default as `kage see`. |
| `{capture:"window"}` | Focused window geometry. |
| `{fresh:true}` | New issue. New Grok session id. Previous thread stays on disk. |
| overlay already open + `grab` | Recapture. New image message in the same session. Mic starts. |

Esc calls `close()` then `shell.hide`. Session process stays in the service.

Do not use slurp. Do not call `omarchy screenshot`. Capture is `kage see` with hyprctl geometry.

## Bindings

Do not steal Print. Omarchy already binds:

- `PRINT` screenshot (tensaku-edit after save)
- `SUPER + PRINT` color picker
- `SUPER + CTRL + PRINT` OCR

Suggested chords in `~/.config/hypr/bindings.lua`. Unbind first if the user already mapped them.

```lua
o.bind("SUPER + SHIFT + A", "Ask kage",
  "omarchy-shell shell summon kage.ask '{\"capture\":\"monitor\"}'")
o.bind("SUPER + SHIFT + W", "Ask kage (window)",
  "omarchy-shell shell summon kage.ask '{\"capture\":\"window\"}'")
o.bind("SUPER + SHIFT + N", "Ask kage (new)",
  "omarchy-shell shell summon kage.ask '{\"fresh\":true,\"capture\":\"monitor\"}'")
```

`--all` monitors is a later payload. A stitched desktop is a bad default for vision.

## Overlay

Keep the grab on screen the whole time. Chat under it, or in a side column if the image is wide.

```
[ annotated screenshot ]

Grok: that's the Hyprland bind conflict on Super+Shift+S
You:  no, the notification under it. why does it keep firing
Grok: ...

[ rec ]  [ recapture ]  [ ask | do ]  [ send ]
```

The composer is always there. First send is image plus transcript. Later turns are usually text. Drawing or recapture attaches a new PNG to that turn.

Esc hides. Chord shows the same thread. `fresh:true` starts a new issue.

Do not reuse tensaku-edit. That editor is for saving a screenshot, not composing a prompt.

## Capture and marks

The overlay shells out to kage. It does not call grim itself.

```
kage see [--window ADDRESS] --path $DRAFT/raw.png
```

Parse the JSON. Keep `snapshot_id`, `path`, `width`, `height`, monitor scale, focused client, window list. After send, Grok can `kage see` again and click in compositor pixels instead of guessing from the picture.

Draw on a copy. Burn marks into `annotated.png` so vision sees the circle. Also write `markers.json` in compositor pixels:

```json
{"marks":[{"kind":"circle","x":1420,"y":380,"r":64}]}
```

Pixels for the model's eyes. Numbers for `kage click --at`. Freehand highlighter is enough for the first cut. Arrows and numbered pins wait.

## Voice

Voxtype is already on this box. Do not add a second STT stack.

Default voxtype output is `type` through ydotool. That would dump words into whatever was focused. The overlay owns the text field. Record with `voxtype record start` / `stop` (or `pw-record` plus `voxtype transcribe` if the daemon will not return the string). Put the transcript in the composer. The user edits before send.

First grab: mic starts with the screenshot. Circle while you talk.

After that: tap rec or hold-to-talk. Always-on while the overlay is up will pick up you muttering at the code. Visible rec light. Esc or rec-again cancels. Hide or send stops the mic.

## Draft files

Screenshots stay files. Never base64 in JSON, MCP, or the Grok prompt payload if a path works.

```
$XDG_RUNTIME_DIR/kage/ask/<issue-id>/
  raw.png
  annotated.png
  snapshot.json          # kage see stdout
  markers.json
  prompt.txt             # composer, last edit before send
```

Mode 0700, same as other kage snapshots. One directory per issue, not per turn. Later turns add `annotated-2.png` (or a `turns/` subdir) rather than overwriting the first grab.

## Session

The overlay is the Grok client. Do not inject into whatever coding TUI is focused. A screenshot of a lock screen does not belong in the kage implementer thread.

Default: one sticky session per issue, held by `Service.qml`.

| Chord | Session |
|---|---|
| Summon / grab | Resume the current issue. Create on first send. |
| `fresh:true` | New session id. Store it as current. |

Persist the current id at `~/.config/kage/ask-session` (plain text UUID). Grok already knows `grok --resume <id>`. The same thread can be opened later in a TUI with that id. The overlay does not need a private transcript format.

Preferred transport: `grok agent stdio` (or `serve` on localhost) started by the service, then ACP `session/new` / `session/load` / `session/prompt` with text plus image path. Stream `session/update` into the overlay.

Bootstrap without ACP: `grok -p --resume $ID --prompt-json ... --output-format streaming-json`. That starts a process per send. Fine to prove the loop. Wrong once follow-ups are the point.

cwd: `$HOME` unless the focused window is a terminal with a git cwd, in which case that cwd is an explicit "this project" toggle, off by default. Do not guess the repo every time.

New-per-send is not the default. It is `fresh:true`.

## Ask vs Do

Two modes on the same thread.

**Ask** (default). Grok can `kage see`, read files, search. Click, type, press, hotkey stay behind kage's input gate and should be denied from this session unless the user flips the switch.

**Do.** Input is allowed the usual way (`allow_input` in config, `KAGE_ALLOW_INPUT=1`, or `--yes` if the plugin passes it). Recapture after the agent acts. Same rule as see-after-click.

Without the switch, "why is this button grey" becomes a yolo operator on a polkit dialog.

## Recapture

The picture goes stale the moment anything moves.

1. Same pixels: talk only. No new grab.
2. Same grab, new mark: draw, send. New image message, same session.
3. Screen changed: recapture. That PNG is the latest attachment. Old images stay in the thread.

If mode is Do, the service recaptures when a kage input tool call finishes, with a short "updated" flash so the user knows the picture moved.

## What Grok receives

Each send is one user turn:

- annotated PNG (path)
- composer text
- optional marker JSON inlined as text
- snapshot JSON path, so the agent can use window ids from that grab

The kage skill already tells the agent to read the PNG at `path` and not guess coordinates. This plugin should pass the same paths rather than inventing a parallel protocol.

Grok `--prompt-json` is the headless shape. ACP `session/prompt` content blocks are the long-lived shape. Do not base64 the screenshot into either.

## Non-goals (this plugin)

- Replacing Omarchy screenshot, tensaku-edit, or slurp
- Screen recording (Omarchy already has that)
- Shipping inside the kage binary
- First-party `omarchy.ask` id (this is kage's plugin, installed by the user)
- Driving Claude/Cursor/Codex from the overlay in the first cut (kage CLI/MCP still can; the overlay talks to Grok)
- Always-on microphone
- Region picker

## Ship order

1. Plugin skeleton: manifest, overlay `open`/`close`, Hypr summon bind, `kage see` of the focused monitor, image on screen.
2. One pen. Voxtype into the composer. Editable text. Esc hides.
3. Send = create or resume a Grok session. Stream the reply in the overlay. Ask mode only.
4. Recapture and window payload.
5. `fresh:true`. Ask vs Do. Recapture after Do.

Bar widget, marker JSON, and "this project" cwd come after that loop works on this machine.
