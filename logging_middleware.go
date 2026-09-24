package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
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
			reqID := spyWriter.ResponseWriter.Header().Get("X-Request-ID")

			info = append(info, 
				slog.Attr{Key: "method",     Value: slog.StringValue(r.Method)},
				slog.Attr{Key: "path",       Value: slog.StringValue(r.URL.Path)},
				slog.Attr{Key: "client_ip",  Value: slog.StringValue(redactIP(r.RemoteAddr))},
				slog.Attr{Key: "request_id", Value: slog.AnyValue(reqID)},
				slog.Duration("duration", time.Since(start)),
				slog.Int("request_body_bytes", spyReader.bytesRead),
				slog.Int("response_status", spyWriter.statusCode),
				slog.Int("response_body_bytes", spyWriter.bytesWritten),
			)
			if logContext.Username != "" {
				info = append(info, slog.Attr{Key: "user", Value: slog.StringValue(logContext.Username)})
			}
			if logContext.Error != nil {
				info = append(info, slog.Attr{Key: "error", Value: slog.AnyValue(logContext.Error)})
			}
			logger.Info("Served request", info...)
		})
	}
}

func httpError(ctx context.Context, w http.ResponseWriter, status int, err error) {
	if logCtx, ok := ctx.Value(logCtxKey).(*LogCtx); ok {
		logCtx.Error = err
	}
	switch status {
	case 401, 403, 500:
		http.Error(w, http.StatusText(status), status)
	default:
		http.Error(w, err.Error(), status)
	}
	
}

func redactIP(ip string) string {
	host, _, err := net.SplitHostPort(ip)
	if err != nil {
		host = ip
	}

	parsedIP := net.ParseIP(host)
	if parsedIP == nil {
		return ip
	}
	ipv4 := parsedIP.To4()
	if ipv4== nil {
		return ip
	}

	redacted := fmt.Sprintf("%d.%d.%d.x", ipv4[0], ipv4[1], ipv4[2])
	return redacted
}

type LogCtx struct {
	Username string
	Error    error
}

const logCtxKey contextKey = "log_context"
