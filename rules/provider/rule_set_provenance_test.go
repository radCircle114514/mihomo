package provider

import (
	"reflect"
	"testing"

	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
)

type provenanceTunnel struct {
	P.Tunnel
	providers map[string]P.RuleProvider
}

func (t provenanceTunnel) RuleProviders() map[string]P.RuleProvider { return t.providers }

type nestedProvenanceProvider struct {
	P.RuleProvider
	matched bool
}

func (p nestedProvenanceProvider) Match(_ *C.Metadata, helper C.RuleMatchHelper) bool {
	if helper.MatchedProviders != nil {
		*helper.MatchedProviders = append(*helper.MatchedProviders, "nested")
	}
	return p.matched
}

func TestRuleSetMatchedProviders(t *testing.T) {
	previous := tunnel
	t.Cleanup(func() { tunnel = previous })
	tunnel = provenanceTunnel{providers: map[string]P.RuleProvider{
		"domains": NewInlineProvider("domains", P.Domain, []string{"example.com"}, nil),
	}}

	for _, tt := range []struct {
		name, host, provider, adapter string
		matched                       bool
		want                          []string
	}{
		{"match", "example.com", "domains", "DIRECT", true, []string{"existing", "domains"}},
		{"miss", "other.com", "domains", "DIRECT", false, []string{"existing"}},
		{"missing provider", "example.com", "missing", "", false, []string{"existing"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := NewRuleSet(tt.provider, "DIRECT", false, false)
			if err != nil {
				t.Fatal(err)
			}
			providers := []string{"existing"}
			matched, adapter := rule.Match(&C.Metadata{Host: tt.host}, C.RuleMatchHelper{MatchedProviders: &providers})
			if matched != tt.matched || adapter != tt.adapter || !reflect.DeepEqual(providers, tt.want) {
				t.Fatalf("got (%v, %q, %v), want (%v, %q, %v)", matched, adapter, providers, tt.matched, tt.adapter, tt.want)
			}
			matched, adapter = rule.Match(&C.Metadata{Host: tt.host}, C.RuleMatchHelper{})
			if matched != tt.matched || adapter != tt.adapter {
				t.Fatalf("without tracking got (%v, %q), want (%v, %q)", matched, adapter, tt.matched, tt.adapter)
			}
		})
	}
}

func TestRuleSetRollsBackFailedProvider(t *testing.T) {
	previous := tunnel
	t.Cleanup(func() { tunnel = previous })
	tunnel = provenanceTunnel{providers: map[string]P.RuleProvider{
		"success": nestedProvenanceProvider{matched: true},
		"failure": nestedProvenanceProvider{matched: false},
	}}
	for _, tt := range []struct {
		name    string
		matched bool
		want    []string
	}{
		{"success", true, []string{"existing", "nested", "success"}},
		{"failure", false, []string{"existing"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := NewRuleSet(tt.name, "DIRECT", false, false)
			if err != nil {
				t.Fatal(err)
			}
			providers := []string{"existing"}
			matched, _ := rule.Match(&C.Metadata{}, C.RuleMatchHelper{MatchedProviders: &providers})
			if matched != tt.matched || !reflect.DeepEqual(providers, tt.want) {
				t.Fatalf("got (%v, %v), want (%v, %v)", matched, providers, tt.matched, tt.want)
			}
		})
	}
}
