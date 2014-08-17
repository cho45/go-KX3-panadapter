package panadapter

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/andrebq/gas"
	"code.google.com/p/go.net/websocket"
	"github.com/cho45/go-KX3-panadapter/kx3hq"
	"github.com/mattn/go-pubsub"
)

type JSONRPCRequest struct {
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
	Id     uint          `json:"id"`
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

type JSONRPCEventResponseResult struct {
	Event string `json:"event"`
	Value interface{} `json:"value"`
}

type SentEvent struct {
	Char string `json:"char"`
	Buffer int `json:"buffer"`
}


func ServWebSocket() error {
	MAX_BUFFER_SIZE := 9
	MIN_BUFFER_SIZE := 5

	localBuffer := make(chan byte, 255)
	deviceBuffer := make([]byte, 0)
	// decodeBuffer := make(chan byte)
	sendBuffer := make([]byte, MAX_BUFFER_SIZE)
	pubsub := pubsub.New()

	go func() {
		for {
			ret, err := kx3.Command("TB;", kx3hq.RSP_TB)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			bufferedCount, err := strconv.ParseInt(ret[1], 10, 32)
			if err != nil {
				panic(err)
			}

			// device decoded texts
			// for _, char := range ret[3] {
			//	decodeBuffer <- byte(char)
			// }

			sent := deviceBuffer[:len(deviceBuffer)-int(bufferedCount)]
			deviceBuffer = deviceBuffer[len(deviceBuffer)-int(bufferedCount):]
			for _, char := range sent {
				pubsub.Pub(&SentEvent{
					Char : string(char),
					Buffer: int(bufferedCount),
				})
			}

			// fill device text buffer
			if int(bufferedCount) <= MIN_BUFFER_SIZE && 0 < len(localBuffer) {
			exhaust:
				for {
					select {
					case char := <-localBuffer:
						sendBuffer = append(sendBuffer, char)
						if len(sendBuffer) >= MAX_BUFFER_SIZE {
							break exhaust
						}
					default:
						break exhaust
					}
				}
				if len(sendBuffer) > 0 {
					log.Printf("Sending... %s", sendBuffer)
					kx3.Send(fmt.Sprintf("KY %s;", sendBuffer))
					deviceBuffer = append(deviceBuffer, sendBuffer...)
					sendBuffer = sendBuffer[:0]
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	send := func(text string) {
		for _, char := range text {
			localBuffer <- byte(char)
		}
	}

	http.Handle("/cw", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("New websocket: %v", ws)
		subFunc := func (ev *SentEvent) {
			event := &JSONRPCEventResponse{
				Result : &JSONRPCEventResponseResult{
					Event : "sent",
					Value : ev,
				},
			}
			websocket.JSON.Send(ws, event)
		}

		pubsub.Sub(subFunc)
		defer pubsub.Leave(subFunc)

		var req JSONRPCRequest
		for {
			if err := websocket.JSON.Receive(ws, &req); err != nil {
				break
			}
			log.Printf("Request: %v", req)
			res := &JSONRPCResponse{Id: req.Id}

			switch req.Method {
			case "device_buffer":
				res.Result = string(deviceBuffer)
			case "speed":
				s, ok := req.Params[0].(float64)
				if ok {
					ret, err := kx3.Command(fmt.Sprintf("KS%03d;KS;", int(s)), kx3hq.RSP_KS)
					fmt.Printf("SetSpeed: %v %v", ret, err)
					if err == nil {
						res.Result = ret[1]
					} else {
						res.Error = err.Error()
					}
				} else {
					ret, err := kx3.Command(fmt.Sprintf("KS;"), kx3hq.RSP_KS)
					fmt.Printf("GetSpeed: %v %v", ret, err)
					if err == nil {
						res.Result = ret[1]
					} else {
						res.Error = err.Error()
					}
				}
			case "tone":
				ret, err := kx3.Command(fmt.Sprintf("CW;"), kx3hq.RSP_CW)
				fmt.Printf("GetTone: %v %v", ret, err)
				if err == nil {
					pitch, err := strconv.ParseUint(ret[1], 10, 32)
					if err == nil {
						res.Result = pitch * 10
					} else {
						res.Error = err.Error()
					}
				} else {
					res.Error = err.Error()
				}
			case "send":
				s, ok := req.Params[0].(string)
				if ok {
					send(s)
				} else {
					res.Error = "invalid request params"
				}
				res.Result = ok
			case "stop":
			clear:
				for {
					select {
					case <-localBuffer:
					default:
						break clear
					}
				}
				kx3.Send("RX;")
				res.Result = true
			case "back":
				res.Result = true
			default:
				res.Error = "unkown method"
			}
			log.Printf("Response: %v", res)
			websocket.JSON.Send(ws, res)
		}
		log.Printf("Closed websocket: %v", ws)
	}))

	dir, err := gas.Abs("github.com/cho45/go-KX3-panadapter/cwclient")
	if err != nil {
		panic(err)
	}
	http.Handle("/", http.FileServer(http.Dir(dir)))

	log.Printf("websocket server listen: %s", config.Server.Listen)
	err = http.ListenAndServe(config.Server.Listen, nil)
	if err != nil {
		log.Printf("http.ListenAndServe failed with  %s", err)
	}
	return err
}
