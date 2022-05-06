package utils

import (
	"io"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(logdir string) {
	rotation := &lumberjack.Logger{
		Filename:   path.Join(logdir, "operator.log"),
		MaxSize:    200, // MiB
		MaxAge:     14,  // Days
		MaxBackups: 5,   // Number
		LocalTime:  false,
		Compress:   false,
	}
	multiWriter := io.MultiWriter(os.Stderr, rotation)
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	customFormatter.DisableQuote = true
	customFormatter.QuoteEmptyFields = true
	logrus.SetFormatter(customFormatter)
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetOutput(multiWriter)
}
