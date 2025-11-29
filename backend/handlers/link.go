package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"myapp/models"
	"myapp/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// LinkTokenResponse представляет ответ с токеном для связывания
type LinkTokenResponse struct {
	Token             string    `json:"token"`
	ExpirationDatetime time.Time `json:"expiration_datetime"`
}

// LinkRequest представляет запрос на связывание пользователей
type LinkRequest struct {
	Token string `json:"token"`
}

// LinkResponse представляет ответ после успешного связывания
type LinkResponse struct {
	LinkedUserName string `json:"linked_user_name"`
	Message        string `json:"message"`
}

// DeleteLinkResponse представляет ответ после удаления связи
type DeleteLinkResponse struct {
	Message string `json:"message"`
}

// GenerateLinkTokenHandler обрабатывает GET /link_token
func GenerateLinkTokenHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodGet {
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

		// Генерируем уникальный токен
		token, err := generateToken()
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Устанавливаем срок действия токена (24 часа)
		expirationTime := time.Now().Add(24 * time.Hour)

		// Создаем запись токена
		linkToken := models.LinkToken{
			Token:              token,
			UserID:             user.UserID,
			ExpirationDatetime: expirationTime,
		}

		// Сохраняем в БД
		if err := db.Create(&linkToken).Error; err != nil {
			http.Error(w, "Failed to create link token", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := LinkTokenResponse{
			Token:             token,
			ExpirationDatetime: expirationTime,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// LinkUsersHandler обрабатывает POST /link
func LinkUsersHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Получаем имя пользователя из заголовка
		username, err := utils.ExtractUsername(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Находим текущего пользователя
		var currentUser models.User
		if err := db.Where("login = ?", username).First(&currentUser).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Парсим запрос
		var req LinkRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация токена
		if req.Token == "" {
			http.Error(w, "Token is required", http.StatusBadRequest)
			return
		}

		// Находим токен в БД
		var linkToken models.LinkToken
		if err := db.Where("token = ?", req.Token).First(&linkToken).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Invalid token", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to verify token", http.StatusInternalServerError)
			return
		}

		// Проверяем срок действия токена
		if time.Now().After(linkToken.ExpirationDatetime) {
			// Удаляем просроченный токен
			db.Delete(&linkToken)
			http.Error(w, "Token has expired", http.StatusBadRequest)
			return
		}

		// Проверяем, не пытается ли пользователь связаться с самим собой
		if linkToken.UserID == currentUser.UserID {
			http.Error(w, "Cannot link to yourself", http.StatusBadRequest)
			return
		}

		// Находим пользователя, которому принадлежит токен
		var targetUser models.User
		if err := db.First(&targetUser, linkToken.UserID).Error; err != nil {
			http.Error(w, "Target user not found", http.StatusNotFound)
			return
		}

		// Проверяем, нет ли уже связи между пользователями
		var existingRelation models.Relation
		err = db.Where(
			"(first_user_id = ? AND second_user_id = ?) OR (first_user_id = ? AND second_user_id = ?)",
			currentUser.UserID, targetUser.UserID, targetUser.UserID, currentUser.UserID,
		).First(&existingRelation).Error

		if err == nil {
			// Связь уже существует
			http.Error(w, "Users are already linked", http.StatusConflict)
			return
		} else if err != gorm.ErrRecordNotFound {
			// Другая ошибка при поиске
			http.Error(w, "Failed to check existing relation", http.StatusInternalServerError)
			return
		}

		// Создаем связь между пользователями
		relation := models.Relation{
			FirstUserID:  currentUser.UserID,
			SecondUserID: targetUser.UserID,
		}

		if err := db.Create(&relation).Error; err != nil {
			http.Error(w, "Failed to create relation", http.StatusInternalServerError)
			return
		}

		// Удаляем одноразовый токен
		if err := db.Delete(&linkToken).Error; err != nil {
			// Логируем ошибку, но не прерываем операцию, так как связь уже создана
			// В реальном приложении здесь можно добавить логирование
		}

		// Возвращаем ответ
		response := LinkResponse{
			LinkedUserName: targetUser.Name,
			Message:        "Users linked successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteLinkHandler обрабатывает DELETE /link/{userId}
func DeleteLinkHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Получаем имя пользователя из заголовка
		username, err := utils.ExtractUsername(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Находим текущего пользователя
		var currentUser models.User
		if err := db.Where("login = ?", username).First(&currentUser).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		// Извлекаем userId из URL
		// Ожидаем формат: /link/123
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		targetUserID, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 32)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		// Проверяем, не пытается ли пользователь удалить связь с самим собой
		if uint(targetUserID) == currentUser.UserID {
			http.Error(w, "Cannot delete link to yourself", http.StatusBadRequest)
			return
		}

		// Находим и удаляем связь
		var relation models.Relation
		err = db.Where(
			"(first_user_id = ? AND second_user_id = ?) OR (first_user_id = ? AND second_user_id = ?)",
			currentUser.UserID, targetUserID, targetUserID, currentUser.UserID,
		).First(&relation).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Relation not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to find relation", http.StatusInternalServerError)
			return
		}

		// Удаляем связь
		if err := db.Delete(&relation).Error; err != nil {
			http.Error(w, "Failed to delete relation", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := DeleteLinkResponse{
			Message: "Relation deleted successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// generateToken генерирует случайный токен
func generateToken() (string, error) {
	bytes := make([]byte, 32) // 32 байта = 256 бит
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
