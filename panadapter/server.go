package panadapter

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"code.google.com/p/go.net/websocket"
	"github.com/andrebq/gas"
	"github.com/cho45/go-KX3-panadapter/kx3hq"
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
	Event string      `json:"event"`
	Value interface{} `json:"value"`
}

type SentEvent struct {
	Char   string `json:"char"`
	Buffer int    `json:"buffer"`
}

func ServWebSocket() error {
	http.Handle("/cw", websocket.Handler(func(ws *websocket.Conn) {
		log.Printf("New websocket: %v", ws)
		subFunc := func(ev *kx3hq.EventTextSent) {
			event := &JSONRPCEventResponse{
				Result: &JSONRPCEventResponseResult{
					Event: "sent",
					Value: &SentEvent{
						Char:   ev.Text,
						Buffer: ev.BufferCount,
					},
				},
			}
			websocket.JSON.Send(ws, event)
		}

		statusFunc := func(ev *kx3hq.EventStatusChange) {
			switch (ev.Status) {
			case kx3hq.STATUS_OPENED:
				websocket.JSON.Send(ws, &JSONRPCEventResponse{
					Result : &JSONRPCEventResponseResult{
						Event: "opened",
						Value: nil,
					},
				})
			case kx3hq.STATUS_CLOSED:
				websocket.JSON.Send(ws, &JSONRPCEventResponse{
					Result : &JSONRPCEventResponseResult{
						Event: "closed",
						Value: nil,
					},
				})
			}
		}

		kx3.On(subFunc)
		defer kx3.Off(subFunc)

		kx3.On(statusFunc)
		defer kx3.Off(statusFunc)

		ev := "opened"
		if kx3.Status != kx3hq.STATUS_OPENED {
			ev = "closed"
		}
		websocket.JSON.Send(ws, &JSONRPCEventResponse{
			Result : &JSONRPCEventResponseResult{
				Event: ev,
				Value: nil,
			},
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
				res.Result = string(kx3.DeviceBuffer)
			case "speed":
				if len(req.Params) > 0 {
					s, ok := req.Params[0].(float64)
					if ok {
						ret, err := kx3.Command(fmt.Sprintf("KS%03d;KS;", int(s)), kx3hq.RSP_KS)
						fmt.Printf("SetSpeed: %v %v", ret, err)
						if err != nil {
							res.Error = err.Error()
							continue
						}
					}
				}

				ret, err := kx3.Command(fmt.Sprintf("KS;"), kx3hq.RSP_KS)
				fmt.Printf("GetSpeed: %v %v", ret, err)
				if err == nil {
					res.Result = ret[1]
				} else {
					res.Error = err.Error()
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
					kx3.SendText(s)
				} else {
					res.Error = "invalid request params"
				}
				res.Result = ok
			case "stop":
				err := kx3.StopTX()
				if err != nil {
					res.Error = err.Error()
					res.Result = false
				} else {
					res.Result = true
				}
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

	log.Printf("websocket server listen: http://%s/client.html", config.Server.Listen)
	err = http.ListenAndServe(config.Server.Listen, nil)
	if err != nil {
		log.Printf("http.ListenAndServe failed with  %s", err)
	}
	return err
}
