package bench

import (
	"io"
	"log/slog"

	"github.com/phsym/zeroslog"
)

type slogZeroPhsymBench struct {
	slogBench
}

// slog frontend with Phsym Zerolog backend.
func newSlogZeroPhsym(w io.Writer) *slog.Logger {
	return slog.New(zeroslog.NewHandler(newZerolog(w), nil))
}

func newSlogZeroPhsymWithCtx(w io.Writer, attr []slog.Attr) *slog.Logger {
	return slog.New(zeroslog.NewHandler(newZeroLogWithContext(w), nil))
}

func (b *slogZeroPhsymBench) new(w io.Writer) logBenchmark {
	return &slogBench{
		l: newSlogZeroPhsym(w),
	}
}

func (b *slogZeroPhsymBench) newWithCtx(w io.Writer) logBenchmark {
	return &slogBench{
		l: newSlogZeroPhsymWithCtx(w, slogAttrs()),
	}
}

func (b *slogZeroPhsymBench) name() string {
	return "SlogZeroPhsym"
}
