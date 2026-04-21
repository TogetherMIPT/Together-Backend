package handlers

import (
	"encoding/json"
	"myapp/middleware"
	"myapp/models"
	"net/http"

	"gorm.io/gorm"
)

type ClientSurveyRequest struct {
	WithPsychologist bool   `json:"with_psychologist"`
	TherapyRequest   string `json:"therapy_request"`
	TherapyApproach  string `json:"therapy_approach"`
	WeeklyMeetings   int    `json:"weekly_meetings"`
}

type ClientSurveyResponse struct {
	ClientSurveyID uint   `json:"client_survey_id"`
	Message        string `json:"message"`
}

// ClientSurveyHandler обрабатывает POST /client_survey — сохраняет (создаёт или обновляет)
// результат первичного опроса клиента.
func ClientSurveyHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctxUser := middleware.GetUserFromContext(r)
		if ctxUser == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req ClientSurveyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.WeeklyMeetings < 0 {
			http.Error(w, "weekly_meetings must be non-negative", http.StatusBadRequest)
			return
		}

		var survey models.ClientSurvey
		err := db.Where("user_id = ?", ctxUser.UserID).First(&survey).Error

		if err == gorm.ErrRecordNotFound {
			survey = models.ClientSurvey{
				UserID:           ctxUser.UserID,
				WithPsychologist: req.WithPsychologist,
				TherapyRequest:   req.TherapyRequest,
				TherapyApproach:  req.TherapyApproach,
				WeeklyMeetings:   req.WeeklyMeetings,
			}
			if err := db.Create(&survey).Error; err != nil {
				http.Error(w, "Failed to save survey", http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			http.Error(w, "Failed to save survey", http.StatusInternalServerError)
			return
		} else {
			if err := db.Model(&survey).Updates(map[string]interface{}{
				"with_psychologist": req.WithPsychologist,
				"therapy_request":   req.TherapyRequest,
				"therapy_approach":  req.TherapyApproach,
				"weekly_meetings":   req.WeeklyMeetings,
			}).Error; err != nil {
				http.Error(w, "Failed to update survey", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ClientSurveyResponse{
			ClientSurveyID: survey.ClientSurveyID,
			Message:        "Survey saved successfully",
		})
	}
}

// GetClientSurveyHandler обрабатывает GET /client_survey — возвращает текущий опрос клиента.
func GetClientSurveyHandler(db *gorm.DB) http.HandlerFunc {
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

		var survey models.ClientSurvey
		err := db.Where("user_id = ?", ctxUser.UserID).First(&survey).Error
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "Survey not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, "Failed to fetch survey", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(survey)
	}
}
