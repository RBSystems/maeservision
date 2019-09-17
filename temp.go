// Example program that uses blakjack/webcam library
// for working with V4L2 devices.
// The application reads frames from device and writes them to stdout
// If your device supports motion formats (e.g. H264 or MJPEG) you can
// use it's output as a video stream.
// Example usage: go run stdout_streamer.go | vlc -
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

var col = color.NRGBA{255, 0, 0, 255}

var (
	dhtMarker = []byte{255, 196}
	dht       = []byte{1, 162, 0, 0, 1, 5, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 1, 0, 3, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 16, 0, 2, 1, 3, 3, 2, 4, 3, 5, 5, 4, 4, 0, 0, 1, 125, 1, 2, 3, 0, 4, 17, 5, 18, 33, 49, 65, 6, 19, 81, 97, 7, 34, 113, 20, 50, 129, 145, 161, 8, 35, 66, 177, 193, 21, 82, 209, 240, 36, 51, 98, 114, 130, 9, 10, 22, 23, 24, 25, 26, 37, 38, 39, 40, 41, 42, 52, 53, 54, 55, 56, 57, 58, 67, 68, 69, 70, 71, 72, 73, 74, 83, 84, 85, 86, 87, 88, 89, 90, 99, 100, 101, 102, 103, 104, 105, 106, 115, 116, 117, 118, 119, 120, 121, 122, 131, 132, 133, 134, 135, 136, 137, 138, 146, 147, 148, 149, 150, 151, 152, 153, 154, 162, 163, 164, 165, 166, 167, 168, 169, 170, 178, 179, 180, 181, 182, 183, 184, 185, 186, 194, 195, 196, 197, 198, 199, 200, 201, 202, 210, 211, 212, 213, 214, 215, 216, 217, 218, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 17, 0, 2, 1, 2, 4, 4, 3, 4, 7, 5, 4, 4, 0, 1, 2, 119, 0, 1, 2, 3, 17, 4, 5, 33, 49, 6, 18, 65, 81, 7, 97, 113, 19, 34, 50, 129, 8, 20, 66, 145, 161, 177, 193, 9, 35, 51, 82, 240, 21, 98, 114, 209, 10, 22, 36, 52, 225, 37, 241, 23, 24, 25, 26, 38, 39, 40, 41, 42, 53, 54, 55, 56, 57, 58, 67, 68, 69, 70, 71, 72, 73, 74, 83, 84, 85, 86, 87, 88, 89, 90, 99, 100, 101, 102, 103, 104, 105, 106, 115, 116, 117, 118, 119, 120, 121, 122, 130, 131, 132, 133, 134, 135, 136, 137, 138, 146, 147, 148, 149, 150, 151, 152, 153, 154, 162, 163, 164, 165, 166, 167, 168, 169, 170, 178, 179, 180, 181, 182, 183, 184, 185, 186, 194, 195, 196, 197, 198, 199, 200, 201, 202, 210, 211, 212, 213, 214, 215, 216, 217, 218, 226, 227, 228, 229, 230, 231, 232, 233, 234, 242, 243, 244, 245, 246, 247, 248, 249, 250}
	sosMarker = []byte{255, 218}
)

func readChoice(s string) int {
	var i int
	for true {
		print(s)
		_, err := fmt.Scanf("%d\n", &i)
		if err != nil || i < 1 {
			println("Invalid input. Try again")
		} else {
			break
		}
	}
	return i
}

// FrameSizes .
type FrameSizes []webcam.FrameSize

func (slice FrameSizes) Len() int {
	return len(slice)
}

//For sorting purposes
func (slice FrameSizes) Less(i, j int) bool {
	ls := slice[i].MaxWidth * slice[i].MaxHeight
	rs := slice[j].MaxWidth * slice[j].MaxHeight
	return ls < rs
}

//For sorting purposes
func (slice FrameSizes) Swap(i, j int) {
	slice[i], slice[j] = slice[j], slice[i]
}

func setupCam() (*webcam.Webcam, error) {
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		return nil, fmt.Errorf("couldn't open webcam: %v", err)

	}
	defer cam.Close()

	formatDesc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range formatDesc {
		formats = append(formats, f)
	}

	var format webcam.PixelFormat
	for _, value := range formats {
		if fmt.Sprintf("%s", formatDesc[value]) == "Motion-JPEG" {
			format = value
			break
		}
	}

	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	/*fmt.Fprintf(os.Stderr, "Supported frame sizes for format %s\n", formatDesc[format])
	sort.Sort(frames)

	for i, value := range frames {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, value.GetString())
	}
	choice := readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(frames)))
	size := frames[choice-1]
	*/
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

func processFrame(frame []byte) {
	img, err := jpeg.Decode(bytes.NewReader(frame))
	if err != nil {
		fmt.Printf("could not decode jpeg from frame")
		panic(err)
	}
	nimg := pigo.ImgToNRGBA(img)
	usePigo(nimg)
	print(".")
	buf := new(bytes.Buffer)
	err = jpeg.Encode(buf, nimg, nil)
	if err != nil {
		fmt.Printf("failed")
	}
	os.Stdout.Write(buf.Bytes())
	os.Stdout.Sync()

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
	}
}

func addMotionDht(frame []byte) []byte {
	jpegParts := bytes.Split(frame, sosMarker)
	return append(jpegParts[0], append(dhtMarker, append(dht, append(sosMarker, jpegParts[1]...)...)...)...)
}

func main() {
	/*cam, err := setupCam()
	if err != nil {
		fmt.Printf("Error setting up camera: %v\n", err)
		os.Exit(1)
	}
	*/
	cam, err := webcam.Open("/dev/video0")
	if err != nil {
		//	return nil, fmt.Errorf("couldn't open webcam: %v", err)

	}
	defer cam.Close()

	formatDesc := cam.GetSupportedFormats()
	var formats []webcam.PixelFormat
	for f := range formatDesc {
		formats = append(formats, f)
	}

	var format webcam.PixelFormat
	for _, value := range formats {
		if fmt.Sprintf("%s", formatDesc[value]) == "Motion-JPEG" {
			format = value
			break
		}
	}

	frames := FrameSizes(cam.GetSupportedFrameSizes(format))
	/*fmt.Fprintf(os.Stderr, "Supported frame sizes for format %s\n", formatDesc[format])
	sort.Sort(frames)

	for i, value := range frames {
		fmt.Fprintf(os.Stderr, "[%d] %s\n", i+1, value.GetString())
	}
	choice := readChoice(fmt.Sprintf("Choose format [1-%d]: ", len(frames)))
	size := frames[choice-1]
	*/
	var size webcam.FrameSize
	for _, value := range frames {
		if fmt.Sprintf("%s", value.GetString()) == "1600x1200" {
			size = value
			break
		}
	}

	f, w, h, err := cam.SetImageFormat(format, uint32(size.MaxWidth), uint32(size.MaxHeight))

	if err != nil {
		fmt.Printf("Error setting up cam img format: %v\n", err)
		//		return nil, fmt.Errorf("Couldn't setup cam img format: %v", err)
	}

	fmt.Fprintf(os.Stderr, "Resulting image format: %s (%dx%d)\n", formatDesc[f], w, h)

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
			fmt.Printf("Error waiting for frame: %v", err)
			panic(err.Error())
		}

		frame, err := cam.ReadFrame()
		if len(frame) != 0 {
			//processFrame(frame)
			img, err := jpeg.Decode(bytes.NewReader(addMotionDht(frame)))
			if err != nil {
				fmt.Printf("could not decode jpeg from frame: %v\n", err)
				os.Exit(1)
			}
			nimg := pigo.ImgToNRGBA(img)
			usePigo(nimg)
			print(".")
			buf := new(bytes.Buffer)
			err = jpeg.Encode(buf, nimg, nil)
			if err != nil {
				fmt.Printf("failed")
			}
			os.Stdout.Write(buf.Bytes())
			os.Stdout.Sync()
		} else if err != nil {
			fmt.Printf("Error reading frame: %v", err)
			panic(err.Error())
		}
	}
}
