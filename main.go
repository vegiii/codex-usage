package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

const requestTimeout = 15 * time.Second

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

type usageWindow struct {
	Remaining *float64 `json:"remaining"`
	ResetsAt  *int64   `json:"resetsAt"`
}

type usage struct {
	FiveHour *usageWindow `json:"fiveHour"`
	Weekly   *usageWindow `json:"weekly"`
}

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: codex-usage")
		os.Exit(2)
	}
	current, err := fetchLimits(context.Background(), "codex")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(normalize(current)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Match reported durations so unexpected quota windows are never mislabeled.
func normalize(current *limits) usage {
	var result usage
	for _, w := range []*window{current.Primary, current.Secondary} {
		if w == nil {
			continue
		}
		value := &usageWindow{ResetsAt: w.ResetsAt}
		if w.UsedPercent != nil && *w.UsedPercent >= 0 && *w.UsedPercent <= 100 {
			remaining := 100 - *w.UsedPercent
			value.Remaining = &remaining
		}
		switch w.WindowDurationMins {
		case 300:
			result.FiveHour = value
		case 10080:
			result.Weekly = value
		}
	}
	return result
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
		"clientInfo": map[string]string{"name": "codex_usage", "version": "1.1.1"},
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
