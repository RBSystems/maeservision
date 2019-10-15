package helpers

import (
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/blackjack/webcam"
	"github.com/byuoitav/room-auth-ms/structs"
	"github.com/byuoitav/wso2services/wso2requests"
	"github.com/oliamb/cutter"
)

var svc *rekognition.Rekognition

// RekognitionResult contains the name and face of the person recognized
type RekognitionResult struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	IsLive bool   `json:"isLive"`
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
	//Channel for rekognition recognized faces
	rekognitionChan := make(chan []byte)
	//Channel for all images to make up the live feed
	liveChan := make(chan []byte)
	// Start the managers
	go rekognitionManager(rekognitionChan)
	go liveManager(liveChan)

	// Start the Camera
	cam, err := StartCam()
	if err != nil {
		fmt.Printf("error starting cam: %v\n", err)
		os.Exit(1)
	}
	defer cam.Close()

	//Start Streaming
	println("Press Enter to start streaming")
	fmt.Scanf("\n")
	err = cam.StartStreaming()
	if err != nil {
		fmt.Printf("Error starting stream: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("starting")
	timeout := uint32(5) //5 seconds

	//Main loop
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
			hasFace, bytes, err := ImgHasFace(frame)
			if err != nil {
				fmt.Printf("Error with yuyv: %v\n", err)
			}
			liveChan <- bytes
			if hasFace {
				fmt.Println("New face found")
				rekognitionChan <- bytes

			}
		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}

	}
}

// recognize returns the person who is recognized in the photo
func recognize(bytes []byte) []structs.WSO2CredentialPerson {
	var toReturn []structs.WSO2CredentialPerson
	fmt.Println("Face recognized")
	image := &rekognition.Image{
		Bytes: bytes,
	}
	collectionID := "maeservision"
	var maxFaces int64
	maxFaces = 100
	input := &rekognition.SearchFacesByImageInput{
		CollectionId: &collectionID,
		//FaceMatchThreshold: 80,
		Image:    image,
		MaxFaces: &maxFaces,
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
		toReturn = append(toReturn, output.Values[0])
	}
	return toReturn
}

func getFeatures(img []byte) []*rekognition.FaceDetail {
	var toReturn []*rekognition.FaceDetail
	image := &rekognition.Image{
		Bytes: img,
	}
	var attr []*string
	var all string
	all = "ALL"
	attr = append(attr, &all)
	input := &rekognition.DetectFacesInput{
		Image:      image,
		Attributes: attr,
	}
	resp, err := svc.DetectFaces(input)
	if err != nil {
		fmt.Println("failed to detect faces: ", err)
		return toReturn
	}
	return resp.FaceDetails

}

//RekognitionManager receives a channel of byte arrays (representing jpegs)
//and then displays who the recognized faces are
func rekognitionManager(rekognitionChan chan ([]byte)) {
	for {
		select {
		case img := <-rekognitionChan:
			var wg sync.WaitGroup
			wg.Add(1)
			var people []structs.WSO2CredentialPerson
			go func() {
				defer wg.Done()
				people = recognize(img)
			}()
			wg.Add(1)
			var faceDetails []*rekognition.FaceDetail
			go func() {
				defer wg.Done()
				faceDetails = getFeatures(img)
			}()
			wg.Wait()
			if len(people) > 0 {
				for _, person := range people {
					//TODO update recognize to also return bounding boxes
					//TODO finish cutter
					//TODO resize image after cutter
					//TODO update html to properly show the faces (maybe an id to know when to reset?
					//TODO update html to have a place to show these sweet face details
					croppedImg, err := cutter.Crop(img, cutterConfig{})
					image := base64.StdEncoding.EncodeToString(img)
					for _, client := range clients {
						client.send <- RekognitionResult{Name: person.Basic.FirstName.Value, Image: image, IsLive: false}
					}
				}

			}
		}
	}
}

func liveManager(liveChan chan ([]byte)) {
	for {
		select {
		case img := <-liveChan:
			image := base64.StdEncoding.EncodeToString(img)
			for _, client := range clients {
				client.send <- RekognitionResult{Name: "", Image: image, IsLive: true}
			}
		}
	}
}
