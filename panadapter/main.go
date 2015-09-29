package panadapter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/http"
	//	"time"

	"github.com/andrebq/gas"
	"github.com/cho45/go-KX3-panadapter/kx3hq"
	"github.com/gordonklaus/portaudio"
	"github.com/mjibson/go-dsp/fft"
	"github.com/mjibson/go-dsp/window"
	"golang.org/x/net/websocket"
)

type JSONRPCRequest struct {
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
	Id     uint                   `json:"id"`
}

type JSONRPCResponse struct {
	Id     uint        `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type JSONRPCEventResponse struct {
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
}

type Server struct {
	Config *Config

	running   bool
	kx3       *kx3hq.KX3Controller
	fftResult chan []float64
	sessions  []*ServerSession
}

type ServerSession struct {
	ws *websocket.Conn

	byteOrder   binary.ByteOrder
	initialized bool
}

func (self *Server) Init() {
	self.running = true
	self.fftResult = make(chan []float64, 1)
	self.sessions = make([]*ServerSession, 0)
}

func (self *Server) Start() error {
	portaudio.Initialize()
	defer portaudio.Terminate()

	var err error

	if err = self.startAudio(); err != nil {
		return err
	}

	//	go func() {
	//		result := make([]float64, 1024)
	//		for {
	//			for i, _ := range result {
	//				result[i] = float64(i) * 0.01
	//			}
	//			self.fftResult <- result
	//			time.Sleep(1 * time.Second)
	//		}
	//	}()

	if err = self.startHttp(); err != nil {
		return err
	}

	return nil
}

func (self *Server) startHttp() error {
	http.Handle("/stream", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("New websocket: %v", ws)

		session := &ServerSession{ws: ws}

		self.sessions = append(self.sessions, session)
		var req *JSONRPCRequest
		for {
			if err := websocket.JSON.Receive(ws, &req); err != nil {
				log.Printf("Error %v", err)
				break
			}
			log.Printf("Request: %v", req)
			res := &JSONRPCResponse{Id: req.Id}

			switch req.Method {
			case "init":
				if req.Params["byteOrder"].(string) == "BIG_ENDIAN" {
					session.byteOrder = binary.BigEndian
				} else {
					session.byteOrder = binary.LittleEndian
				}
				session.initialized = true

				res.Result = map[string]interface{}{
					"config": self.Config,
				}
			case "echo":
				self.handleEcho(req, res, session)
			default:
				res.Error = "unkown method"
			}

			log.Printf("Response: %v", res)
			websocket.JSON.Send(ws, res)
		}
		log.Printf("Closed websocket: %v", ws)
		for i, v := range self.sessions {
			if v == session {
				// remove this socket
				self.sessions = append(self.sessions[:i], self.sessions[i+1:]...)
				break
			}
		}
	}))

	go func() {
		bufferLittle := new(bytes.Buffer)
		bufferBig := new(bytes.Buffer)

		getBytes := func(result []float64, byteOrder binary.ByteOrder) []byte {
			var buffer *bytes.Buffer
			if byteOrder == binary.BigEndian {
				buffer = bufferBig
			} else {
				buffer = bufferLittle
			}

			if buffer.Len() > 0 {
				return buffer.Bytes()
			}

			err := binary.Write(buffer, byteOrder, result)
			if err != nil {
				fmt.Println("binary.Write failed:", err)
			}
			return buffer.Bytes()
		}

		for self.running {
			result := <-self.fftResult

			bufferLittle.Reset()
			bufferBig.Reset()

			for _, session := range self.sessions {
				if !session.initialized {
					continue
				}
				websocket.Message.Send(session.ws, getBytes(result, session.byteOrder))
			}
		}
	}()

	var err error

	dir, err := gas.Abs("github.com/cho45/go-KX3-panadapter/static")
	if err != nil {
		panic(err)
	}
	http.Handle("/", http.FileServer(http.Dir(dir)))

	log.Printf("websocket server listen: %d", self.Config.Server.Listen)
	err = http.ListenAndServe(self.Config.Server.Listen, nil)
	return err
}

func (self *Server) handleEcho(req *JSONRPCRequest, res *JSONRPCResponse, session *ServerSession) {
	res.Result = req.Params
}

func (self *Server) openAudioStream(in []int32) (*portaudio.Stream, error) {
	var device *portaudio.DeviceInfo
	var stream *portaudio.Stream
	var err error

	if self.Config.Input != nil {
		devices, err := portaudio.Devices()
		for _, deviceInfo := range devices {
			if deviceInfo.Name == self.Config.Input.Name {
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
				SampleRate:      self.Config.Input.SampleRate,
				FramesPerBuffer: len(in),
				Flags:           portaudio.NoFlag,
			}, in)
			if err != nil {
				return nil, err
			}
		} else {
			log.Printf("No matched devices: (required: '%s')", self.Config.Input.Name)
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
			return nil, err
		}

		stream, err = portaudio.OpenDefaultStream(2, 0, device.DefaultSampleRate, len(in), in)
		if err != nil {
			return nil, err
		}
	}

	return stream, nil
}

func (self *Server) startAudio() error {
	in := make([]int32, self.Config.FftSize)
	stream, err := self.openAudioStream(in)
	if err = stream.Start(); err != nil {
		return err
	}

	go func() {
		defer stream.Close()
		defer stream.Stop()

		fftSize := self.Config.FftSize
		halfFftSize := fftSize / 2
		phaseI := make([]float64, fftSize)
		phaseQ := make([]float64, fftSize)
		complexIQ := make([]complex128, fftSize)
		fftResult := make([]float64, fftSize)
		fftCorrection := func(freq float64) float64 {
			return math.Pow(2.0, freq/41000)
		}
		fftBinBandWidth := stream.Info().SampleRate / float64(fftSize)

		for self.running {
			if err = stream.Read(); err != nil {
				log.Printf("portaudio: stream.Read() failed: %s", err)
				continue
			}

			if len(self.sessions) == 0 {
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
			for i := 0; i < halfFftSize; i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i+halfFftSize] = 20 * math.Log10(power*fftCorrection(float64(i)*fftBinBandWidth))
			}
			// imag
			for i := halfFftSize; i < fftSize; i++ {
				power := math.Sqrt(real(result[i])*real(result[i]) + imag(result[i])*imag(result[i]))
				fftResult[i-halfFftSize] = 20 * math.Log10(power*fftCorrection(float64(fftSize-i)*fftBinBandWidth))
			}

			self.fftResult <- fftResult
		}
	}()
	return nil
}
