package ksynth

import (
	"testing"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/plugin/pkg/fall"
)

func TestHostsParse(t *testing.T) {
	tests := []struct {
		inputFileRules      string
		shouldErr           bool
		expectedListen      string
		expectedOrigins     []string
		expectedFallthrough fall.F
	}{
		{
			`hosts
`,
			false, "127.0.0.1:8080", nil, fall.Zero,
		},
		{
			`hosts {
                          listen 127.0.0.1:8081
                        }`,
			false, "127.0.0.1:8081", nil, fall.Zero,
		},
		{
			`hosts miek.nl.`,
			false, "127.0.0.1:8080", []string{"miek.nl."}, fall.Zero,
		},
		{
			`hosts miek.nl. pun.gent.`,
			false, "127.0.0.1:8080", []string{"miek.nl.", "pun.gent."}, fall.Zero,
		},
		{
			`hosts {
				fallthrough
			}`,
			false, "127.0.0.1:8080", nil, fall.Root,
		},
		{
			`hosts {
				fallthrough
			}`,
			false, "127.0.0.1:8080", nil, fall.Root,
		},
		{
			`hosts miek.nl. {
				fallthrough
			}`,
			false, "127.0.0.1:8080", []string{"miek.nl."}, fall.Root,
		},
		{
			`hosts miek.nl 10.0.0.9/8 {
				fallthrough
			}`,
			false, "127.0.0.1:8080", []string{"miek.nl.", "10.in-addr.arpa."}, fall.Root,
		},
		{
			`hosts {
				fallthrough
			}
			hosts {
				fallthrough
			}`,
			true, "127.0.0.1:8080", nil, fall.Root,
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.inputFileRules)
		h, err := ksynthParse(c)

		if err == nil && test.shouldErr {
			t.Fatalf("Test %d expected errors, but got no error", i)
		} else if err != nil && !test.shouldErr {
			t.Fatalf("Test %d expected no errors, but got '%v'", i, err)
		} else if !test.shouldErr {
			if h.options.listen != test.expectedListen {
				t.Fatalf("Test %d expected %v, got %v", i, test.expectedListen, h.options.listen)
			}
		} else {
			if !h.Fall.Equal(test.expectedFallthrough) {
				t.Fatalf("Test %d expected fallthrough of %v, got %v", i, test.expectedFallthrough, h.Fall)
			}
			if len(h.Origins) != len(test.expectedOrigins) {
				t.Fatalf("Test %d expected %v, got %v", i, test.expectedOrigins, h.Origins)
			}
			for j, name := range test.expectedOrigins {
				if h.Origins[j] != name {
					t.Fatalf("Test %d expected %v for %d th zone, got %v", i, name, j, h.Origins[j])
				}
			}
		}
	}
}
