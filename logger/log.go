package logger

import (
	"fmt"
	"log"
	"os"
)

func init() {
	// logFileLocation, _ := os.OpenFile("data.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0744)
	// log.SetOutput(logFileLocation)
	log.SetOutput(os.Stdout)
	log.SetPrefix(fmt.Sprintf("[PID=%d] ", os.Getpid()))
}
