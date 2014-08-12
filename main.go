//#!go run

package main

import (
	"container/ring"
	"fmt"
	"log"
	"math"
	"runtime"

	"code.google.com/p/portaudio-go/portaudio"
	"github.com/go-gl/gl"
	"github.com/go-gl/glfw"
	"github.com/mjibson/go-dsp/fft"
)

var (
	running bool
)

func listen(fftSize int) chan []float64 {
	ch := make(chan []float64, 1)
	buf := make([]complex128, fftSize)
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
				buf[i/2] = complex(float64(in[i])/0x1000000, float64(in[i+1])/0x1000000)
			}

			// window.Apply(buf, window.Hamming)
			result := fft.FFT(buf)
			// real
			for i := 0; i < len(buf)/2; i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i+len(buf)/2] = 20 * math.Log10(power)
			}
			// imag
			for i := len(buf) / 2; i < len(buf); i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i-len(buf)/2] = 20 * math.Log10(power)
			}

			ch <- fftResult
		}
	}()

	return ch
}

func main() {
	var err error

	runtime.GOMAXPROCS(runtime.NumCPU())
	running = true

	if err = glfw.Init(); err != nil {
		log.Fatalf("%v\n", err)
		return
	}

	defer glfw.Terminate()

	width := 1024
	height := 500
	historySize := 500
	fftSize := 2048
	fftBinSize := fftSize
	dynamicRange := 100.0

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

	buffer := ring.New(historySize)
	buffer.Value = make([]float64, fftBinSize)
	for p := buffer.Next(); p != buffer; p = p.Next() {
		v := make([]float64, fftBinSize)
		p.Value = v
	}

	historyBitmap := make([]byte, fftBinSize*historySize*3)

	texture := gl.GenTexture()

	for running && glfw.WindowParam(glfw.Opened) == 1 {
		fftResult := <-fftResultChan
		copy(buffer.Value.([]float64), fftResult)
		buffer = buffer.Prev()

		gl.Clear(gl.COLOR_BUFFER_BIT)

		// draw fft history
		i := 0
		buffer.Do(func(v interface{}) {
			result := v.([]float64)
			for x := 0; x < fftBinSize; x++ {
				p := result[x] / dynamicRange
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

				historyBitmap[i] = byte(r)
				historyBitmap[i+1] = byte(g)
				historyBitmap[i+2] = byte(b)
				i += 3
			}
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
		gl.Vertex2d(0, 0)
		gl.TexCoord2d(1, 0)
		gl.Vertex2d(2, 0)
		gl.TexCoord2d(1, 1)
		gl.Vertex2d(2, 2)
		gl.TexCoord2d(0, 1)
		gl.Vertex2d(0, 2)
		gl.End()
		texture.Unbind(gl.TEXTURE_2D)

		//		gl.PixelZoom(float32(width)/float32(fftBinSize), float32(height)/float32(historySize))
		//		gl.DrawPixels(fftBinSize, historySize, gl.RGB, gl.UNSIGNED_BYTE, historyBitmap)
		gl.PopMatrix()

		// draw fft
		gl.PushMatrix()
		gl.Translatef(-1.0, -1.0, 0.0)
		gl.Color3f(1.0, 0.0, 0.0)
		gl.Begin(gl.LINE_STRIP)
		unit := 2.0 / float64(len(fftResult))
		for i := 0; i < len(fftResult); i++ {
			x := float64(i) * unit
			p := fftResult[i] / dynamicRange
			if p < 0 {
				p = 0
			}
			gl.Vertex2d(x, p)
		}
		gl.End()
		gl.PopMatrix()

		// draw Text
		withPixelContext(func() {
			gl.PointSize(5.0)
			gl.Begin(gl.POINTS)
			gl.Color3d(1.0, 1.0, 1.0)
			gl.Vertex2d(10, 10)
			gl.End()
		})

		// done
		glfw.SwapBuffers()
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
	gl.ClearColor(1, 1, 1, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.LoadIdentity()
}

func onMouseBtn(button, state int) {
	//	mouse[button] = state
}

func onKey(key, state int) {
	switch key {
	case glfw.KeyEsc:
		running = state == 0
	case 67: // 'c'
		gl.Clear(gl.COLOR_BUFFER_BIT)
	}
}

//package main
//
//import (
//    "fmt"
//)
//
//func sum(a int, b int) int {
//    return a + b
//}
//
//func main() {
//    fmt.Println(sum(1, 2))
//}
//
//import (
//    "fmt"
//    "time"
//)
//
//func throttle(fun func(), wait time.Duration) func() {
//    var cancelCh chan bool = nil
//    return func() {
//        if cancelCh != nil {
//            cancelCh <- true
//        }
//        cancelCh = make(chan bool)
//        go func() {
//            select {
//            case <- time.After(wait):
//                fun();
//            case <- cancelCh:
//                cancelCh = nil
//            }
//        } ()
//    }
//}
//
//func main() {
//    fun := throttle(func () {
//        fmt.Println("Hello, World")
//    }, 1000 * time.Millisecond);
//
//    fun()
//    fun()
//    fun()
//
//    time.Sleep(5 * time.Second)
//}
