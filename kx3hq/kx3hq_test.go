//#!go test -v github.com/cho45/go-KX3-panadapter/kx3hq ; echo
package kx3hq

import (
	"bufio"
	"fmt"
	"log"
	"io/ioutil"
	"os"
	"testing"

	"github.com/kr/pty"
)

func TestKX3HQ(t *testing.T) {
	var err error
	var result []string

	file, err := ioutil.TempFile("", "test")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())
	_ = file

	master, slave, err := pty.Open()
	if err != nil {
		panic(err)
	}

	resCh := make(chan string)
	go func() {
		reader := bufio.NewReaderSize(master, 4096)
		for {
			res := <-resCh
			command, err := reader.ReadString(';')
			log.Printf("COMMAND: %s %s --> %s", command, err, res)
			master.WriteString(res)
		}
	}()

	kx3 := New()
	err = kx3.Open(slave.Name(), 38400)
	if err != nil {
		t.Errorf("Open() returns: %s", err)
		return
	}

	if kx3.Status != STATUS_OPENED {
		t.Errorf("Opened but status is not opened: %d", kx3.Status)
	}

	resCh <- ""
	_, err = kx3.Command("FA;", RSP_FA)
	if err != ErrTimeout {
		t.Errorf("Open() does not return timeout: %s", err)
	}


	resCh <- fmt.Sprintf("FA%011d;", int(7e6))
	result, err = kx3.Command("FA;", RSP_FA)
	if err != nil {
		t.Errorf("Command failed with: %s", err)
	}
	if result[1] != "00007000000" {
		t.Errorf("result[1] is not expected: %v", result)
	}


	resCh <- "TB002CQ;"
	result, err = kx3.Command("TB;", RSP_TB)
	if err != nil {
		t.Errorf("Command failed with: %s", err)
	}
	if result[3] != "CQ" {
		t.Errorf("result[3] is not expected: %v", result)
	}

	resCh <- "TB005CQ;CQ;"
	result, err = kx3.Command("TB;", RSP_TB)
	if err != nil {
		t.Errorf("Command failed with: %s", err)
	}
	if result[3] != "CQ;CQ" {
		t.Errorf("result[3] is not expected: %v", result)
	}

	resCh <- "TB002CQ;"
	result, err = kx3.Command("TB;", RSP_TB)
	if err != nil {
		t.Errorf("Command failed with: %s", err)
	}
	if result[3] != "CQ" {
		t.Errorf("result[3] is not expected: %v", result)
	}

	close(resCh)
}
