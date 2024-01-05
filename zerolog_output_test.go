package bench

import (
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

var zeroLoggers = []logBenchmark{
	//&zerologBench{},
	//&slogZeroSamberBench{},
	//&slogZeroPhsymBench{},
	//&slogBench{},

	//&phusLogBench{},
	//&slogZapBench{},
	//&apexBench{},
	&logrusBench{},
}

func Test_Event_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEvent(logMsg)
	}

	for _, capture := range hdlrData {
		event, err := basicFields(t, capture, logMsg, numBasicFields)
		require.NoError(t, err)
		noContext(t, event)
	}
}

func Test_Disabled_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabled(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventFmt_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEventFmt(logMsgFmt, logMsgArgs...)
	}

	for _, capture := range hdlrData {
		event, err := basicFields(t, capture, logMsgFormatted, numFormattedFields)
		require.NoError(t, err)
		noContext(t, event)
	}
}

func Test_EventDisabledFmt_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabledFmt(logMsgFmt, logMsgArgs...)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventCtx_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logEventCtx(logMsg)
	}

	for _, capture := range hdlrData {
		entry, err := basicFields(t, capture, logMsg, numContextFields)
		require.NoError(t, err)
		contextFields(t, entry)
	}
}

func Test_EventDisabledCtx_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.new(hdlrData[i])
		logger.logDisabledCtx(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func Test_EventCtxWeak_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(loggers))
	for i, benchmark := range loggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logEventCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		entry, err := basicFields(t, capture, logMsg, numContextFields)
		require.NoError(t, err)
		contextFields(t, entry)
	}
}

func Test_EventDisabledCtxWeak_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logDisabledCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func xTest_EventAccumulatedCtx_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logEvent(logMsg)
	}

	for _, capture := range hdlrData {
		entry, err := basicFields(t, capture, logMsg, numAccumulatedFields)
		require.NoError(t, err)
		contextFields(t, entry)
	}
}

func xTest_EventDisabledAccumulatedCtx_zerolog(t *testing.T) {
	hdlrData := make([]*logCapture, len(zeroLoggers))
	for i, benchmark := range zeroLoggers {
		hdlrData[i] = newLogCapture(benchmark)
		logger := benchmark.newWithCtx(hdlrData[i])
		logger.logDisabledCtxWeak(logMsg)
	}

	for _, capture := range hdlrData {
		assert.True(t, capture.empty())
	}
}

func basicFields(t *testing.T, capture *logCapture, message string, numFields int) (*logEntry, error) {
	fmt.Printf(
		"--------------------------------------------------------------------------\n"+
			"%s (%d):\n  %s\n", capture.name(), capture.numFields(), capture.String())
	entry, err := capture.jsonObject()
	assert.NoError(t, err)
	assert.Equal(t, "info", strings.ToLower(entry.Level))
	_, err = time.Parse(time.RFC3339Nano, entry.Time)
	assert.NoError(t, err)
	if entry.Msg == "" {
		// Note: Some handlers use 'message' instead of 'msg'.
		fmt.Printf(">>> %s uses 'message' instead of 'msg'\n", capture.name())
		assert.Equal(t, message, entry.Message)
	} else {
		assert.Equal(t, message, entry.Msg)
	}
	if capture.name() == (&slogZeroPhsymBench{}).name() {
		// Note: The phsym handler adds an extra time field in the JSON output.
		assert.Equal(t, numFields+1, capture.numFields())
	} else {
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
