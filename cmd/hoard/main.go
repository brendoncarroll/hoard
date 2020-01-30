package main

import (
	"log"

	"github.com/brendoncarroll/hoard/pkg/hoardcmd"
)

func main() {
	if err := hoardcmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
