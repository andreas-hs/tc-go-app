package logging

import (
	"github.com/sirupsen/logrus"
)

func LogError(logger *logrus.Logger, msg string, err error) {
	logger.Errorf("%s: %v", msg, err)
}

func LogFatal(logger *logrus.Logger, msg string, err error) {
	logger.Fatalf("%s: %v", msg, err)
}

func LogWarn(logger *logrus.Logger, msg string) {
	logger.Warn(msg)
}

func LogInfo(logger *logrus.Logger, msg string) {
	logger.Info(msg)
}
