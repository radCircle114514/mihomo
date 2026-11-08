package config

import (
	"strings"
	"testing"

	P "github.com/metacubex/mihomo/constant/provider"
)

func TestTimeBlacklistConfig(t *testing.T) {
	raw, err := UnmarshalRawConfig([]byte("time-blacklist:\n  example:\n    start: '02:00'\n    end: '06:00'\n"))
	if err != nil {
		t.Fatal(err)
	}
	providers := map[string]P.RuleProvider{"example": nil}
	windows, err := parseTimeBlacklist(raw.TimeBlacklist, providers)
	if err != nil {
		t.Fatal(err)
	}
	if got := windows["example"]; got.Start != 120 || got.End != 360 {
		t.Fatalf("unexpected window: %+v", got)
	}
	if _, err := parseTimeBlacklist(raw.TimeBlacklist, nil); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected missing provider error, got %v", err)
	}
	raw.TimeBlacklist["example"] = RawTimeWindow{Start: "bad", End: "06:00"}
	if _, err := parseTimeBlacklist(raw.TimeBlacklist, providers); err == nil {
		t.Fatal("expected invalid window error")
	}
}
