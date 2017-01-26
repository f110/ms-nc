package main

import (
	"flag"
	"os"
)

func main() {
	var listen int
	flag.IntVar(&listen, "l", 0, "listen")
	flag.Parse()

	if listen != 0 {
		NewListener(os.Stdout)
		return
	}

	NewWriter(os.Args[1], os.Stdin)
}
