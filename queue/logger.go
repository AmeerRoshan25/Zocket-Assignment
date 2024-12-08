package queue

import (
	"github.com/sirupsen/logrus"
)

// Shared logger instance
var Logger = logrus.New()

func InitLogger() {
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.SetLevel(logrus.InfoLevel)
}
