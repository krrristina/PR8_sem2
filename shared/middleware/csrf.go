package middleware

import (
	"net/http"
)

// CSRF проверяет X-CSRF-Token на опасных запросах (POST/PATCH/DELETE)
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET и HEAD не меняют состояние — проверять не нужно
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		// Берём csrf_token из cookie
		cookie, err := r.Cookie("csrf_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "forbidden: missing csrf cookie", http.StatusForbidden)
			return
		}

		// Берём X-CSRF-Token из заголовка запроса
		headerToken := r.Header.Get("X-CSRF-Token")
		if headerToken == "" {
			http.Error(w, "forbidden: missing X-CSRF-Token header", http.StatusForbidden)
			return
		}

		// Сравниваем — должны совпадать
		if cookie.Value != headerToken {
			http.Error(w, "forbidden: csrf token mismatch", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
