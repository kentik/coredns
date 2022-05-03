package ksynth

import (
	"testing"

	"github.com/coredns/coredns/plugin/pkg/fall"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
)

func TestOptimize(t *testing.T) {
	h := Ksynth{
		Next: test.NextHandler(dns.RcodeNameError, nil),
		KsynthListen: &KsynthListen{
			hmap:    newMap(),
			options: newOptions(),
			updates: map[string]*Update{},
		},
		Fall: fall.Zero,
	}
	h.optimizer, _ = GetPolicy(DefaultPolicy)
	_, updates := h.parse(ksynthExample)

	if len(updates) != 2 {
		t.Errorf("Expected 2 ips seen in updates, instead %d %d", len(updates), len(ksynthExample))
		return
	}

	if len(updates["localhost"].TargetIPs) != 2 {
		t.Errorf("Expected 2 ips for localhost")
		return
	}
}
