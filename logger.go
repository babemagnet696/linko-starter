package main

import (
	"bufio"
	"io"
	"log/slog"
	"os"
)

type closeFunc func() error

func initializeLogger(logFile string) (*slog.Logger, closeFunc, error) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	if logFile == "" {
		return logger, func()error{return nil}, nil
	}
	
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	
	bufferedFile := bufio.NewWriterSize(file, 8192)
	multiWriter := io.MultiWriter(os.Stderr, bufferedFile)
	logger = slog.New(slog.NewTextHandler(multiWriter, nil))
	return logger, func()error{
		defer file.Close()
		err := bufferedFile.Flush()
		if err != nil {
			return err
		}
		return nil
	}, nil
}