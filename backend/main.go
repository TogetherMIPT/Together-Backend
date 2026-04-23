package main

import (
	"log"
	"myapp/crypto"
	"myapp/database"
	"myapp/handlers"
	"myapp/middleware"
	"myapp/services"
	"net/http"
	"os"
)

func main() {
	// Загружаем ключ шифрования из окружения (обязателен в production)
	encKey := os.Getenv("ENCRYPTION_KEY")
	if encKey == "" {
		log.Fatal("ENCRYPTION_KEY environment variable is required")
	}
	crypto.EncryptionKey = encKey

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

	// Запускаем планировщик ежедневных email-уведомлений (20:00 по Москве)
	services.StartDailyEmailScheduler(database.DB)

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
		case http.MethodPut:
			handlers.RenameChatHandler(database.DB)(w, r)
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

	// Эндпоинты для ежедневного опроса состояния
	mux.Handle("/survey", withAuth(handlers.DailySurveyHandler(database.DB)))
	mux.Handle("/survey/status", withAuth(handlers.SurveyStatusHandler(database.DB)))
	mux.Handle("/survey/history", withAuth(handlers.SurveyHistoryHandler(database.DB)))

	// Эндпоинт для первичного опроса клиента
	mux.Handle("/client_survey", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			handlers.ClientSurveyHandler(database.DB)(w, r)
		case http.MethodGet:
			handlers.GetClientSurveyHandler(database.DB)(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})))

	// Получаем порт из переменной окружения или используем 8080 по умолчанию
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Оборачиваем весь роутер в CORS + HTTPS redirect middleware.
	// TLS-терминация выполняется на уровне reverse proxy (nginx/traefik).
	handler := middleware.CORSMiddleware(mux) // для корректной работы на удаленном сервере
	//handler := middleware.CORSMiddleware(middleware.HTTPSRedirectMiddleware(mux))

	addr := ":" + port
	log.Printf("Starting HTTP server on %s", addr)
	logEndpoints()
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func logEndpoints() {
	log.Printf("Available endpoints:")
	log.Printf("  PUBLIC (no auth required):")
	log.Printf("    POST   /register            - Register new user")
	log.Printf("    POST   /login               - Login and get session token")
	log.Printf("    POST   /logout              - Logout and invalidate session")
	log.Printf("  PROTECTED (requires Authorization header with session token):")
	log.Printf("    PUT    /profile             - Update user profile")
	log.Printf("    GET    /profile/{id}        - Get own profile by ID")
	log.Printf("    POST   /message             - Send message to chat")
	log.Printf("    GET    /msg_batch/{chatId}  - Get message batch by chat ID (params: limit, offset)")
	log.Printf("    GET    /chats/{userId}      - Get own chats")
	log.Printf("    POST   /chat/{userId}       - Create new chat")
	log.Printf("    PUT    /chat/{chatId}       - Rename own chat")
	log.Printf("    DELETE /chat/{chatId}       - Delete own chat")
	log.Printf("    GET    /link_token          - Generate link token for user linking")
	log.Printf("    POST   /link                - Link users using token")
	log.Printf("    DELETE /link/{userId}       - Delete link between users")
	log.Printf("    POST   /survey              - Submit daily mood survey")
	log.Printf("    GET    /survey/status       - Check if user completed today's survey")
	log.Printf("    GET    /survey/history      - Get last month survey history with LLM summary and recommendations")
	log.Printf("    POST   /client_survey       - Save client onboarding survey")
	log.Printf("    GET    /client_survey       - Get client onboarding survey")
}
