//#!go run
package panadapter

import (
	"encoding/json"
	"os"
)

type Config struct {
	Port        PortConfig   `json:"port"`
	Window      WindowConfig `json:"window"`
	HistorySize int          `json:"historySize"`
	FftSize     int          `json:"fftSize"`
}

type PortConfig struct {
	Name     string `json:"name"`
	Baudrate int    `json:"baudrate"`
}

type WindowConfig struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func ReadConfig(filename string) (*Config, error) {
	c := &Config{}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
