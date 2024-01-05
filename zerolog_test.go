package bench

import (
	"io"
	"time"

	"github.com/rs/zerolog"
)

func (u user) MarshalZerologObject(e *zerolog.Event) {
	e.Str("name", u.Name).
		Int("age", u.Age).
		Time("dob", u.DOB)
}

func (uu users) MarshalZerologArray(a *zerolog.Array) {
	for _, u := range uu {
		a.Object(u)
	}
}

func zerologFields(e *zerolog.Event) *zerolog.Event {
	e.
		Int("bytes", ctxBodyBytes).
		Str("request", ctxRequest).
		Float64("elapsed_time_ms", ctxTimeElapsedMs).
		Object("user", ctxUser).
		Time("now", ctxTime).
		Strs("months", ctxMonths).
		Ints("primes", ctxFirst10Primes).
		Array("users", ctxUsers).
		Err(ctxErr)

	return e
}

func zerologCtx(c zerolog.Context) zerolog.Context {
	c.
		Int("bytes", ctxBodyBytes).
		Str("request", ctxRequest).
		Float64("elapsed_time_ms", ctxTimeElapsedMs).
		Object("user", ctxUser).
		Time("now", ctxTime).
		Strs("months", ctxMonths).
		Ints("primes", ctxFirst10Primes).
		Array("users", ctxUsers).
		Err(ctxErr)

	return c
}

func newZerolog(w io.Writer) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	return zerolog.New(w).Level(zerolog.InfoLevel).With().Timestamp().Logger()
}

func newZeroLogWithContext(w io.Writer) zerolog.Logger {
	return zerologCtx(newZerolog(w).With()).Logger()
}

type zerologBench struct {
	l zerolog.Logger
}

func (b *zerologBench) new(w io.Writer) logBenchmark {
	return &zerologBench{
		l: newZerolog(w),
	}
}

func (b *zerologBench) newWithCtx(w io.Writer) logBenchmark {
	return &zerologBench{
		l: newZeroLogWithContext(w),
	}
}

func (b *zerologBench) name() string {
	return "Zerolog"
}

func (b *zerologBench) logEvent(msg string) {
	b.l.Info().Msg(msg)
}

func (b *zerologBench) logEventFmt(msg string, args ...any) {
	b.l.Info().Msgf(msg, args...)
}

func (b *zerologBench) logEventCtx(msg string) {
	zerologFields(b.l.Info()).Msg(msg)
}

func (b *zerologBench) logEventCtxWeak(msg string) {
	b.l.Info().Fields(alternatingKeyValuePairs()).Msg(msg)
}

func (b *zerologBench) logDisabled(msg string) {
	b.l.Debug().Msg(msg)
}

func (b *zerologBench) logDisabledFmt(msg string, args ...any) {
	b.l.Debug().Msgf(msg, args...)
}

func (b *zerologBench) logDisabledCtx(msg string) {
	zerologFields(b.l.Debug()).Msg(msg)
}

func (b *zerologBench) logDisabledCtxWeak(msg string) {
	b.l.Debug().Fields(alternatingKeyValuePairs()).Msg(msg)
}
