package main

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

// MatBuffer is a matrix ring buffer, which stores the last frames added to it.
type MatBuffer struct {
	imgs   []*gocv.Mat
	times  []time.Time
	writes int
}

// NewMatBuffer creates a new MatBuffer with enough frames to store the given
// duration at the given FPS.
func NewMatBuffer(duration time.Duration, fps float64) *MatBuffer {
	frames := int(fps * duration.Seconds())
	b := MatBuffer{
		imgs:  make([]*gocv.Mat, frames),
		times: make([]time.Time, frames),
	}
	for i := range b.imgs {
		m := gocv.NewMat()
		b.imgs[i] = &m
	}
	return &b
}

// Close closes the buffer. A closed buffer can no longer be used.
func (b *MatBuffer) Close() error {
	var err error
	for _, img := range b.imgs {
		if err == nil {
			err = img.Close()
		}
		img.Close()
	}
	return err
}

// Add adds a new frame with the given timestamp to the buffer. If the buffer is
// full, the oldest frame is discarded.
func (b *MatBuffer) Add(img *gocv.Mat, t time.Time) {
	i := b.writes % len(b.imgs)
	img.CopyTo(b.imgs[i])
	b.times[i] = t
	b.writes++
}

// Duration returns the duration between the first and last frame added.
func (b *MatBuffer) Duration() time.Duration {
	oldest, newest := b.TimeWindow()
	return newest.Sub(oldest)
}

// Count returns the number of frames in the buffer.
func (b *MatBuffer) Count() int {
	return len(b.imgs)
}

// TimeWindow returns the timestamps of the first and last frames added.
// If no frames were added, the zero-value times are returned for both.
func (b *MatBuffer) TimeWindow() (time.Time, time.Time) {
	if b.writes == 0 {
		return time.Time{}, time.Time{}
	} else if b.writes <= len(b.imgs) {
		return b.times[0], b.times[b.writes-1]
	}
	var (
		first = b.writes % len(b.imgs)
		last  = (b.writes - 1 + len(b.imgs)) % len(b.imgs)
	)
	return b.times[first], b.times[last]
}

// FPS returns the average FPS of current contents of the buffer. Note that this
// may be different from the FPS with which the buffer was created.
func (b *MatBuffer) FPS() float64 {
	if b.writes < 2 {
		return 0
	}
	seconds := b.Duration().Seconds()
	if b.writes < len(b.imgs) {
		return float64(b.writes) / seconds
	}
	return float64(len(b.imgs)) / seconds
}

// Slice returns the buffer as a slice of matrices.
func (b *MatBuffer) Slice() []*gocv.Mat {
	if b.writes <= len(b.imgs) {
		return b.imgs[0:b.writes]
	}
	i := b.writes % len(b.imgs)
	return append(b.imgs[i:], b.imgs[:i]...)
}

// WriteFile writes the buffer as a video to the specified filename, using the
// specified "FourCC" codec (e.g. "mp4v"), with the given video dimensions.
func (b *MatBuffer) WriteFile(filename, codec string) error {
	imgs := b.Slice()
	if len(imgs) < 2 {
		return fmt.Errorf("need at least 2 frames")
	}

	var (
		width  = imgs[0].Cols()
		height = imgs[0].Rows()
	)

	vw, err := gocv.VideoWriterFile(filename, codec, b.FPS(), width, height, true)
	if err != nil {
		return fmt.Errorf("opening writer failed: %w", err)
	}
	defer vw.Close()

	for _, img := range imgs {
		if img.Cols() != width || img.Rows() != height {
			return fmt.Errorf("not all frames have the same dimensions")
		}
		if err := vw.Write(*img); err != nil {
			return fmt.Errorf("writing image failed: %w", err)
		}
	}
	return nil
}
