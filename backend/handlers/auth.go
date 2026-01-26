package handlers

import (
	"encoding/json"
	"myapp/models"
	"myapp/utils"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginResponse представляет ответ на вход
type LoginResponse struct {
	Token     string    `json:"token"`
	UserID    uint      `json:"user_id"`
	Login     string    `json:"login"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// LogoutResponse представляет ответ на выход
type LogoutResponse struct {
	Message string `json:"message"`
}

// LoginHandler обрабатывает POST /login
func LoginHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Парсим запрос
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация обязательных полей
		if req.Login == "" {
			http.Error(w, "Login is required", http.StatusBadRequest)
			return
		}
		if req.Password == "" {
			http.Error(w, "Password is required", http.StatusBadRequest)
			return
		}

		// Находим пользователя по логину
		var user models.User
		if err := db.Where("login = ?", req.Login).First(&user).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Invalid login or password", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Failed to find user", http.StatusInternalServerError)
			return
		}

		// Проверяем пароль
		if !utils.CheckPassword(req.Password, user.Password) {
			http.Error(w, "Invalid login or password", http.StatusUnauthorized)
			return
		}

		// Генерируем токен сессии
		token := uuid.New().String()
		expiresAt := time.Now().Add(24 * time.Hour) // Сессия действительна 24 часа

		// Создаем сессию
		session := models.Session{
			Token:              token,
			UserID:             user.UserID,
			ExpirationDatetime: expiresAt,
		}

		if err := db.Create(&session).Error; err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := LoginResponse{
			Token:     token,
			UserID:    user.UserID,
			Login:     user.Login,
			ExpiresAt: expiresAt,
			Message:   "Login successful",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}

// LogoutHandler обрабатывает POST /logout
func LogoutHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

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

		// Удаляем сессию
		result := db.Where("token = ?", token).Delete(&models.Session{})
		if result.Error != nil {
			http.Error(w, "Failed to logout", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "Session not found or already expired", http.StatusUnauthorized)
			return
		}

		// Возвращаем ответ
		response := LogoutResponse{
			Message: "Logout successful",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
