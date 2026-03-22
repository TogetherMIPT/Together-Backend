package handlers

import (
	"encoding/json"
	"myapp/middleware"
	"myapp/models"
	"net/http"
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
