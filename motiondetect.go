package main

import (
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

var (
	red   = color.RGBA{255, 0, 0, 0}
	green = color.RGBA{0, 255, 0, 0}
	blue  = color.RGBA{0, 0, 255, 0}

	ContourColor = red
	RectColor    = blue
)

const (
	ContourThickness = 1
	RectThickness    = 2
)

// MotionDetector
type MotionDetector struct {
	Threshold          float32
	DilateSize         int
	MinimumContourArea float64

	DrawContours bool
	DrawRects    bool

	deltaMat     gocv.Mat
	threshMat    gocv.Mat
	bgSubtractor gocv.BackgroundSubtractorMOG2
}

// NewMotionDetector returns a MotionDetector with reasonable defaults.
func NewMotionDetector() *MotionDetector {
	return &MotionDetector{
		Threshold:          25,
		DilateSize:          3,
		MinimumContourArea: 3000,
		DrawContours:       true,
		DrawRects:          true,
		deltaMat:           gocv.NewMat(),
		threshMat:          gocv.NewMat(),
		bgSubtractor:       gocv.NewBackgroundSubtractorMOG2WithParams(500, 16, false),
	}
}

// Detected returns true if motion has been detected in the given image,
// compared to the image given the last time it was called. The image will also
// be marked up with rectangles and contours where the motion was detected,
// based on the values of DrawRects and DrawContours, respectively.
func (m *MotionDetector) Detected(img *gocv.Mat) bool {
	// first phase of cleaning up image, obtain foreground only
	m.bgSubtractor.Apply(*img, &m.deltaMat)

	// remaining cleanup of the image to use for finding contours.
	// first use threshold
	gocv.Threshold(m.deltaMat, &m.threshMat, m.Threshold, 255, gocv.ThresholdBinary)

	// then dilate
	kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(m.DilateSize, m.DilateSize))
	defer kernel.Close()
	gocv.Dilate(m.threshMat, &m.threshMat, kernel)

	// now find contours
	contours := gocv.FindContours(m.threshMat, gocv.RetrievalExternal, gocv.ChainApproxSimple)

	hasMarkup := m.DrawContours || m.DrawRects

	motionDetected := false
	for i := 0; i < contours.Size(); i++ {
		var (
			contour = contours.At(i)
			area    = gocv.ContourArea(contour)
		)
		if area < m.MinimumContourArea {
			continue
		}
		motionDetected = true
		if !hasMarkup {
			break
		}

		if m.DrawContours {
			gocv.DrawContours(img, contours, i, ContourColor, ContourThickness)
		}
		if m.DrawRects {
			rect := gocv.BoundingRect(contour)
			gocv.Rectangle(img, rect, RectColor, RectThickness)
		}
	}
	return motionDetected
}

// Close closes the detector & cleans up all resources.
func (m *MotionDetector) Close() {
	m.deltaMat.Close()
	m.threshMat.Close()
	m.bgSubtractor.Close()
}
