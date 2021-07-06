package common

import (
	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/promlog"
)

var logger log.Logger
var level int

func GetLogger() log.Logger {
	return logger
}

func SetLogger(log log.Logger) {
	logger = log
}

const (
	levelDebug int = 1 << iota
	levelInfo
	levelWarn
	levelError
)

func Setlevel(l promlog.AllowedLevel) {

	// Set updates the value of the allowed level.
	switch l.String() {
	case "debug":
		level = levelDebug
	case "info":
		level = levelInfo
	case "warn":
		level = levelWarn
	case "error":
		level = levelError
	}
}

func IsDebugEnabled() bool {
	return level == levelDebug
}

func IsInfoEnabled() bool {
	return level == levelDebug|levelInfo
}
