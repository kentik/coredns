package ksynth

import (
	"github.com/coredns/coredns/plugin"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ksynthEntries is the number of entries in ksynth.
	ksynthEntries = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: "ksynth",
		Name:      "entries",
		Help:      "The number of entries in ksynth.",
	}, []string{})
	// ksynthUpdateTime is the timestamp of the last update from firehose.
	ksynthUpdateTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: plugin.Namespace,
		Subsystem: "ksynth",
		Name:      "update_timestamp_seconds",
		Help:      "The timestamp of the last update from Kentik Firehose.",
	})
	// ksynthErrors is a counter of errors processing updates.
	ksynthErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "ksynth",
		Name:      "errors",
		Help:      "The total number of errors seen by ksynth processing updates.",
	}, []string{})
)
