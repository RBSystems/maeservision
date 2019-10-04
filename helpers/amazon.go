package helpers

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rekognition"
	"github.com/byuoitav/room-auth-ms/structs"
	"github.com/byuoitav/wso2services/wso2requests"
)

var svc *rekognition.Rekognition

func init() {
	sess, err := session.NewSession(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}
	svc = rekognition.New(sess)

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
func RekognitionManager(byteChan chan ([]byte)) {
	for {
		select {
		case img := <-byteChan:
			resp := recognize(img)
			if resp.Basic.NetID.Value != "" {
				fmt.Println(resp.Basic.FirstName.Value)
			}
		}
	}
}
