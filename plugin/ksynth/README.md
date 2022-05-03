# ksynth

## Name

*ksynth* - enables serving zone data from the kentik firehose.

## Description

Ksynth returns DNS requests based on data from Kentik Synthetic Testing. It will consume a firehose of
results from Kentik and always retrurn the "best" A or AAAA record, based on a pre-defined policy.

If you want to pass the request to the rest of the plugin chain if there is no match in the *ksynth*
plugin, you must specify the `fallthrough` option.

This plugin can only be used once per Server Block.

## Syntax

~~~
hosts [ZONES...] {
    ttl SECONDS
    no_reverse
    fallthrough [ZONES...]
    listen INTERFACE
    path URL
    policy OPTIMIZATION_POLICY
}
~~~

* **ZONES** zones it should be authoritative for. If empty, the zones from the configuration block
   are used.
* `no_reverse` disable the automatic generation of the `in-addr.arpa` or `ip6.arpa` entries for the hosts
* `fallthrough` If zone matches and no record can be generated, pass request to the next plugin.
  If **[ZONES...]** is omitted, then fallthrough happens for all zones for which the plugin
  is authoritative. If specific zones are listed (for example `in-addr.arpa` and `ip6.arpa`), then only
  queries for those zones will be subject to fallthrough.
* **OPTIMIZATION_POLICY** is one of avg_ping_time,max_ping_time,min_packet_loss, as explained below.

## Metrics

If monitoring is enabled (via the *prometheus* plugin) then the following metrics are exported:

- `coredns_ksynth_entries{}` - The combined number of entries in hosts and Corefile.
- `coredns_ksynth_update_timestamp_seconds{}` - The timestamp of the last update from Kentik Firehose.
- `coredns_ksynth_errors{}` - The total number of errors seen by ksynth processing updates.

## Examples

Listen on 127.0.0.1:8090 and return the result with min packet loss.

~~~
example.hosts example.org {
    ksynth {
      listen 127.0.0.1:8090
      ttl 30
      policy min_packet_loss
    }
    whoami
}
~~~

## Docker Compose

Run with docker-compose like so:

```
KENTIK_API_TOKEN=my_kentik_token KENTIK_EMAIL=bar@email.com docker-compose up
```

## Optimization Policies

* avg_ping_time: Return the record matching the requested domain with the lowest average ping time as recorded in the last round of tests from all agents. If all targets of this test are down, `SERVFAIL` will be returned. 

* max_ping_time: Return the record matching the requested domain with the lowest max ping time as recorded in the last round of tests from all agents. If all targets of this test are down, `SERVFAIL` will be returned.

* min_packet_loss: Return the record matching the requested domain with the lowest percent of packet loss as recorded in the last round of tests from all agents. If all targets of this test are down, `SERVFAIL` will be returned.

## See also

An overview of Kentik Synthetics: https://www.kentik.com/product/synthetics/
