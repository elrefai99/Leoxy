package watcher

import (
	"testing"
	"time"
)

func TestWatcherStep(t *testing.T) {
	base := time.Unix(1000, 0)
	breaches := 0
	watcher := &Watcher{
		limit:    100,
		cooldown: time.Minute,
		onBreach: func(Sample) { breaches++ },
	}

	steps := []struct {
		name   string
		at     time.Duration
		memory uint64
		want   bool
	}{
		{"below limit", 0, 99, false},
		{"first breach", time.Second, 100, true},
		{"inside cooldown", 30 * time.Second, 200, false},
		{"after cooldown", 62 * time.Second, 200, true},
		{"back below limit", 63 * time.Second, 10, false},
	}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			got := watcher.step(Sample{At: base.Add(step.at), MemoryBytes: step.memory})
			if got != step.want {
				t.Fatalf("step = %v, want %v", got, step.want)
			}
		})
	}
	if breaches != 2 {
		t.Fatalf("breaches = %d, want 2", breaches)
	}
}
