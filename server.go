package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
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
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	svc := rekognition.New(sess)
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
			bytes, err := helpers.ImgFromYUYV(frame)
			if err != nil {
				fmt.Printf("Error with yuyv: %v\n", err)
			} else if len(bytes) > 0 {
				/*print("*")
				os.Stdout.Write(bytes())
				os.Stdout.Sync()
				*/
				fmt.Println("eeeeee")
				image := &rekognition.Image{
					Bytes: bytes,
				}
				collectionID := "maeservision"
				input := &rekognition.SearchFacesByImageInput{
					CollectionId: &collectionID,
					//FaceMatchThreshold: 80,
					Image: image,
				}

				resp, err := svc.SearchFacesByImage(input)
				if err != nil {
					fmt.Println("failedd to serach faces: ", err)
					return
				}
				for _, face := range resp.FaceMatches {
					fmt.Println(*face.Face.ExternalImageId)
				}
				/*				params := &rekognition.DetectFacesInput{
									Image: image,
									Attributes: []

								}
				*/

			}
		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}

	}
}
