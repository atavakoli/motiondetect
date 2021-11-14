package main

import (
	"time"
)

// FPSCounter measures average frames per second.
type FPSCounter struct {
	FPS float64

	ticks     int
	frames    []int
	durations []time.Duration

	totalFrames   int
	totalDuration time.Duration

	done   chan struct{}
	ticker *time.Ticker
}

// NewFPSCounter creates a new FPSCounter that keeps track of the average
// FPS for the given last number of seconds. The counter is not started
// automateically; this must be done by the caller.
func NewFPSCounter(seconds int) *FPSCounter {
	return &FPSCounter{
		frames:    make([]int, seconds),
		durations: make([]time.Duration, seconds),
		done:      make(chan struct{}),
	}
}

// Start starts the counter and keeps track of average FPS, where a new frame is
// counted on each call to NextFrame. Start must not be called more than once
// for a given FPSCounter, unless Stop was called first.
func (c *FPSCounter) Start() {
	go c.runTicker()
}

func (c *FPSCounter) runTicker() {
	lastTime := time.Now()
	c.ticker = time.NewTicker(time.Second)
	<-c.ticker.C
	for {
		select {
		case <-c.done:
			break
		case t := <-c.ticker.C:
			lastDuration := t.Sub(lastTime)
			lastTime = t

			idx := c.ticks % len(c.frames)
			c.durations[idx] = lastDuration
			c.totalFrames += c.frames[idx]
			c.totalDuration += c.durations[idx]

			c.ticks++
			idx = c.ticks % len(c.frames)
			c.totalFrames -= c.frames[idx]
			c.totalDuration -= c.durations[idx]
			c.frames[idx] = 0
			c.durations[idx] = time.Duration(0)

			c.FPS = float64(c.totalFrames) / c.totalDuration.Seconds()
		}
	}
	c.ticker.Stop()
}

// NextFrame registers to the counter that a new frame has passed.
func (c *FPSCounter) NextFrame() {
	c.frames[c.ticks % len(c.frames)]++
}

// Duration returns the total duration over which the counter is currently
// tracking.
func (c *FPSCounter) Duration() time.Duration {
	return c.totalDuration
}

// Stop stops the counter.
func (c *FPSCounter) Stop() {
	c.done <- struct{}{}
}
