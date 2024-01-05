package bench

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numBasicFields       = 3
	numFormattedFields   = 7 // The formatted string includes four extra commas.
	numContextFields     = 63
	numAccumulatedFields = 123
)

func Test_Event(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEvent(logMsg)
	}

	for _, capture := range hdlrData {
		event, err := basicFields(t, capture, logMsg, numBasicFields)
		if errors.Is(err, &badJSONerror{}) {
			fmt.Printf(">>> Bad JSON from %s handler: %s\n", capture.name(), err)
		} else {
			require.NoError(t, err)
			noContext(t, event)
		}
	}
}

func Test_Disabled(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabled(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventFmt(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEventFmt(logMsgFmt, logMsgArgs...)
	}

	for _, capture := range hdlrData {
		event, err := basicFields(t, capture, logMsgFormatted, numFormattedFields)
		if errors.Is(err, &badJSONerror{}) {
			fmt.Printf(">>> Bad JSON from %s handler: %s\n", capture.name(), err)
		} else {
			require.NoError(t, err)
			noContext(t, event)
		}
	}
}

func Test_EventDisabledFmt(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabledFmt(logMsgFmt, logMsgArgs...)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventCtx(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEventCtx(logMsg)
	}

	for _, capture := range hdlrData {
		event, err := basicFields(t, capture, logMsg, numContextFields)
		if errors.Is(err, &badJSONerror{}) {
			fmt.Printf(">>> Bad JSON from %s handler: %s\n", capture.name(), err)
		} else {
			require.NoError(t, err)
			contextFields(t, event)
		}
	}
}

func Test_EventDisabledCtx(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabledCtx(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventCtxWeak(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logEventCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		entry, err := basicFields(t, capture, logMsg, numContextFields)
		if errors.Is(err, &badJSONerror{}) {
			fmt.Printf(">>> Bad JSON from %s handler: %s\n", capture.name(), err)
		} else {
			require.NoError(t, err)
			contextFields(t, entry)
		}
	}
}

func Test_EventDisabledCtxWeak(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logDisabledCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventAccumulatedCtx(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logEvent(logMsg)
	}

	for _, capture := range hdlrData {
		entry, err := basicFields(t, capture, logMsg, numAccumulatedFields)
		if errors.Is(err, &badJSONerror{}) {
			fmt.Printf(">>> Bad JSON from %s handler: %s\n", capture.name(), err)
		} else {
			require.NoError(t, err)
			contextFields(t, entry)
		}
	}
}

func xTest_EventDisabledAccumulatedCtx(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logDisabledCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

type badJSONerror struct {
	wrapped error
}

func (e *badJSONerror) Is(target error) bool {
	var badJSONerror *badJSONerror
	ok := errors.As(target, &badJSONerror)
	return ok
}

func (e *badJSONerror) Error() string {
	return "parse error: " + e.wrapped.Error()
}

func (e *badJSONerror) Unwrap() error {
	return e.wrapped
}

func basicFields(t *testing.T, capture *logCapture, message string, numFields int) (*logEntry, error) {
	fmt.Printf(
		"--------------------------------------------------------------------------\n"+
			"%s (%d):\n  %s\n", capture.name(), capture.numFields(), capture.String())

	entry, err := capture.jsonObject()
	if err != nil {
		return nil, &badJSONerror{wrapped: err}
	}

	if entry.Level != "" {
		assert.Equal(t, "info", strings.ToLower(entry.Level))
	} else if entry.Lvl != "" {
		fmt.Printf(">>> %s uses 'lvl' instead of 'level\n", capture.name())
		assert.Equal(t, "info", strings.ToLower(entry.Lvl))
	} else {
		assert.Fail(t, "No level field")
	}
	if entry.Time != "" {
		_, err = time.Parse(time.RFC3339Nano, entry.Time)
	} else if entry.Timestamp != "" {
		fmt.Printf(">>> %s uses 'timestamp' instead of 'time\n", capture.name())
		_, err = time.Parse(time.RFC3339Nano, entry.Timestamp)
	} else if entry.T != "" {
		fmt.Printf(">>> %s uses 't' instead of 'time\n", capture.name())
		_, err = time.Parse(time.RFC3339Nano, entry.T)
	} else {
		assert.Fail(t, "No time field")
	}
	assert.NoError(t, err)
	if entry.Msg != "" {
		assert.Equal(t, message, entry.Msg)
	} else if entry.Message != "" {
		fmt.Printf(">>> %s uses 'message' instead of 'msg'\n", capture.name())
		assert.Equal(t, message, entry.Message)
	} else {
		assert.Fail(t, "No message field")
	}
	switch capture.name() {
	case (&slogZeroPhsymBench{}).name():
	case (&apexBench{}).name():
		fmt.Printf(">>> %s has an extra field\n", capture.name())
		assert.Equal(t, numFields+1, capture.numFields())
	default:
		assert.Equal(t, numFields, capture.numFields())
	}
	return entry, nil
}

func contextFields(t *testing.T, entry *logEntry) {
	assert.Equal(t, ctxBodyBytes, entry.Bytes)
	assert.Equal(t, ctxTimeElapsedMs, entry.ElapsedMS)
	assert.Equal(t, ctxErr.Error(), entry.Error)
	if assert.Equal(t, len(ctxMonths), len(entry.Months)) {
		for i, month := range ctxMonths {
			assert.Equal(t, month, entry.Months[i])
		}
	}
	assert.WithinDuration(t, ctxTime, entry.Now, 0)
	if assert.Len(t, entry.Primes, len(ctxFirst10Primes)) {
		for i, prime := range ctxFirst10Primes {
			assert.Equal(t, prime, entry.Primes[i])
		}
	}
	assert.Equal(t, ctxRequest, entry.Request)
	checkUser(t, &entry.User)

	if assert.Len(t, entry.Users, len(ctxUsers)) {
		for _, u := range entry.Users {
			checkUser(t, &u)
		}
	}
}

func checkUser(t *testing.T, u *user) {
	assert.Equal(t, ctxUser.Name, u.Name)
	assert.Equal(t, ctxUser.Age, u.Age)
	assert.Equal(t, ctxUser.DOB, u.DOB)
}

func noContext(t *testing.T, entry *logEntry) {
	assert.Empty(t, entry.Bytes)
	assert.Empty(t, entry.ElapsedMS)
	assert.Empty(t, entry.Error)
	assert.Empty(t, entry.Months)
	assert.Empty(t, entry.Now)
	assert.Empty(t, entry.Primes)
	assert.Empty(t, entry.Request)
	assert.Empty(t, entry.User.Name)
	assert.Empty(t, entry.User.Age)
	assert.Empty(t, entry.User.DOB)
	assert.Empty(t, entry.Users)
}
