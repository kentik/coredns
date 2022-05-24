package ksynth

import (
	"context"
	"net"
	"strings"

	"github.com/coredns/coredns/request"

	"github.com/kentik/odyssey/pkg/synthetics"
)

// For any unknown dns requests which come in, see about creating them as tests in kentik.
func (ks *KsynthListen) handleUnknowns() {
	log.Infof("Starting unknown handler")
	// If there are kentik api credencials, use them here to manage tests.
	if ks.options.kentikEmail != "" && ks.options.kentikApiToken != "" {
		ks.client = synthetics.NewClient(ks.options.kentikEmail, ks.options.kentikApiToken, NewLogger())
		log.Infof("Kentik API Client Initialized")
	}

	seen := map[string]bool{}  // Track which requests we've seen, only work on new ones.
	resolver := &net.Resolver{ // Uses the server's local resolver to get IPs. Better way?
		PreferGo:     false,
		StrictErrors: false,
	}

	ctx := context.Background()
	for {
		r, ok := <-ks.unknowns
		if !ok {
			return
		}

		// If we don't have a client here, don't try to do any processing.
		if ks.client == nil {
			continue
		}

		// Otherwise, now lets see about creating one of these in kentik.
		state := request.Request{Req: r}
		qname := state.Name()
		if _, ok := seen[qname]; ok { // We've already seen this.
			continue
		}
		seen[qname] = true

		log.Infof("Looking up %s", qname)
		if ans, err := resolver.LookupHost(ctx, qname); err == nil {
			if len(ans) == 0 {
				log.Infof("Could not resolve %s to any IPs")
				continue
			}

			go ks.createSynthTest(ctx, qname, ans) // Go ahead and create the tests here.
		} else {
			log.Errorf("Could not resolve %s to any IPs -- %v", err)
		}
	}
}

func (ks *KsynthListen) createSynthTest(ctx context.Context, qname string, ips []string) {
	// Now Look at which agents are closest to the requester's IP.
	// For now, we're just using a hard coded list of agents for every one.
	// Trim qname trailing .
	if strings.HasSuffix(qname, ".") {
		qname = qname[0 : len(qname)-1]
	}

	pingPeriod := 60 // Seconds
	pingDelay := 0   // Milliseconds
	protocol := "icmp"
	count := 5
	agentIDs := []string{"2287"}

	// Consider just ipv4 for now?
	nip := []string{}
	for _, ip := range ips {
		ipr := net.ParseIP(ip)
		if ipr == nil {
			continue
		}
		if ipr.To4() == nil { // Is IPv6
			continue
		}
		nip = append(nip, ip)
	}

	if len(nip) == 0 {
		return
	}

	// Now pick some and set up some tests for these guys. Just do a basic ping test for now.
	log.Infof("Creating test for %s", qname)
	test, err := ks.client.CreateTest(ctx, &synthetics.Test{
		Name: qname,
		Type: synthetics.TestTypeIP,
		Settings: &synthetics.TestSettings{
			Period: int(pingPeriod),
			Tasks:  []string{synthetics.TestTaskPing},
			Ping: &synthetics.PingTest{
				Protocol: protocol,
				Count:    count,
				Delay:    pingDelay,
				Timeout:  1000, // MS
			},
			Trace: &synthetics.TraceTest{
				Count:    3,
				Protocol: "udp",
				Timeout:  22500, // MS
				Limit:    20,
			},
			IP: &synthetics.TestIP{
				Targets: nip,
			},
			AgentIDs: agentIDs,
		},
	})

	if err != nil {
		log.Errorf("Cannot create test for %s: %v", qname, err)
	} else {
		log.Infof("Created test %s for %s", test.ID, qname)
	}
}
