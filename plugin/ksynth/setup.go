package ksynth

import (
	"strconv"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
)

var log = clog.NewWithPlugin("ksynth")

func init() { plugin.Register("ksynth", setup) }

func setup(c *caddy.Controller) error {
	h, err := ksynthParse(c)
	if err != nil {
		return plugin.Error("ksynth", err)
	}

	c.OnStartup(func() error {
		go h.listen()
		return nil
	})

	c.OnShutdown(func() error {
		return nil
	})

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		h.Next = next
		return h
	})

	return nil
}

func ksynthParse(c *caddy.Controller) (Ksynth, error) {
	h := Ksynth{
		KsynthListen: &KsynthListen{
			hmap:    newMap(),
			options: newOptions(),
		},
	}

	i := 0
	for c.Next() {
		if i > 0 {
			return h, plugin.ErrOnce
		}
		i++

		args := c.RemainingArgs()

		h.Origins = plugin.OriginsFromArgsOrServerBlock(args, c.ServerBlockKeys)

		for c.NextBlock() {
			switch c.Val() {
			case "fallthrough":
				h.Fall.SetZonesFromArgs(c.RemainingArgs())
			case "no_reverse":
				h.options.autoReverse = false
			case "ttl":
				remaining := c.RemainingArgs()
				if len(remaining) < 1 {
					return h, c.Errf("ttl needs a time in second")
				}
				ttl, err := strconv.Atoi(remaining[0])
				if err != nil {
					return h, c.Errf("ttl needs a number of second")
				}
				if ttl <= 0 || ttl > 65535 {
					return h, c.Errf("ttl provided is invalid")
				}
				h.options.ttl = uint32(ttl)
			case "listen":
				listen := c.RemainingArgs()
				if len(listen) != 1 {
					return h, c.Errf("listen needs an argument")
				}
				h.options.listen = listen[0]
			case "path":
				path := c.RemainingArgs()
				if len(path) != 1 {
					return h, c.Errf("path needs an argument")
				}
				h.options.path = path[0]
			case "policy":
				policy := c.RemainingArgs()
				if len(policy) != 1 {
					return h, c.Errf("policy needs an argument")
				}
				pol, err := GetPolicy(policy[0])
				if err != nil {
					return h, err
				}
				h.optimizer = pol
				h.options.policy = policy[0]
			default:
				return h, c.Errf("unknown property '%s'", c.Val())
			}
		}
	}

	if h.optimizer == nil {
		h.optimizer, _ = GetPolicy(DefaultPolicy)
	}

	return h, nil
}
