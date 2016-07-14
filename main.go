package main

import (
	"log"
	"os"
)

func AssertOK(err error, format string, args ...interface{}) {
	if err == nil {
		return
	}
	args = append(args, err)
	log.Fatalf(format+": %s", args...)
}

func main() {
	if len(os.Args) == 1 {
		log.Fatalf("Usage: %s <fname>", os.Args[0])
	}

	f, err := os.Open(os.Args[1])
	AssertOK(err, "Error on open")

	DumpBox(f, OutputContext(""))
}
