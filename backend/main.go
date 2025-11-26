package main

import (
	"log"
	"myapp/database"
	"myapp/handlers"
	"net/http"
	"os"
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

	log.Println("Database connected and migrated successfully!")

	// Настраиваем роутинг
	mux := http.NewServeMux()

	// Пользовательские эндпоинты
	mux.HandleFunc("/register", handlers.RegisterHandler(database.DB))
	mux.HandleFunc("/profile", handlers.UpdateProfileHandler(database.DB))
	mux.HandleFunc("/profile/", handlers.GetProfileHandler(database.DB))

	// Эндпоинт для сообщений (если нужен)
	mux.HandleFunc("/message", handlers.MessageHandler(database.DB))

	// Получаем порт из переменной окружения или используем 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Запускаем HTTP сервер
	addr := ":" + port
	log.Printf("Starting HTTP server on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  POST   /register      - Register new user")
	log.Printf("  PUT    /profile       - Update user profile (requires X-USER-NAME header)")
	log.Printf("  GET    /profile/{id}  - Get user profile by ID")
	log.Printf("  POST   /message       - Send message to chat")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
