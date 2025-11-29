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

	// Эндпоинты для чатов
	mux.HandleFunc("/msg_batch/", handlers.GetMessageBatchHandler(database.DB))
	mux.HandleFunc("/chats/", handlers.GetChatsHandler(database.DB))
	mux.HandleFunc("/chat/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateChatHandler(database.DB)(w, r)
		case http.MethodDelete:
			handlers.DeleteChatHandler(database.DB)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Эндпоинты для связей пользователей
	mux.HandleFunc("/link_token", handlers.GenerateLinkTokenHandler(database.DB))
	mux.HandleFunc("/link", handlers.LinkUsersHandler(database.DB))
	mux.HandleFunc("/link/", handlers.DeleteLinkHandler(database.DB))

	// Получаем порт из переменной окружения или используем 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Запускаем HTTP сервер
	addr := ":" + port
	log.Printf("Starting HTTP server on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  POST   /register            - Register new user")
	log.Printf("  PUT    /profile             - Update user profile (requires X-USER-NAME header)")
	log.Printf("  GET    /profile/{id}        - Get user profile by ID")
	log.Printf("  POST   /message             - Send message to chat")
	log.Printf("  GET    /msg_batch/{chatId}  - Get message batch by chat ID (params: limit, offset)")
	log.Printf("  GET    /chats/{userId}      - Get all chats by user ID")
	log.Printf("  POST   /chat/{userId}       - Create new chat for user")
	log.Printf("  DELETE /chat/{chatId}       - Delete chat by chat ID")
	log.Printf("  GET    /link_token          - Generate link token for user linking (requires X-USER-NAME header)")
	log.Printf("  POST   /link                - Link users using token (requires X-USER-NAME header)")
	log.Printf("  DELETE /link/{userId}       - Delete link between users (requires X-USER-NAME header)")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
