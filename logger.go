package main

import (
	"bufio"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	debugHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(debugHandler)
	if logFile == "" {
		return logger, func() error { return nil }, nil
	}

	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}

	bufferedFile := bufio.NewWriterSize(file, 8192)
	infoHandler := slog.NewTextHandler(bufferedFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger = slog.New(slog.NewMultiHandler(
		debugHandler,
		infoHandler,
	))
	return logger, func() error {
		defer file.Close()
		err := bufferedFile.Flush()
		if err != nil {
			return err
		}
		return nil
	}, nil
}
