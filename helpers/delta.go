package helpers

import (
	"fmt"
	"time"

	pigo "github.com/esimov/pigo/core"
)

var lastFaces []pigo.Detection

var last time.Time

const deltaTimeout = 5

// IsDelta checks an incoming picture position and the time and finds
// if it is different enough to be a delta picture
func IsDelta(dets []pigo.Detection) bool {
	if len(dets) == 0 {
		return false
	}
	if len(lastFaces) == 0 {
		lastFaces = dets
		last = time.Now()
		return true
	}
	if time.Since(last).Seconds() > deltaTimeout {
		fmt.Println("By time")
		last = time.Now()
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
	last = time.Now()
	return true
}

func checkIntersection(a, b pigo.Detection) bool {
	aLeft := a.Col - a.Scale/2
	bLeft := b.Col - b.Scale/2
	aWidth := a.Scale
	bWidth := a.Scale
	aTop := a.Row - a.Scale/2
	bTop := b.Row - b.Scale/2
	aHeight := a.Scale
	bHeight := b.Scale
	if aLeft < bLeft {
		if aLeft+aWidth > bLeft {
			if aTop < bTop {
				if aTop+aHeight > bTop {
					return true
				}
			} else {
				if bTop+bHeight > aTop {
					return true
				}
			}
		}
	} else {
		if bLeft+bWidth > aLeft {
			if aTop < bTop {
				if aTop+aHeight >= bTop {
					return true
				}
			} else {
				if bTop+bHeight >= aTop {
					return true
				}
			}
		}
	}
	return false
}
