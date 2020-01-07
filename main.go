package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const version = "0.5.2"
const website = "https://github.com/nxadm/tailtrigger"
const author = "Claudio Ramirez <pub.claudio@gmail.com>"
const defaultConfigFile = "tailtrigger.yaml"
const defaultTimeoutSec = 5

var configFile string
var debug bool
var timeoutSec int

func main() {
	readCliParams()
	fileWatches, foundErrors := importConfigFile(configFile)
	if foundErrors != nil {
		os.Stderr.WriteString("Errors found:\n")
		for _, err := range foundErrors {
			fmt.Fprintf(os.Stderr, "%s.\n", err)
		}
		os.Stderr.WriteString("Bailing out...\n")
		os.Exit(1)
	}

	for _, fileWatch := range fileWatches {
		fw := fileWatch
		log.Printf("Starting triggers for %s at %s\n",
			fw.FileName, time.Now())
		go fw.watch()
	}
	select {} // Wait forever
}
