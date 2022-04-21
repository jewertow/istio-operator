package common

import (
	"math"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

type Backoff struct {
	backoff *wait.Backoff
	mux     sync.Mutex

	interval    time.Duration
	maxDuration time.Duration
}

func NewBackoff(interval, maxDuration time.Duration) *Backoff {
	return &Backoff{
		backoff: &wait.Backoff{
			Duration: interval,
			Factor:   2,
			Steps:    math.MaxInt32,
			Cap:      maxDuration,
		},
		mux:         sync.Mutex{},
		interval:    interval,
		maxDuration: maxDuration,
	}
}

func (b *Backoff) Step() time.Duration {
	b.mux.Lock()
	defer b.mux.Unlock()

	return b.backoff.Step()
}

func (b *Backoff) Reset() {
	b.mux.Lock()
	defer b.mux.Unlock()

	b.backoff.Duration = b.interval
	b.backoff.Factor = 2
	b.backoff.Steps = math.MaxInt32
	b.backoff.Cap = b.maxDuration
}
