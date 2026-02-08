package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"myapp/models"
	"gorm.io/gorm"
)

// LLMService представляет клиент для работы с LLM API
type LLMService struct {
	client      *http.Client
	baseURL     string
	modelName   string
	maxRetries  int
	timeout     time.Duration
}

// LLMGenerateRequest структура для запроса к LLM API
type LLMGenerateRequest struct {
	Prompt      string  `json:"prompt"`
	MaxLength   int     `json:"max_length,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// LLMGenerateResponse структура ответа от LLM API
type LLMGenerateResponse struct {
	Response         string `json:"response"`
	Model            string `json:"model"`
	ProcessingTimeMs int    `json:"processing_time_ms"`
}

// NewLLMService создаёт новый экземпляр сервиса LLM
func NewLLMService() *LLMService {
	llmHost := getEnv("LLM_HOST", "llm")
	llmPort := getEnv("LLM_PORT", "8000")
	
	return &LLMService{
		client: &http.Client{
			Timeout: 120 * time.Second, // Долгие генерации требуют большего таймаута
		},
		baseURL:    fmt.Sprintf("http://%s:%s", llmHost, llmPort),
		modelName:  getEnv("MODEL_NAME", "nikrog/rugpt3small_finetuned_psychology_v2"),
		maxRetries: 3,
		timeout:    120 * time.Second,
	}
}

// GetChatContext формирует контекст из истории сообщений
func GetChatContext(db *gorm.DB, chatID uint, maxTokens int) string {
	var messages []models.Message
	db.Where("chat_id = ?", chatID).
		Order("creation_datetime asc").
		Limit(100).
		Find(&messages)

	var contextBuilder strings.Builder
	tokenCount := 0

	for _, msg := range messages {
		var prefix string
		if msg.IsFromUser {
			prefix = "Пользователь: "
		} else {
			prefix = "Психолог: "
		}

		// Оценка токенов: ~1 токен на 4 символа (для русского текста)
		// В реальной реализации используйте токенизатор модели
		messageStr := prefix + msg.MessageText + "\n"
		messageTokens := len(messageStr) / 4

		if tokenCount+messageTokens > maxTokens {
			// Если не помещается текущее сообщение, прекращаем
			// Можно также обрезать старые сообщения для сохранения контекста
			remainingTokens := maxTokens - tokenCount
			if remainingTokens > 0 {
				// Обрезаем сообщение до оставшихся токенов
				allowedChars := remainingTokens * 4
				if len(messageStr) > allowedChars {
					messageStr = messageStr[:allowedChars] + "..."
				}
				contextBuilder.WriteString(messageStr)
			}
			break
		}

		contextBuilder.WriteString(messageStr)
		tokenCount += messageTokens
	}

	return contextBuilder.String()
}

// GetLLMResponse получает ответ от LLM модели через API
func (s *LLMService) GetLLMResponse(context string, userMessage string, options ...LLMOption) (string, error) {
	// Применяем опции
	opts := &LLMOptions{
		MaxLength:   200,
		Temperature: 0.7,
	}
	for _, opt := range options {
		opt(opts)
	}

	// Формируем промпт с контекстом
	// Формат: [Контекст] + [Новое сообщение пользователя] + [Промпт для модели]
	
	// Базовый промпт для психологической модели
	basePrompt := "Ты — профессиональный психолог, который помогает людям разобраться в их чувствах и проблемах. Отвечай эмпатично, поддерживая и задавая уточняющие вопросы. Не давай медицинских советов, а направляй к специалистам при необходимости.\n"
	
	// Собираем финальный промпт
	// Ограничиваем общий размер контекста ~512 токенами (для модели rugpt3-small)
	maxContextTokens := 512 - opts.MaxLength/4 // Оставляем место для генерации
	
	// Обрезаем контекст если слишком длинный
	estimatedContextTokens := len(context) / 4
	if estimatedContextTokens > maxContextTokens {
		// Обрезаем контекст с конца (оставляем последние сообщения)
		allowedChars := maxContextTokens * 4
		if len(context) > allowedChars {
			// Находим последний полный диалог для сохранения структуры
			cutPos := allowedChars
			if cutPos > len(context) {
				cutPos = len(context)
			}
			context = "... " + context[cutPos:]
		}
	}
	
	prompt := basePrompt
	if context != "" {
		prompt += "\n=== История диалога ===\n" + context + "\n"
	}
	prompt += "\n=== Текущий вопрос ===\nПользователь: " + userMessage + "\nПсихолог:"
	
	// Логируем промпт для отладки
	// log.Printf("LLM Prompt (%d chars):\n%s", len(prompt), prompt)
	
	// Вызываем LLM API
	response, err := s.generate(prompt, opts.MaxLength, opts.Temperature)
	if err != nil {
		return "", fmt.Errorf("ошибка генерации ответа: %w", err)
	}
	
	return response, nil
}

// generate вызывает LLM API для генерации текста
func (s *LLMService) generate(prompt string, maxLength int, temperature float64) (string, error) {
	requestBody := LLMGenerateRequest{
		Prompt:      prompt,
		MaxLength:   maxLength,
		Temperature: temperature,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	url := s.baseURL + "/generate"
	
	var lastErr error
	for attempt := 1; attempt <= s.maxRetries; attempt++ {
		resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("попытка %d: %w", attempt, err)
			log.Printf("Ошибка вызова LLM API (попытка %d/%d): %v", attempt, s.maxRetries, err)
			
			if attempt < s.maxRetries {
				waitTime := time.Duration(attempt) * 2 * time.Second
				log.Printf("Повторная попытка через %v...", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("ошибка чтения ответа: %w", err)
			continue
		}

		// Проверяем статус ответа
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("LLM API вернул статус %d: %s", resp.StatusCode, string(body))
			log.Printf("LLM API ошибка (попытка %d/%d): %v", attempt, s.maxRetries, lastErr)
			
			if attempt < s.maxRetries && resp.StatusCode >= 500 {
				// Серверные ошибки можно повторить
				waitTime := time.Duration(attempt) * 3 * time.Second
				time.Sleep(waitTime)
				continue
			}
			return "", lastErr
		}

		// Парсим ответ
		var llmResp LLMGenerateResponse
		if err := json.Unmarshal(body, &llmResp); err != nil {
			return "", fmt.Errorf("ошибка декодирования ответа: %w. Ответ: %s", err, string(body))
		}

		log.Printf("LLM ответ получен за %d мс (модель: %s)", llmResp.ProcessingTimeMs, llmResp.Model)
		
		// Очищаем ответ от лишних префиксов
		response := strings.TrimSpace(llmResp.Response)
		
		// Убираем повтор промпта если он есть в ответе
		if strings.HasPrefix(response, "Психолог:") {
			response = strings.TrimPrefix(response, "Психолог:")
			response = strings.TrimSpace(response)
		}
		
		return response, nil
	}

	return "", fmt.Errorf("не удалось получить ответ от LLM после %d попыток: %w", s.maxRetries, lastErr)
}

// HealthCheck проверяет доступность LLM сервиса
func (s *LLMService) HealthCheck() error {
	resp, err := s.client.Get(s.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("ошибка подключения к LLM сервису: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("LLM сервис недоступен, статус: %d, тело: %s", resp.StatusCode, string(body))
	}

	var healthResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return fmt.Errorf("ошибка декодирования health check: %w", err)
	}

	log.Printf("LLM сервис доступен: %+v", healthResp)
	return nil
}

// Опции для настройки генерации
type LLMOptions struct {
	MaxLength   int
	Temperature float64
}

// LLMOption функциональный паттерн для опций
type LLMOption func(*LLMOptions)

// WithMaxLength устанавливает максимальную длину генерации
func WithMaxLength(length int) LLMOption {
	return func(o *LLMOptions) {
		o.MaxLength = length
	}
}

// WithTemperature устанавливает температуру генерации
func WithTemperature(temp float64) LLMOption {
	return func(o *LLMOptions) {
		o.Temperature = temp
	}
}

// getEnv вспомогательная функция для получения переменных окружения
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
