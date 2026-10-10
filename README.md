# Codex Usage

KDE Plasma 6 widget showing remaining Codex 5-hour and weekly quotas and their reset times.

I built this for my own use, but you’re welcome to use it and adapt it to your needs.

## Requirements

- KDE Plasma 6 with `plasma5support` and `kpackagetool6`.
- Codex CLI on Plasma's `PATH`, signed in using `codex login`.
- Go 1.22+ and Git.

## Install

```bash
git clone https://github.com/vegiii/codex-usage.git ~/Documents/Git/codex-usage &&
cd ~/Documents/Git/codex-usage &&
CGO_ENABLED=0 go build -o bin/codex-usage . &&
mkdir -p ~/.local/bin &&
install -m 755 bin/codex-usage ~/.local/bin/codex-usage &&
kpackagetool6 --type Plasma/Applet --install plasmoid
```

Open **Add Widgets** and add **Codex Usage**.

## Update

Close widget settings first. This restarts Plasma to load the update.

```bash
cd ~/Documents/Git/codex-usage &&
git pull --ff-only &&
CGO_ENABLED=0 go build -o bin/codex-usage . &&
install -m 755 bin/codex-usage ~/.local/bin/codex-usage &&
kpackagetool6 --type Plasma/Applet --upgrade plasmoid &&
systemctl --user restart plasma-plasmashell.service
```

## License

[MIT](LICENSE).
