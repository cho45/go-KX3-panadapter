//#!go build github.com/cho45/go-KX3-panadapter/{kx3hq,panadapter} && go run

package main

import (
	"flag"
	"fmt"
	"runtime"
	"github.com/cho45/go-KX3-panadapter/panadapter"
	"os"
)

func main() {
	var numcpu int
	var configPath string

	flag.IntVar(&numcpu, "numcpu", runtime.NumCPU(), "cpu num (default = runtime.NumCPU())")
	flag.StringVar(&configPath, "config", "config.json", "path to config.json")
	flag.Parse()

	runtime.GOMAXPROCS(numcpu)

	config, err := panadapter.ReadConfig(configPath)
	if err != nil {
		fmt.Printf("Error on reading config: %s", err)
		os.Exit(255)
	}

	panadapter.Start(config)
}
