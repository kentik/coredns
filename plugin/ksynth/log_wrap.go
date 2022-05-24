package ksynth

/**
This is a dumb wrapper around the coredns log to let us use it as a logr logger.
*/

import (
	"github.com/go-logr/logr"
)

type corednsLogSink struct {
	name      string
	keyValues map[string]interface{}
}

var _ logr.LogSink = &corednsLogSink{}

func (_ *corednsLogSink) Init(info logr.RuntimeInfo) {
}

func (_ corednsLogSink) Enabled(level int) bool {
	return true
}

func (l corednsLogSink) Info(level int, msg string, kvs ...interface{}) {
	log.Infof(msg, kvs)
}

func (l corednsLogSink) Error(err error, msg string, kvs ...interface{}) {
	log.Errorf(msg, kvs)
}

func (l corednsLogSink) WithName(name string) logr.LogSink {
	return &corednsLogSink{
		name:      l.name + "." + name,
		keyValues: l.keyValues,
	}
}

func (l corednsLogSink) WithValues(kvs ...interface{}) logr.LogSink {
	newMap := make(map[string]interface{}, len(l.keyValues)+len(kvs)/2)
	for k, v := range l.keyValues {
		newMap[k] = v
	}
	for i := 0; i < len(kvs); i += 2 {
		newMap[kvs[i].(string)] = kvs[i+1]
	}
	return &corednsLogSink{
		name:      l.name,
		keyValues: newMap,
	}
}

func NewLogger() logr.Logger {
	sink := &corednsLogSink{}
	return logr.New(sink)
}
