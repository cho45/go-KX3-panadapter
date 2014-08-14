package kx3hq

import (
	"bufio"
	"errors"
	"github.com/tarm/goserial"
	"io"
	"log"
	"sync"
	"time"
)

const MODE_LSB = "1"
const MODE_USB = "2"
const MODE_CW = "3"
const MODE_FM = "4"
const MODE_AM = "5"
const MODE_DATA = "6"
const MODE_CW_REV = "7"
const MODE_DATA_REV = "8"

//const MODE = map[string]string{
//	"1": "LSB",
//	"2": "USB",
//	"3": "CW",
//	"4": "FM",
//	"5": "AM",
//	"6": "DATA",
//	"7": "CW-REV",
//	"9": "DATA-REV",
//}

type KX3Controller struct {
	port     io.ReadWriteCloser
	resultCh chan string
	reader   *bufio.Reader
	mutex    *sync.Mutex
}

func (s *KX3Controller) Open(name string, baudrate int) error {
	port, err := serial.OpenPort(&serial.Config{
		Name: name,
		Baud: baudrate,
	})
	if err != nil {
		return err
	}
	s.port = port
	s.resultCh = make(chan string)
	s.mutex = &sync.Mutex{}

	go func() {
		reader := bufio.NewReaderSize(s.port, 4096)
		// read go routine
		defer close(s.resultCh)
		defer s.port.Close()
		for {
			command, err := reader.ReadString(';')
			if err != nil {
				if err == io.EOF {
					break
				} else {
					break
				}
			}
			s.resultCh <- command
		}
		log.Println("reader thread is done")
	}()
	return nil
}

func (s *KX3Controller) Command(command string) (string, error) {
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
	_, err := s.port.Write([]byte(command))
	if err != nil {
		return "", err
	}

	select {
	case ret := <-s.resultCh:
		return ret, nil
	case <-time.After(1000 * time.Millisecond):
		return "", errors.New("timeout")
	}
}

func (s *KX3Controller) Close() error {
	return s.port.Close()
}
