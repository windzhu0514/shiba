package main

import (
	"os"
	"time"

	"github.com/windzhu0514/shiba/log"
)

func main() {
	log.Debug("debug log")
	defer log.Close()
	log.SetLevel(log.InfoLevel)
	log.Debug("debug log again")
	logger := log.Clone("log_name")
	logger.Errorf("clone error log")
	logger.Debug("clone debug log")
	logger.Info("clone info log")
	defer logger.Close()

	ml := log.New("test", os.Stdout, log.Config{
		EncoderMode:   log.EncoderModeJson,
		RotatorMode:   log.RotateModeDaily,
		Level:         log.DebugLevel,
		WithoutCaller: false,
		FileName:      "test.log",
		MaxAge:        3,
		UTCTime:       false,
		Compress:      true,
	})
	defer ml.Close()

	ml.Debug("test log")
	ml.Error("test log")
	for {
		ml.Debug("test debug log")
		ml.Error("test error log")
		time.Sleep(5 * time.Second)
	}
}
