package shiba

import (
	"strings"
	"time"

	"github.com/windzhu0514/shiba/log"
)

var defaultLogger log.Logger

type cronLogger struct {
	logger log.Logger
}

func (l cronLogger) Info(msg string, keysAndValues ...interface{}) {
	if l.logger == nil {
		return
	}

	keysAndValues = formatTimes(keysAndValues)
	l.logger.Debugf(
		formatString(len(keysAndValues)),
		append([]interface{}{msg}, keysAndValues...)...)
}

func (l cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if l.logger == nil {
		return
	}

	keysAndValues = formatTimes(keysAndValues)
	l.logger.Errorf(
		formatString(len(keysAndValues)+2),
		append([]interface{}{msg, "error", err}, keysAndValues...)...)
}

// formatString returns a logfmt-like format string for the number of
// key/values.
func formatString(numKeysAndValues int) string {
	var sb strings.Builder
	sb.WriteString("%s")
	if numKeysAndValues > 0 {
		sb.WriteString(", ")
	}
	for i := 0; i < numKeysAndValues/2; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("%v=%v")
	}
	return sb.String()
}

// formatTimes formats any time.Time values as RFC3339.
func formatTimes(keysAndValues []interface{}) []interface{} {
	var formattedArgs []interface{}
	for _, arg := range keysAndValues {
		if t, ok := arg.(time.Time); ok {
			arg = t.Format(time.RFC3339)
		}
		formattedArgs = append(formattedArgs, arg)
	}
	return formattedArgs
}
