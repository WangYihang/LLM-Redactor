package logging

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

type Loggers struct {
	System    zerolog.Logger
	Traffic   zerolog.Logger
	Detection zerolog.Logger
}

func New(appLogFile, trafficLogFile, detectionLogFile string) *Loggers {
	openFile := func(path string) *os.File {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Fallback to stderr if we can't open a log file
			return nil
		}
		return f
	}

	appFile := openFile(appLogFile)
	trafficFile := openFile(trafficLogFile)
	detectionFile := openFile(detectionLogFile)

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}

	var appWriter io.Writer
	if appFile != nil {
		appWriter = zerolog.MultiLevelWriter(consoleWriter, appFile)
	} else {
		appWriter = consoleWriter
	}

	sysLog := zerolog.New(appWriter).
		With().
		Timestamp().
		Logger()

	trafficLog := zerolog.New(trafficFile).
		With().
		Timestamp().
		Logger()

	detectionLog := zerolog.New(detectionFile).
		With().
		Timestamp().
		Logger()

	return &Loggers{
		System:    sysLog,
		Traffic:   trafficLog,
		Detection: detectionLog,
	}
}
