//#!go build github.com/cho45/go-KX3-panadapter/kx3hq && go run

package main

import (
	"container/ring"
	"errors"
	"fmt"
	"github.com/cho45/go-KX3-panadapter/kx3hq"
	"log"
	"math"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"time"

	"code.google.com/p/portaudio-go/portaudio"
	"github.com/andrebq/gas"
	"github.com/go-gl/gl"
	"github.com/go-gl/glfw"
	"github.com/go-gl/gltext"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

var (
	running    bool
	sampleRate float64
	fonts      [16]*gltext.Font

	fftSize int
	buffer  *ring.Ring

	kx3          *kx3hq.KX3Controller
	rigFrequency float64
	rigMode      string
)

func listen(fftSize int) chan []float64 {
	ch := make(chan []float64, 1)
	phaseI := make([]float64, fftSize)
	phaseQ := make([]float64, fftSize)
	complexIQ := make([]complex128, fftSize)
	fftResult := make([]float64, fftSize)
	go func() {
		portaudio.Initialize()
		defer portaudio.Terminate()

		//		devices, err := portaudio.Devices()
		//		var device *portaudio.DeviceInfo
		//		for _, deviceInfo := range devices {
		//			if deviceInfo.MaxInputChannels >= 2 {
		//				device = deviceInfo
		//				break
		//			}
		//		}
		//
		//		if device != nil {
		//			log.Printf("Use %v", device)
		//		} else {
		//			log.Fatalf("No devices found with stereo input")
		//			for _, deviceInfo := range devices {
		//				log.Fatalf("%v", deviceInfo)
		//			}
		//		}
		//
		//		in := make([]int32, fftSize)
		//
		//		stream, err := portaudio.OpenStream(portaudio.StreamParameters{
		//			Input: portaudio.StreamDeviceParameters{
		//				Device:   device,
		//				Channels: 2,
		//				Latency:  device.DefaultHighInputLatency,
		//			},
		//			Output: portaudio.StreamDeviceParameters{
		//				Device:   nil,
		//				Channels: 0,
		//				Latency:  0,
		//			},
		//			SampleRate:      device.DefaultSampleRate,
		//			FramesPerBuffer: len(in),
		//			Flags:           portaudio.NoFlag,
		//		}, in)
		//		if err != nil {
		//			panic(err)
		//		}
		//		defer stream.Close()

		device, err := portaudio.DefaultInputDevice()
		if err != nil {
			panic(err)
		}

		sampleRate = device.DefaultSampleRate

		in := make([]int32, fftSize)
		stream, err := portaudio.OpenDefaultStream(2, 0, device.DefaultSampleRate, len(in), in)
		if err != nil {
			panic(err)
		}
		defer stream.Close()

		if err = stream.Start(); err != nil {
			panic(err)
		}
		defer stream.Stop()

		for running {
			if err = stream.Read(); err != nil {
				log.Printf("portaudio: stream.Read() failed: %s", err)
				continue
			}

			for i := 0; i < len(in); i += 2 {
				// left
				phaseI[i/2] = float64(in[i]) / 0x1000000
				// right
				phaseQ[i/2] = float64(in[i+1]) / 0x1000000
			}

			windowFunc := window.Hamming
			window.Apply(phaseI, windowFunc)
			window.Apply(phaseQ, windowFunc)

			for i := 0; i < fftSize; i++ {
				complexIQ[i] = complex(phaseI[i], phaseQ[i])
			}

			result := fft.FFT(complexIQ)
			// real
			for i := 0; i < len(complexIQ)/2; i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i+len(complexIQ)/2] = 20 * math.Log10(power)
			}
			// imag
			for i := len(complexIQ) / 2; i < len(complexIQ); i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i-len(complexIQ)/2] = 20 * math.Log10(power)
			}

			ch <- fftResult
		}
	}()

	return ch
}

func main() {
	var err error

	go func() {
		kx3 = &kx3hq.KX3Controller{}
		if err := kx3.Open("/dev/tty.usbserial-A402PY11", 38400); err != nil {
			panic(err)
		}
		log.Printf("Connected")
		defer kx3.Close()
		var mode string
		var err error
		var freqI string
		var freq float64
		var match bool
		for {
			mode, err = kx3.Command("MD;")
			if err != nil {
				log.Printf("Error on command: %s", err)
			}
			match, err = regexp.MatchString("^MD.;", mode)
			if match {
				rigMode = mode[2:3]
			}

			freqI, err = kx3.Command("FA;")
			if err != nil {
				log.Printf("Error on command: %s", err)
			}
			match, err = regexp.MatchString("^FA", freqI)
			if match {
				freq, err = strconv.ParseFloat(freqI[2:13], 64)
				ShiftFFTHistory(freq - rigFrequency)
				rigFrequency = freq
				if err != nil {
					panic(err)
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	fftSize = 2048

	height := 600
	width := int(float32(height) * 2.35) // CinemaScope
	historySize := 500
	fftBinSize := fftSize
	dynamicRange := 80.0

	runtime.GOMAXPROCS(runtime.NumCPU())
	running = true

	if err = glfw.Init(); err != nil {
		log.Fatalf("%v\n", err)
		return
	}
	defer glfw.Terminate()

	if err = glfw.OpenWindow(width, height, 8, 8, 8, 8, 0, 0, glfw.Windowed); err != nil {
		log.Fatalf("%v\n", err)
		return
	}
	defer glfw.CloseWindow()

	glfw.SetWindowTitle("Go KX3 Panadapter")
	glfw.SetSwapInterval(1)
	glfw.SetKeyCallback(onKey)
	glfw.SetMouseButtonCallback(onMouseBtn)
	glfw.SetWindowSizeCallback(onResize)

	fftResultChan := listen(fftSize)

	buffer = ring.New(historySize)
	buffer.Value = make([]byte, fftBinSize*3)
	for p := buffer.Next(); p != buffer; p = p.Next() {
		p.Value = make([]byte, fftBinSize*3)
	}

	historyBitmap := make([]byte, fftBinSize*historySize*3)

	texture := gl.GenTexture()

	file, err := gas.Abs("code.google.com/p/freetype-go/testdata/luxisr.ttf")
	if err != nil {
		log.Printf("Find font file: %v", err)
		return
	}
	for i := range fonts {
		fonts[i], err = loadFont(file, int32(i+12))
		if err != nil {
			log.Printf("LoadFont: %v", err)
			return
		}
		defer fonts[i].Release()
	}

	for running && glfw.WindowParam(glfw.Opened) == 1 {
		fftResult := <-fftResultChan
		current := buffer.Value.([]byte)

		for i := 0; i < fftBinSize; i++ {
			p := fftResult[i] / dynamicRange
			if p < 0 {
				p = 0
			}

			r := 0.0
			g := 0.0
			b := 0.0

			switch {
			case p > 5.0/6.0:
				// yellow -> red
				p = (p - (5 / 6.0)) / (1 / 6.0)
				r = 255
				g = 255 * p
				b = 255 * p
			case p > 4.0/6.0:
				// yellow -> red
				p = (p - (4 / 6.0)) / (1 / 6.0)
				r = 255
				g = 255 * (1 - p)
				b = 0
			case p > 3.0/6.0:
				// green -> yellow
				p = (p - (3 / 6.0)) / (1 / 6.0)
				r = 255 * p
				g = 255
				b = 0
			case p > 2.0/6.0:
				// light blue -> green
				p = (p - (2 / 6.0)) / (1 / 6.0)
				r = 0
				g = 255
				b = 255 * (1 - p)
			case p > 1.0/6.0:
				// blue -> light blue
				p = (p - (1 / 6.0)) / (1 / 6.0)
				r = 0
				g = 255 * p
				b = 255
			case p > 0:
				// black -> blue
				p = p / (1 / 6.0)
				r = 0
				g = 0
				b = 255 * p
			}

			current[i*3] = byte(r)
			current[i*3+1] = byte(g)
			current[i*3+2] = byte(b)
		}

		gl.Clear(gl.COLOR_BUFFER_BIT)

		// draw fft history
		i := 0
		buffer.Do(func(v interface{}) {
			copy(historyBitmap[i:], v.([]byte))
			i += fftBinSize * 3
		})

		gl.PushMatrix()
		gl.Translatef(-1.0, -1.0, 0.0)
		gl.Enable(gl.TEXTURE_2D)
		texture.Bind(gl.TEXTURE_2D)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB, fftBinSize, historySize, 0, gl.RGB, gl.UNSIGNED_BYTE, historyBitmap)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.Begin(gl.QUADS)
		gl.Color3f(1.0, 1.0, 1.0)
		gl.TexCoord2d(0, 0)
		gl.Vertex2d(0, 0.5)
		gl.TexCoord2d(1, 0)
		gl.Vertex2d(2, 0.5)
		gl.TexCoord2d(1, 1)
		gl.Vertex2d(2, 2)
		gl.TexCoord2d(0, 1)
		gl.Vertex2d(0, 2)
		gl.End()
		texture.Unbind(gl.TEXTURE_2D)

		gl.PopMatrix()

		// draw grid
		withPixelContext(func() {
			w, h := GetWindowSizeF()
			gl.Begin(gl.LINES)
			for freq := 0.0; freq < sampleRate/2; freq += 1000.0 {
				if int(freq)%5000 == 0 {
					gl.Color4f(1.0, 1.0, 1.0, 0.3)
				} else {
					gl.Color4f(1.0, 1.0, 1.0, 0.1)
				}
				gl.Vertex2f(w/2.0+w*float32(freq/sampleRate), h*0.75)
				gl.Vertex2f(w/2.0+w*float32(freq/sampleRate), h)
				gl.Vertex2f(w/2.0-w*float32(freq/sampleRate), h*0.75)
				gl.Vertex2f(w/2.0-w*float32(freq/sampleRate), h)
			}
			gl.End()
		})

		// draw fft
		gl.PushMatrix()
		gl.Translatef(-1.0, -1.0, 0.0)
		gl.Color3f(0.2, 0.2, 0.7)
		gl.Begin(gl.LINE_STRIP)
		unit := 2.0 / float64(len(fftResult))
		for i := 0; i < len(fftResult); i++ {
			x := float64(i) * unit
			p := fftResult[i] / dynamicRange
			if p < 0 {
				p = 0
			}
			gl.Vertex2d(x, p*0.5)
		}
		gl.End()
		gl.PopMatrix()

		// draw Text
		withPixelContext(func() {
			// Frequency labels
			w, h := GetWindowSizeF()
			for freq := 0.0; freq < sampleRate/2; freq += 5000.0 {
				drawString(w/2.0+w*float32(freq/sampleRate), h*0.75, 12, fmt.Sprintf("%fMHz", (rigFrequency+freq)/1000/1000))
				drawString(w/2.0-w*float32(freq/sampleRate), h*0.75, 12, fmt.Sprintf("%fMHz", (rigFrequency-freq)/1000/1000))
			}

			//			gl.Color4f(1, 1, 1, 1)
			//			gl.Rectd(10, 10, 100, 100)
			//			gl.Rectd(-10, -10, 10, 10)

			// Mouse
			x, y := glfw.MousePos()
			_, ry := RelativeMousePos()
			fx := float32(x)
			fy := float32(y)
			if ry < 0.5 {
				fy += 50.0
			} else {
				fy -= 50.0
			}
			drawString(fx, fy, 12, fmt.Sprintf("%fMHz", FreqFromMousePos()/1000/1000))
		})

		// done
		glfw.SwapBuffers()

		buffer = buffer.Prev()
	}
}

func withPixelContext(cb func()) {
	w, h := glfw.WindowSize()

	gl.MatrixMode(gl.PROJECTION)
	gl.PushMatrix()
	gl.LoadIdentity()
	gl.Ortho(0, float64(w), float64(h), 0, -1.0, 1.0)
	gl.MatrixMode(gl.MODELVIEW)
	gl.PushMatrix()
	gl.LoadIdentity()
	defer func() {
		gl.PopMatrix()
		gl.MatrixMode(gl.PROJECTION)
		gl.PopMatrix()
	}()

	cb()
}

func onResize(w int, h int) {
	fmt.Printf("resize %d, %d\n", w, h)
	gl.DrawBuffer(gl.FRONT_AND_BACK)

	gl.MatrixMode(gl.PROJECTION)
	gl.LoadIdentity()
	gl.Viewport(0, 0, w, h)
	gl.Ortho(0, float64(w), float64(h), 0, -1.0, 1.0)
	gl.ClearColor(0, 0, 0, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.LoadIdentity()
}

func onMouseBtn(button, state int) {
	//	mouse[button] = state
	x, y := RelativeMousePos()
	fmt.Printf("onMouseBtn %d %d / x:%f, y:%f / %fHz\n", button, state, x, y, FreqFromMousePos())

	switch {
	case button == glfw.MouseLeft && state == glfw.KeyPress:
		freq := FreqFromMousePos()
		switch rigMode {
		case kx3hq.MODE_CW_REV:
			freq -= 600
		case kx3hq.MODE_CW:
			freq += 600
		default:
		}

		ret, err := kx3.Command(fmt.Sprintf("FA%011d;FA;", int(freq)))
		fmt.Printf("change: %s, %s", ret, err)
	}
}

func onKey(key, state int) {
	switch key {
	case glfw.KeyEsc:
		running = state == 0
	case 67: // 'c'
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}
}

func loadFont(file string, scale int32) (*gltext.Font, error) {
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer fd.Close()

	return gltext.LoadTruetype(fd, scale, 32, 127, gltext.LeftToRight)
}

func drawString(x, y float32, size int, str string) error {
	font := fonts[size-12]
	if font == nil {
		return errors.New("undefined size")
	}

	gl.Enable(gl.BLEND)

	sw, sh := font.Metrics(str)
	gl.Color4f(0.0, 0.0, 0.0, 0.7)
	gl.Rectf(x-5, y, x+float32(sw)+5, y+float32(sh))

	gl.Color3d(1.0, 1.0, 1.0)
	err := font.Printf(x, y, "%s", str)
	if err != nil {
		return err
	}
	// font.Printf does not unbind texture
	gl.Disable(gl.TEXTURE_2D)

	return nil
}

func RelativeMousePos() (float32, float32) {
	x, y := glfw.MousePos()
	w, h := GetWindowSizeF()
	return float32(x)/float32(w) - 0.5, float32(y) / float32(h)
}

func GetWindowSizeF() (float32, float32) {
	w, h := glfw.WindowSize()
	return float32(w), float32(h)
}

func FreqFromMousePos() float64 {
	x, _ := RelativeMousePos()
	return rigFrequency + float64(x*float32(sampleRate))
}

func ShiftFFTHistory (freqDiff float64) {
	if freqDiff == 0.0 {
		return
	}

	freqRes := sampleRate / float64(fftSize)
	shift := int(freqDiff/freqRes) * 3
	// log.Printf("shift %d", shift)
	buffer.Do(func(v interface{}) {
		bytes := v.([]byte)
		if shift < 0 {
			for i := len(bytes)-1; -shift < i; i-- {
				bytes[i] = bytes[i+shift]
			}
			for i := 0; i < -shift; i++ {
				bytes[i] = 0
			}
		} else {
			for i := 0; i < len(bytes)-shift; i++ {
				bytes[i] = bytes[i+shift]
			}
			for i := len(bytes) - shift; i < len(bytes); i++ {
				bytes[i] = 0
			}
		}
	})
}
