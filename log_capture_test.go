package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-logfmt/logfmt"
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

// logfmtObject exists to parse the logfmt puked out by logf.
func (lc *logCapture) logfmtObject() (*logEntry, error) {
	var entry logEntry
	var err error

	const fmtTimeLogfmt = "2006-01-02 15:04:05.999999999 -0700 MST"
	const fmtTimeUser = "2006-01-02 15:04:05 -0700 MST"

	parseUser := func(userString string) (user, error) {
		u := user{}
		parts := strings.Split(strings.Trim(userString, "{}"), " ")
		dob := strings.Join(parts[:4], " ")
		if u.DOB, err = time.Parse(fmtTimeUser, dob); err != nil {
			return u, fmt.Errorf("parse user DOB '%s': %w", dob, err)
		}
		u.Name = strings.Join(parts[4:len(parts)-1], " ")
		age := parts[len(parts)-1]
		if u.Age, err = strconv.Atoi(age); err != nil {
			return u, fmt.Errorf("convert user age '%s': %w", age, err)
		}
		return u, nil
	}

	d := logfmt.NewDecoder(strings.NewReader(lc.String()))
	d.ScanRecord()
	for d.ScanKeyval() {
		switch string(d.Key()) {
		case "level":
			entry.Level = string(d.Value())
		case "timestamp":
			entry.Time = string(d.Value())
		case "message":
			entry.Msg = string(d.Value())
		case "bytes":
			if entry.Bytes, err = strconv.Atoi(string(d.Value())); err != nil {
				return nil, fmt.Errorf("convert bytes '%s': %w", d.Value(), err)
			}
		case "elapsed_time_ms":
			if entry.ElapsedMS, err = strconv.ParseFloat(string(d.Value()), 64); err != nil {
				return nil, fmt.Errorf("convert elapsed MS '%s': %w", d.Value(), err)
			}
		case "error":
			entry.Error = string(d.Value())
		case "months":
			entry.Months = strings.Split(strings.Trim(string(d.Value()), "[]"), " ")
		case "now":
			now := string(d.Value())
			now = now[:len(now)-15] // Remove ' m=+#.#########' from end.
			if entry.Now, err = time.Parse(fmtTimeLogfmt, now); err != nil {
				return nil, fmt.Errorf("convert now time '%s': %w", now, err)
			}
		case "primes":
			ps := strings.Split(strings.Trim(string(d.Value()), "[]"), " ")
			entry.Primes = make([]int, len(ps))
			for i, prime := range ps {
				if entry.Primes[i], err = strconv.Atoi(prime); err != nil {
					return nil, fmt.Errorf("convert prime '%s': %w", prime, err)
				}
			}
		case "request":
			entry.Request = string(d.Value())
		case "user":
			if entry.User, err = parseUser(string(d.Value())); err != nil {
				return nil, fmt.Errorf("parse user '%s': %w", d.Value(), err)
			}
		case "users":
			// TODO: This line doesn't work (spaces within individual user records):
			us := strings.Split(strings.Trim(string(d.Value()), "[]"), "} {")
			entry.Users = make([]user, len(us))
			for i, u := range us {
				if entry.Users[i], err = parseUser(u); err != nil {
					return nil, fmt.Errorf("parse user '%s': %w", u, err)
				}
			}
		default:
			fmt.Printf(">>> unhandled logfmt field %s: %s\n", d.Key(), d.Value())
		}
	}

	return &entry, d.Err()
}

func (lc *logCapture) name() string {
	return lc.loggerName
}

// duplicateFields returns a map with duplicate field counts.
func (lc *logCapture) duplicateFields() (int, map[string]int, error) {
	if lc.name() == (&logfBench{}).name() {
		// Handle counting for logfmt.
		fields := 0
		counts := make(map[string]int)
		d := logfmt.NewDecoder(strings.NewReader(lc.String()))
		d.ScanRecord()
		for d.ScanKeyval() {
			counts[string(d.Key())]++
		}
		for key, count := range counts {
			fields++
			if count < 2 {
				delete(counts, key)
			}
		}
		return fields, counts, nil
	} else {
		// Handle counting for JSON.
		counter := newFieldDuplicateCounter(string(lc.Bytes()))
		if err := counter.topLevel(); err != nil {
			return 0, nil, fmt.Errorf("top level: %w", err)
		}
		return counter.fieldCount(), counter.duplicateFields(), nil
	}
}
