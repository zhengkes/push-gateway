package main

import (
	"sync"
	"time"
)

var metricHistory *history

func initCache() {
	metricHistory = newHistory()
}

func newHistory() *history {
	h := history{
		Data: make(map[string]metricValue),
	}

	go h.Clean()
	return &h
}

type history struct {
	sync.RWMutex
	Data map[string]metricValue
}

func (h *history) Set(key string, item metricValue) {
	h.Lock()
	defer h.Unlock()
	h.Data[key] = item
}

func (h *history) Get(key string) (metricValue, bool) {
	h.RLock()
	defer h.RUnlock()

	item, exists := h.Data[key]
	return item, exists
}

func (h *history) Clean() {
	ticker := time.NewTicker(10 * time.Minute)
	for {
		select {
		case <-ticker.C:
			h.clean()
		}
	}
}

func (h *history) clean() {
	h.Lock()
	defer h.Unlock()
	now := time.Now().Unix()
	for key, item := range h.Data {
		if now-item.Timestamp > 10*item.Step {
			delete(h.Data, key)
		}
	}
}
