package main

import (
	"log"
	"myapp/database"
)

func main() {
	// Получаем конфигурацию БД
	dbConfig := database.GetDefaultConfig()

	// Подключаемся к базе данных
	if err := database.Connect(dbConfig); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Выполняем миграции
	if err := database.Migrate(database.DB); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Application started successfully!")

	// Пример создания пользователя с хешированным паролем
	// password := "SecurePass123!"
	// hashedPassword, err := utils.HashPassword(password)
	// if err != nil {
	// 	log.Printf("Error hashing password: %v", err)
	// 	return
	// }

	// user := &models.User{
	// 	Name:      "Иван Иванов",
	// 	Email:     "ivan@example.com",
	// 	Login:     "ivan123",
	// 	Country:   "Russia",
	// 	City:      "Moscow",
	// 	Birthdate: time.Date(1990, 5, 15, 0, 0, 0, 0, time.UTC),
	// 	Gender:    "male",
	// 	Password:  hashedPassword,
	// }

	// // Создаем пользователя в БД
	// if err := database.DB.Create(user).Error; err != nil {
	// 	log.Printf("Error creating user: %v", err)
	// } else {
	// 	log.Printf("User created with ID: %d", user.UserID)
	// }

	// Пример создания токена для пользователя
	// token := &models.LinkToken{
	// 	Token:              uuid.New().String(),
	// 	UserID:             user.UserID,
	// 	ExpirationDatetime: time.Now().Add(24 * time.Hour), // Токен действителен 24 часа
	// }

	// if err := database.DB.Create(token).Error; err != nil {
	// 	log.Printf("Error creating token: %v", err)
	// } else {
	// 	log.Printf("Token created: %s", token.Token)
	// }

	// Пример создания чата
	// chat := &models.Chat{
	// 	UserID:   user.UserID,
	// 	ChatName: "Мой первый чат",
	// 	IsActive: true,
	// }

	// if err := database.DB.Create(chat).Error; err != nil {
	// 	log.Printf("Error creating chat: %v", err)
	// } else {
	// 	log.Printf("Chat created with ID: %d", chat.ChatID)
	// }

	// Пример создания сообщения
	// message := &models.Message{
	// 	ChatID:      chat.ChatID,
	// 	MessageText: "Привет, это первое сообщение!",
	// }

	// if err := database.DB.Create(message).Error; err != nil {
	// 	log.Printf("Error creating message: %v", err)
	// } else {
	// 	log.Printf("Message created with ID: %d", message.MessageID)
	// }

	// Пример поиска пользователя с загрузкой связанных данных
	// var foundUser models.User
	// if err := database.DB.Preload("Chats").Preload("LinkTokens").First(&foundUser, user.UserID).Error; err != nil {
	// 	log.Printf("Error finding user: %v", err)
	// } else {
	// 	log.Printf("Found user: %s with %d chats and %d tokens",
	// 		foundUser.Name, len(foundUser.Chats), len(foundUser.LinkTokens))
	// }
}
