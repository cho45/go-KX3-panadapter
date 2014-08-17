package panadapter

import (
	"container/ring"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"

	"code.google.com/p/portaudio-go/portaudio"
	"github.com/andrebq/gas"
	"github.com/cho45/go-KX3-panadapter/kx3hq"
	"github.com/go-gl/gl"
	"github.com/go-gl/glfw"
	"github.com/go-gl/gltext"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
)

var (
	config       *Config
	running      bool
	sampleRate   float64
	dynamicRange float64
	fonts        [16]*gltext.Font

	fftSize int
	buffer  *ring.Ring

	kx3          *kx3hq.KX3Controller
	rigFrequency float64
	rigMode      string

	forceUpdateEntire bool
)

func StartFFT(fftSize int) (chan []float64, chan error) {
	ch := make(chan []float64, 1)
	errCh := make(chan error)

	phaseI := make([]float64, fftSize)
	phaseQ := make([]float64, fftSize)
	complexIQ := make([]complex128, fftSize)
	fftResult := make([]float64, fftSize)
	go func() {
		portaudio.Initialize()
		defer portaudio.Terminate()

		in := make([]int32, fftSize)

		var device *portaudio.DeviceInfo
		var stream *portaudio.Stream
		var err error

		if config.Input != nil {
			devices, err := portaudio.Devices()
			for _, deviceInfo := range devices {
				if deviceInfo.Name == config.Input.Name {
					device = deviceInfo
					break
				}
			}

			if device != nil {
				log.Printf("Use %v", device)
				stream, err = portaudio.OpenStream(portaudio.StreamParameters{
					Input: portaudio.StreamDeviceParameters{
						Device:   device,
						Channels: 2,
						Latency:  device.DefaultHighInputLatency,
					},
					Output: portaudio.StreamDeviceParameters{
						Device:   nil,
						Channels: 0,
						Latency:  0,
					},
					SampleRate:      config.Input.SampleRate,
					FramesPerBuffer: len(in),
					Flags:           portaudio.NoFlag,
				}, in)
				if err != nil {
					errCh <- err
					return
				}
				defer stream.Close()
			} else {
				log.Printf("No matched devices: (required: '%s')", config.Input.Name)
				for _, deviceInfo := range devices {
					log.Printf("Found Device... %s %.1f", deviceInfo.Name, deviceInfo.DefaultSampleRate)
				}
				log.Printf("Fallback to default input device")
			}
		}

		if device == nil {
			device, err = portaudio.DefaultInputDevice()
			log.Printf("Use %v", device)
			if err != nil {
				errCh <- err
				return
			}

			stream, err = portaudio.OpenDefaultStream(2, 0, device.DefaultSampleRate, len(in), in)
			if err != nil {
				errCh <- err
				return
			}
			defer stream.Close()
		}

		sampleRate = stream.Info().SampleRate
		log.Printf("Opened : %s %.1f", device.Name, sampleRate)

		if err = stream.Start(); err != nil {
			errCh <- err
			return
		}
		defer stream.Stop()

		errCh <- nil
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

	return ch, errCh
}

func Serial() {
	connect := func() {
		kx3 = &kx3hq.KX3Controller{}
		if err := kx3.Open(config.Port.Name, config.Port.Baudrate); err != nil {
			log.Printf("Error on Open: %s", err)
			return
		}
		log.Printf("Connected")
		defer kx3.Close()
		var err error
		var freq float64
		var matched []string
		for {
			matched, err = kx3.Command("MD;", kx3hq.RSP_MD)
			if err != nil {
				// timeout when KX3 does not respond (eg. changing band)
				log.Printf("Error on command: %s", err)
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			rigMode = matched[1]

			matched, err = kx3.Command("FA;", kx3hq.RSP_FA)
			if err != nil {
				log.Printf("Error on command: %s", err)
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			freq, err = strconv.ParseFloat(matched[1], 64)
			ShiftFFTHistory(freq - rigFrequency)
			rigFrequency = freq

			time.Sleep(100 * time.Millisecond)
		}
	}

	for {
		connect()
		log.Printf("Sleep 3sec for retrying")
		time.Sleep(3000 * time.Millisecond)
	}
}

func Start(c *Config) {
	var err error
	config = c

	running = true
	dynamicRange = 80.0

	fftSize = config.FftSize
	height := config.Window.Height
	width := config.Window.Width
	historySize := config.HistorySize
	fftBinSize := fftSize

	go Serial()

	if config.Server != nil {
		go ServWebSocket()
	}

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
	gl.Init()
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 8)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 8)

	drawBuffers := make([]gl.Buffer, 2)
	gl.GenBuffers(drawBuffers)
	for _, buffer := range drawBuffers {
		buffer.Bind(gl.PIXEL_UNPACK_BUFFER)
		gl.BufferData(gl.PIXEL_UNPACK_BUFFER, fftBinSize*historySize*4, nil, gl.STREAM_DRAW)
		buffer.Unbind(gl.PIXEL_UNPACK_BUFFER)
	}

	texture := gl.GenTexture()
	texture.Bind(gl.TEXTURE_2D)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, fftBinSize, historySize, 0, gl.BGRA, gl.UNSIGNED_INT_8_8_8_8_REV, nil)
	texture.Unbind(gl.TEXTURE_2D)

	fftResultChan, listenErrCh := StartFFT(fftSize)
	if err = <-listenErrCh; err != nil {
		log.Fatalf("Failed to Open Device with %s", err)
		return
	}

	buffer = ring.New(historySize)
	buffer.Value = make([]uint32, fftBinSize)
	for p := buffer.Next(); p != buffer; p = p.Next() {
		p.Value = make([]uint32, fftBinSize)
	}

	file, err := gas.Abs("github.com/cho45/go-KX3-panadapter/assets/Roboto/Roboto-Bold.ttf")
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

	currentDrawBuffer := 0
	currentDrawLine := 0

	for running && glfw.WindowParam(glfw.Opened) == 1 {
		fftResult := <-fftResultChan
		current := buffer.Value.([]uint32)

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

			// current[i] = uint32(b) << 24 | uint32(g) << 16 | uint32(r) << 8 | 255
			current[i] = 255<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)<<0
		}

		gl.Clear(gl.COLOR_BUFFER_BIT)

		gl.Enable(gl.TEXTURE_2D)

		texture.Bind(gl.TEXTURE_2D)

		if !forceUpdateEntire {
			drawBuffer := drawBuffers[currentDrawBuffer]
			drawBuffer.Bind(gl.PIXEL_UNPACK_BUFFER)
			historyBitmap := *(*[]uint32)(gl.MapBufferSlice(gl.PIXEL_UNPACK_BUFFER, gl.READ_WRITE, 4))
			copy(historyBitmap[currentDrawLine*fftSize:], current)
			gl.UnmapBuffer(gl.PIXEL_UNPACK_BUFFER)
			drawBuffer.Unbind(gl.PIXEL_UNPACK_BUFFER)

			currentDrawLine++
			if currentDrawLine >= historySize {
				currentDrawLine = 0
				currentDrawBuffer = (currentDrawBuffer + 1) % 2
			}
		} else {
			drawBuffer := drawBuffers[0]
			drawBuffer.Bind(gl.PIXEL_UNPACK_BUFFER)
			historyBitmap := *(*[]uint32)(gl.MapBufferSlice(gl.PIXEL_UNPACK_BUFFER, gl.READ_WRITE, 4))
			// draw fft history
			i := 0
			buffer.Do(func(v interface{}) {
				copy(historyBitmap[i:], v.([]uint32))
				i += fftBinSize
			})
			gl.UnmapBuffer(gl.PIXEL_UNPACK_BUFFER)
			drawBuffer.Unbind(gl.PIXEL_UNPACK_BUFFER)

			currentDrawLine = 0
			currentDrawBuffer = 1

			forceUpdateEntire = false
		}

		historyHeight := 1.5
		fftHeight := 2 - historyHeight

		pxHeight := historyHeight / float64(historySize)

		gl.PushMatrix()
		gl.Translated(-1.0, -2.5+(pxHeight*float64(currentDrawLine)), 0.0)
		drawBuffers[currentDrawBuffer].Bind(gl.PIXEL_UNPACK_BUFFER)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, fftBinSize, historySize, gl.BGRA, gl.UNSIGNED_INT_8_8_8_8_REV, nil)
		drawBuffers[currentDrawBuffer].Unbind(gl.PIXEL_UNPACK_BUFFER)
		gl.Begin(gl.QUADS)
		gl.TexCoord2d(0, 1)
		gl.Vertex2d(0, 2-historyHeight)
		gl.TexCoord2d(1, 1)
		gl.Vertex2d(2, 2-historyHeight)
		gl.TexCoord2d(1, 0)
		gl.Vertex2d(2, 2)
		gl.TexCoord2d(0, 0)
		gl.Vertex2d(0, 2)
		gl.End()
		gl.Translated(0, (pxHeight * float64(historySize-1)), 0.0)
		drawBuffers[(currentDrawBuffer+1)%2].Bind(gl.PIXEL_UNPACK_BUFFER)
		gl.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, fftBinSize, historySize, gl.BGRA, gl.UNSIGNED_INT_8_8_8_8_REV, nil)
		drawBuffers[(currentDrawBuffer+1)%2].Unbind(gl.PIXEL_UNPACK_BUFFER)
		gl.Begin(gl.QUADS)
		gl.TexCoord2d(0, 1)
		gl.Vertex2d(0, 0.5)
		gl.TexCoord2d(1, 1)
		gl.Vertex2d(2, 0.5)
		gl.TexCoord2d(1, 0)
		gl.Vertex2d(2, 2)
		gl.TexCoord2d(0, 0)
		gl.Vertex2d(0, 2)
		gl.End()
		gl.PopMatrix()
		texture.Unbind(gl.TEXTURE_2D)

		gl.Color4d(0, 0, 0, 1)
		gl.Rectd(-1, -0.5, 1, -1)

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
			gl.Vertex2d(x, p*fftHeight)
		}
		gl.End()
		gl.PopMatrix()

		// draw Text
		withPixelContext(func() {
			// Frequency labels
			w, h := GetWindowSizeF()
			for freq := 0.0; freq < sampleRate/2; freq += 5000.0 {
				drawString(w/2.0+w*float32(freq/sampleRate)-20, h*0.75, 12, fmt.Sprintf("%fMHz", (rigFrequency+freq)/1000/1000))
				drawString(w/2.0-w*float32(freq/sampleRate)-20, h*0.75, 12, fmt.Sprintf("%fMHz", (rigFrequency-freq)/1000/1000))
			}

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
			drawString(fx, fy, 14, fmt.Sprintf("%fMHz", FreqFromMousePos()/1000/1000))
		})

		// done
		glfw.SwapBuffers()

		buffer = buffer.Next()
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

		ret, err := kx3.Command(fmt.Sprintf("FA%011d;FA;", int(freq)), kx3hq.RSP_FA)
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

func ShiftFFTHistory(freqDiff float64) {
	if freqDiff == 0.0 {
		return
	}

	if math.Abs(freqDiff) < sampleRate {
		freqRes := sampleRate / float64(fftSize)
		shift := int(freqDiff / freqRes)
		// log.Printf("shift %d", shift)
		buffer.Do(func(v interface{}) {
			bytes := v.([]uint32)
			if shift < 0 {
				for i := len(bytes) - 1; -shift < i; i-- {
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
	} else {
		buffer.Do(func(v interface{}) {
			bytes := v.([]uint32)
			for i := 0; i < len(bytes); i++ {
				bytes[i] = 0
			}
		})
	}

	forceUpdateEntire = true
}
