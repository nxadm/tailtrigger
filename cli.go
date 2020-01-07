package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
	"os"
	"strconv"
)

/* Flags */
// configFile, debug and timeout are declared in the main file
var help, progVersion, sampleConfig bool
var givenTimeout int

/* Initialize the flags */
func init() {

	usage := `tailtrigger, ` + version + `.
Trigger actions by matching regexes in logfiles.
See ` + website + ` for more information.
Author: ` + author + `

Usage:
  tailtrigger [-c <configuration file>] [-d] [-t seconds]
  tailtrigger [-s]
  tailtrigger [-h]
  tailtrigger [-v]


Parameters:
  -c  | --config        : Configuration file [default: config.yaml].
  -d  | --debug         : Log extra runtime information.
  -t  | --timeout       : timeout seconds for actions [default: ` + strconv.Itoa(timeoutSec) + `]
  -s  | --sample-config : Print a sample configuration.
  -h  | --help          : This help message.
  -v  | --version       : Version message.
`

	flag.BoolVarP(&help, "help", "h", false, "")
	flag.BoolVarP(&progVersion, "version", "v", false, "")
	flag.BoolVarP(&debug, "debug", "d", false, "")
	flag.IntVarP(&givenTimeout, "timeout", "t", defaultTimeoutSec, "")
	flag.BoolVarP(&sampleConfig, "sample-config", "s", false, "")
	flag.StringVarP(&configFile, "config", "c", defaultConfigFile, "")
	flag.Usage = func() { fmt.Println(usage) } // Set a custom usage message
	flag.Parse()
}

func readCliParams() {
	switch {
	case help == true:
		flag.Usage()
		os.Exit(0)
	case progVersion == true:
		fmt.Println(version)
		os.Exit(0)
	case sampleConfig == true:
		fmt.Println(sampleConfigFile)
		os.Exit(0)
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		os.Stderr.WriteString("Can not find " + configFile + ".\n")
		os.Exit(1)
	}
	timeoutSec = givenTimeout
}
