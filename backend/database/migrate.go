package database

import (
	"log"
	"myapp/models"

	"gorm.io/gorm"
)

// Migrate выполняет автоматическую миграцию всех моделей
func Migrate(db *gorm.DB) error {
	log.Println("Starting database migration...")

	// Автомиграция создает таблицы, добавляет недостающие колонки и индексы
	// Но НЕ удаляет неиспользуемые колонки для защиты данных
	err := db.AutoMigrate(
		&models.User{},
		&models.LinkToken{},
		&models.Chat{},
		&models.Message{},
		&models.Relation{},
	)

	if err != nil {
		return err
	}

	// Создание дополнительных индексов и ограничений при необходимости
	if err := createAdditionalConstraints(db); err != nil {
		return err
	}

	log.Println("Database migration completed successfully")
	return nil
}

// createAdditionalConstraints создает дополнительные ограничения и индексы
func createAdditionalConstraints(db *gorm.DB) error {
	// Создаем составной уникальный индекс для таблицы relations
	// чтобы избежать дублирования связей между пользователями
	if err := db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_relation 
		ON relations (
			LEAST(first_user_id, second_user_id), 
			GREATEST(first_user_id, second_user_id)
		)
	`).Error; err != nil {
		log.Printf("Warning: Could not create unique index for relations: %v", err)
		// Не останавливаем миграцию, если этот индекс не создался
	}

	// Создаем индекс для поиска по email
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_user_email 
		ON users (email)
	`).Error; err != nil {
		log.Printf("Warning: Could not create index for email: %v", err)
	}

	// Создаем индекс для поиска активных чатов
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chat_active 
		ON chats (is_active)
	`).Error; err != nil {
		log.Printf("Warning: Could not create index for is_active: %v", err)
	}

	return nil
}

// DropTables удаляет все таблицы (используется для тестирования)
func DropTables(db *gorm.DB) error {
	log.Println("Dropping all tables...")
	
	// Удаляем таблицы в обратном порядке зависимостей
	return db.Migrator().DropTable(
		&models.Message{},
		&models.Relation{},
		&models.Chat{},
		&models.LinkToken{},
		&models.User{},
	)
}
