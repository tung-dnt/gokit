package recovery 

import (
	"fmt"
	"log/slog"
	"net/http"
)

// Recovery returns a middleware that recovers from panics, logs the error,
// and responds with 500 Internal Server Error.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "error", fmt.Sprint(rec), "path", r.URL.Path)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
