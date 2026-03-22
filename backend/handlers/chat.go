package handlers

import (
	"encoding/json"
	"myapp/middleware"
	"myapp/models"
	"net/http"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// MessageBatchResponse представляет ответ с батчем сообщений
type MessageBatchResponse struct {
	Messages []MessageInfo `json:"messages"`
	Total    int64         `json:"total"`
	Limit    int           `json:"limit"`
	Offset   int           `json:"offset"`
}

// MessageInfo представляет информацию о сообщении
type MessageInfo struct {
	MessageID   uint   `json:"message_id"`
	ChatID      uint   `json:"chat_id"`
	MessageText string `json:"message_text"`
	IsFromUser  bool   `json:"is_from_user"`
	CreatedAt   string `json:"created_at"`
}

// ChatInfo представляет информацию о чате
type ChatInfo struct {
	ChatID    uint   `json:"chat_id"`
	ChatName  string `json:"chat_name"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ChatsResponse представляет ответ со списком чатов
type ChatsResponse struct {
	Chats []ChatInfo `json:"chats"`
	Total int64      `json:"total"`
}

// CreateChatResponse представляет ответ на создание чата
type CreateChatResponse struct {
	ChatID  uint   `json:"chat_id"`
	Message string `json:"message"`
}

// DeleteChatResponse представляет ответ на удаление чата
type DeleteChatResponse struct {
	Message string `json:"message"`
}

// RenameChatRequest представляет запрос на переименование чата
type RenameChatRequest struct {
	ChatName string `json:"chat_name"`
}

// RenameChatResponse представляет ответ на переименование чата
type RenameChatResponse struct {
	Message string `json:"message"`
}

// GetMessageBatchHandler обрабатывает GET /msg_batch/{chatId}
func GetMessageBatchHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем chatId из URL
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "Chat ID is required", http.StatusBadRequest)
			return
		}

		chatID, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 32)
		if err != nil {
			http.Error(w, "Invalid chat ID", http.StatusBadRequest)
			return
		}

		// Получаем параметры limit и offset из query string
		query := r.URL.Query()
		limit := 50 // значение по умолчанию
		offset := 0

		if limitStr := query.Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}

		if offsetStr := query.Get("offset"); offsetStr != "" {
			if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
				offset = o
			}
		}

		// Проверяем существование чата и принадлежность текущему пользователю
		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var chat models.Chat
		if err := db.First(&chat, chatID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to fetch chat", http.StatusInternalServerError)
			return
		}

		if chat.UserID != currentUser.UserID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Получаем общее количество сообщений
		var total int64
		db.Model(&models.Message{}).Where("chat_id = ?", chatID).Count(&total)

		// Получаем сообщения с пагинацией
		var messages []models.Message
		if err := db.Where("chat_id = ?", chatID).
			Order("creation_datetime ASC").
			Limit(limit).
			Offset(offset).
			Find(&messages).Error; err != nil {
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}

		// Преобразуем в ответ
		messageInfos := make([]MessageInfo, len(messages))
		for i, msg := range messages {
			messageInfos[i] = MessageInfo{
				MessageID:   msg.MessageID,
				ChatID:      msg.ChatID,
				MessageText: msg.MessageText,
				IsFromUser:  msg.IsFromUser,
				CreatedAt:   msg.CreationDatetime.Format("2006-01-02T15:04:05Z"),
			}
		}

		response := MessageBatchResponse{
			Messages: messageInfos,
			Total:    total,
			Limit:    limit,
			Offset:   offset,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// GetChatsHandler обрабатывает GET /chats/{userId}
func GetChatsHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем userId из URL
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

		// Проверяем, что запрашивается только собственный список чатов
		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if uint(userID) != currentUser.UserID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Получаем общее количество чатов
		var total int64
		db.Model(&models.Chat{}).Where("user_id = ?", userID).Count(&total)

		// Получаем чаты пользователя
		var chats []models.Chat
		if err := db.Where("user_id = ?", userID).
			Order("creation_datetime DESC").
			Find(&chats).Error; err != nil {
			http.Error(w, "Failed to fetch chats", http.StatusInternalServerError)
			return
		}

		// Преобразуем в ответ
		chatInfos := make([]ChatInfo, len(chats))
		for i, chat := range chats {
			chatInfos[i] = ChatInfo{
				ChatID:    chat.ChatID,
				ChatName:  chat.ChatName,
				IsActive:  chat.IsActive,
				CreatedAt: chat.CreationDatetime.Format("2006-01-02T15:04:05Z"),
				UpdatedAt: chat.UpdatedDatetime.Format("2006-01-02T15:04:05Z"),
			}
		}

		response := ChatsResponse{
			Chats: chatInfos,
			Total: total,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CreateChatHandler обрабатывает POST /chat/{userId}
func CreateChatHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Используем ID аутентифицированного пользователя, игнорируя userId из URL
		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Создаем новый чат
		chat := models.Chat{
			UserID:   currentUser.UserID,
			IsActive: true,
			ChatName: "New Chat",
		}

		if err := db.Create(&chat).Error; err != nil {
			http.Error(w, "Failed to create chat", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := CreateChatResponse{
			ChatID:  chat.ChatID,
			Message: "Chat created successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// RenameChatHandler обрабатывает PUT /chat/{chatId}
func RenameChatHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "Chat ID is required", http.StatusBadRequest)
			return
		}

		chatID, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 32)
		if err != nil {
			http.Error(w, "Invalid chat ID", http.StatusBadRequest)
			return
		}

		var req RenameChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(req.ChatName) == "" {
			http.Error(w, "Chat name cannot be empty", http.StatusBadRequest)
			return
		}

		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var chat models.Chat
		if err := db.First(&chat, chatID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to fetch chat", http.StatusInternalServerError)
			return
		}

		if chat.UserID != currentUser.UserID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		if err := db.Model(&chat).Update("chat_name", req.ChatName).Error; err != nil {
			http.Error(w, "Failed to rename chat", http.StatusInternalServerError)
			return
		}

		response := RenameChatResponse{Message: "Chat renamed successfully"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteChatHandler обрабатывает DELETE /chat/{chatId}
func DeleteChatHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Проверка метода
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем chatId из URL
		pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(pathParts) < 2 {
			http.Error(w, "Chat ID is required", http.StatusBadRequest)
			return
		}

		chatID, err := strconv.ParseUint(pathParts[len(pathParts)-1], 10, 32)
		if err != nil {
			http.Error(w, "Invalid chat ID", http.StatusBadRequest)
			return
		}

		// Проверяем существование чата и принадлежность текущему пользователю
		currentUser := middleware.GetUserFromContext(r)
		if currentUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var chat models.Chat
		if err := db.First(&chat, chatID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Chat not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to fetch chat", http.StatusInternalServerError)
			return
		}

		if chat.UserID != currentUser.UserID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Удаляем чат (CASCADE удалит связанные сообщения)
		if err := db.Delete(&chat).Error; err != nil {
			http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
			return
		}

		// Возвращаем ответ
		response := DeleteChatResponse{
			Message: "Chat deleted successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
