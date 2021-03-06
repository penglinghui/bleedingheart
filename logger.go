package main

import (
	"fmt"
	"log"
	"os"
)

var debugEnabled = true
var logger = log.New(os.Stderr, "", log.Lshortfile|log.Ltime)

func debugln(vals ...interface{}) {
	if debugEnabled {
		s := fmt.Sprintln(vals...)
		logger.Output(2, "DEBUG: "+s)
	}
}

func debugf(format string, vals ...interface{}) {
	if debugEnabled {
		s := fmt.Sprintf(format, vals...)
		logger.Output(2, "DEBUG: "+s)
	}
}

func infoln(vals ...interface{}) {
	s := fmt.Sprintln(vals...)
	logger.Output(2, "INFO: "+s)
}

func infof(format string, vals ...interface{}) {
	s := fmt.Sprintf(format, vals...)
	logger.Output(2, "INFO: "+s)
}

func okln(vals ...interface{}) {
	s := fmt.Sprintln(vals...)
	logger.Output(2, "OK: "+s)
}

func okf(format string, vals ...interface{}) {
	s := fmt.Sprintf(format, vals...)
	logger.Output(2, "OK: "+s)
}

func warnln(vals ...interface{}) {
	s := fmt.Sprintln(vals...)
	logger.Output(2, "WARNING: "+s)
}

func warnf(format string, vals ...interface{}) {
	s := fmt.Sprintf(format, vals...)
	logger.Output(2, "WARNING: "+s)
}

func fatalln(vals ...interface{}) {
	s := fmt.Sprintln(vals...)
	logger.Output(2, "FATAL: "+s)
	os.Exit(3)
}

func fatalf(format string, vals ...interface{}) {
	s := fmt.Sprintf(format, vals...)
	logger.Output(2, "FATAL: "+s)
	os.Exit(3)
}

func fatalErr(err error) {
	if err != nil {
		fatalf(err.Error())
	}
}
