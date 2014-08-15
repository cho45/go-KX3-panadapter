//#!go build github.com/cho45/go-KX3-panadapter/{kx3hq,panadapter} && go run

package main

import (
	"fmt"
	"runtime"
	"github.com/cho45/go-KX3-panadapter/panadapter"
	"os"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	config, err := panadapter.ReadConfig("./config.json")
	if err != nil {
		fmt.Printf("Error on reading config: %s", err)
		os.Exit(255)
	}

	panadapter.Start(config)
}
