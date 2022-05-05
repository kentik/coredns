package ksynth

import (
	"fmt"
	"time"

	"github.com/coredns/coredns/plugin"
)

const (
	LowestAvgPingTime = "avg_ping_time"
	LowestMaxPingTime = "max_ping_time"
	MinPacketLoss     = "min_packet_loss"
	AllUp             = "all_up"

	MAX_LOSS = 110.

	MaxTimeSeen = -1 * 60 * time.Second
)

type optimizer func([]*Update, *KsynthListen) map[string]*Update

func common(w []*Update, ks *KsynthListen, pol func([]*Update, map[string]*Update)) map[string]*Update {
	now := time.Now()
	maxTime := now.Add(MaxTimeSeen)
	ks.RLock()

	// Make a copy of the updates map.
	hosts := map[string]*Update{}
	for h, u := range ks.updates {
		hosts[h] = u
	}
	ks.RUnlock() // Now we have a local copy to work on, don't need to worry about locking.

	for i, h := range w {
		if h == nil || !h.IsUp() {
			w[i] = nil
			continue
		}
		h.Init()

		hostname := plugin.Name(h.Host).Normalize()
		h.Seen = now
		if _, ok := hosts[hostname]; !ok { // Nothing here for this host, just add it in.
			hosts[hostname] = h
			w[i] = nil
			continue
		}

		// If the old entry for this event is too old, replace it with the new one.
		if hosts[hostname].Seen.Before(maxTime) {
			hosts[hostname] = h
			w[i] = nil
			continue
		}
	}

	// Now actually apply the policy to this set of updates.
	// w at this time should be only new entries who haven't been automatically added to the update map.
	pol(w, hosts)

	for _, h := range hosts {
		h.Finalize()
	}

	return hosts
}

func AvgPingTime(w []*Update, ks *KsynthListen) map[string]*Update {
	avg := func(updates []*Update, all map[string]*Update) {
		for _, h := range updates {
			if h == nil {
				continue // h already handled outside.
			}

			// See if we have a better value.
			hostname := plugin.Name(h.Host).Normalize()
			if h.PingTimeAvg < all[hostname].PingTimeAvg {
				all[hostname] = h
			}
		}
	}

	return common(w, ks, avg)
}

func AllUpPolicy(w []*Update, ks *KsynthListen) map[string]*Update {
	all := func(updates []*Update, all map[string]*Update) {
		for _, h := range updates {
			if h == nil {
				continue // h already handled outside.
			}

			// If host is up, add to list.
			hostname := plugin.Name(h.Host).Normalize()
			all[hostname].TargetIPs[h.IP.String()] = h.IP
		}
	}

	return common(w, ks, all)
}

func MaxPingTime(w []*Update, ks *KsynthListen) map[string]*Update {
	max := func(updates []*Update, all map[string]*Update) {
		for _, h := range updates {
			if h == nil {
				continue // h already handled outside.
			}

			// See if we have a better value.
			hostname := plugin.Name(h.Host).Normalize()
			if h.PingTimeMax < all[hostname].PingTimeMax {
				all[hostname] = h
			}
		}
	}

	return common(w, ks, max)
}

// Here we have to track packet loss outselves because it doesn't come in pre-computed.
func PacketLoss(w []*Update, ks *KsynthListen) map[string]*Update {
	loss := func(updates []*Update, all map[string]*Update) {
		for _, h := range updates {
			if h == nil {
				continue // h already handled outside.
			}

			if h.PingSent > 0 {
				h.PacketLost = (float64(h.PingLost) / float64(h.PingSent)) * 100.
			} else {
				h.PacketLost = MAX_LOSS
			}
		}

		for _, h := range updates {
			if h == nil {
				continue // h already handled outside.
			}

			hostname := plugin.Name(h.Host).Normalize()
			if all[hostname] == nil || h.PacketLost < all[hostname].PacketLost {
				all[hostname] = h
			}
		}
	}

	return common(w, ks, loss)
}

func GetPolicy(pol string) (optimizer, error) {
	switch pol {
	case LowestAvgPingTime:
		return AvgPingTime, nil
	case LowestMaxPingTime:
		return MaxPingTime, nil
	case MinPacketLoss:
		return PacketLoss, nil
	case AllUp:
		return AllUpPolicy, nil
	default:
		return nil, fmt.Errorf("Invalid optimizing policy: %s", pol)
	}
}
