package main

import (
	"log"
	"myapp/database"
	"myapp/handlers"
	"myapp/middleware"
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

	// Создаем middleware для аутентификации
	authMiddleware := middleware.AuthMiddleware(database.DB)

	// Вспомогательная функция для применения middleware
	withAuth := func(handler http.HandlerFunc) http.Handler {
		return authMiddleware(http.HandlerFunc(handler))
	}

	// Публичные эндпоинты (без аутентификации)
	mux.HandleFunc("/register", handlers.RegisterHandler(database.DB))
	mux.HandleFunc("/login", handlers.LoginHandler(database.DB))
	mux.HandleFunc("/logout", handlers.LogoutHandler(database.DB))

	// Защищённые эндпоинты (требуют аутентификации)
	mux.Handle("/profile", withAuth(handlers.UpdateProfileHandler(database.DB)))
	mux.Handle("/profile/", withAuth(handlers.GetProfileHandler(database.DB)))

	// Эндпоинт для сообщений
	mux.Handle("/message", withAuth(handlers.MessageHandler(database.DB)))

	// Эндпоинты для чатов
	mux.Handle("/msg_batch/", withAuth(handlers.GetMessageBatchHandler(database.DB)))
	mux.Handle("/chats/", withAuth(handlers.GetChatsHandler(database.DB)))
	mux.Handle("/chat/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.CreateChatHandler(database.DB)(w, r)
		case http.MethodDelete:
			handlers.DeleteChatHandler(database.DB)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Эндпоинты для связей пользователей
	mux.Handle("/link_token", withAuth(handlers.GenerateLinkTokenHandler(database.DB)))
	mux.Handle("/link", withAuth(handlers.LinkUsersHandler(database.DB)))
	mux.Handle("/link/", withAuth(handlers.DeleteLinkHandler(database.DB)))

	// Получаем порт из переменной окружения или используем 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Запускаем HTTP сервер
	addr := ":" + port
	log.Printf("Starting HTTP server on %s", addr)
	log.Printf("Available endpoints:")
	log.Printf("  PUBLIC (no auth required):")
	log.Printf("    POST   /register            - Register new user")
	log.Printf("    POST   /login               - Login and get session token")
	log.Printf("    POST   /logout              - Logout and invalidate session")
	log.Printf("  PROTECTED (requires Authorization header with session token):")
	log.Printf("    PUT    /profile             - Update user profile")
	log.Printf("    GET    /profile/{id}        - Get user profile by ID")
	log.Printf("    POST   /message             - Send message to chat")
	log.Printf("    GET    /msg_batch/{chatId}  - Get message batch by chat ID (params: limit, offset)")
	log.Printf("    GET    /chats/{userId}      - Get all chats by user ID")
	log.Printf("    POST   /chat/{userId}       - Create new chat for user")
	log.Printf("    DELETE /chat/{chatId}       - Delete chat by chat ID")
	log.Printf("    GET    /link_token          - Generate link token for user linking")
	log.Printf("    POST   /link                - Link users using token")
	log.Printf("    DELETE /link/{userId}       - Delete link between users")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
