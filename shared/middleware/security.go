package middleware

import "net/http"

// SecurityHeaders добавляет заголовки безопасности к каждому ответу
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Запрещает браузеру угадывать тип контента
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Базовая политика безопасности контента
		w.Header().Set("Content-Security-Policy", "default-src 'none'")

		// Запрещает отображение в iframe (защита от clickjacking)
		w.Header().Set("X-Frame-Options", "DENY")

		next.ServeHTTP(w, r)
	})
}
