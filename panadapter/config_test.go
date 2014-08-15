package panadapter

import (
	"io/ioutil"
	"os"
	"testing"
)

func createTemporaryFile(content string) *os.File {
	file, err := ioutil.TempFile("", "test")
	if err != nil {
		panic(err)
	}
	file.Write([]byte(content))
	file.Close()
	return file
}

func TestReadConfig(t *testing.T) {
	{
		file := createTemporaryFile(`
			{
				"port" : {
					"name" : "/dev/tty.usbserial-A402PY11",
					"baudrate" : 38400
				},
				"window" : {
					"width" : 1410,
					"height" : 600
				},
				"historySize" : 500,
				"fftSize" : 2048
			}
		`)
		defer os.Remove(file.Name())

		config, err := ReadConfig(file.Name())
		if err != nil {
			t.Errorf("ReadConfig returns error: %s", err)
		}
		if config.FftSize != 2048 {
			t.Error("missmatch config.FftSize")
		}
		if config.Input != nil {
			t.Errorf("Input must be nil")
		}
	}

	{
		file := createTemporaryFile(`
			{
				"port" : {
					"name" : "/dev/tty.usbserial-A402PY11",
					"baudrate" : 38400
				},
				"window" : {
					"width" : 1410,
					"height" : 600
				},
				"historySize" : 500,
				"fftSize" : 2048,
				"input": {
					"name": "Built-In Mic",
					"samplerate" : 44100
				}
			}
		`)
		defer os.Remove(file.Name())

		config, err := ReadConfig(file.Name())
		if err != nil {
			t.Errorf("ReadConfig returns error: %s", err)
		}
		if config.Input.Name != "Built-In Mic" {
			t.Error("missmatch config")
		}
		if config.Input.SampleRate != 44100 {
			t.Error("missmatch config")
		}
	}
}
