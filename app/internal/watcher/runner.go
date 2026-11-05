package watcher

import "time"

type Watcher struct {
	limit    uint64
	cooldown time.Duration
	last     time.Time
	onBreach func(Sample)
}

func (w *Watcher) step(sample Sample) bool {
	if sample.MemoryBytes < w.limit {
		return false
	}
	if !w.last.IsZero() && sample.At.Sub(w.last) < w.cooldown {
		return false
	}
	w.last = sample.At
	w.onBreach(sample)
	return true
}
