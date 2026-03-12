package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

type Loggers struct {
	System    zerolog.Logger
	Data      zerolog.Logger
	Detection zerolog.Logger
}

func New(logFile string, detectionLogFile string) *Loggers {
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: "15:04:05",
	}
	sysLog := zerolog.New(consoleWriter).
		With().
		Timestamp().
		Logger()

	openFile := func(path string) *os.File {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			sysLog.Fatal().Err(err).Str("path", path).Msg("cannot open log file")
		}
		return f
	}

	dataLog := zerolog.New(openFile(logFile)).
		With().
		Timestamp().
		Logger()

	detectionLog := zerolog.New(openFile(detectionLogFile)).
		With().
		Timestamp().
		Logger()

	return &Loggers{
		System:    sysLog,
		Data:      dataLog,
		Detection: detectionLog,
	}
}
