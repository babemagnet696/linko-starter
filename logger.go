package main

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os"

	pkgerr "github.com/pkg/errors"

	"boot.dev/linko/internal/build"
	"boot.dev/linko/internal/linkoerr"
)

type closeFunc func() error

type stackTracer interface {
	error
	StackTrace() pkgerr.StackTrace
}

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	})
	env := os.Getenv("ENV")
	hostname, _ := os.Hostname()

	logger := slog.New(debugHandler)
	if logFile == "" {
		logger = logger.With(
			slog.String("git_sha", build.GitSHA),
			slog.String("build_time", build.BuildTime),
			slog.String("env", env),
			slog.String("hostname", hostname),
		)
		return logger, func() error { return nil }, nil
	}

	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}

	bufferedFile := bufio.NewWriterSize(file, 8192)
	infoHandler := slog.NewJSONHandler(bufferedFile, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: replaceAttr,
	})

	logger = slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", env),
		slog.String("hostname", hostname),
	)
	return logger, func() error {
		defer file.Close()
		err := bufferedFile.Flush()
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key != "error" {
		return a
	}

	err, ok := a.Value.Any().(error)
	if !ok {
		return a
	}

	var attrs []slog.Attr
	if multiErr, ok := errors.AsType[multiError](err); ok {
		for i, err := range multiErr.Unwrap() {
			attrs = append(attrs, slog.GroupAttrs(fmt.Sprintf("error_%d", i+1), errorAttrs(err)...))
		}
		return slog.GroupAttrs("errors", attrs...)
	}

	attrs = errorAttrs(err)
	return slog.GroupAttrs("error", attrs...)
}

type multiError interface {
	error
	Unwrap() []error
}

func errorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("message", err.Error()),
	}
	if stackErr, ok := errors.AsType[stackTracer](err); ok {
		attrs = append(attrs, slog.Attr{
			Key:   "stack_trace",
			Value: slog.StringValue(fmt.Sprintf("%+v", stackErr.StackTrace())),
		})
	}
	attrs = append(attrs, linkoerr.Attrs(err)...)
	return attrs
}
