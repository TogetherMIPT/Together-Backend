package utils

import (
	"errors"
	"myapp/models"
	"net/http"

	"gorm.io/gorm"
)

// ExtractUsername извлекает имя пользователя из заголовка
func ExtractUsername(r *http.Request) (string, error) {
	username := r.Header.Get("X-USER-NAME")
	if username == "" {
		return "", errors.New("unauthorized: X-USER-NAME header is required")
	}
	return username, nil
}

// ValidateUserAndChat проверяет существование пользователя и чата
func ValidateUserAndChat(db *gorm.DB, username string, chatID uint) (*models.User, *models.Chat, error) {
	var user models.User
	if err := db.Where("login = ?", username).First(&user).Error; err != nil {
		return nil, nil, errors.New("user not found")
	}

	var chat models.Chat
	if err := db.Where("chat_id = ? AND user_id = ?", chatID, user.UserID).First(&chat).Error; err != nil {
		return nil, nil, errors.New("chat not found")
	}

	return &user, &chat, nil
}
