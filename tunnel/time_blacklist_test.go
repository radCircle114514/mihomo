package tunnel

import (
	"testing"
	"time"

	C "github.com/metacubex/mihomo/constant"
	P "github.com/metacubex/mihomo/constant/provider"
	RP "github.com/metacubex/mihomo/rules/provider"
)

type blacklistTestProxy struct {
	C.Proxy
	name      string
	proxyType C.AdapterType
}

func (p blacklistTestProxy) Name() string                     { return p.name }
func (p blacklistTestProxy) Type() C.AdapterType              { return p.proxyType }
func (p blacklistTestProxy) Unwrap(*C.Metadata, bool) C.Proxy { return nil }
func (p blacklistTestProxy) SupportUDP() bool                 { return true }

type blacklistTestRule struct {
	provider string
	target   string
	matched  bool
}

func (r blacklistTestRule) RuleType() C.RuleType    { return C.RuleSet }
func (r blacklistTestRule) Adapter() string         { return r.target }
func (r blacklistTestRule) Payload() string         { return r.provider }
func (r blacklistTestRule) ProviderNames() []string { return []string{r.provider} }
func (r blacklistTestRule) Match(_ *C.Metadata, helper C.RuleMatchHelper) (bool, string) {
	if r.matched && helper.MatchedProviders != nil {
		*helper.MatchedProviders = append(*helper.MatchedProviders, r.provider)
	}
	return r.matched, r.target
}

func TestTimeBlacklistRoutingAndReload(t *testing.T) {
	oldRules, oldSubRules, oldProviders, oldBlacklist, oldProxies, oldProxyProviders := rules, subRules, ruleProviders, timeBlacklist, proxies, providers
	t.Cleanup(func() {
		UpdateRulesWithTimeBlacklist(oldRules, oldSubRules, oldProviders, oldBlacklist)
		UpdateProxies(oldProxies, oldProxyProviders)
	})
	UpdateProxies(map[string]C.Proxy{
		"DIRECT": blacklistTestProxy{name: "DIRECT", proxyType: C.Direct},
		"REJECT": blacklistTestProxy{name: "REJECT", proxyType: C.Reject},
		"PASS":   blacklistTestProxy{name: "PASS", proxyType: C.Pass},
	}, nil)
	now := time.Now()
	minute := now.Hour()*60 + now.Minute()
	active := C.TimeWindow{Start: (minute + 1439) % 1440, End: (minute + 2) % 1440}
	configured := map[string]C.TimeWindow{"blocked": active}
	routing := []C.Rule{
		blacklistTestRule{provider: "blocked", target: "PASS", matched: true},
		blacklistTestRule{provider: "blocked", target: "DIRECT", matched: true},
	}
	UpdateRulesWithTimeBlacklist(routing, nil, nil, configured)
	for _, network := range []C.NetWork{C.TCP, C.UDP} {
		proxy, _, err := match(&C.Metadata{NetWork: network}, C.RuleMatchHelper{})
		if err != nil || proxy.Name() != "REJECT" {
			t.Fatalf("network %v: got proxy %v, error %v", network, proxy, err)
		}
	}
	UpdateRules(routing, nil, nil) // A reload without a blacklist must clear it.
	proxy, _, err := match(&C.Metadata{NetWork: C.TCP}, C.RuleMatchHelper{})
	if err != nil || proxy.Name() != "DIRECT" {
		t.Fatalf("after reload: got proxy %v, error %v", proxy, err)
	}
}

func TestTimeBlacklistWithRuleProvider(t *testing.T) {
	oldRules, oldSubRules, oldProviders, oldBlacklist, oldProxies, oldProxyProviders := rules, subRules, ruleProviders, timeBlacklist, proxies, providers
	RP.SetTunnel(Tunnel)
	t.Cleanup(func() {
		UpdateRulesWithTimeBlacklist(oldRules, oldSubRules, oldProviders, oldBlacklist)
		UpdateProxies(oldProxies, oldProxyProviders)
		RP.SetTunnel(nil)
	})
	UpdateProxies(map[string]C.Proxy{
		"DIRECT": blacklistTestProxy{name: "DIRECT", proxyType: C.Direct},
		"REJECT": blacklistTestProxy{name: "REJECT", proxyType: C.Reject},
	}, nil)
	provider := RP.NewInlineProvider("example", P.Domain, []string{"example.com"}, nil)
	rule, err := RP.NewRuleSet("example", "DIRECT", false, false)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	minute := now.Hour()*60 + now.Minute()
	window := C.TimeWindow{Start: (minute + 1439) % 1440, End: (minute + 2) % 1440}
	UpdateRulesWithTimeBlacklist([]C.Rule{rule}, nil, map[string]P.RuleProvider{"example": provider}, map[string]C.TimeWindow{"example": window})
	for _, tc := range []struct{ host, want string }{
		{"example.com", "REJECT"},
		{"other.com", "DIRECT"},
	} {
		proxy, _, err := match(&C.Metadata{Host: tc.host, NetWork: C.TCP}, C.RuleMatchHelper{})
		if err != nil || proxy.Name() != tc.want {
			t.Errorf("host %s: got proxy %v, error %v; want %s", tc.host, proxy, err, tc.want)
		}
	}
}
