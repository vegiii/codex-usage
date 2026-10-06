# codex-usage

A small terminal display for your Codex usage limits. Refreshes every 60 seconds.

```text
5 hour    █████████████████░░░  83% left   reset 3h 29m
Weekly    ███████████████████░  95% left   reset Sat 18:53

Refreshes in 57s
```

Bars and percentages show **quota remaining**, matching Codex `/status`.
The refresh countdown ticks every second; quota is fetched every 60 seconds.
Reset times use your local timezone.
Window names follow the durations Codex reports. Missing data is shown as
unavailable; failed refreshes keep the previous reading marked as stale.

## Requirements

- Codex CLI on `PATH`, signed in with your ChatGPT account (`codex login`).
- On Windows, use the native `codex.exe`; `.cmd`/`.bat` launchers are not supported.
- A terminal with Unicode and ANSI support, such as Konsole or Windows Terminal.
- Go 1.22 or newer **to build only**. No extra Go libraries or runtime are needed.

## Build

Run these commands from this repository with Go installed:

```bash
# Linux
CGO_ENABLED=0 go build -o bin/codex-usage .

# Windows (64-bit Intel/AMD), built on Linux
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o bin/codex-usage.exe .
```

For Windows on ARM, use `GOARCH=arm64` instead.
To build directly on Windows, run `go build -o bin/codex-usage.exe .`.

## Run

On Linux, install the built executable:

```bash
mkdir -p ~/.local/bin
install -m 755 bin/codex-usage ~/.local/bin/codex-usage
codex-usage
```

Make sure `~/.local/bin` is on your `PATH`.

On Windows, copy `bin/codex-usage.exe` into a folder on your user `Path`,
then open a new terminal and run `codex-usage`. You can also run it from its
folder with `./codex-usage.exe` in PowerShell.

Press **Ctrl+C** to exit. For a single reading:

```bash
codex-usage --once
```

Redirecting output also prints once, without terminal control sequences.
Rebuild and replace the executable after changing the source.

## How it works

Each refresh starts `codex app-server --stdio`, initializes the connection,
reads `account/rateLimits/read`, then stops that child process. Codex handles
authentication. The app does not read credential files or start model turns.
It shows the main `codex` quota bucket, not additional model-specific buckets.

The code is in `main.go`: `run` controls refreshing, `fetchLimits` communicates
with Codex, and `formatWindow` draws a usage bar and formats its reset time.

[Codex app-server documentation](https://developers.openai.com/codex/app-server)
