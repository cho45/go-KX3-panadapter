//#!go run
package main

import (
	"code.google.com/p/portaudio-go/portaudio"
	"fmt"
)

func main() {
	portaudio.Initialize()
	defer portaudio.Terminate()
	devices, err := portaudio.Devices()
	if err != nil {
		panic(err)
	}
	for _, deviceInfo := range devices {
		if deviceInfo.MaxInputChannels == 0 {
			continue
		}
		fmt.Printf("Input device: '%s' %.1fHz\n", deviceInfo.Name, deviceInfo.DefaultSampleRate)
	}
}
