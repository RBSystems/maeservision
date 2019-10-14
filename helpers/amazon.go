package helpers

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/blackjack/webcam"
	"github.com/byuoitav/room-auth-ms/structs"
	"github.com/byuoitav/wso2services/wso2requests"
)

var svc *rekognition.Rekognition

// RekognitionResult contains the name and face of the person recognized
type RekognitionResult struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

func init() {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	svc = rekognition.New(sess)

}

// StartRekognition starts the webcam and begins passing images up to Amazon Rekognition
func StartRekognition() {
	byteChan := make(chan []byte)
	go rekognitionManager(byteChan)
	cam, err := StartCam()
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
		//		fmt.Println("Picture time")
		log.Println("Wait for frame")
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
		log.Println("ReadFrame")
		frame, err := cam.ReadFrame()
		if len(frame) != 0 {
			bytes, err := ImgFromYUYV(frame)
			if err != nil {
				fmt.Printf("Error with yuyv: %v\n", err)
			} else if len(bytes) > 0 {
				fmt.Println("Face found")
				byteChan <- bytes
			}

		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}
		log.Println("Finished")

	}
}

// recognize returns the person who is recognized in the photo
func recognize(bytes []byte) structs.WSO2CredentialPerson {
	var toReturn structs.WSO2CredentialPerson
	fmt.Println("Face recognized")
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
		fmt.Println("failed to search faces: ", err)
		return toReturn
	}
	fmt.Printf("%d faces found\n", len(resp.FaceMatches))
	if len(resp.FaceMatches) == 0 {
		fmt.Println("No faces found")
		return toReturn
	}
	for _, face := range resp.FaceMatches {
		fmt.Println(*face.Face.ExternalImageId)
		var output structs.WSO2CredentialResponse
		err := wso2requests.MakeWSO2Request("GET", "https://api.byu.edu:443/byuapi/persons/v3/?credentials.credential_type=NET_ID&credentials.credential_id="+*face.Face.ExternalImageId, "", &output)
		if err != nil {
			fmt.Printf("Error when making WSO2 request %v", err)
			return toReturn
		}
		return output.Values[0]
	}
	return toReturn
}

//RekognitionManager receives a channel of byte arrays (representing jpegs)
//and then displays who the recognized faces are
func rekognitionManager(byteChan chan ([]byte)) {
	for {
		select {
		case img := <-byteChan:
			image := base64.StdEncoding.EncodeToString(img)
			for _, client := range clients {
				client.send <- RekognitionResult{Name: "danny", Image: image}
			}

			/*
				resp := recognize(img)
				if resp.Basic.NetID.Value != "" {
					image := base64.StdEncoding.EncodeToString(img)
					for _, client := range clients {
						client.send <- RekognitionResult{Name: resp.Basic.FirstName.Value, Image: image}
					}
				}
			*/
		}
	}
}
