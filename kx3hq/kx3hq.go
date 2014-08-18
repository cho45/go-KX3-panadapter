package kx3hq

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/mattn/go-pubsub"
	"github.com/tarm/goserial"
)

const (
	MODE_LSB      = "1"
	MODE_USB      = "2"
	MODE_CW       = "3"
	MODE_FM       = "4"
	MODE_AM       = "5"
	MODE_DATA     = "6"
	MODE_CW_REV   = "7"
	MODE_DATA_REV = "8"
)

const (
	STATUS_INIT = iota
	STATUS_OPENED
	STATUS_CLOSED
)

var (
	RSP_CW = regexp.MustCompile("^CW([0-9]{2});$")
	RSP_TB = regexp.MustCompile("^TB([0-9])([0-9]{2})(.*);$")
	RSP_MD = regexp.MustCompile("^MD([0-9]);$")
	RSP_FA = regexp.MustCompile("^FA([0-9]{11});$")
	RSP_KS = regexp.MustCompile("^KS([0-9]{3});$")
	RSP_IS = regexp.MustCompile("^IS(.)([0-9]{4});$")
)

var (
	ErrTimeout       = errors.New("timeout")
	ErrRigIsBusy     = errors.New("rig is busy")
	ErrInvalidStatus = errors.New("invalid status")
	ErrAlreayClosed  = errors.New("already closed")
)

type KX3Controller struct {
	port       io.ReadWriteCloser
	resultCh   chan string
	writeCh    chan string
	writeResCh chan error
	mutex      *sync.Mutex
	pubsub     *pubsub.PubSub
	Status     int

	DeviceBuffer []byte
	localBuffer  chan byte
}

type EventStatusChange struct {
	Status int
}

type EventTextSent struct {
	Text        string
	BufferCount int
}

type EventTextDecoded struct {
	Text string
}

func New() *KX3Controller {
	return &KX3Controller{
		mutex:  &sync.Mutex{},
		pubsub: pubsub.New(),
		Status: STATUS_INIT,
	}
}

func (s *KX3Controller) Open(name string, baudrate int) error {
	s.pubsub.Pub(&EventStatusChange{Status: s.Status})
	port, err := serial.OpenPort(&serial.Config{
		Name: name,
		Baud: baudrate,
	})
	if err != nil {
		s.Status = STATUS_CLOSED
		return err
	}
	s.port = port
	s.resultCh = make(chan string)
	s.writeCh = make(chan string, 1)
	s.writeResCh = make(chan error, 1)
	s.Status = STATUS_OPENED
	s.pubsub.Pub(&EventStatusChange{Status: s.Status})

	// reader thread
	go func() {
		reader := bufio.NewReaderSize(s.port, 4096)
		for s.Status == STATUS_OPENED {
			command, err := reader.ReadString(';')
			if err != nil {
				if err == io.EOF {
					break
				} else {
					panic(err)
				}
			}

			matched := RSP_TB.FindStringSubmatch(command)
			if matched != nil {
				length, err := strconv.ParseInt(matched[2], 10, 32)
				if err != nil {
					s.resultCh <- command
					continue
				}
				remain, err := reader.Peek(int(length) - len(matched[3]))
				if err != nil {
					if err == io.EOF {
						break
					} else {
						panic(err)
					}
				}
				command += string(remain)
			}

			s.resultCh <- command
		}
		log.Println("reader thread is done")
	}()

	// writer thread
	go func() {
		for s.Status == STATUS_OPENED {
			command := <-s.writeCh
			_, err = s.port.Write([]byte(command))
			s.writeResCh <- err
		}
		log.Println("writer thread is done")
	}()

	return nil
}

// Block until response
// Command("FA;")
// Command("FA00007100000;FA;")
func (s *KX3Controller) Command(command string, re *regexp.Regexp) ([]string, error) {
	if s.Status != STATUS_OPENED {
		return nil, ErrInvalidStatus
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
clear:
	for {
		select {
		case <-s.resultCh:
		default:
			break clear
		}
	}
	s.writeCh <- command
	err := <-s.writeResCh
	if err != nil {
		return nil, err
	}

	select {
	case ret := <-s.resultCh:
		if ret != "?;" {
			matched := re.FindStringSubmatch(ret)
			if matched != nil {
				return matched, nil
			} else {
				return nil, fmt.Errorf("regexp unmatched: %v -> \"%s\"", re, ret)
			}
		} else {
			return nil, ErrRigIsBusy
		}
	case <-time.After(1000 * time.Millisecond):
		return nil, ErrTimeout
	}
}

func (s *KX3Controller) Send(command string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.writeCh <- command
	err := <-s.writeResCh
	if err != nil {
		return err
	}
	return nil
}

func (s *KX3Controller) StartTextBufferObserver() {
	MAX_BUFFER_SIZE := 9
	MIN_BUFFER_SIZE := 5

	s.localBuffer = make(chan byte, 255)
	s.DeviceBuffer = make([]byte, 0)
	sendBuffer := make([]byte, MAX_BUFFER_SIZE)

	log.Println("StartTextBufferObserver")
	go func() {
		for s.Status == STATUS_OPENED {
			ret, err := s.Command("TB;", RSP_TB)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			bufferedCount, err := strconv.ParseInt(ret[1], 10, 32)
			if err != nil {
				panic(err)
			}

			if ret[3] != "" {
				// device decoded texts
				s.pubsub.Pub(&EventTextDecoded{Text: ret[3]})
			}

			sent := s.DeviceBuffer[:len(s.DeviceBuffer)-int(bufferedCount)]
			s.DeviceBuffer = s.DeviceBuffer[len(s.DeviceBuffer)-int(bufferedCount):]
			for _, char := range sent {
				s.pubsub.Pub(&EventTextSent{
					Text:        string(char),
					BufferCount: int(bufferedCount),
				})
			}

			// fill device text buffer
			if int(bufferedCount) <= MIN_BUFFER_SIZE && 0 < len(s.localBuffer) {
			exhaust:
				for {
					select {
					case char := <-s.localBuffer:
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
					s.Send(fmt.Sprintf("KY %s;", sendBuffer))
					s.DeviceBuffer = append(s.DeviceBuffer, sendBuffer...)
					sendBuffer = sendBuffer[:0]
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
		log.Println("TextBufferObserver done")
	}()
}

func (s *KX3Controller) SendText(text string) error {
	for _, char := range text {
		s.localBuffer <- byte(char)
	}
	return nil
}

func (s *KX3Controller) StopTX() error {
clear:
	for {
		select {
		case <-s.localBuffer:
		default:
			break clear
		}
	}
	return s.Send("RX;")
}

func (s *KX3Controller) Close() error {
	log.Println("KX3Cotroller#Close")
	if s.Status != STATUS_CLOSED {
		s.Status = STATUS_CLOSED
		// kernel panic
		// err := s.port.Close()
		// log.Printf("Closed with: %s", err)
		close(s.resultCh)
		close(s.writeCh)
		close(s.writeResCh)
		s.pubsub.Pub(&EventStatusChange{Status: s.Status})
		return nil
	} else {
		return ErrAlreayClosed
	}
	return nil
}

func (s *KX3Controller) On(f interface{}) {
	s.pubsub.Sub(f)
}

func (s *KX3Controller) Off(f interface{}) {
	s.pubsub.Leave(f)
}
