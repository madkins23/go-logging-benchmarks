package bench

import (
	"io"
	"log/slog"

	samber "github.com/samber/slog-zerolog/v2"
)

type slogZeroSamberBench struct {
	slogBench
}

// slog frontend with Samber Zerolog backend.
func newSlogZeroSamber(w io.Writer) *slog.Logger {
	l := newZerolog(w)
	return slog.New(samber.Option{Logger: &l}.NewZerologHandler())
}

func newSlogZeroSamberWithCtx(w io.Writer, attr []slog.Attr) *slog.Logger {
	l := newZeroLogWithContext(w)
	return slog.New(samber.Option{Logger: &l}.NewZerologHandler().WithAttrs(attr))
}

func (b *slogZeroSamberBench) name() string {
	return "SlogZeroSamber"
}
