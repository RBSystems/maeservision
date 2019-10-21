package helpers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/blackjack/webcam"
	pigo "github.com/esimov/pigo/core"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
)

const fmtYUYV = 0x56595559

var push time.Time

// FrameSizes .
type FrameSizes []webcam.FrameSize

// StartCam starts a webcam connection
func StartCam() (*webcam.Webcam, error) {
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
	push = time.Now()
	cam.SetBufferCount(1)
	return cam, nil

}

var classifier *pigo.Pigo

func init() {
	push = time.Now()
	cascadeFile, err := ioutil.ReadFile("/home/caleb/go/src/github.com/esimov/pigo/cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	pigogo := pigo.NewPigo()
	classifier, err = pigogo.Unpack(cascadeFile)
	if err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}
}

func usePigo(src *image.NRGBA) []pigo.Detection {
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

	angle := 0.0 // cascade rotation angle. 0.0 is 0 radians and 1.0 is 2*pi radians

	//	drawSrc := &image.Uniform{color.RGBA{255, 0, 0, 255}}
	// Run the classifier over the obtained leaf nodes and return the detection results.
	// The result contains quadruplets representing the row, column, scale and detection score.
	dets := classifier.RunCascade(cParams, angle)
	dets = classifier.ClusterDetections(dets, 0.2)
	var toReturn []pigo.Detection
	for _, det := range dets {
		if det.Q < 12 {
			//			fmt.Printf("Lame face found: %v\n", det.Q)
			continue
		}
		//Rectangle drawing
		//x := det.Col - det.Scale/2
		//y := det.Row - det.Scale/2
		//Rect(src, x, y, x+det.Scale, y+det.Scale)
		toReturn = append(toReturn, det)
	}
	return toReturn
}

// FrameToJPEG converts a camera frame into a JPEG image
func FrameToJPEG(frame []byte) ([]byte, error) {
	yuyv := image.NewYCbCr(image.Rect(0, 0, 1600, 1200), image.YCbCrSubsampleRatio422)
	for i := range yuyv.Cb {
		ii := i * 4
		yuyv.Y[i*2] = frame[ii]
		yuyv.Y[i*2+1] = frame[ii+2]
		yuyv.Cb[i] = frame[ii+1]
		yuyv.Cr[i] = frame[ii+3]
	}
	nimg := pigo.ImgToNRGBA(yuyv)

	//Get jpeg form of face
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, nimg, nil)
	return buf.Bytes(), err
}

// ImgHasFace receives a byte array that is a YUYV frame from a webcam and processes
// said frame using pigo. It will then encode that image to a jpeg and return it and a boolean
// as to whether it contains a face or not
func ImgHasFace(frame []byte) (bool, []byte, error) {
	hasFace := false
	buf := bytes.NewBuffer(frame)
	img, err := jpeg.Decode(buf)
	if err != nil {
		return hasFace, []byte{}, err
	}
	nimg := pigo.ImgToNRGBA(img)
	dets := usePigo(nimg)
	/*for i, det := range dets {
		x := det.Col - det.Scale/2
		y := det.Row - det.Scale/2
		fmt.Printf("%v Q: %v left: %v --- top: %v --- right: %v --- bottom: %v\n", i, det.Q, x, y, x+det.Scale, y+det.Scale)

	}
	*/
	if len(dets) > 0 {
		if IsDelta(dets, push) {
			for _, det := range dets {
				x := det.Col - det.Scale/2
				y := det.Row - det.Scale/2
				left := x
				top := y
				width := det.Scale
				height := det.Scale

				croppedImg, err := cutter.Crop(nimg, cutter.Config{
					Width:  width,
					Height: height,
					Anchor: image.Point{left, top},
					Mode:   cutter.TopLeft,
				})
				if err != nil {
					fmt.Printf("error cropping image: %v", err)
					continue
				}

				resized := resize.Resize(uint(width), uint(height), croppedImg, resize.NearestNeighbor)

				buf := new(bytes.Buffer)
				err = jpeg.Encode(buf, resized, nil)
				if err != nil {
					fmt.Printf("error encoding jpeg after resize: %v", err)
					continue
				}
				fmt.Printf("Q: %v\n", det.Q)
				image := base64.StdEncoding.EncodeToString(buf.Bytes())

				for _, client := range clients {
					client.send <- RekognitionResult{Name: "", Image: image, Type: "cut"}
					fmt.Println("\nhere")
				}
			}

			push = time.Now()
			hasFace = true
			fmt.Println("Is delta")
		}
	}

	return hasFace, frame, nil
}
