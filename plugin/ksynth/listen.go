package ksynth

// Listen for updates, add to our in memory map.
// Serve requests out of this map.
import (
	"compress/gzip"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coredns/coredns/plugin"
	jsoniter "github.com/json-iterator/go"
	"github.com/kentik/ktranslate/pkg/eggs/kmux"
)

const (
	DefaultPath   = "/input"
	DefaultHost   = "127.0.0.1:8080"
	DefaultPolicy = LowestAvgPingTime
)

var json = jsoniter.ConfigFastest

type options struct {
	// automatically generate IP to Hostname PTR entries
	// for host entries we parse
	autoReverse bool

	// The TTL of the record we generate
	ttl uint32

	// IP and port to listen on for updates to come in.
	listen string

	// Path to accept updates on.
	path string

	// Policy to use for picking which entry to use.
	policy string
}

func newOptions() *options {
	return &options{
		autoReverse: true,
		ttl:         3600,
		listen:      DefaultHost,
		path:        DefaultPath,
		policy:      DefaultPolicy,
	}
}

// Map contains the IPv4/IPv6 and reverse mapping.
type Map struct {
	// Key for the list of literal IP addresses must be a FQDN lowercased host name.
	name4 map[string][]net.IP
	name6 map[string][]net.IP

	// Key for the list of host names must be a literal IP address
	// including IPv6 address without zone identifier.
	// We don't support old-classful IP address notation.
	addr map[string][]string
}

func newMap() *Map {
	return &Map{
		name4: make(map[string][]net.IP),
		name6: make(map[string][]net.IP),
		addr:  make(map[string][]string),
	}
}

// Len returns the total number of addresses in the hostmap, this includes V4/V6 and any reverse addresses.
func (h *Map) Len() int {
	l := 0
	for _, v4 := range h.name4 {
		l += len(v4)
	}
	for _, v6 := range h.name6 {
		l += len(v6)
	}
	for _, a := range h.addr {
		l += len(a)
	}
	return l
}

type Update struct {
	IP          net.IP  `json:"dst_addr"`
	Host        string  `json:"test_name"`
	PingTimeStd float64 `json:"ping_std_rtt"`
	PingTimeMax float64 `json:"ping_max_rtt"`
	PingTimeJit float64 `json:"ping_jit_rtt"`
	PingTimeAvg float64 `json:"ping_avg_rtt"`
	TestID      int     `json:"test_id"`
	TaskID      int     `json:"task_id"`
	ResultType  string  `json:"result_type_str"`
	AgentName   string  `json:"agent_name"`
	PingSent    int     `json:"fetch_status_|_ping_sent_|_trace_time"`
	PingLost    int     `json:"fetch_ttlb_|_ping_lost"`
}

// KsynthListen contains known host entries.
type KsynthListen struct {
	sync.RWMutex

	// list of zones we are authoritative for
	Origins []string

	// hosts maps for lookups
	hmap *Map

	// updates keep track of current state of best times by host.
	updates map[string]Update

	options *options

	optimizer optimizer
}

func (ks *KsynthListen) listen() {
	log.Infof("Listening on %s%s with policy %v", ks.options.listen, ks.options.path, ks.options.policy)
	r := kmux.NewRouter()
	r.HandleFunc(ks.options.path+"/ksynth/batch", ks.wrap(ks.readBatch))
	server := &http.Server{Addr: ks.options.listen, Handler: r}
	err := server.ListenAndServe()

	// err is always non-nil -- the http server stopped.
	if err != http.ErrServerClosed {
		log.Errorf("There was an error when bringing up the HTTP system on %s: %v.", ks.options.listen, err)
		panic(err)
	}
	log.Infof("HTTP server shut down on %s -- %v", ks.options.listen, err)
}

func (ks *KsynthListen) readBatch(w http.ResponseWriter, r *http.Request) {
	var wrapper []Update

	// Decode body in gzip format if the request header is set this way.
	body := r.Body
	if r.Header.Get("Content-Encoding") == "gzip" {
		z, err := gzip.NewReader(r.Body)
		if err != nil {
			panic(http.StatusInternalServerError)
		}
		body = z
	}
	defer body.Close()

	if err := json.NewDecoder(body).Decode(&wrapper); err != nil {
		panic(http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)

	newMap, newUpdates := ks.parse(wrapper)
	log.Debugf("Parsed hosts file into %d entries", newMap.Len())
	if newMap.Len() == 0 {
		return // Don't change if the update has 0 entries.
	}

	ks.Lock()
	ks.hmap = newMap
	ks.updates = newUpdates
	ksynthEntries.WithLabelValues().Set(float64(ks.hmap.Len()))
	ksynthUpdateTime.Set(float64(time.Now().UnixNano()))
	ks.Unlock()
}

func (h *KsynthListen) optimize(w []Update) map[string]Update {

	// Sort by host. Pick out the lowest pingtime for each host here, return just the entry for this host.
	hosts := h.optimizer(w)

	log.Infof("Reduced entries from %d to %d in optimization", len(w), len(hosts))

	// Now add in any entries which arn't present in this update but were set previously.
	// XX Should we do this?
	h.RLock()
	old := h.updates
	h.RUnlock()

	for hostname, update := range old {
		if _, ok := hosts[hostname]; !ok { // Nothing here for this host, just add it in.
			hosts[hostname] = update
		}
	}

	return hosts
}

func (h *KsynthListen) parse(w []Update) (*Map, map[string]Update) {
	hmap := newMap()

	updates := h.optimize(w)
	for _, update := range updates {
		if update.IP == nil {
			continue
		}

		family := 0
		if update.IP.To4() != nil {
			family = 1
		} else {
			family = 2
		}

		name := plugin.Name(update.Host).Normalize()
		if plugin.Zones(h.Origins).Matches(name) == "" {
			// name is not in Origins
			continue
		}
		switch family {
		case 1:
			hmap.name4[name] = append(hmap.name4[name], update.IP)
		case 2:
			hmap.name6[name] = append(hmap.name6[name], update.IP)
		default:
			continue
		}
		if !h.options.autoReverse {
			continue
		}
		hmap.addr[update.IP.String()] = append(hmap.addr[update.IP.String()], name)
	}

	return hmap, updates
}

func (ks *KsynthListen) wrap(f handler) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				ksynthErrors.WithLabelValues().Inc()
				if code, ok := r.(int); ok {
					http.Error(w, http.StatusText(code), code)
					return
				}
				panic(r)
			}
		}()

		if err := r.ParseForm(); err != nil {
			panic(http.StatusBadRequest)
		}

		f(w, r)
	}
}

type handler func(http.ResponseWriter, *http.Request)

// lookupStaticHost looks up the IP addresses for the given host from the hosts file.
func (h *KsynthListen) lookupStaticHost(m map[string][]net.IP, host string) []net.IP {
	h.RLock()
	defer h.RUnlock()

	if len(m) == 0 {
		return nil
	}

	ips, ok := m[host]
	if !ok {
		return nil
	}
	ipsCp := make([]net.IP, len(ips))
	copy(ipsCp, ips)
	return ipsCp
}

// LookupStaticHostV4 looks up the IPv4 addresses for the given host from the hosts file.
func (h *KsynthListen) LookupStaticHostV4(host string) []net.IP {
	host = strings.ToLower(host)
	return h.lookupStaticHost(h.hmap.name4, host)
}

// LookupStaticHostV6 looks up the IPv6 addresses for the given host from the hosts file.
func (h *KsynthListen) LookupStaticHostV6(host string) []net.IP {
	host = strings.ToLower(host)
	return h.lookupStaticHost(h.hmap.name6, host)
}

// LookupStaticAddr looks up the hosts for the given address from the hosts file.
func (h *KsynthListen) LookupStaticAddr(addr string) []string {
	addr = net.ParseIP(addr).String()
	if addr == "" {
		return nil
	}

	h.RLock()
	defer h.RUnlock()
	hosts1 := h.hmap.addr[addr]

	if len(hosts1) == 0 {
		return nil
	}

	return hosts1
}
