package main

import (
	"github.com/windzhu0514/shiba/log"
)

func main() {
	log.Debug("debug log")
	log.SetLevel(log.InfoLevel)
	log.Debug("debug log again")
	logger := log.Clone("log_name")
	logger.Errorf("clone error log")
	logger.Debug("clone debug log")
	logger.Info("clone info log")
}
