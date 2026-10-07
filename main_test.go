package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	used := 17.0
	invalid := 101.0
	reset := int64(2000000000)
	current := &limits{
		Primary:   &window{UsedPercent: &used, WindowDurationMins: 10080, ResetsAt: &reset},
		Secondary: &window{UsedPercent: &invalid, WindowDurationMins: 300},
	}
	got := normalize(current)
	if got.Weekly == nil || *got.Weekly.Remaining != 83 || *got.Weekly.ResetsAt != reset {
		t.Fatalf("weekly quota was not mapped by duration: %+v", got)
	}
	if got.FiveHour == nil || got.FiveHour.Remaining != nil {
		t.Fatal("invalid usage should be unavailable")
	}
	current.Primary.WindowDurationMins = 60
	current.Secondary = nil
	got = normalize(current)
	if got.FiveHour != nil || got.Weekly != nil {
		t.Fatal("unexpected windows must not be mislabeled")
	}
}

func fakeCodex(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "codex")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFetchLimits(t *testing.T) {
	for _, tc := range []struct{ name, response, wantError string }{
		{"codex bucket", `{"rateLimitsByLimitId":{"codex":{"primary":{"usedPercent":17,"windowDurationMins":300}}}}`, ""},
		{"legacy fallback", `{"rateLimits":{"primary":{"usedPercent":17,"windowDurationMins":300}}}`, ""},
		{"wrong bucket", `{"rateLimits":{"limitId":"other"}}`, "no Codex quota"},
		{"missing windows", `{"rateLimits":{}}`, "no quota windows"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			binary := fakeCodex(t, "read -r init\nprintf '%s\\n' '{\"method\":\"notice\"}' '{\"id\":1,\"result\":{}}'\nread -r initialized\nread -r request\nprintf '%s\\n' '{\"id\":2,\"result\":"+tc.response+"}'\n")
			result, err := fetchLimits(context.Background(), binary)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("got %v", err)
				}
			} else if err != nil || result.Primary == nil || *result.Primary.UsedPercent != 17 {
				t.Fatalf("got %+v, %v", result, err)
			}
		})
	}
}

func TestFetchFailures(t *testing.T) {
	binary := fakeCodex(t, "exit 0\n")
	if _, err := fetchLimits(context.Background(), binary); err == nil {
		t.Fatal("expected error after premature exit")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fetchLimits(ctx, binary); err == nil {
		t.Fatal("expected cancellation error")
	}
	if _, err := fetchLimits(context.Background(), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected missing executable error")
	}
}

func TestFetchTimeout(t *testing.T) {
	binary := fakeCodex(t, "read -r request\nread -r request\n")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := fetchLimits(ctx, binary); err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected bounded request failure, got %v", err)
	}
}
