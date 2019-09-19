package main

import (
	"fmt"
	"os"

	"github.com/blackjack/webcam"
	"github.com/byuoitav/maeservision/helpers"
)

func main() {
	cam, err := helpers.StartCam()
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
			err := helpers.ImgFromYUYV(frame)
			if err != nil {
				fmt.Printf("Error with yuyv: %v\n", err)
			}
		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}

	}
}
