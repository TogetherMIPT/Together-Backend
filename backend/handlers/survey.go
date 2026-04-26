package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"myapp/middleware"
	"myapp/models"
	"myapp/services"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DailySurveyRequest представляет запрос с данными ежедневного опроса
type DailySurveyRequest struct {
	UserID        uint `json:"user_id"`
	MoodAnswer    int  `json:"mood_answer"`
	AnxietyAnswer int  `json:"anxiety_answer"`
	ControlAnswer int  `json:"control_answer"`
}

// DailySurveyResponse представляет ответ после сохранения опроса
type DailySurveyResponse struct {
	SurveyID uint   `json:"survey_id"`
	Message  string `json:"message"`
}

// DailySurveyHandler обрабатывает POST /survey
func DailySurveyHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req DailySurveyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация ответов (допустимые значения: 1, 2, 3)
		for _, answer := range []int{req.MoodAnswer, req.AnxietyAnswer, req.ControlAnswer} {
			if answer < 1 || answer > 3 {
				http.Error(w, "Each answer must be 1, 2, or 3", http.StatusBadRequest)
				return
			}
		}

		// Используем только ID аутентифицированного пользователя
		ctxUser := middleware.GetUserFromContext(r)
		if ctxUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		survey := models.DailySurvey{
			UserID:        ctxUser.UserID,
			MoodAnswer:    req.MoodAnswer,
			AnxietyAnswer: req.AnxietyAnswer,
			ControlAnswer: req.ControlAnswer,
		}

		if err := db.Create(&survey).Error; err != nil {
			http.Error(w, "Failed to save survey", http.StatusInternalServerError)
			return
		}

		response := DailySurveyResponse{
			SurveyID: survey.SurveyID,
			Message:  "Survey saved successfully",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	}
}

// SurveyStatusResponse представляет ответ о статусе прохождения опроса
type SurveyStatusResponse struct {
	Completed bool `json:"completed"`
}

// SurveyStatusHandler обрабатывает GET /survey/status
// Возвращает информацию о том, проходил ли клиент сегодня опрос
func SurveyStatusHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctxUser := middleware.GetUserFromContext(r)
		if ctxUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		now := time.Now()
		startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var count int64
		if err := db.Model(&models.DailySurvey{}).
			Where("user_id = ? AND creation_datetime >= ? AND creation_datetime < ?", ctxUser.UserID, startOfDay, endOfDay).
			Count(&count).Error; err != nil {
			http.Error(w, "Failed to check survey status", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SurveyStatusResponse{Completed: count > 0})
	}
}

// SurveyHistoryEntry представляет одну запись опроса в ответе истории
type SurveyHistoryEntry struct {
	SurveyID         uint      `json:"survey_id"`
	MoodAnswer       int       `json:"mood_answer"`
	AnxietyAnswer    int       `json:"anxiety_answer"`
	ControlAnswer    int       `json:"control_answer"`
	CreationDatetime time.Time `json:"creation_datetime"`
}

// SurveyHistoryResponse представляет ответ эндпоинта истории опросов
type SurveyHistoryResponse struct {
	Surveys         []SurveyHistoryEntry `json:"surveys"`
	Summary         string               `json:"summary"`
	Recommendations string               `json:"recommendations"`
}

// Убирает вопросы и фразы, характерные для диалога
func sanitizeLLMResponse(text string) string {
	// Удаляем прямые вопросы (упрощённая эвристика)
	questionPatterns := []string{
		`?\s*`, // вопросительный знак в конце
		`(?i)хочешь обсудить`,
		`(?i)давай разберём`,
		`(?i)расскажи подробнее`,
		`(?i)были ли у тебя`,
		`(?i)чувствуешь ли`,
		`(?i)как ты обычно`,
	}
	
	result := text
	for _, pattern := range questionPatterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}
	
	// Обрезаем последние предложения, если они содержат призыв к диалогу
	dialogEndings := []string{
		"Если хочешь, можем обсудить",
		"Давай поговорим об этом",
		"Напиши, если нужны уточнения",
	}
	for _, ending := range dialogEndings {
		if idx := strings.Index(result, ending); idx != -1 {
			result = strings.TrimSpace(result[:idx])
		}
	}
	
	return strings.TrimSpace(result)
}

// SurveyHistoryHandler обрабатывает GET /survey/history
// Возвращает историю ответов пользователя за последний месяц,
// а также резюме и рекомендации от LLM.
func SurveyHistoryHandler(db *gorm.DB) http.HandlerFunc {
	llmService := services.NewLLMService()

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctxUser := middleware.GetUserFromContext(r)
		if ctxUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		since := time.Now().AddDate(0, -1, 0)

		var surveys []models.DailySurvey
		if err := db.Where("user_id = ? AND creation_datetime >= ?", ctxUser.UserID, since).
			Order("creation_datetime ASC").
			Find(&surveys).Error; err != nil {
			http.Error(w, "Failed to fetch survey history", http.StatusInternalServerError)
			return
		}

		entries := make([]SurveyHistoryEntry, len(surveys))
		for i, s := range surveys {
			entries[i] = SurveyHistoryEntry{
				SurveyID:         s.SurveyID,
				MoodAnswer:       s.MoodAnswer,
				AnxietyAnswer:    s.AnxietyAnswer,
				ControlAnswer:    s.ControlAnswer,
				CreationDatetime: s.CreationDatetime,
			}
		}

		summary := ""
		recommendations := ""

		if len(surveys) > 0 {
			now := time.Now()
			today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

			var cached models.SurveyRecommendation
			cacheErr := db.Where("user_id = ? AND date = ?", ctxUser.UserID, today).
				First(&cached).Error

			if cacheErr == nil {
				// Используем кэшированные рекомендации
				summary = cached.Summary
				recommendations = cached.Recommendations
			} else {
				// Генерируем новые рекомендации через LLM
				var totalMood, totalAnxiety, totalControl int
				for _, s := range surveys {
					totalMood += s.MoodAnswer
					totalAnxiety += s.AnxietyAnswer
					totalControl += s.ControlAnswer
				}
				count := len(surveys)
				avgMood := float64(totalMood) / float64(count)
				avgAnxiety := float64(totalAnxiety) / float64(count)
				avgControl := float64(totalControl) / float64(count)

				details := buildSurveyDetails(surveys)

				summaryPrompt := fmt.Sprintf(
					"Ты — аналитик психологического состояния. Твоя задача — дать краткое объективное резюме на основе данных опросов.\n\n"+
					"ПРАВИЛА ОТВЕТА:\n"+
					"- Пиши в третьем лице (о пользователе), не обращайся к нему напрямую.\n"+
					"- НЕ задавай никаких вопросов пользователю.\n"+
					"- НЕ предлагай продолжить диалог или обсудить что-то.\n"+
					"- Используй только факты из предоставленных данных.\n"+
					"- Объём: 3-5 предложений.\n\n"+
					"Данные:\n"+
					"Пользователь проходил ежедневные опросы о своём состоянии. "+
						"Каждый показатель оценивается по шкале от 1 до 3 (1 — плохо/низко, 2 — нейтрально/средне, 3 — хорошо/высоко).\n\n"+
						"Статистика за последние %d опросов:\n"+
						"- Среднее настроение: %.2f/3\n"+
						"- Средний уровень тревожности: %.2f/3\n"+
						"- Средний уровень ощущения контроля над жизнью: %.2f/3\n\n"+
						"Детальная история опросов:\n%s\n"+
						"Сформируй краткое резюме о психологическом состоянии пользователя за последний месяц.",
					count, avgMood, avgAnxiety, avgControl, details,
				)

				var err error
				summary, err = llmService.GetLLMResponse("", summaryPrompt)
				if err != nil {
					log.Printf("LLM summary error: %v", err)
					summary = "Не удалось сформировать резюме."
				} else {
					summary = sanitizeLLMResponse(summary) // пост-обработка ответа от LLM-модели
					}

				recPrompt := fmt.Sprintf(
					"Ты — эксперт по ментальному здоровью. Сформируй список практических рекомендаций на основе статистики опросов.\n\n"+
					"ПРАВИЛА ОТВЕТА:\n"+
					"- Пиши в повелительном или безличном формате (например, «Попробуйте...», «Рекомендуется...»).\n"+
					"- НЕ задавай вопросов пользователю.\n"+
					"- НЕ используй фразы типа «если хочешь, обсудим», «давай разберём».\n"+
					"- Дай 3-5 конкретных, выполнимых рекомендаций.\n"+
					"- Избегай общих фраз, привязывай советы к данным.\n\n"+
					"Данные:\n"+
					"Пользователь проходил ежедневные опросы о своём состоянии (шкала 1–3, где 1 — плохо/низко, 2 — нейтрально/средне, 3 — хорошо/высоко). "+
						"Среднее настроение: %.2f/3, средняя тревожность: %.2f/3, средний контроль: %.2f/3 за %d опросов за последний месяц.\n"+
						"Дай конкретные практические рекомендации, как улучшить психологическое состояние пользователя.",
					avgMood, avgAnxiety, avgControl, count,
				)

				recommendations, err = llmService.GetLLMResponse("", recPrompt)
				if err != nil {
					log.Printf("LLM recommendations error: %v", err)
					recommendations = "Не удалось сформировать рекомендации."
				} else {
					recommendations = sanitizeLLMResponse(recommendations) // пост-обработка ответа от LLM-модели
					}
				// Сохраняем результат в кэш
				rec := models.SurveyRecommendation{
					UserID:          ctxUser.UserID,
					Date:            today,
					Summary:         summary,
					Recommendations: recommendations,
				}
				if saveErr := db.Create(&rec).Error; saveErr != nil {
					log.Printf("Failed to cache survey recommendations: %v", saveErr)
				}
			}
		}

		resp := SurveyHistoryResponse{
			Surveys:         entries,
			Summary:         summary,
			Recommendations: recommendations,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// buildSurveyDetails форматирует список опросов для передачи в LLM
func buildSurveyDetails(surveys []models.DailySurvey) string {
	var sb strings.Builder
	for _, s := range surveys {
		sb.WriteString(fmt.Sprintf("  %s — настроение: %d, тревожность: %d, контроль: %d\n",
			s.CreationDatetime.Format("02.01.2006"), s.MoodAnswer, s.AnxietyAnswer, s.ControlAnswer))
	}
	return sb.String()
}
