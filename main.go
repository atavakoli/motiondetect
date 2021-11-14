package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
	"time"

	"gocv.io/x/gocv"
)

var (
	Width  int
	Height int
	MaxFPS float64

	Detector *MotionDetector
	DetectionEnabled bool

	BufferDuration time.Duration = 5 * time.Second

	fps = NewFPSCounter(5)

	FieldChanged = 'a'

	Done bool
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile = flag.String("memprofile", "", "write memory profile to file")
	matprofile = flag.String("matprofile", "", "write matrix memory profile to file")
)


func Status(s string) string {
	return fmt.Sprintf(
		"[%dx%d @ %0.0f/%0.0ffps] [a=%v d=%v t=%v (%s)]: %s",
		Width, Height,
		fps.FPS, MaxFPS,
		Detector.MinimumContourArea, Detector.DilateSize, Detector.Threshold,
		string(FieldChanged),
		s,
	)
}

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		Done = true
	}()
}

func PollInput(window *gocv.Window) {
	switch k := window.PollKey(); k {
	case 3: // ctrl+c
		Done = true
	default:
		switch rk := rune(k); rk {
		case 'm':
			DetectionEnabled = !DetectionEnabled
		case 'c':
			Detector.DrawContours = !Detector.DrawContours
		case 'r':
			Detector.DrawRects = !Detector.DrawRects
		case 'a', 'd', 't':
			FieldChanged = rk
		case '-', '=':
			dir := 1
			if rk == '-' {
				dir = -1
			}
			switch FieldChanged {
			case 'a':
				Detector.MinimumContourArea += float64(100 * dir)
				if Detector.MinimumContourArea <= 0 {
					Detector.MinimumContourArea = 100
				}
			case 'd':
				Detector.DilateSize += 1 * dir
				if Detector.DilateSize <= 0 {
					Detector.DilateSize = 1
				}
			case 't':
				Detector.Threshold += float32(1 * dir)
				if Detector.Threshold <= 0 {
					Detector.Threshold = 1
				}
			}
		}
	}
}

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		log.Println("Profiling CPU to", *cpuprofile)
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if len(flag.Args()) < 1 {
		fmt.Println("USAGE: camera [camera ID]")
		return
	}

	// parse args
	deviceID := flag.Arg(0)

	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		log.Fatalf("Error opening video capture device %v: %v", deviceID, err)
	}
	defer webcam.Close()

	window := gocv.NewWindow("Motion Window")
	defer window.Close()

	imgSrc := gocv.NewMat()
	defer imgSrc.Close()

	img := gocv.NewMat()
	defer img.Close()

	Width = int(webcam.Get(gocv.VideoCaptureFrameWidth))
	Height = int(webcam.Get(gocv.VideoCaptureFrameHeight))
	MaxFPS = webcam.Get(gocv.VideoCaptureFPS)

	var status string
	var statusColor color.RGBA

	Detector = NewMotionDetector()
	defer Detector.Close()

	SetupCloseHandler()

	fmt.Printf("Start reading device: %v\n", deviceID)

	fps.Start()
	defer fps.Stop()

	buffer := NewMatBuffer(BufferDuration, MaxFPS)
	log.Printf("Buffering %v @ %0.1ffps", BufferDuration, MaxFPS)
	defer buffer.Close()

	for !Done {
		if ok := webcam.Read(&imgSrc); !ok {
			fmt.Printf("Device closed: %v\n", deviceID)
			return
		}
		if imgSrc.Empty() {
			continue
		}

		// Flip horizontally (mirror view)
		gocv.Flip(imgSrc, &img, 1)

		if !DetectionEnabled {
			status = "Motion detection disabled"
			statusColor = blue
		} else if Detector.Detected(&img) {
			status = "Motion detected"
			statusColor = red
		} else {
			status = "Ready"
			statusColor = green
		}

		gocv.PutText(&img, Status(status), image.Pt(10, 20), gocv.FontHersheyPlain, 1.2, statusColor, 2)
		for i := range fps.frames {
			s := fmt.Sprintf("%d: %d %v", i, fps.frames[i], fps.durations[i])
			gocv.PutText(&img, s, image.Pt(10, 50+20*i), gocv.FontHersheyPlain, 1.2, blue, 2)
		}

		buffer.Add(&img, time.Now())
		window.IMShow(img)
		fps.NextFrame()

		PollInput(window)
	}

	log.Printf("Saving (%v @ %0.0ffps)", buffer.Duration(), buffer.FPS())
	if err := buffer.WriteFile("video.mp4", "mp4v"); err != nil {
		log.Fatalf("Error saving buffer: %v", err)
	}
	log.Println("Done")

	if *memprofile != "" {
		log.Println("Profiling memory to", *memprofile)

		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		runtime.GC()
		if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

	if *matprofile != "" {
		log.Println("Profiling matrix memory to", *matprofile)

		f, err := os.Create(*matprofile)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err := gocv.MatProfile.WriteTo(f, 1); err != nil {
			log.Fatal("could not write matrix memory profile: ", err)
		}
	}
}
