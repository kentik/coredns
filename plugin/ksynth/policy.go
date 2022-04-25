package ksynth

import (
	"fmt"

	"github.com/coredns/coredns/plugin"
)

const (
	LowestAvgPingTime = "avg_ping_time"
	LowestMaxPingTime = "max_ping_time"
	MinPacketLoss     = "min_packet_loss"

	MAX_LOSS = 110.
)

type optimizer func([]Update) map[string]Update

func AvgPingTime(w []Update) map[string]Update {
	hosts := map[string]Update{}

	for _, h := range w {
		hostname := plugin.Name(h.Host).Normalize()
		if _, ok := hosts[hostname]; !ok { // Nothing here for this host, just add it in.
			hosts[hostname] = h
			continue
		}

		// See if we have a better value.
		if h.PingTimeAvg < hosts[hostname].PingTimeAvg {
			hosts[hostname] = h
		}
	}

	return hosts
}

func MaxPingTime(w []Update) map[string]Update {
	hosts := map[string]Update{}

	for _, h := range w {
		hostname := plugin.Name(h.Host).Normalize()
		if _, ok := hosts[hostname]; !ok { // Nothing here for this host, just add it in.
			hosts[hostname] = h
			continue
		}

		// See if we have a better value.
		if h.PingTimeMax < hosts[hostname].PingTimeMax {
			hosts[hostname] = h
		}
	}

	return hosts
}

// Here we have to track packet loss outselves because it doesn't come in pre-computed.
func PacketLoss(w []Update) map[string]Update {
	hosts := map[string]Update{}
	lossPct := make([]float64, len(w))
	lossPctHost := map[string]float64{}
	for i, h := range w {
		if h.PingSent > 0 {
			lossPct[i] = (float64(h.PingLost) / float64(h.PingSent)) * 100.
		} else {
			lossPct[i] = MAX_LOSS
		}
	}

	for i, h := range w {
		hostname := plugin.Name(h.Host).Normalize()
		if _, ok := hosts[hostname]; !ok { // Nothing here for this host, just add it in.
			hosts[hostname] = h
			lossPctHost[hostname] = lossPct[i]
			continue
		}

		// See if we have a better value.
		if lossPct[i] < lossPctHost[hostname] {
			hosts[hostname] = h
			lossPctHost[hostname] = lossPct[i]
		}
	}

	return hosts
}

func GetPolicy(pol string) (optimizer, error) {
	switch pol {
	case LowestAvgPingTime:
		return AvgPingTime, nil
	case LowestMaxPingTime:
		return MaxPingTime, nil
	case MinPacketLoss:
		return PacketLoss, nil
	default:
		return nil, fmt.Errorf("Invalid optimizing policy: %s", pol)
	}
}
