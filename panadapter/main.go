package panadapter

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

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
	Result *JSONRPCEventResponseResult `json:"result"`
	Error  interface{}                 `json:"error"`
}

type JSONRPCEventResponseResult struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type Server struct {
	Config *Config

	running   bool
	kx3       *kx3hq.KX3Controller
	fftResult chan []float64
	sessions  []*ServerSession

	rigMode      string
	rigFrequency float64
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

	if err = self.startSerial(); err != nil {
		return err
	}

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
					"config":       self.Config,
					"rigFrequency": self.rigFrequency,
					"rigMode":      self.rigMode,
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

func (self *Server) broadcastNotification(event *JSONRPCEventResponse) {
	for _, session := range self.sessions {
		if !session.initialized {
			continue
		}
		websocket.JSON.Send(session.ws, event)
	}
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

func (self *Server) startSerial() error {
	connect := func() {
		self.kx3 = kx3hq.New()
		if err := self.kx3.Open(self.Config.Port.Name, self.Config.Port.Baudrate); err != nil {
			log.Printf("Error on Open: %s", err)
			return
		}
		log.Printf("Connected")
		self.kx3.StartTextBufferObserver()
		defer self.kx3.Close()
		var err error
		var freq float64
		var matched []string
		for self.running {
			matched, err = self.kx3.Command("MD;", kx3hq.RSP_MD)
			if err != nil {
				// timeout when KX3 does not respond (eg. changing band)
				log.Printf("Error on command: %s", err)
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			self.rigMode = matched[1]

			matched, err = self.kx3.Command("FA;", kx3hq.RSP_FA)
			if err != nil {
				log.Printf("Error on command: %s", err)
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			freq, err = strconv.ParseFloat(matched[1], 64)
			if freq != self.rigFrequency {
				self.rigFrequency = freq
				self.broadcastNotification(&JSONRPCEventResponse{
					Result: &JSONRPCEventResponseResult{
						Type: "frequencyChanged",
						Data: map[string]interface{}{
							"rigFrequency": self.rigFrequency,
						},
					},
				})
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	go func() {
		for self.running {
			connect()
			log.Printf("Sleep 3sec for retrying")
			time.Sleep(3000 * time.Millisecond)
		}
	}()

	return nil
}
