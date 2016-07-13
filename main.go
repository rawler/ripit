package main

import (
	"log"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("Usage: %s <fname>", os.Args[0])
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatalf("Error on open: %s", err)
	}

	DumpBox(f, OutputContext(""))
}
