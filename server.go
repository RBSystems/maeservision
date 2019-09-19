package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"

	"github.com/blackjack/webcam"
	pigo "github.com/esimov/pigo/core"
)

const fmtYUYV = 0x56595559

var col = color.NRGBA{255, 0, 0, 255}

// FrameSizes .
type FrameSizes []webcam.FrameSize

func startCam() (*webcam.Webcam, error) {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		return nil, fmt.Errorf("couldn't open webcam: %v", err)
	}

	formatDesc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range formatDesc {
		formats = append(formats, f)
	}

	var format webcam.PixelFormat
	format = fmtYUYV

	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	var size webcam.FrameSize

	for _, value := range frames {
		if fmt.Sprintf("%s", value.GetString()) == "1600x1200" {
			size = value
			break
		}
	}

	f, w, h, err := cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		return nil, fmt.Errorf("Couldn't setup cam img format: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Resulting image format: %s (%dx%d)\n", formatDesc[f], w, h)
	return cam, nil

}

// HLine draws a horizontal line
func HLine(img *image.NRGBA, x1, y, x2 int, col color.Color) {
	for ; x1 <= x2; x1++ {
		img.Set(x1, y, col)
	}
}

// VLine draws a veritcal line
func VLine(img *image.NRGBA, x, y1, y2 int, col color.Color) {
	for ; y1 <= y2; y1++ {
		img.Set(x, y1, col)
	}
}

// Rect draws a rectangle utilizing HLine() and VLine()
func Rect(img *image.NRGBA, x1, y1, x2, y2 int, col color.Color) {
	HLine(img, x1, y1, x2, col)
	HLine(img, x1, y2, x2, col)
	VLine(img, x1, y1, y2, col)
	VLine(img, x2, y1, y2, col)
}

func usePigo(src *image.NRGBA) {
	cascadeFile, err := ioutil.ReadFile("/home/caleb/go/src/github.com/esimov/pigo/cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	pixels := pigo.RgbToGrayscale(src)
	cols, rows := src.Bounds().Max.X, src.Bounds().Max.Y

	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.15,
		ScaleFactor: 1.1,

		ImageParams: pigo.ImageParams{
			Pixels: pixels,
			Rows:   rows,
			Cols:   cols,
			Dim:    cols,
		},
	}

	pigo := pigo.NewPigo()

	classifier, err := pigo.Unpack(cascadeFile)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}

	angle := 0.0 // cascade rotation angle. 0.0 is 0 radians and 1.0 is 2*pi radians

	//	drawSrc := &image.Uniform{color.RGBA{255, 0, 0, 255}}
	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cParams, angle)
	dets = classifier.ClusterDetections(dets, 0.2)
	for _, det := range dets {
		if det.Q < 5 {
			continue
		}
		x := det.Col - det.Scale/2
		y := det.Row - det.Scale/2
		Rect(src, x, y, x+det.Scale, y+det.Scale, col)
		print("Q")
	}
}

func imgFromYUYV(frame []byte) error {
	yuyv := image.NewYCbCr(image.Rect(0, 0, 1600, 1200), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		yuyv.Y[i*2] = frame[ii]
		yuyv.Y[i*2+1] = frame[ii+2]
		yuyv.Cb[i] = frame[ii+1]
		yuyv.Cr[i] = frame[ii+3]
	}

	nimg := pigo.ImgToNRGBA(yuyv)
	usePigo(nimg)
	//	f, err := os.Create("yuyv.jpg")
	//	defer f.Close()
	//	if err != nil {
	//		return err
	//	}
	buf := new(bytes.Buffer)
	err := jpeg.Encode(buf, nimg, nil)
	print("*")
	os.Stdout.Write(buf.Bytes())
	os.Stdout.Sync()
	return err
}

func main() {
	cam, err := startCam()
	if err != nil {
		fmt.Printf("error starting cam: %v\n", err)
		os.Exit(1)
	}
	defer cam.Close()

	println("Press Enter to start streaming")
	fmt.Scanf("\n")
	err = cam.StartStreaming()
	if err != nil {
		fmt.Printf("Error starting stream: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("starting")
	timeout := uint32(5) //5 seconds
	for {
		err = cam.WaitForFrame(timeout)

		switch err.(type) {
		case nil:
		case *webcam.Timeout:
			fmt.Fprint(os.Stderr, err.Error())
			continue
		default:
			fmt.Printf("Error waiting for frame: %v\n", err)
			panic(err.Error())
		}

		frame, err := cam.ReadFrame()
		if len(frame) != 0 {
			err := imgFromYUYV(frame)
			if err != nil {
				fmt.Printf("Error with yuyv: %v\n", err)
			}
		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}

	}
}
