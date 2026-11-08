package logic

import (
	"reflect"
	"testing"

	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/rules/common"
)

type provenanceRule struct {
	common.Base
	name    string
	adapter string
	matched bool
}

func (r provenanceRule) RuleType() C.RuleType { return C.RuleSet }
func (r provenanceRule) Adapter() string      { return r.adapter }
func (r provenanceRule) Payload() string      { return r.name }
func (r provenanceRule) Match(_ *C.Metadata, helper C.RuleMatchHelper) (bool, string) {
	// Simulate even a poorly behaved child that writes before returning false.
	if helper.MatchedProviders != nil && r.name != "" {
		*helper.MatchedProviders = append(*helper.MatchedProviders, r.name)
	}
	return r.matched, r.adapter
}

func leaf(name, adapter string, matched bool) C.Rule {
	return &provenanceRule{name: name, adapter: adapter, matched: matched}
}

func expr(kind C.RuleType, rules ...C.Rule) *Logic {
	return &Logic{ruleType: kind, adapter: "DIRECT", rules: rules}
}

func subrule(predicate C.Rule, children ...C.Rule) *Logic {
	return &Logic{ruleType: C.SubRules, adapter: "child", rules: []C.Rule{predicate}, subRules: map[string][]C.Rule{"child": children}}
}

func TestMatchedProviderProvenance(t *testing.T) {
	tests := []struct {
		name      string
		rule      C.Rule
		matched   bool
		adapter   string
		want      []string
		checkPass func(string) bool
	}{
		{"and success", expr(C.AND, leaf("a", "", true), leaf("b", "", true)), true, "DIRECT", []string{"existing", "a", "b"}, nil},
		{"and failure", expr(C.AND, leaf("a", "", true), leaf("b", "", false)), false, "DIRECT", []string{"existing"}, nil},
		{"or skips failed and branch", expr(C.OR, expr(C.AND, leaf("a", "", true), leaf("b", "", false)), leaf("c", "", true), leaf("unused", "", true)), true, "DIRECT", []string{"existing", "c"}, nil},
		{"or all fail", expr(C.OR, leaf("a", "", false), leaf("b", "", false)), false, "", []string{"existing"}, nil},
		{"not matching child", expr(C.NOT, leaf("a", "", true)), false, "", []string{"existing"}, nil},
		{"not nonmatching child", expr(C.NOT, leaf("a", "", false)), true, "DIRECT", []string{"existing"}, nil},
		{"double not", expr(C.NOT, expr(C.NOT, leaf("a", "", true))), true, "DIRECT", []string{"existing"}, nil},
		{"subrule success", subrule(leaf("gate", "", true), leaf("result", "DIRECT", true)), true, "DIRECT", []string{"existing", "gate", "result"}, nil},
		{"subrule gate fails", subrule(leaf("gate", "", false), leaf("unused", "DIRECT", true)), false, "", []string{"existing"}, nil},
		{"subrule has no match", subrule(leaf("gate", "", true), leaf("miss", "DIRECT", false)), false, "", []string{"existing"}, nil},
		{"subrule skips pass", subrule(leaf("gate", "", true), leaf("skip", "PASS-RULE", true), leaf("result", "DIRECT", true)), true, "DIRECT", []string{"existing", "gate", "result"}, nil},
		{"subrule skips custom pass", subrule(leaf("gate", "", true), leaf("skip", "custom", true), leaf("result", "DIRECT", true)), true, "DIRECT", []string{"existing", "gate", "result"}, func(a string) bool { return a == "custom" }},
		{"subrule only pass", subrule(leaf("gate", "", true), leaf("skip", "PASS-RULE", true)), false, "", []string{"existing"}, nil},
		{"nested or and not", expr(C.AND, leaf("a", "", true), expr(C.OR, expr(C.NOT, leaf("b", "", true)), leaf("c", "", true))), true, "DIRECT", []string{"existing", "a", "c"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			providers := []string{"existing"}
			matched, adapter := tt.rule.Match(&C.Metadata{}, C.RuleMatchHelper{MatchedProviders: &providers, CheckPassRule: tt.checkPass})
			if matched != tt.matched || adapter != tt.adapter || !reflect.DeepEqual(providers, tt.want) {
				t.Fatalf("got (%v, %q, %v), want (%v, %q, %v)", matched, adapter, providers, tt.matched, tt.adapter, tt.want)
			}
			matched, adapter = tt.rule.Match(&C.Metadata{}, C.RuleMatchHelper{CheckPassRule: tt.checkPass})
			if matched != tt.matched || adapter != tt.adapter {
				t.Fatalf("without tracking got (%v, %q), want (%v, %q)", matched, adapter, tt.matched, tt.adapter)
			}
		})
	}
}

func TestNilTrackingDoesNotAllocate(t *testing.T) {
	metadata := &C.Metadata{}
	for name, rule := range map[string]C.Rule{
		"logical": expr(C.AND, leaf("a", "", true), expr(C.OR, leaf("b", "", false), leaf("c", "", true))),
		"subrule": subrule(leaf("gate", "", true), leaf("skip", "PASS-RULE", true), leaf("result", "DIRECT", true)),
	} {
		t.Run(name, func(t *testing.T) {
			if allocations := testing.AllocsPerRun(100, func() { rule.Match(metadata, C.RuleMatchHelper{}) }); allocations != 0 {
				t.Fatalf("nil tracking allocated %v times per match", allocations)
			}
		})
	}
}

func TestNestedSubruleDiscardsPassBranches(t *testing.T) {
	groups := map[string][]C.Rule{}
	nested := &Logic{ruleType: C.SubRules, adapter: "nested", rules: []C.Rule{leaf("inner-gate", "", true)}, subRules: groups}
	groups["nested"] = []C.Rule{leaf("skip", "PASS-RULE", true), leaf("winner", "DIRECT", true)}
	groups["outer"] = []C.Rule{nested}
	outer := &Logic{ruleType: C.SubRules, adapter: "outer", rules: []C.Rule{leaf("outer-gate", "", true)}, subRules: groups}
	providers := []string{"existing"}
	matched, adapter := outer.Match(&C.Metadata{}, C.RuleMatchHelper{MatchedProviders: &providers})
	// The existing nested traversal evaluates the winning subrule twice.
	want := []string{"existing", "outer-gate", "inner-gate", "winner", "winner"}
	if !matched || adapter != "DIRECT" || !reflect.DeepEqual(providers, want) {
		t.Fatalf("got (%v, %q, %v), want (true, DIRECT, %v)", matched, adapter, providers, want)
	}
}
