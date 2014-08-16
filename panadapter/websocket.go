package panadapter

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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
	MAX_BUFFER_SIZE := 8
	MIN_BUFFER_SIZE := 2

	localBuffer := make(chan byte, 255)
	deviceBuffer := make([]byte, 10)
	decodeBuffer := make(chan byte)
	sendBuffer := make([]byte, MAX_BUFFER_SIZE)
	pubsub := pubsub.New()

	go func() {
		for {
			ret, err := kx3.Command("TB;", kx3hq.RSP_TB)
			if err != nil {
				continue
			}

			bufferedCount, err := strconv.ParseInt(ret[1], 10, 32)
			if err != nil {
				panic(err)
			}

			// device decoded texts
			for _, char := range ret[3] {
				decodeBuffer <- byte(char)
			}

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

	port := 51234

	http.Handle("/", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("New websocket: %v", ws)
		pubsub.Sub(func (ev *SentEvent) {
			event := &JSONRPCEventResponse{
				Result : &JSONRPCEventResponseResult{
					Event : "sent",
					Value : ev,
				},
			}
			websocket.JSON.Send(ws, event)
		})

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
				res.Result = 20
			case "tone":
				res.Result = true
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
	log.Printf("websocket server listen: %d", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	return err
}
