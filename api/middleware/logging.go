package middleware

import (
	"log"
	"net/http"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush passes through to the underlying writer when it implements
// http.Flusher. Without this method, an SSE handler's
// `w.(http.Flusher)` assertion on the wrapped writer fails — Go does
// NOT promote interfaces through struct embedding for type assertions
// even though method calls on the embedded value still resolve. This
// broke /api/studies/events with a 500 "streaming not supported"
// whenever the request was routed through Logging (i.e. every request
// in production).
func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start).Round(time.Millisecond))
	})
}
