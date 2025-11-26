package handlers

import (
	"encoding/json"
	"myapp/models"
	"myapp/services"
	"myapp/utils"
	"net/http"

	"gorm.io/gorm"
)

type MessageRequest struct {
	ChatID  uint   `json:"chat_id"`
	Message string `json:"message"`
}

type MessageResponse struct {
	Response string `json:"response"`
}

func MessageHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Заглушка авторизации
		username, err := utils.ExtractUsername(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Парсим запрос
		var req MessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Проверяем существование пользователя и чата
		_, chat, err := utils.ValidateUserAndChat(db, username, req.ChatID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Получаем контекст чата
		context := services.GetChatContext(db, req.ChatID, 2000)

		// Получаем ответ от модели
		llmResponse := services.GetLLMResponse(context, req.Message)

		// Сохраняем сообщение пользователя
		userMessage := models.Message{
			ChatID:      req.ChatID,
			MessageText: req.Message,
			IsFromUser:  true,
		}
		db.Create(&userMessage)

		// Сохраняем ответ модели
		assistantMessage := models.Message{
			ChatID:      req.ChatID,
			MessageText: llmResponse,
			IsFromUser:  false,
		}
		db.Create(&assistantMessage)

		// Обновляем время последней активности чата
		db.Model(&chat).Update("last_activity", db.NowFunc())

		// Возвращаем ответ
		response := MessageResponse{
			Response: llmResponse,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
