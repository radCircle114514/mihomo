package logic_test

import (
	"reflect"
	"testing"

	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
	"github.com/metacubex/mihomo/rules/logic"
	RP "github.com/metacubex/mihomo/rules/provider"
)

type ruleProviderTunnel struct {
	P.Tunnel
	providers map[string]P.RuleProvider
}

func (t ruleProviderTunnel) RuleProviders() map[string]P.RuleProvider { return t.providers }

func TestParsedProviderProvenance(t *testing.T) {
	RP.SetTunnel(ruleProviderTunnel{providers: map[string]P.RuleProvider{
		"first":  RP.NewInlineProvider("first", P.Domain, []string{"example.com"}, nil),
		"miss":   RP.NewInlineProvider("miss", P.Domain, []string{"other.com"}, nil),
		"winner": RP.NewInlineProvider("winner", P.Domain, []string{"example.com"}, nil),
	}})
	t.Cleanup(func() { RP.SetTunnel(nil) })
	rule, err := logic.NewOR("((AND,((RULE-SET,first),(RULE-SET,miss))),(RULE-SET,winner))", "DIRECT", ParseRule)
	if err != nil {
		t.Fatal(err)
	}
	var providers []string
	matched, adapter := rule.Match(&C.Metadata{Host: "example.com"}, C.RuleMatchHelper{MatchedProviders: &providers})
	if !matched || adapter != "DIRECT" || !reflect.DeepEqual(providers, []string{"winner"}) {
		t.Fatalf("got (%v, %q, %v), want (true, DIRECT, [winner])", matched, adapter, providers)
	}
}
