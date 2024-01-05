package bench

import (
	"bytes"
	"encoding/json"
	"time"
)

type logCapture struct {
	bytes.Buffer
	loggerName string
}

type LogEntryBasicFields struct {
	Level     string `json:"level"`
	Lvl       string `json:"lvl"` // Note Used by Log15 handler instead of 'level'
	Msg       string `json:"msg"`
	Message   string `json:"message"` // Note: Used by the Phsym handler instead of 'msg'
	Time      string `json:"time"`
	Timestamp string `json:"timestamp"` // Note: Used by Apex handler instead of 'time'
	T         string `json:"t"`         // Note: Used by Log15 handler instead of 'time'
}

type logEntryContextFields struct {
	Bytes     int       `json:"bytes"`
	ElapsedMS float64   `json:"elapsed_time_ms"`
	Error     string    `json:"error"`
	Months    []string  `json:"months"`
	Now       time.Time `json:"now"`
	Primes    []int     `json:"primes"`
	Request   string    `json:"request"`
	User      user      `json:"user"`
	Users     []user    `json:"users"`
}

type logEntry struct {
	LogEntryBasicFields
	logEntryContextFields
}

func newLogCapture(benchmark logBenchmark) *logCapture {
	return &logCapture{loggerName: benchmark.name()}
}

func (lc *logCapture) empty() bool {
	return lc.Buffer.Len() < 1
}

func (lc *logCapture) jsonObject() (*logEntry, error) {
	var entry logEntry
	return &entry, json.Unmarshal(lc.Bytes(), &entry)
}

func (lc *logCapture) name() string {
	return lc.loggerName
}

// numFields returns the number of separate fields captured.
// This covers situations where fields are duplicated in the output.
// JSON unmarshalling will not treat this as an error or return both fields.
func (lc *logCapture) numFields() int {
	return bytes.Count(lc.Bytes(), []byte{','}) + 1
}
