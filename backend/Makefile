.PHONY: help run build test clean deps migrate docker-up docker-down

# Переменные
APP_NAME=myapp
MAIN_PATH=main.go
BUILD_DIR=./bin

# Помощь по командам
help:
	@echo "Available commands:"
	@echo "  make run         - Run the application"
	@echo "  make build       - Build the application"
	@echo "  make test        - Run tests"
	@echo "  make deps        - Download dependencies"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make docker-up   - Start PostgreSQL in Docker"
	@echo "  make docker-down - Stop PostgreSQL in Docker"
	@echo "  make migrate     - Run database migrations"

# Запуск приложения
run:
	go run $(MAIN_PATH)

# Сборка приложения
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Тестирование
test:
	go test -v ./...

# Тестирование с покрытием
test-cover:
	go test -v -cover ./...

# Загрузка зависимостей
deps:
	go mod download
	go mod tidy

# Очистка артефактов сборки
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Запуск PostgreSQL в Docker
docker-up:
	docker run --name postgres-dev \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_DB=myapp_db \
		-p 5432:5432 \
		-d postgres:15-alpine

# Остановка PostgreSQL в Docker
docker-down:
	docker stop postgres-dev
	docker rm postgres-dev

# Проверка форматирования кода
fmt:
	go fmt ./...

# Линтинг кода
lint:
	golangci-lint run ./...

# Установка golangci-lint
install-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Запуск приложения с hot-reload (требует установки air)
dev:
	air

# Установка air для hot-reload
install-air:
	go install github.com/cosmtrek/air@latest

# Создание миграции (простой вариант - просто запускает приложение)
migrate:
	@echo "Running migrations..."
	go run $(MAIN_PATH)
