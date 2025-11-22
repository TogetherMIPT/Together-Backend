package services

import (
	"myapp/models"
	"strings"

	"gorm.io/gorm"
)

// GetChatContext формирует контекст из истории сообщений
func GetChatContext(db *gorm.DB, chatID uint, maxTokens int) string {
	var messages []models.Message
	db.Where("chat_id = ?", chatID).
		Order("creation_datetime asc"). // Берем в хронологическом порядке
		Limit(100).
		Find(&messages)

	var contextBuilder strings.Builder
	tokenCount := 0

	for _, msg := range messages {
		var prefix string
		if msg.IsFromUser {
			prefix = "User: "
		} else {
			prefix = "Assistant: "
		}

		messageStr := prefix + msg.MessageText + "\n"

		// здесь должен быть токенизатор модели LLM
		if tokenCount+len(messageStr) > maxTokens {
			break
		}

		contextBuilder.WriteString(messageStr)
		tokenCount += len(messageStr)
	}

	return contextBuilder.String()
}

// GetLLMResponse получает ответ от LLM модели
func GetLLMResponse(context string, userMessage string) string {
	// Заглушка - в реальности здесь интеграция с rugpt3-small
	// Можно добавить логику для обрезки контекста под конкретную модель

	if context == "" {
		return "Это ответ на ваше сообщение: " + userMessage
	}

	return "На основе контекста нашего разговора, отвечаю: " + userMessage + ". [Контекст был учтен]"
}
