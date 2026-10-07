# Codex Usage

Compact KDE Plasma 6 widget showing Codex 5-hour and weekly quota remaining. Refreshes every 60 seconds.

## Requirements

- KDE Plasma 6 with `plasma5support` and `kpackagetool6`.
- Codex CLI on Plasma's `PATH`, signed in using `codex login`.
- Go 1.22 or newer to build.

## Build and install

```bash
cd ~/Documents/Git/codex-usage &&
CGO_ENABLED=0 go build -o bin/codex-usage . &&
mkdir -p ~/.local/bin &&
install -m 755 bin/codex-usage ~/.local/bin/codex-usage &&
kpackagetool6 --type Plasma/Applet --install plasmoid
```

Open **Add Widgets** and add **Codex Usage**. Pin the popup to keep it open when switching applications. Settings control the panel text size.

## Update

Close widget settings, then run:

```bash
cd ~/Documents/Git/codex-usage &&
CGO_ENABLED=0 go build -o bin/codex-usage . &&
install -m 755 bin/codex-usage ~/.local/bin/codex-usage &&
kpackagetool6 --type Plasma/Applet --upgrade plasmoid &&
systemctl --user restart plasma-plasmashell.service
```

The panel and desktop briefly disappear while Plasma reloads.

## Remove

Remove the widget from your panel or desktop, then run:

```bash
kpackagetool6 --type Plasma/Applet --remove local.codex.usage &&
rm ~/.local/bin/codex-usage
```
