package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

type logCapture struct {
	bytes.Buffer
	loggerName string
}

type logEntry struct {
	// Basic fields:
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Message string `json:"message"` // Note: Used by the Phsym handler instead of 'msg'
	Time    string `json:"time"`

	// Context fields:
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

func newLogCapture(benchmark logBenchmark) *logCapture {
	return &logCapture{loggerName: benchmark.name()}
}

func (lc *logCapture) empty() bool {
	return lc.Buffer.Len() < 1
}

func (lc *logCapture) jsonObject() (*logEntry, error) {
	b := lc.Bytes()
	if found, err := regexp.Match(`^{"fields":`, b); err != nil {
		return nil, fmt.Errorf("looking for '{fields:'")
	} else if found {
		b = b[7:]
	}
	var entry logEntry
	return &entry, json.Unmarshal(b, &entry)
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
