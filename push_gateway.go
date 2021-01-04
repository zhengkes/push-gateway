package main

import (
	"fmt"
	"sync"
	"time"
)


var Gauge *gaugeMetric
var once = &sync.Once{}

type gaugeMetric struct {
	sync.RWMutex
	prefix string
	//ident,metric
	metrics map[string]map[string]*metricValue
}

func newGauge(prefix string) *gaugeMetric {
	return &gaugeMetric{
		prefix:  prefix,
		metrics: make(map[string]map[string]*metricValue),
	}
}

//tags:xxx=xxx
func (g *gaugeMetric) Set(ident, metric string, val interface{}, tags ...string) {
	g.Lock()
	defer g.Unlock()
	metric = g.prefix + "." + metric
	if value, exists := g.metrics[ident]; exists {
		if value_1, exists_1 := value[metric]; exists_1 {
			value_1.setGauge(ident, metric, val, tags...)
		} else {
			value[metric] = new(metricValue)
			value[metric].setGauge(ident, metric, val, tags...)
		}
	} else {
		g.metrics[ident] = make(map[string]*metricValue)
		g.metrics[ident][metric] = new(metricValue)
		g.metrics[ident][metric].setGauge(ident, metric, val, tags...)
	}
}
func (g *gaugeMetric) dump() []*metricValue {
	g.Lock()
	defer g.Unlock()
	metrics := make([]*metricValue, 0)
	for _, v := range g.metrics {
		for _, v_1 := range v {
			metrics = append(metrics, v_1)
		}
	}
	return metrics
}

func Init(params ...string) error {
	var prefix string
	if len(params) == 0 {
		return fmt.Errorf("param can not be empty")
	}
	prefix = params[0]
	if len(params) < 2 {
		if err := parse(); err != nil {
			return err
		}
	} else {
		if err := parse(params[1]); err != nil {
			return err
		}
	}
	once.Do(func() {
		initRpcClients()
	})
	Gauge = newGauge(prefix)

	go func() {
		t1 := time.NewTicker(time.Duration(config.Duration) * time.Second)
		for {
			<-t1.C
			gauges := Gauge.dump()
			push(gauges)
		}
	}()
	return nil
}

func push(items []*metricValue) {
	if err := rpcPush(config.Remote.Addresses, items); err != nil {
		return
	}
}

