package bench

import (
	"errors"
	"fmt"
	"io"
	"regexp/syntax"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

// duplicateFieldCounter manually parses JSON to count duplicate fields.
// The purpose of this is to find log output that has duplicate fields.
// The encoding/json package just chooses one of the fields when this happens,
// so it isn't useful in getting this result.
type duplicateFieldCounter struct {
	counts  map[string]uint
	charLoc int
	recent  []rune
	reader  *strings.Reader
}

func newFieldDuplicateCounter(text string) *duplicateFieldCounter {
	return &duplicateFieldCounter{
		counts: make(map[string]uint),
		reader: strings.NewReader(text),
	}
}

const (
	msgParseArray      = "parse array"
	msgParseField      = "parse field"
	msgParseKeyword    = "parse keyword"
	msgParseNumber     = "parse number"
	msgParseObject     = "parse object"
	msgParseString     = "parse string"
	msgParseValue      = "parse value"
	msgUnreadRune      = "unread rune"
	fmtExpectedColon   = "expected colon, got '%c'"
	msgUnexpectedColon = "unexpected colon"
	fmtExpectedComma   = "expected comma, got '%c'"
	msgUnexpectedComma = "unexpected comma"
)

func (ctr *duplicateFieldCounter) duplicateFields() map[string]int {
	duplicates := make(map[string]int)
	for field, count := range ctr.counts {
		if count > 1 {
			duplicates[field] = int(count)
		}
	}
	return duplicates
}

func (ctr *duplicateFieldCounter) fieldCount() int {
	return len(ctr.counts)
}

func (ctr *duplicateFieldCounter) topLevel() error {
	return ctr.readLoop(func(r rune) error {
		if unicode.IsSpace(r) {
			return nil
		} else if r == '{' {
			err := ctr.wrapCallError(ctr.parseObject(true), msgParseObject, true, true)
			if errors.Is(err, io.EOF) {
				return nil
			} else {
				return err
			}
		} else {
			return errUnexpected(r)
		}
	})
}

func (ctr *duplicateFieldCounter) parseArray() error {
	expectComma := false
	return ctr.readLoop(func(r rune) error {
		if unicode.IsSpace(r) {
			return nil
		} else if r == ',' {
			if expectComma {
				expectComma = false
				return nil
			} else {
				return fmt.Errorf(msgUnexpectedComma)
			}
		} else if r == ']' {
			return errFinished
		} else if expectComma {
			return fmt.Errorf(fmtExpectedComma, r)
		} else {
			if err := ctr.unreadRune(); err != nil {
				return ctr.wrapCallError(err, msgUnreadRune, false, false)
			}
			expectComma = true
			return ctr.wrapCallError(ctr.parseValue(), msgParseValue, false, false)
		}
	})
}

func (ctr *duplicateFieldCounter) parseField(field string) error {
	expectColon := true
	return ctr.readLoop(func(r rune) error {
		if unicode.IsSpace(r) {
			return nil
		} else if r == ':' {
			if expectColon {
				expectColon = false
				return nil
			} else {
				return fmt.Errorf(msgUnexpectedColon)
			}
		} else if expectColon {
			return fmt.Errorf(fmtExpectedColon, r)
		} else if r == '{' {
			if field == "fields" {
				// Special case for Apex wherein additional fields are in 'fields' object.
				delete(ctr.counts, field)
				return ctr.wrapCallError(ctr.parseObject(true), msgParseObject, true, false)
			} else {
				return ctr.wrapCallError(ctr.parseObject(false), msgParseObject, true, false)
			}
		} else if r == '[' {
			return ctr.wrapCallError(ctr.parseArray(), msgParseArray, true, false)
		} else if err := ctr.unreadRune(); err != nil {
			return ctr.wrapCallError(err, msgUnreadRune, false, false)
		} else {
			return ctr.wrapCallError(ctr.parseValue(), msgParseValue, true, false)
		}
	})
}

func (ctr *duplicateFieldCounter) parseKeyword(first rune) (string, error) {
	var builder strings.Builder
	builder.WriteRune(first)
	return builder.String(), ctr.readLoop(func(r rune) error {
		if syntax.IsWordChar(r) {
			builder.WriteRune(r)
		} else if err := ctr.unreadRune(); err != nil {
			return ctr.wrapCallError(err, msgUnreadRune, false, false)
		} else {
			return errFinished
		}
		return nil
	})
}

func (ctr *duplicateFieldCounter) parseNumber() error {
	foundDecimal := false
	foundExponent := false
	foundExponentSign := false
	return ctr.readLoop(func(r rune) error {
		if unicode.IsDigit(r) {
			return nil
		} else if r == '.' {
			if foundExponent {
				return fmt.Errorf("decimal in exponent")
			} else if foundDecimal {
				return fmt.Errorf("second decimal")
			} else {
				foundDecimal = true
				return nil
			}
		} else if r == 'e' || r == 'E' {
			if foundExponent {
				return fmt.Errorf("second exponent")
			} else {
				foundExponent = true
				return nil
			}
		} else if r == '-' || r == '+' {
			if !foundExponent {
				return fmt.Errorf("sign before exponent")
			} else if foundExponentSign {
				return fmt.Errorf("second exponent sign")
			} else {
				foundExponentSign = true
				return nil
			}
		} else if err := ctr.unreadRune(); err != nil {
			return ctr.wrapCallError(err, msgUnreadRune, false, false)
		} else {
			return errFinished
		}
	})
}

func (ctr *duplicateFieldCounter) parseObject(countFields bool) error {
	expectComma := false
	return ctr.readLoop(func(r rune) error {
		switch {
		case unicode.IsSpace(r):
			return nil
		case r == ',':
			if expectComma {
				expectComma = false
				return nil
			} else {
				return fmt.Errorf(msgUnexpectedComma)
			}
		case r == '"':
			if expectComma {
				return fmt.Errorf(fmtExpectedComma, r)
			}
			field, err := ctr.parseString()
			if err != nil {
				return ctr.wrapCallError(err, msgParseString, false, false)
			}
			field = strings.ToLower(field)
			if countFields {
				ctr.counts[field]++
			}
			expectComma = true
			return ctr.wrapCallError(ctr.parseField(field), msgParseField, false, false)
		case r == '}':
			return errFinished
		default:
			return errUnexpected(r)
		}
	})
}

func (ctr *duplicateFieldCounter) parseString() (string, error) {
	var builder strings.Builder
	err := ctr.readLoop(func(r rune) error {
		var err error
		switch r {
		case '\\':
			builder.WriteRune(r)
			if r, err = ctr.readRune(); err != nil {
				return fmt.Errorf("read escaped rune: %w", err)
			} else {
				builder.WriteRune(r)
				return nil
			}
		case '"':
			return errFinished
		default:
			builder.WriteRune(r)
			return nil
		}
	})
	return builder.String(), err
}

var goodKeywords = map[string]bool{
	"true":  true,
	"false": true,
	"null":  true,
}

func (ctr *duplicateFieldCounter) parseValue() error {
	return ctr.readLoop(func(r rune) error {
		if unicode.IsSpace(r) {
			return nil
		} else if unicode.IsLetter(r) {
			if keyword, err := ctr.parseKeyword(r); err != nil {
				return ctr.wrapCallError(err, msgParseKeyword, false, false)
			} else if !goodKeywords[strings.ToLower(keyword)] {
				return fmt.Errorf("bad keyword '%s'", keyword)
			}
		} else if r == '"' {
			_, err := ctr.parseString()
			return ctr.wrapCallError(err, msgParseString, true, false)
		} else if unicode.IsDigit(r) || r == '-' {
			return ctr.wrapCallError(ctr.parseNumber(), msgParseNumber, true, false)
		} else if r == '[' {
			return ctr.wrapCallError(ctr.parseArray(), msgParseArray, true, false)
		} else if r == '{' {
			return ctr.wrapCallError(ctr.parseObject(false), msgParseObject, true, false)
		} else {
			return errUnexpected(r)
		}
		return nil
	})
}

type loopFn func(r rune) error

func (ctr *duplicateFieldCounter) readLoop(fn loopFn) error {
	for {
		r, err := ctr.readRune()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("read rune: %w", err)
		}
		if err := fn(r); errors.Is(err, errFinished) {
			break
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (ctr *duplicateFieldCounter) readRune() (rune, error) {
	r, _, err := ctr.reader.ReadRune()
	if err == nil {
		ctr.charLoc++
		ctr.recent = append(ctr.recent, r)
		if len(ctr.recent) > 15 {
			ctr.recent = ctr.recent[5:]
		}
	}
	return r, err
}

func (ctr *duplicateFieldCounter) unreadRune() error {
	err := ctr.reader.UnreadRune()
	if err == nil {
		if len(ctr.recent) > 0 {
			ctr.charLoc--
			ctr.recent = ctr.recent[:len(ctr.recent)-1]
		}
	}
	return err
}

func (ctr *duplicateFieldCounter) wrapCallError(err error, msg string, finished bool, status bool) error {
	if errors.Is(err, errFinished) {
		return err
	} else if err != nil {
		if status {
			return fmt.Errorf("%s %s: %w", msg, ctr.wrapStatusStatus(), err)
		} else {
			return fmt.Errorf("%s: %w", msg, err)
		}
	} else if finished {
		return errFinished
	} else {
		return nil
	}
}

func (ctr *duplicateFieldCounter) wrapStatusStatus() string {
	var builder strings.Builder
	var r rune
	builder.WriteRune('@')
	builder.WriteString(strconv.Itoa(ctr.charLoc))
	builder.WriteString(" '")
	for _, r = range ctr.recent {
		builder.WriteRune(r)
	}
	builder.WriteString("<^>")
	for i := 0; i < 15; i++ {
		if r, err := ctr.readRune(); err == nil {
			builder.WriteRune(r)
		} else {
			if errors.Is(err, io.EOF) {
				builder.WriteString("<EOF>")
			}
			break
		}
	}
	builder.WriteString("' ")
	builder.WriteString(strconv.Itoa(ctr.fieldCount()))
	builder.WriteString(" fields")
	duplicates := ctr.duplicateFields()
	if len(duplicates) > 0 {
		builder.WriteRune(',')
		for field, count := range duplicates {
			builder.WriteByte(' ')
			builder.WriteString(field)
			builder.WriteByte(':')
			builder.WriteString(strconv.Itoa(int(count)))
		}
	} else {
		builder.WriteString(", no duplicates")
	}
	return builder.String()
}

var errFinished = errors.New("finished readLoop")

func errUnexpected(r rune) error {
	return fmt.Errorf("unexpected rune: %c", r)
}

var duplicateFieldsTestCases = []string{
	`{"time":"2024-01-06T09:26:20.402581532-08:00","level":"INFO","msg":"The quick brown fox jumps over the lazy dog","bytes":123456789,"request":"GET /icons/ubuntu-logo.png HTTP/1.1","elapsed_time_ms":11.398466,"user":{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},"now":"2024-01-06T09:26:20.398051436-08:00","months":["January","February","March","April","May","June","July","August","September","October","November","December"],"primes":[2,3,5,7,11,13,17,23,29,31],"users":[{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23}],"error":"failed to open file: /home/dev/new.txt","bytes":123456789,"request":"GET /icons/ubuntu-logo.png HTTP/1.1","elapsed_time_ms":11.398466,"user":{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},"now":"2024-01-06T09:26:20.398051436-08:00","months":["January","February","March","April","May","June","July","August","September","October","November","December"],"primes":[2,3,5,7,11,13,17,23,29,31],"users":[{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23},{"dob":"2000-09-09T00:00:00Z","name":"John Doe","age":23}],"error":"failed to open file: /home/dev/new.txt"}`,
	`{"time":"2024-01-06T11:42:26.288035029-08:00","level":"info","bytes":123456789,"request":"GET /icons/ubuntu-logo.png HTTP/1.1","elapsed_time_ms":11.398466,"user":{"name":"John Doe","age":23,"dob":"2000-09-09T00:00:00Z"},"now":"2024-01-06T11:42:26.275-08:00","months":["January","February","March","April","May","June","July","August","September","October","November","December"],"primes":[2,3,5,7,11,13,17,23,29,31],"users":"[{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23},{\"dob\":\"2000-09-09T00:00:00Z\",\"name\":\"John Doe\",\"age\":23}]","error":"failed to open file: /home/dev/new.txt","message":"The quick brown fox jumps over the lazy dog"}`,
	`{"alpha": 32, , "bravo": 14}`,
}

func Test_FieldDuplicateCounter(t *testing.T) {
	for _, testCase := range duplicateFieldsTestCases {
		fmt.Printf(">>> %s\n", testCase)
		counter := newFieldDuplicateCounter(testCase)
		require.NoError(t, counter.topLevel())
		fmt.Printf(">>> %s\n", counter.wrapStatusStatus())
	}
}
