package helpers

import (
	"fmt"
	"time"

	pigo "github.com/esimov/pigo/core"
)

var lastFaces []pigo.Detection

// IsDelta checks an incoming picture position and the time and finds
// if it is different enough to be a delta picture
func IsDelta(dets []pigo.Detection, last time.Time) bool {
	if len(dets) == 0 {
		return false
	}
	if len(lastFaces) == 0 {
		lastFaces = dets
		return true
	}
	if time.Since(last).Seconds() > 5 {
		fmt.Println("By time")
		return true
	}
	intersection := false
	for _, det := range dets {
		for _, face := range lastFaces {
			if checkIntersection(det, face) {
				intersection = true
				break
			}
		}
		if intersection {
			break
		}
	}
	if intersection {
		return false
	}
	lastFaces = dets
	return true
}

func checkIntersection(a, b pigo.Detection) bool {
	if a.Col < b.Col {
		if a.Col+a.Scale > b.Col-b.Scale {
			if a.Row < b.Row {
				if a.Row+a.Scale >= b.Row-b.Scale {
					return true
				}
			} else {
				if b.Row+b.Scale >= a.Row-a.Scale {
					return true
				}
			}
		}
	} else {
		if b.Col+b.Scale > a.Col-a.Scale {
			if a.Row < b.Row {
				if a.Row+a.Scale >= b.Row-b.Scale {
					return true
				}
			} else {
				if b.Row+b.Scale >= a.Row-a.Scale {
					return true
				}
			}
		}
	}
	return false
}
