package main

import (
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	pigo "github.com/esimov/pigo/core"
)

/*var wg = &sync.WaitGroup{}

func main() {

		fmt.Printf("Starting image check\n")
		wg.Add(1)
		go facecheck()
		fmt.Printf("Started!\n")
		wg.Wait()
		fmt.Printf("Finished\n")

}
*/

/* var haarCascadeFile = "haar.xml"
var blue = color.RGBA{0, 0, 255, 0}
var green = color.RGBA{0, 255, 0, 0}

func toImage(m *gocv.Mat, imge image.Image) error {
	typ := m.Type()
	if typ != gocv.MatTypeCV8UC1 && typ != gocv.MatTypeCV8UC3 && typ !=
		gocv.MatTypeCV8UC4 {
		return errors.New("ToImage supports only MatType CV8UC1, CV8UC3 and CV8UC4")
	}

	width := m.Cols()
	height := m.Rows()
	step := m.Step()
	data := m.ToBytes()
	channels := m.Channels()

	switch img := imge.(type) {
	case *image.NRGBA:
		c := color.NRGBA{
			R: uint8(0),
			G: uint8(0),
			B: uint8(0),
			A: uint8(255),
		}
		for y := 0; y < height; y++ {
			for x := 0; x < step; x = x + channels {
				c.B = uint8(data[y*step+x])
				c.G = uint8(data[y*step+x+1])
				c.R = uint8(data[y*step+x+2])
				if channels == 4 {
					c.A = uint8(data[y*step+x+3])
				}
				img.SetNRGBA(int(x/channels), y, c)
			}
		}

	case *image.Gray:
		c := color.Gray{Y: uint8(0)}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c.Y = uint8(data[y*step+x])
				img.SetGray(x, y, c)
			}
		}
	}
	return nil
}

func grayscale(dst []uint8, src *image.NRGBA) []uint8 {
	rows, cols := src.Bounds().Dx(), src.Bounds().Dy()
	if dst == nil || len(dst) != rows*cols {
		dst = make([]uint8, rows*cols)
	}
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			dst[r*cols+c] = uint8(
				0.299*float64(src.Pix[r*4*cols+4*c+0]) +
					0.587*float64(src.Pix[r*4*cols+4*c+1]) +
					0.114*float64(src.Pix[r*4*cols+4*c+2]),
			)
		}
	}
	return dst
}

func pigoSetup(width, height int) (*image.NRGBA, []uint8, *pigo.Pigo,
	pigo.CascadeParams, pigo.ImageParams) {
	goImg := image.NewNRGBA(image.Rect(0, 0, width, height))
	grayGoImg := make([]uint8, width*height)
	cParams := pigo.CascadeParams{
		MinSize:     20,
		MaxSize:     1000,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
	}
	imgParams := pigo.ImageParams{
		Pixels: grayGoImg,
		Rows:   height,
		Cols:   width,
		Dim:    width,
	}
	classifier := pigo.NewPigo()

	pigoCascadeFile, err = ioutil.ReadFile("/home/caleb/go/src/github.com/esimov/pigo/cascade/facefinder")
	if err != nil {
		log.Fatalf("Error reading the cascade file: %v", err)
	}

	var err error
	if classifier, err = classifier.Unpack(pigoCascadeFile); err != nil {
		log.Fatalf("Error reading the cascade file: %s", err)
	}
	return goImg, grayGoImg, classifier, cParams, imgParams
}

func main() {
	fmt.Printf("Starting code\n")
	var err error
	// open webcam
	if webcam, err = gocv.VideoCaptureDevice(0); err != nil {
		log.Fatal(err)
	}
	defer webcam.Close()
	width := int(webcam.Get(gocv.VideoCaptureFrameWidth))
	height := int(webcam.Get(gocv.VideoCaptureFrameHeight))

	// open display window
	window := gocv.NewWindow("Face Detect")
	defer window.Close()

	// prepare image matrix
	img := gocv.NewMat()
	defer img.Close()

	// set up pigo
	goImg, grayGoImg, pigoClass, cParams, imgParams := pigoSetup(width,
		height)

	// create classifier and load model
	classifier := gocv.NewCascadeClassifier()
	if !classifier.Load(haarCascadeFile) {
		log.Fatalf("Error reading cascade file: %v\n", haarCascadeFile)
	}
	defer classifier.Close()

	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("cannot read device %d\n", deviceID)
			return
		}
		if img.Empty() {
			continue
		}
		// use PIGO
		if err = toImage(&img, goImg); err != nil {
			log.Fatal(err)
		}

		grayGoImg = grayscale(grayGoImg, goImg)
		imgParams.Pixels = grayGoImg
		dets := pigoClass.RunCascade(imgParams, cParams)
		dets = pigoClass.ClusterDetections(dets, 0.3)

		for _, det := range dets {
			if det.Q < 5 {
				continue
			}
			x := det.Col - det.Scale/2
			y := det.Row - det.Scale/2
			r := image.Rect(x, y, x+det.Scale, y+det.Scale)
			gocv.Rectangle(&img, r, green, 3)
		}

		// use GoCV
		rects := classifier.DetectMultiScale(img)
		for _, r := range rects {
			gocv.Rectangle(&img, r, blue, 3)
		}

		window.IMShow(img)
		if window.WaitKey(1) >= 0 {
			break
		}
	}
}
*/

func facecheck() {
	defer wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fmt.Println("Face check")
			resp, err := http.Get("http://10.66.76.14/cgi-bin/view.cgi?action=snapshot")
			if err != nil {
				fmt.Printf("Couldn't execute get: %v\n", err)
				os.Exit(1)
			}
			if resp.StatusCode/100 != 2 {
				fmt.Printf("Non-200 status code: %v\n", resp.StatusCode)
				os.Exit(1)
			}
			defer resp.Body.Close()
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading resp body: %v", err)
				os.Exit(1)
			}
			err = ioutil.WriteFile("pic.jpg", bytes, 0666)
			if err != nil {
				fmt.Printf("Couldn't write to file: %v", err)
				os.Exit(1)
			}

			err = ioutil.WriteFile("out.jpg", bytes, 0666)
			if err != nil {
				fmt.Printf("Couldn't write to out file: %v", err)
				os.Exit(1)
			}

			cascadeFile, err := ioutil.ReadFile("/home/caleb/go/src/github.com/esimov/pigo/cascade/facefinder")
			if err != nil {
				log.Fatalf("Error reading the cascade file: %v", err)
			}
			src, err := pigo.GetImage("pic.jpg")

			if err != nil {
				fmt.Printf("Error getting image in pigo: %v", err)
				os.Exit(1)
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
			if len(dets) > 0 {
				fmt.Printf("Detected!\n")
				//draw.DrawMask(dst, dst.Bounds(), drawSrc, image.ZP, &circle{image.Point{det.Col, det.Row}, det.Scale}, image.ZP, draw.Over)
				file, err := os.Create(fmt.Sprintf("./faces/%v.jpg", time.Now()))
				defer file.Close()
				if err != nil {
					fmt.Printf("os.Create failed: %v", err)
					os.Exit(1)
				}
				jpeg.Encode(file, src, nil)
			}
		}
	}
}
