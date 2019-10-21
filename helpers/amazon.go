package helpers

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/blackjack/webcam"
	"github.com/byuoitav/room-auth-ms/structs"
	"github.com/byuoitav/wso2services/wso2requests"
	"github.com/nfnt/resize"
	"github.com/oliamb/cutter"
)

var svc *rekognition.Rekognition

// RekognitionResult contains the name and face of the person recognized
type RekognitionResult struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Type    string `json:"type"`
	Glasses bool   `json:"glasses"`
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
	/*func rekognitionManager(rekognitionChan chan ([]byte)) {
	for {
		select {
		case img := <-rekognitionChan:
	*/
	matches := recognize(img)
	faceDetails := getFeatures(img)

	if len(matches) > 0 {
		fmt.Printf("Matches found: %v\n", len(matches))
		for _, match := range matches {
			//TODO update recognize to also return bounding boxes
			//TODO finish cutter
			//TODO resize image after cutter
			//TODO update html to properly show the faces (maybe an id to know when to reset?
			//TODO update html to have a place to show these sweet face details
			jpg, err := jpeg.Decode(bytes.NewReader(img))
			if err != nil {
				fmt.Printf("error decoding image for cropping: %v\n", err)
				continue
			}
			imgWidth := float64(jpg.Bounds().Max.X - jpg.Bounds().Min.X)
			imgHeight := float64(jpg.Bounds().Max.Y - jpg.Bounds().Min.Y)
			left := int(*faceDetails[0].BoundingBox.Left * imgWidth)
			top := int(*faceDetails[0].BoundingBox.Top * imgHeight)
			width := int(*faceDetails[0].BoundingBox.Width * imgWidth)
			height := int(*faceDetails[0].BoundingBox.Height * imgHeight)
			//fmt.Printf("Width: %v\n Height:%v\n", imgWidth, imgHeight)
			//fmt.Printf("Ratios: \nLeft: %v\nTop: %v\nWidth: %v\nHeight: %v\n", *match.Face.Face.BoundingBox.Left, *match.Face.Face.BoundingBox.Top, *match.Face.Face.BoundingBox.Width, *match.Face.Face.BoundingBox.Height)
			//fmt.Printf("BoundingBox: \nLeft: %v\nTop: %v\nWidth: %v\n Height: %v\n", left, top, width, height)

			croppedImg, err := cutter.Crop(jpg, cutter.Config{
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
			image := base64.StdEncoding.EncodeToString(buf.Bytes())
			fmt.Printf("Found: %v\n", match.Person.Basic.FirstName.Value)
			for _, client := range clients {
				client.send <- RekognitionResult{Name: match.Person.Basic.FirstName.Value, Image: image, Type: "recognized", Glasses: *faceDetails[0].Eyeglasses.Value}
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
				client.send <- RekognitionResult{Name: "", Image: image, Type: "live"}
			}
		}
	}
}

func pigoManager(img []byte, rekognitionChan chan ([]byte)) {
	hasFace, _, err := ImgHasFace(img)
	if err != nil {
		fmt.Printf("Error with yuyv: %v\n", err)
		return
	}
	if hasFace {
		//rekognitionChan <- bytes
		go rekognitionManager(img)
	}
}
