package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func WithLogging(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	if logger == nil {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	logger = logger.With("component", "logger_middleware")

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			defer func() {
				logger.Infow("Served",
					"method", r.Method,
					"path", r.URL.Path,
					"proto", r.Proto,
					"reqId", middleware.GetReqID(r.Context()),
					"duration", time.Since(start),
					"status", ww.Status(),
					"size", ww.BytesWritten(),
				)
			}()

			logger.Infow("Started request",
				"method", r.Method,
				"path", r.URL.Path,
				"proto", r.Proto,
				"reqId", middleware.GetReqID(r.Context()),
				"remote", r.RemoteAddr,
			)

			next.ServeHTTP(ww, r)
		}

		return http.HandlerFunc(fn)
	}
}
