//#!go build github.com/cho45/go-KX3-panadapter/{kx3hq,panadapter} && go run

package main

import (
	"flag"
	"fmt"
	"github.com/cho45/go-KX3-panadapter/panadapter"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
)

func main() {
	var numcpu int
	var configPath string
	var cpuprofile string

	flag.IntVar(&numcpu, "numcpu", runtime.NumCPU(), "cpu num (default = runtime.NumCPU())")
	flag.StringVar(&configPath, "config", "config.json", "path to config.json")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.Parse()

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	runtime.GOMAXPROCS(numcpu)

	config, err := panadapter.ReadConfig(configPath)
	if err != nil {
		fmt.Printf("Error on reading config: %s", err)
		os.Exit(255)
	}

	server := &panadapter.Server{Config: config}
	server.Init()
	err = server.Start()
	if err != nil {
		fmt.Printf("Error", err)
		os.Exit(255)
	}
}
