package logrus

import (
	"github.com/sirupsen/logrus"

	"github.com/slok/kubewebhook/v2/pkg/log"
)

type logger struct {
	*logrus.Entry
}

// NewLogrus returns a new log.Logger for a logrus implementation.
func NewLogrus(l *logrus.Entry) log.Logger {
	return logger{Entry: l}
}

func (l logger) WithValues(kv map[string]interface{}) log.Logger {
	newLogger := l.Entry.WithFields(kv)
	return NewLogrus(newLogger)
}
