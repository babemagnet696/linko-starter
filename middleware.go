package main

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var info []any
			start := time.Now()

			logContext := &LogCtx{}
			r = r.WithContext(context.WithValue(r.Context(), logCtxKey, logContext))

			spyReader := &spyReadCloser{ReadCloser: r.Body}
			spyWriter := &spyResponseWriter{ResponseWriter: w}
			r.Body = spyReader

			next.ServeHTTP(spyWriter, r)
			info = append(info, 
				slog.Attr{Key: "method",    Value: slog.StringValue(r.Method)},
				slog.Attr{Key: "path",      Value: slog.StringValue(r.URL.Path)},
				slog.Attr{Key: "client_ip", Value: slog.StringValue(r.RemoteAddr)},
				slog.Duration("duration", time.Since(start)),
				slog.Int("request_body_bytes", spyReader.bytesRead),
				slog.Int("response_status", spyWriter.statusCode),
				slog.Int("response_body_bytes", spyWriter.bytesWritten),
			)
			if logContext.Username != "" {
				info = append(info, slog.Attr{Key: "user", Value: slog.StringValue(logContext.Username)})
			}
			logger.Info("Served request", info...)
		})
	}
}



type LogCtx struct {
	Username string
}

const logCtxKey contextKey = "log_context"
