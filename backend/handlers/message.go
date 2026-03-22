package handlers

import (
	"log"
	"time"
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
	MaxLength   int     `json:"max_length,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

type MessageResponse struct {
	Response string `json:"response"`
}

func MessageHandler(db *gorm.DB) http.HandlerFunc {
	// Инициализируем LLM сервис при старте
	llmService := services.NewLLMService()
	
	// Проверяем доступность LLM сервиса
	if err := llmService.HealthCheck(); err != nil {
		log.Printf("LLM сервис недоступен при старте: %v", err)
	} else {
		log.Println("LLM сервис доступен")
	}
	
	// Запускаем фоновую проверку здоровья каждые 5 минут
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			if err := llmService.HealthCheck(); err != nil {
				log.Printf("LLM сервис недоступен: %v", err)
			}
		}
	}()

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
		user, chat, err := utils.ValidateUserAndChat(db, username, req.ChatID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		// Проверяем доступ к подписке
		now := time.Now()
		registeredRecently := now.Sub(user.CreationDatetime) <= 7*24*time.Hour
		paidRecently := user.LastPaymentDatetime != nil && now.Sub(*user.LastPaymentDatetime) <= 30*24*time.Hour
		if !registeredRecently && !paidRecently {
			// Фиксируем дату "оплаты" и сохраняем сообщение об оплате в историю чата
			paymentMessage := "Привет! Это команда Together. Предлагаем пообщаться и обсудить твой опыт использования приложения!"
			db.Model(user).Update("last_payment_datetime", now)

			// Уведомляем администратора об окончании триала
			go services.SendTrialEndAdminNotification(user.Email, user.Login)

			assistantPaymentMsg := models.Message{
				ChatID:      req.ChatID,
				MessageText: paymentMessage,
				IsFromUser:  false,
			}
			db.Create(&assistantPaymentMsg)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(MessageResponse{Response: paymentMessage})
			return
		}

		// Получаем контекст чата
		context := services.GetChatContext(db, req.ChatID, 2000, 30)

		// Сохраняем сообщение пользователя
		userMessage := models.Message{
			ChatID:      req.ChatID,
			MessageText: req.Message,
			IsFromUser:  true,
		}

		if err := db.Create(&userMessage).Error; err != nil {
			log.Printf("Ошибка сохранения сообщения: %v", err)
			http.Error(w, "Failed to save message", http.StatusInternalServerError)
			return
		}

		// Формируем опции для генерации
		opts := []services.LLMOption{}
		if req.MaxLength > 0 {
			opts = append(opts, services.WithMaxLength(req.MaxLength))
		}
		if req.Temperature > 0 {
			opts = append(opts, services.WithTemperature(req.Temperature))
		}
		
		// Получаем ответ от психолога (LLM)
		llmResponse, err := llmService.GetLLMResponse(context, req.Message, opts...)
		if err != nil {
			log.Printf("Ошибка генерации ответа LLM: %v", err)
			http.Error(w, "Failed to generate response", http.StatusInternalServerError)
			return
		}

		// Сохраняем ответ модели
		assistantMessage := models.Message{
			ChatID:      req.ChatID,
			MessageText: llmResponse,
			IsFromUser:  false,
		}

		if err := db.Create(&assistantMessage).Error; err != nil {
			log.Printf("Ошибка сохранения ответа: %v", err)
			// Не возвращаем ошибку клиенту, т.к. сообщение пользователя уже сохранено
		}
		
		// Обновляем время последней активности чата (такого поля у нас в БД нет)
		//db.Model(&chat).Update("last_activity", db.NowFunc())
		if err := db.Model(&chat).Update("is_active", true).Error; err != nil {
			log.Printf("Ошибка активации чата: %v", err)
			http.Error(w, "Failed to activate chat", http.StatusInternalServerError)
			return
		}

		if err := db.Model(&models.Chat{}).
		Where("chat_id <> ? AND is_active = ?", req.ChatID, true).Update("is_active", false).Error; err != nil {
			log.Printf("Ошибка деактивации старых чатов: %v", err)
			http.Error(w, "Failed to deactivate old chats", http.StatusInternalServerError)
			return
		}

		go generateChatNameIfNeeded(db, llmService, req.ChatID, req.Message)

		// Возвращаем ответ
		response := MessageResponse{
			Response: llmResponse,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}


func generateChatNameIfNeeded(db *gorm.DB, llmService services.LLMServiceInterface, chatID uint, firstMessage string) {
	// Небольшая задержка, чтобы не конкурировать с основным запросом
	time.Sleep(500 * time.Millisecond)
	
	// Проверяем, первое ли это сообщение
	var count int64
	if err := db.Model(&models.Message{}).
		Where("chat_id = ?", chatID).
		Count(&count).Error; err != nil {
		log.Printf("Ошибка подсчёта сообщений: %v", err)
		return
	}
	
	// Генерируем название только для первого сообщения пользователя
	// (учитываем, что текущее сообщение уже сохранено)
	if count <= 2 {
		name := llmService.GenerateChatName(firstMessage)
		if name != "" && name != "Новый чат" {
			// Обновляем название чата
			if err := db.Model(&models.Chat{}).
				Where("chat_id = ?", chatID).
				Update("chat_name", name).Error; err != nil {
				log.Printf("Ошибка обновления chat_name: %v", err)
			} else {
				log.Printf("Чат %d: название = \"%s\"", chatID, name)
			}
		}
	}
}