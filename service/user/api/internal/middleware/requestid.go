package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
)

// RequestIDMiddleware 从 OpenTelemetry span context 取 trace ID,写入响应头 X-Request-Id
func RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		if span != nil && span.SpanContext().HasTraceID() {
			traceID := span.SpanContext().TraceID().String()
			w.Header().Set("X-Request-Id", traceID)
		}
		next(w, r)
	}
}