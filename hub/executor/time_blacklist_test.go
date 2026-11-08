package executor

import (
	"strings"
	"testing"
)

func TestParseTimeBlacklist(t *testing.T) {
	const base = `rule-providers:
  example:
    type: inline
    behavior: domain
    payload: [example.com]
rules:
  - RULE-SET,example,DIRECT
`
	cfg, err := ParseWithBytes([]byte(base + "time-blacklist:\n  example:\n    start: '02:00'\n    end: '06:00'\n"))
	if err != nil {
		t.Fatal(err)
	}
	window, ok := cfg.TimeBlacklist["example"]
	if !ok || window.Start != 120 || window.End != 360 {
		t.Fatalf("unexpected blacklist: %+v", cfg.TimeBlacklist)
	}
	_, err = ParseWithBytes([]byte(base + "time-blacklist:\n  missing:\n    start: '02:00'\n    end: '06:00'\n"))
	if err == nil || !strings.Contains(err.Error(), "rule provider \"missing\" not found") {
		t.Fatalf("expected missing provider error, got %v", err)
	}
}
