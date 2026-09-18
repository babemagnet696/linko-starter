package main

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"gopkg.in/natefinch/lumberjack.v2"
	isatty "github.com/mattn/go-isatty"
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
	var handlers []slog.Handler
	debugHandler := tint.NewTextHandler(os.Stderr, &tint.Options{
		Level:       slog.LevelDebug,
		ReplaceAttr: replaceAttr,
		NoColor:     !(isatty.IsTerminal(os.Stderr.Fd()) || isatty.IsCygwinTerminal(os.Stderr.Fd())),
	})
	handlers = append(handlers, debugHandler)

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

	infoHandler := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1,
		MaxAge:     28,
		MaxBackups: 10,
		LocalTime:  false,
		Compress:   true,
	}
	handlers = append(handlers, slog.NewJSONHandler(infoHandler, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
		Level: slog.LevelInfo,
	}))

	logger = slog.New(slog.NewMultiHandler(
		handlers...,
	))
	logger = logger.With(
		slog.String("git_sha", build.GitSHA),
		slog.String("build_time", build.BuildTime),
		slog.String("env", env),
		slog.String("hostname", hostname),
	)
	return logger, func() error {
		err := infoHandler.Close()
		if err != nil {
			fmt.Printf("error closing logger file: %w", err)
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
