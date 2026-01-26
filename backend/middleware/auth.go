package middleware

import (
	"context"
	"myapp/models"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// ContextKey тип для ключей контекста
type ContextKey string

const (
	// UserContextKey ключ для хранения пользователя в контексте
	UserContextKey ContextKey = "user"
	// SessionContextKey ключ для хранения сессии в контексте
	SessionContextKey ContextKey = "session"
)

// AuthMiddleware создает middleware для проверки аутентификации
func AuthMiddleware(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Получаем токен из заголовка
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, "Authorization header is required", http.StatusUnauthorized)
				return
			}

			// Удаляем префикс "Bearer " если есть
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			}

			// Ищем сессию по токену
			var session models.Session
			if err := db.Where("token = ?", token).First(&session).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
					return
				}
				http.Error(w, "Failed to validate session", http.StatusInternalServerError)
				return
			}

			// Проверяем, не истекла ли сессия
			if time.Now().After(session.ExpirationDatetime) {
				// Удаляем просроченную сессию
				db.Delete(&session)
				http.Error(w, "Session expired", http.StatusUnauthorized)
				return
			}

			// Находим пользователя
			var user models.User
			if err := db.First(&user, session.UserID).Error; err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Добавляем пользователя и сессию в контекст
			ctx := context.WithValue(r.Context(), UserContextKey, &user)
			ctx = context.WithValue(ctx, SessionContextKey, &session)

			// Передаем запрос дальше
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext извлекает пользователя из контекста запроса
func GetUserFromContext(r *http.Request) *models.User {
	user, ok := r.Context().Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// GetSessionFromContext извлекает сессию из контекста запроса
func GetSessionFromContext(r *http.Request) *models.Session {
	session, ok := r.Context().Value(SessionContextKey).(*models.Session)
	if !ok {
		return nil
	}
	return session
}
