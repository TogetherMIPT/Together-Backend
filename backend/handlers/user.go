package handlers

import (
	"encoding/json"
	"myapp/crypto"
	"myapp/middleware"
	"myapp/models"
	"myapp/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Country  string `json:"country,omitempty"`
	City     string `json:"city,omitempty"`
	Gender   string `json:"gender,omitempty"`
}

// RegisterResponse представляет ответ на регистрацию
type RegisterResponse struct {
	UserID  uint   `json:"user_id"`
	Login   string `json:"login"`
	Message string `json:"message"`
}

// UpdateProfileRequest представляет запрос на обновление профиля
type UpdateProfileRequest struct {
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Country   string `json:"country,omitempty"`
	City      string `json:"city,omitempty"`
	Gender    string `json:"gender,omitempty"`
	Birthdate string `json:"birthdate,omitempty"` // Формат: YYYY-MM-DD
}

// ProfileResponse представляет ответ с данными профиля
type ProfileResponse struct {
	UserID    uint      `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Login     string    `json:"login"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Birthdate time.Time `json:"birthdate"`
	Gender    string    `json:"gender"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterHandler обрабатывает POST /register
func RegisterHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Парсим запрос
		var req RegisterRequest
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

		// Валидация пароля
		if err := utils.ValidatePassword(req.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Проверяем, не существует ли уже пользователь с таким логином
		var existingUser models.User
		if err := db.Where("login = ?", req.Login).First(&existingUser).Error; err == nil {
			http.Error(w, "User with this login already exists", http.StatusConflict)
			return
		}

		// Хешируем пароль
		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Создаем нового пользователя
		user := models.User{
			Login:    req.Login, // На данный момент логином является email
			Password: hashedPassword,
			Name:     req.Name,
			Email:    req.Login,
			Country:  req.Country,
			City:     req.City,
			Gender:   req.Gender,
		}

		// Сохраняем в БД
		if err := db.Create(&user).Error; err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := RegisterResponse{
			UserID:  user.UserID,
			Login:   user.Login,
			Message: "User registered successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// UpdateProfileHandler обрабатывает PUT /profile
func UpdateProfileHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Получаем имя пользователя из заголовка
		username, err := utils.ExtractUsername(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Находим пользователя
		var user models.User
		if err := db.Where("login = ?", username).First(&user).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Парсим запрос
		var req UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Обновляем поля, если они переданы (чувствительные поля шифруются перед сохранением)
		updates := make(map[string]interface{})

		if req.Name != "" {
			enc, err := crypto.Encrypt(req.Name)
			if err != nil {
				http.Error(w, "Failed to process name", http.StatusInternalServerError)
				return
			}
			updates["name"] = enc
		}
		if req.Email != "" {
			enc, err := crypto.Encrypt(req.Email)
			if err != nil {
				http.Error(w, "Failed to process email", http.StatusInternalServerError)
				return
			}
			updates["email"] = enc
		}
		if req.Country != "" {
			enc, err := crypto.Encrypt(req.Country)
			if err != nil {
				http.Error(w, "Failed to process country", http.StatusInternalServerError)
				return
			}
			updates["country"] = enc
		}
		if req.City != "" {
			enc, err := crypto.Encrypt(req.City)
			if err != nil {
				http.Error(w, "Failed to process city", http.StatusInternalServerError)
				return
			}
			updates["city"] = enc
		}
		if req.Gender != "" {
			enc, err := crypto.Encrypt(req.Gender)
			if err != nil {
				http.Error(w, "Failed to process gender", http.StatusInternalServerError)
				return
			}
			updates["gender"] = enc
		}
		if req.Birthdate != "" {
			birthdate, err := time.Parse("2006-01-02", req.Birthdate)
			if err != nil {
				http.Error(w, "Invalid birthdate format. Use YYYY-MM-DD", http.StatusBadRequest)
				return
			}
			updates["birthdate"] = birthdate
		}

		// Применяем обновления
		if len(updates) > 0 {
			if err := db.Model(&user).Updates(updates).Error; err != nil {
				http.Error(w, "Failed to update profile", http.StatusInternalServerError)
				return
			}
		}

		// Загружаем обновленного пользователя
		if err := db.First(&user, user.UserID).Error; err != nil {
			http.Error(w, "Failed to fetch updated profile", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := ProfileResponse{
			UserID:    user.UserID,
			Name:      user.Name,
			Email:     user.Email,
			Login:     user.Login,
			Country:   user.Country,
			City:      user.City,
			Birthdate: user.Birthdate,
			Gender:    user.Gender,
			CreatedAt: user.CreationDatetime,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetProfileHandler обрабатывает GET /profile/{id}
func GetProfileHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем ID из URL
		// Ожидаем формат: /profile/123
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 32)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Разрешаем просматривать только собственный профиль
		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if uint(userID) != currentUser.UserID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Возвращаем ответ (без пароля!)
		response := ProfileResponse{
			UserID:    currentUser.UserID,
			Name:      currentUser.Name,
			Email:     currentUser.Email,
			Login:     currentUser.Login,
			Country:   currentUser.Country,
			City:      currentUser.City,
			Birthdate: currentUser.Birthdate,
			Gender:    currentUser.Gender,
			CreatedAt: currentUser.CreationDatetime,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
