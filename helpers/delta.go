package helpers

import (
	"time"

	pigo "github.com/esimov/pigo/core"
)

var lastFaces []pigo.Detection

// IsDelta checks an incoming picture position and the time and finds
// if it is different enough to be a delta picture
func IsDelta(dets []pigo.Detection, last time.Time) bool {
	if len(lastFaces) == 0 {
		lastFaces = dets
		return false
	}
	if len(dets) > 0 && time.Since(last).Seconds() > 5 {
		for _, det := range dets {
			if det.Q >= 5 {
				return true
			}
		}
		return false
	}

	for _, det := range dets {
		if det.Q < 5 {
			//		print("D")
			continue
		}
		for _, face := range lastFaces {
			if face.Q < 5 {
				//			print("F")
				continue
			}
			if !checkIntersection(det, face) {
				lastFaces = dets
				return true
			}
		}
	}
	return false
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
