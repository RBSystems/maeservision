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
)

var svc *rekognition.Rekognition

// RekognitionResult contains the name and face of the person recognized
type RekognitionResult struct {
	FirstName     string    `json:"firstName"`
	LastName      string    `json:"lastName"`
	Image         string    `json:"image"`
	Type          string    `json:"type"`
	EmotionNames  []string  `json:"emotionNames"`
	EmotionValues []float64 `json:"emotionValues"`
	NetID         string    `json:"netID"`
}

type personFace struct {
	Person structs.WSO2CredentialPerson
	Face   *rekognition.FaceMatch
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
	//go rekognitionManager(rekognitionChan)
	go liveManager(liveChan)

	// Start the Camera
	cam, err := StartCam()
	if err != nil {
		fmt.Printf("error starting cam: %v\n", err)
		os.Exit(1)
	}
	defer cam.Close()

	//Start Streaming
	//	println("Press Enter to start streaming")
	//	fmt.Scanf("\n")
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
			frame, err = FrameToJPEG(frame)
			if err != nil {
				fmt.Printf("Error converting frame to jpeg: %v\n", err)
				continue
			}
			liveChan <- frame
			go pigoManager(frame, rekognitionChan)

		} else if err != nil {
			fmt.Printf("Error reading frame: %v\n", err)
			panic(err.Error())
		}

	}
}

// recognize returns the person who is recognized in the photo
func recognize(bytes []byte) []personFace {
	var toReturn []personFace
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
	if len(resp.FaceMatches) == 0 {
		fmt.Println("No faces found")
		return toReturn
	}
	for _, face := range resp.FaceMatches {
		var person structs.WSO2CredentialResponse
		err := wso2requests.MakeWSO2Request("GET", "https://api.byu.edu:443/byuapi/persons/v3/?credentials.credential_type=NET_ID&credentials.credential_id="+*face.Face.ExternalImageId, "", &person)
		if err != nil {
			fmt.Printf("Error when making WSO2 request %v", err)
			return toReturn
		}

		/*	var person structs.WSO2CredentialPerson
			person.Basic.FirstName.Value = "Caleb"
		*/
		toReturn = append(toReturn, personFace{
			Person: person.Values[0],
			//Person: person,
			Face: face,
		})
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
func rekognitionManager(img []byte) {
	var wg sync.WaitGroup
	wg.Add(1)
	var matches []personFace
	go func() {
		defer wg.Done()
		matches = recognize(img)
	}()
	var faceDetails []*rekognition.FaceDetail
	wg.Add(1)
	go func() {
		defer wg.Done()
		faceDetails = getFeatures(img)
	}()
	wg.Wait()

	if len(matches) > 0 {
		for _, match := range matches {
			fmt.Printf("Confidence: %v\n", *match.Face.Face.Confidence)
			if *match.Face.Face.Confidence < 85 {
				continue
			}
			image := base64.StdEncoding.EncodeToString(img)
			fmt.Printf("Found: %v %v\n", match.Person.Basic.FirstName.Value, match.Person.Basic.Surname.Value)
			var emotionNames []string
			var emotionValues []float64
			for _, emotion := range faceDetails[0].Emotions {
				if *emotion.Confidence < 10 {
					continue
				}
				emotionNames = append(emotionNames, *emotion.Type)
				emotionValues = append(emotionValues, *emotion.Confidence)
			}
			result := RekognitionResult{FirstName: match.Person.Basic.FirstName.Value,
				LastName:      match.Person.Basic.Surname.Value,
				Image:         image,
				Type:          "recognized",
				EmotionNames:  emotionNames,
				EmotionValues: emotionValues,
				NetID:         match.Person.Basic.NetID.Value,
			}
			for _, client := range clients {
				client.send <- result
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
				client.send <- RekognitionResult{Image: image, Type: "live"}
			}
		}
	}
}

func pigoManager(img []byte, rekognitionChan chan ([]byte)) {
	faces, err := DetectFaces(img)
	if err != nil {
		fmt.Printf("Error with yuyv: %v\n", err)
		return
	}
	if len(faces) > 0 {
		for _, face := range faces {
			go rekognitionManager(face)
		}
	}
}
