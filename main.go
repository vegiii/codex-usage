package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"
)

const (
	refreshInterval = 60 * time.Second
	requestTimeout  = 15 * time.Second
	barWidth        = 20
)

// Codex reports quota used and the next reset time for each window.
type window struct {
	UsedPercent        *float64 `json:"usedPercent"`
	WindowDurationMins int      `json:"windowDurationMins"`
	ResetsAt           *int64   `json:"resetsAt"`
}

type limits struct {
	LimitID   string  `json:"limitId"`
	Primary   *window `json:"primary"`
	Secondary *window `json:"secondary"`
}

type snapshot struct {
	RateLimits          *limits           `json:"rateLimits"`
	RateLimitsByLimitID map[string]limits `json:"rateLimitsByLimitId"`
}

func main() {
	once := flag.Bool("once", false, "print usage once and exit")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: codex-usage [-once]")
		os.Exit(2)
	}
	if err := run(*once); err != nil {
		fmt.Fprintln(os.Stderr, "codex-usage:", cleanText(err.Error()))
		os.Exit(1)
	}
}

func run(once bool) error {
	binary, err := exec.LookPath("codex")
	if err != nil {
		return errors.New("Codex CLI not found on PATH; install Codex and run codex login")
	}
	if strings.HasSuffix(strings.ToLower(binary), ".cmd") || strings.HasSuffix(strings.ToLower(binary), ".bat") {
		return errors.New("use the native Codex CLI (codex.exe) on PATH instead of a .cmd/.bat launcher")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	// Redirected output is a single plain-text snapshot, suitable for scripts.
	info, err := os.Stdout.Stat()
	if err != nil {
		return err
	}
	once = once || info.Mode()&os.ModeCharDevice == 0
	if !once {
		restore, err := prepareTerminal()
		if err != nil {
			return err
		}
		defer restore()
		// Restore the terminal on normal exit and Ctrl+C.
		fmt.Print("\x1b[?1049h\x1b[?25l")
		defer fmt.Print("\x1b[?25h\x1b[?1049l")
		fmt.Print("Loading Codex usage...\r\n")
	}
	var previous *limits
	var updated time.Time
	for {
		current, fetchErr := fetchLimits(ctx, binary)
		if ctx.Err() != nil {
			return nil
		}
		if once && fetchErr != nil {
			return fetchErr
		}
		if fetchErr == nil {
			previous, updated = current, time.Now()
		}
		if !once {
			fmt.Print("\x1b[H\x1b[J")
		}
		if previous != nil {
			output := formatLimits(previous, time.Now())
			if !once {
				output = strings.ReplaceAll(output, "\n", "\r\n")
			}
			fmt.Print(output)
		}
		if once {
			return nil
		}
		if fetchErr != nil {
			if previous != nil {
				fmt.Printf("\r\nStale data (last updated %s).\r\n", updated.Format("Mon 15:04"))
			}
			fmt.Printf("\r\n%s\r\n", cleanText(fetchErr.Error()))
		}
		fmt.Print("\r\n")
		if !waitForRefresh(ctx) {
			return nil
		}
	}
}

// Update only the footer each second; quota requests remain a minute apart.
func waitForRefresh(ctx context.Context) bool {
	deadline := time.Now().Add(refreshInterval)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		seconds := int(math.Ceil(time.Until(deadline).Seconds()))
		if seconds <= 0 {
			fmt.Print("\r\x1b[2KRefreshing...")
			return true
		}
		fmt.Printf("\r\x1b[2KRefreshes in %ds", seconds)
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
		}
	}
}

// Codex owns authentication; this program never reads credential files.
func fetchLimits(parent context.Context, binary string) (*limits, error) {
	ctx, cancel := context.WithTimeout(parent, requestTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "app-server", "--stdio")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start Codex: %w", err)
	}
	defer func() {
		stdin.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()
	encoder, decoder := json.NewEncoder(stdin), json.NewDecoder(stdout)
	request := func(id int, method string, params any) (json.RawMessage, error) {
		if err := encoder.Encode(map[string]any{"id": id, "method": method, "params": params}); err != nil {
			return nil, fmt.Errorf("send request to Codex: %w", err)
		}
		for {
			var response struct {
				ID     *int            `json:"id"`
				Result json.RawMessage `json:"result"`
				Error  *struct {
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := decoder.Decode(&response); err != nil {
				if ctx.Err() != nil {
					return nil, errors.New("Codex request cancelled or timed out (15s)")
				}
				if errors.Is(err, io.EOF) {
					return nil, errors.New("Codex closed unexpectedly; check codex login status and codex app-server --help")
				}
				return nil, fmt.Errorf("read Codex response: %w", err)
			}
			// Notifications can arrive between replies; match the request ID.
			if response.ID == nil || *response.ID != id {
				continue
			}
			if response.Error != nil {
				return nil, fmt.Errorf("Codex: %s", response.Error.Message)
			}
			return response.Result, nil
		}
	}
	_, err = request(1, "initialize", map[string]any{
		"clientInfo": map[string]string{"name": "codex_usage", "version": "0.1.0"},
	})
	if err != nil {
		return nil, err
	}
	if err := encoder.Encode(map[string]string{"method": "initialized"}); err != nil {
		return nil, err
	}
	data, err := request(2, "account/rateLimits/read", nil)
	if err != nil {
		return nil, err
	}
	var result snapshot
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("decode Codex limits: %w", err)
	}
	current, found := result.RateLimitsByLimitID["codex"]
	if !found {
		if result.RateLimits == nil || (result.RateLimits.LimitID != "" && result.RateLimits.LimitID != "codex") {
			return nil, errors.New("no Codex quota available; sign in with your ChatGPT account using codex login")
		}
		current = *result.RateLimits
	}
	if current.Primary == nil && current.Secondary == nil {
		return nil, errors.New("Codex returned no quota windows")
	}
	return &current, nil
}

func formatLimits(current *limits, now time.Time) string {
	return formatWindow(current.Primary, "Primary", now) + "\n" +
		formatWindow(current.Secondary, "Secondary", now) + "\n"
}

func formatWindow(w *window, fallback string, now time.Time) string {
	if w == nil {
		return fmt.Sprintf("%-9s unavailable", fallback)
	}
	label := fallback
	switch w.WindowDurationMins {
	case 300:
		label = "5 hour"
	case 10080:
		label = "Weekly"
	default:
		if w.WindowDurationMins > 0 {
			label = fmt.Sprintf("%d min", w.WindowDurationMins)
		}
	}
	if w.UsedPercent == nil || *w.UsedPercent < 0 || *w.UsedPercent > 100 {
		return fmt.Sprintf("%-9s usage unavailable", label)
	}
	remaining := 100 - *w.UsedPercent
	filled := int(math.Round(remaining * barWidth / 100))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	reset := "unknown"
	if w.ResetsAt != nil && *w.ResetsAt > 0 {
		when := time.Unix(*w.ResetsAt, 0).In(now.Location())
		switch {
		case !when.After(now):
			reset = "pending refresh"
		case w.WindowDurationMins == 300:
			minutes := int(math.Ceil(when.Sub(now).Minutes()))
			reset = fmt.Sprintf("%dh %02dm", minutes/60, minutes%60)
		default:
			reset = when.Format("Mon 15:04")
		}
	}
	return fmt.Sprintf("%-9s %s %3.0f%% left   reset %s", label, bar, remaining, reset)
}

// Keep backend error messages on one line without terminal control characters.
func cleanText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 || (r >= 127 && r <= 159) {
			return ' '
		}
		return r
	}, s)
}
