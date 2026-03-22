package middleware

import (
	"net/http"
	"os"
	"strings"
)

// CORSMiddleware устанавливает заголовки CORS и базовые заголовки безопасности.
// Разрешённый источник задаётся переменной окружения ALLOWED_ORIGIN (по умолчанию "*").
// В production рекомендуется задать конкретный домен фронтенда.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
		if allowedOrigin == "" {
			allowedOrigin = "*"
		}

		origin := r.Header.Get("Origin")
		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && matchOrigin(origin, allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Заголовки безопасности
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Обработка preflight-запросов
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// HTTPSRedirectMiddleware перенаправляет HTTP-запросы на HTTPS.
// Используется при наличии TLS-сертификата или за reverse proxy (X-Forwarded-Proto).
func HTTPSRedirectMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем заголовок от reverse proxy (nginx, traefik и т.д.)
		if r.Header.Get("X-Forwarded-Proto") == "http" {
			target := "https://" + r.Host + r.URL.RequestURI()
			http.Redirect(w, r, target, http.StatusMovedPermanently)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// matchOrigin проверяет, совпадает ли origin с разрешённым (поддерживает несколько через запятую).
func matchOrigin(origin, allowed string) bool {
	for _, a := range strings.Split(allowed, ",") {
		if strings.TrimSpace(a) == origin {
			return true
		}
	}
	return false
}
