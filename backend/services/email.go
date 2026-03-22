package services

import (
	"log"
	"myapp/models"
	"net/smtp"
	"os"
	"time"

	"gorm.io/gorm"
)

func smtpEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func sendDailyNotifications(db *gorm.DB) {
	var users []models.User
	if err := db.Select("email").Find(&users).Error; err != nil {
		log.Printf("Email scheduler: failed to fetch users: %v", err)
		return
	}

	host := smtpEnv("SMTP_HOST", "smtp.gmail.com")
	port := smtpEnv("SMTP_PORT", "587")
	from := smtpEnv("SMTP_USER", "fritata.artemyeva@gmail.com")
	password := smtpEnv("SMTP_PASSWORD", "YY7xDrJh7UoAh1UD6x34")

	auth := smtp.PlainAuth("", from, password, host)
	addr := host + ":" + port

	subject := "Напоминание от Together"
	body := "Здравствуйте, это Together. Пожалуйста, пройдите опрос состояния в приложении."

	sent, skipped := 0, 0
	for _, user := range users {
		if user.Email == "" {
			skipped++
			continue
		}
		msg := []byte(
			"From: " + from + "\r\n" +
				"To: " + user.Email + "\r\n" +
				"Subject: " + subject + "\r\n" +
				"Content-Type: text/plain; charset=UTF-8\r\n" +
				"\r\n" +
				body + "\r\n",
		)
		if err := smtp.SendMail(addr, auth, from, []string{user.Email}, msg); err != nil {
			log.Printf("Email scheduler: failed to send to %s: %v", user.Email, err)
		} else {
			sent++
		}
	}
	log.Printf("Email scheduler: sent %d, skipped %d (no email), total %d users", sent, skipped, len(users))
}

// StartDailyEmailScheduler запускает фоновую горутину, которая каждый день в 20:00 по Москве
// отправляет всем зарегистрированным пользователям уведомление о прохождении опроса.
func StartDailyEmailScheduler(db *gorm.DB) {
	go func() {
		moscowLoc, err := time.LoadLocation("Europe/Moscow")
		if err != nil {
			log.Printf("Email scheduler: cannot load Europe/Moscow, falling back to UTC+3: %v", err)
			moscowLoc = time.FixedZone("MSK", 3*60*60)
		}

		for {
			now := time.Now().In(moscowLoc)
			next := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, moscowLoc)
			if !next.After(now) {
				next = next.Add(24 * time.Hour)
			}
			delay := time.Until(next)
			log.Printf("Email scheduler: next run at %s (in %s)",
				next.Format("2006-01-02 15:04:05 MST"), delay.Round(time.Second))
			time.Sleep(delay)
			sendDailyNotifications(db)
		}
	}()
}

// SendTrialEndAdminNotification отправляет письмо администратору об окончании триала у пользователя.
func SendTrialEndAdminNotification(userEmail, userLogin string) {
	host := smtpEnv("SMTP_HOST", "smtp.gmail.com")
	port := smtpEnv("SMTP_PORT", "587")
	from := smtpEnv("SMTP_USER", "fritata.artemyeva@gmail.com")
	password := smtpEnv("SMTP_PASSWORD", "YY7xDrJh7UoAh1UD6x34")

	auth := smtp.PlainAuth("", from, password, host)
	addr := host + ":" + port

	adminEmail := "fritata.artemyeva@gmail.com"

	identifier := userEmail
	if identifier == "" {
		identifier = userLogin
	}

	subject := "Уведомление: пользователь достиг окончания триала"
	body := "Пользователь " + identifier + " достиг окончания триал."

	msg := []byte(
		"From: " + from + "\r\n" +
			"To: " + adminEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" +
			body + "\r\n",
	)
	if err := smtp.SendMail(addr, auth, from, []string{adminEmail}, msg); err != nil {
		log.Printf("Trial end admin notification: failed to send: %v", err)
	} else {
		log.Printf("Trial end admin notification: sent for user %s", identifier)
	}
}
