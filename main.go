package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"boot.dev/linko/internal/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	httpPort := flag.Int("port", 8899, "port to listen on")
	dataDir := flag.String("data", "./data", "directory to store data")
	flag.Parse()

	status := run(ctx, cancel, *httpPort, *dataDir)
	cancel()
	os.Exit(status)
}

func run(ctx context.Context, cancel context.CancelFunc, httpPort int, dataDir string) int {
	logFile := os.Getenv("LINKO_LOG_FILE")
	tracerCloser, err := initTracing(ctx)
	if err != nil {
		fmt.Printf("error creating tracer: %v", err)
		return 1
	}

	defer func() {
		err := tracerCloser(context.Background())
		if err != nil {
			fmt.Printf("error closing tracer: %v", err)
		}
	}()

	logger, closer, err := initializeLogger(logFile)
	if err != nil {
		fmt.Printf("error creating logger: %v", err)
		return 1
	}

	defer func() {
		err := closer()
		if err != nil {
			fmt.Printf("error closing logger: %s", err)
		}
	}()

	st, err := store.New(logger, dataDir)
	if err != nil {
		logger.Error("failed to create store",
			"error", err,
		)
		return 1
	}
	s := newServer(logger, *st, httpPort, cancel)
	var serverErr error
	go func() {
		serverErr = s.start()
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown server",
			"error", err,
		)
		return 1
	}
	if serverErr != nil {
		logger.Error("server error",
			"error", serverErr,
		)
		return 1
	}
	return 0
}
