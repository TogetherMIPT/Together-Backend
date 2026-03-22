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
				"Пользователь проходил ежедневные опросы о своём состоянии. "+
					"Каждый показатель оценивается по шкале от 1 до 3 (1 — плохо/низко, 2 — нейтрально/средне, 3 — хорошо/высоко).\n\n"+
					"Статистика за последние %d опросов:\n"+
					"- Среднее настроение: %.2f/3\n"+
					"- Средний уровень тревожности: %.2f/3\n"+
					"- Средний уровень ощущения контроля над жизнью: %.2f/3\n\n"+
					"Детальная история опросов:\n%s\n"+
					"Дай краткое резюме о психологическом состоянии пользователя за последний месяц.",
				count, avgMood, avgAnxiety, avgControl, details,
			)

			var err error
			summary, err = llmService.GetLLMResponse("", summaryPrompt)
			if err != nil {
				log.Printf("LLM summary error: %v", err)
				summary = "Не удалось сформировать резюме."
			}

			recPrompt := fmt.Sprintf(
				"Пользователь проходил ежедневные опросы о своём состоянии (шкала 1–3). "+
					"Среднее настроение: %.2f/3, средняя тревожность: %.2f/3, средний контроль: %.2f/3 за %d опросов за последний месяц.\n"+
					"Дай конкретные практические рекомендации, как улучшить психологическое состояние пользователя.",
				avgMood, avgAnxiety, avgControl, count,
			)

			recommendations, err = llmService.GetLLMResponse("", recPrompt)
			if err != nil {
				log.Printf("LLM recommendations error: %v", err)
				recommendations = "Не удалось сформировать рекомендации."
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
