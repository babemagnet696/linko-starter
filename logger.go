package main

import (
	"io"
	"log"
	"os"
)

func initializeLogger(logFile string) (*log.Logger, error) {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	if logFile == "" {
		return logger, nil
	}
	
	file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	
	multiWriter := io.MultiWriter(os.Stderr, file)
	logger = log.New(multiWriter, "", log.LstdFlags)
	return logger, nil
}