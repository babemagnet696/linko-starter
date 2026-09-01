package main

import (
	"bufio"
	"io"
	"log"
	"os"
)

type closeFunc func() error

func initializeLogger(logFile string) (*log.Logger, closeFunc, error) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if logFile == "" {
		return logger, func()error{return nil}, nil
	}
	
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	
	bufferedFile := bufio.NewWriterSize(file, 8192)
	multiWriter := io.MultiWriter(os.Stderr, bufferedFile)
	logger = log.New(multiWriter, "", log.LstdFlags)
	return logger, func()error{
		defer file.Close()
		err := bufferedFile.Flush()
		if err != nil {
			return err
		}
		return nil
	}, nil
}