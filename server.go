package main

import (
	"fmt"
	"image/jpeg"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	pigo "github.com/esimov/pigo/core"
)

/*
type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}
*/

var wg = &sync.WaitGroup{}

func main() {
	fmt.Printf("Starting image check\n")
	wg.Add(1)
	go facecheck()
	fmt.Printf("Started!\n")
	wg.Wait()
	fmt.Printf("Finished\n")
}

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
