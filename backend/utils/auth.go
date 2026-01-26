package utils

import (
	"errors"
	"myapp/middleware"
	"myapp/models"
	"net/http"

	"gorm.io/gorm"
)

// ExtractUsername извлекает имя пользователя из заголовка (deprecated, используйте GetUserFromContext)
func ExtractUsername(r *http.Request) (string, error) {
	// Сначала пробуем получить из контекста (новый способ через middleware)
	user := middleware.GetUserFromContext(r)
	if user != nil {
		return user.Login, nil
	}

	// Fallback на старый способ для обратной совместимости
	username := r.Header.Get("X-USER-NAME")
	if username == "" {
		return "", errors.New("unauthorized: authentication required")
	}
	return username, nil
}

// GetAuthenticatedUser возвращает аутентифицированного пользователя из контекста
func GetAuthenticatedUser(r *http.Request) (*models.User, error) {
	user := middleware.GetUserFromContext(r)
	if user == nil {
		return nil, errors.New("unauthorized: authentication required")
	}
	return user, nil
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

// ValidateAuthenticatedUserChat проверяет, что чат принадлежит аутентифицированному пользователю
func ValidateAuthenticatedUserChat(db *gorm.DB, r *http.Request, chatID uint) (*models.User, *models.Chat, error) {
	user, err := GetAuthenticatedUser(r)
	if err != nil {
		return nil, nil, err
	}

	var chat models.Chat
	if err := db.Where("chat_id = ? AND user_id = ?", chatID, user.UserID).First(&chat).Error; err != nil {
		return nil, nil, errors.New("chat not found or access denied")
	}

	return user, &chat, nil
}
