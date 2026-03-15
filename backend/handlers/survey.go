package handlers

import (
	"encoding/json"
	"myapp/middleware"
	"myapp/models"
	"net/http"

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

		// Определяем user_id: из аутентифицированного контекста или из тела запроса
		var userID uint
		if ctxUser := middleware.GetUserFromContext(r); ctxUser != nil {
			userID = ctxUser.UserID
		} else {
			userID = req.UserID
		}

		if userID == 0 {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		// Проверяем существование пользователя
		var user models.User
		if err := db.First(&user, userID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to find user", http.StatusInternalServerError)
			return
		}

		survey := models.DailySurvey{
			UserID:        userID,
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
