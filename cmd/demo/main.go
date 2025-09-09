package main

import (
	"fmt"
	"os"

	"github.com/b2network/pulsar/examples"
)

func main() {
	if err := examples.RunSimpleDemo(); err != nil {
		fmt.Fprintf(os.Stderr, "Demo failed: %v\n", err)
		os.Exit(1)
	}
}
