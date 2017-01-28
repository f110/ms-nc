package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

var (
	Version = "debug"
)

func startReceiver(port int, args []string) {
	var f io.Writer
	if len(args) == 0 || args[0] == "-" {
		f = os.Stdout
	} else {
		if _, err := os.Stat(args[0]); err == nil {
			fmt.Print(err)
			os.Exit(1)
		}

		o, err := os.OpenFile(args[0], os.O_WRONLY, 0644)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		f = o
	}
	NewReceiver(port, f)
	return
}

func main() {
	var listen int
	var mBits int
	version := false
	flag.IntVar(&listen, "l", 0, "listen")
	flag.IntVar(&mBits, "r", 10, "rate limit (Mbps)")
	flag.BoolVar(&version, "v", false, "show version")
	flag.Parse()

	if version {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if listen != 0 {
		startReceiver(listen, flag.Args())
		return
	}

	if mBits > 2000 {
		log.Print("rate limit should be less than 2,000 Mbps")
		os.Exit(1)
	}
	NewSender(os.Args[1], os.Stdin, float32(mBits)/8.0)
}
