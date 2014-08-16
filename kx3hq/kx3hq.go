package kx3hq

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/tarm/goserial"
	"io"
	"log"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	MODE_LSB = "1"
	MODE_USB = "2"
	MODE_CW = "3"
	MODE_FM = "4"
	MODE_AM = "5"
	MODE_DATA = "6"
	MODE_CW_REV = "7"
	MODE_DATA_REV = "8"
)

const (
	STATUS_INIT = iota
	STATUS_OPENED
	STATUS_CLOSED
)

var (
	RSP_TB = regexp.MustCompile("^TB([0-9])([0-9]{2})(.*);$")
	RSP_MD = regexp.MustCompile("^MD([0-9]);$")
	RSP_FA = regexp.MustCompile("^FA([0-9]{11});$")
)

type KX3Controller struct {
	port       io.ReadWriteCloser
	resultCh   chan string
	writeCh    chan string
	writeResCh chan error
	reader     *bufio.Reader
	status     int
	mutex      *sync.Mutex
}

func (s *KX3Controller) Open(name string, baudrate int) error {
	s.status = STATUS_INIT
	port, err := serial.OpenPort(&serial.Config{
		Name: name,
		Baud: baudrate,
	})
	if err != nil {
		s.status = STATUS_CLOSED
		return err
	}
	s.port = port
	s.mutex = &sync.Mutex{}
	s.resultCh = make(chan string)
	s.writeCh = make(chan string, 1)
	s.writeResCh = make(chan error, 1)
	s.status = STATUS_OPENED

	// reader thread
	go func() {
		reader := bufio.NewReaderSize(s.port, 4096)
		for s.status == STATUS_OPENED {
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
		for s.status == STATUS_OPENED {
			command := <-s.writeCh
			_, err = s.port.Write([]byte(command))
			s.writeResCh <- err
		}
	}()
	return nil
}

// Block until response
// Command("FA;")
// Command("FA00007100000;FA;")
func (s *KX3Controller) Command(command string, re *regexp.Regexp) ([]string, error) {
	if s.status != STATUS_OPENED {
		return nil, errors.New("invalid status")
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
				return nil, errors.New(fmt.Sprintf("regexp unmatched: %v -> \"%s\"", re, ret))
			}
		} else {
			return nil, errors.New("rig is busy")
		}
	case <-time.After(1000 * time.Millisecond):
		return nil, errors.New("timeout")
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

func (s *KX3Controller) Close() error {
	log.Println("KX3Cotroller#Close")
	if s.status != STATUS_CLOSED {
		s.status = STATUS_CLOSED
		err := s.port.Close()
		close(s.resultCh)
		close(s.writeCh)
		close(s.writeResCh)
		log.Printf("Closed with: %s", err)
		return err
	} else {
		return errors.New("already closed")
	}
}
