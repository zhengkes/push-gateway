package main

import (
	"fmt"
	"sync"
	"time"
)

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

type counterMetric struct {
	sync.RWMutex
	prefix  string
	metrics map[string]int
}

var Counter *counterMetric
var Gauge *gaugeMetric

var once = &sync.Once{}

func newCounter(prefix string) *counterMetric {
	return &counterMetric{
		metrics: make(map[string]int),
		prefix:  prefix,
	}
}

func (c *counterMetric) Set(metric string, value int) {
	c.Lock()
	defer c.Unlock()
	if _, exists := c.metrics[metric]; exists {
		c.metrics[metric] += value
	} else {
		c.metrics[metric] = value
	}
}

func (c *counterMetric) dump() map[string]int {
	c.Lock()
	defer c.Unlock()
	metrics := make(map[string]int)
	for key, value := range c.metrics {
		newKey := c.prefix + "." + key
		metrics[newKey] = value
		c.metrics[key] = 0
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
		if err := Parse(); err != nil {
			return err
		}
	} else {
		if err := Parse(params[1]); err != nil {
			return err
		}
	}
	//初始化rpc和本地缓存,保存counter类型,转化为gauge类型
	once.Do(func() {
		initRpcClients()
		initCache()
	})
	Counter = newCounter(prefix)
	Gauge = newGauge(prefix)
	go cron()
	return nil
}

func cron() {
	t1 := time.NewTicker(time.Duration(config.Duration) * time.Second)
	for {
		<-t1.C
		counters := Counter.dump()
		gauges := Gauge.dump()

		items := make([]*metricValue, 0)
		for metric, value := range counters {
			items = append(items, newMetricValue(metric, int64(value)))
		}
		items = append(items, gauges...)
		push(items)
	}
}

func push(items []*metricValue) {
	if err := rpcPush(config.Remote.Addresses, items); err != nil {
		return
	}
}

func newMetricValue(metric string, value interface{}) *metricValue {
	item := &metricValue{
		Metric:       metric,
		Timestamp:    time.Now().Unix(),
		ValueUntyped: value,
		CounterType:  "GAUGE",
		Step:         int64(config.Duration),
	}
	return item
}
