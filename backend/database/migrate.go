package database

import (
	"log"

	"gorm.io/gorm"
)

// Migrate выполняет миграцию базы данных в правильном порядке
func Migrate(db *gorm.DB) error {
	log.Println("Starting database migration...")

	// Сначала создаем таблицы без внешних ключей
	if err := createTablesWithoutFKs(db); err != nil {
		return err
	}

	// Затем добавляем внешние ключи
	if err := addForeignKeys(db); err != nil {
		return err
	}

	log.Println("Database migration completed successfully")

	// Применяем миграции для добавления новых колонок в существующие таблицы
	if err := applyColumnMigrations(db); err != nil {
		log.Printf("Warning: Could not apply column migrations: %v", err)
	}

	// Создание дополнительных индексов после успешной миграции
	if err := createAdditionalConstraints(db); err != nil {
		log.Printf("Warning: Could not create additional constraints: %v", err)
	}

	return nil
}

// createTablesWithoutFKs создает таблицы без внешних ключей
func createTablesWithoutFKs(db *gorm.DB) error {
	// Временно отключаем AutoMigrate для создания таблиц без FK
	// Создаем таблицы через Raw SQL в правильном порядке

	tables := []string{
		`CREATE TABLE IF NOT EXISTS users (
			user_id BIGSERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL,
			login VARCHAR(100) UNIQUE NOT NULL,
			country VARCHAR(100),
			city VARCHAR(100),
			birthdate DATE,
			gender VARCHAR(10),
			creation_datetime TIMESTAMPTZ DEFAULT NOW(),
			password VARCHAR(255) NOT NULL,
			last_payment_datetime TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS link_tokens (
			token VARCHAR(255) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			creation_datetime TIMESTAMPTZ DEFAULT NOW(),
			expiration_datetime TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chats (
			chat_id BIGSERIAL PRIMARY KEY,
			creation_datetime TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT NOT NULL,
			is_active BOOLEAN DEFAULT TRUE,
			chat_name VARCHAR(255),
			updated_datetime TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			message_id BIGSERIAL PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			creation_datetime TIMESTAMPTZ DEFAULT NOW(),
			message_text TEXT,
			is_from_user BOOLEAN NOT NULL DEFAULT FALSE
		)`,
		`CREATE TABLE IF NOT EXISTS relations (
			relation_id BIGSERIAL PRIMARY KEY,
			first_user_id BIGINT NOT NULL,
			second_user_id BIGINT NOT NULL,
			creation_datetime TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			creation_datetime TIMESTAMPTZ DEFAULT NOW(),
			expiration_datetime TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS daily_surveys (
			survey_id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			mood_answer SMALLINT NOT NULL,
			anxiety_answer SMALLINT NOT NULL,
			control_answer SMALLINT NOT NULL,
			creation_datetime TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS survey_recommendations (
			recommendation_id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			date DATE NOT NULL,
			summary TEXT,
			recommendations TEXT,
			creation_datetime TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS client_surveys (
			client_survey_id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL UNIQUE,
			with_psychologist BOOLEAN NOT NULL,
			therapy_request TEXT,
			therapy_approach VARCHAR(255),
			weekly_meetings SMALLINT NOT NULL DEFAULT 0,
			creation_datetime TIMESTAMPTZ DEFAULT NOW()
		)`,
	}

	for _, tableSQL := range tables {
		if err := db.Exec(tableSQL).Error; err != nil {
			log.Printf("Error creating table: %v", err)
			return err
		}
	}

	return nil
}

// addForeignKeys добавляет внешние ключи после создания всех таблиц
func addForeignKeys(db *gorm.DB) error {
	foreignKeys := []string{
		`ALTER TABLE link_tokens 
		 ADD CONSTRAINT fk_link_tokens_user 
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE chats 
		 ADD CONSTRAINT fk_chats_user 
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE messages 
		 ADD CONSTRAINT fk_messages_chat 
		 FOREIGN KEY (chat_id) REFERENCES chats(chat_id) ON DELETE CASCADE`,

		`ALTER TABLE relations 
		 ADD CONSTRAINT fk_relations_first_user 
		 FOREIGN KEY (first_user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE relations
		 ADD CONSTRAINT fk_relations_second_user
		 FOREIGN KEY (second_user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE sessions
		 ADD CONSTRAINT fk_sessions_user
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE daily_surveys
		 ADD CONSTRAINT fk_daily_surveys_user
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE survey_recommendations
		 ADD CONSTRAINT fk_survey_recommendations_user
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,

		`ALTER TABLE client_surveys
		 ADD CONSTRAINT fk_client_surveys_user
		 FOREIGN KEY (user_id) REFERENCES users(user_id) ON DELETE CASCADE`,
	}

	for _, fkSQL := range foreignKeys {
		if err := db.Exec(fkSQL).Error; err != nil {
			log.Printf("Warning: Could not add foreign key (might already exist): %v", err)
			// Не прерываем выполнение, если FK уже существует
		}
	}

	return nil
}

// applyColumnMigrations добавляет новые колонки в существующие таблицы
func applyColumnMigrations(db *gorm.DB) error {
	migrations := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_payment_datetime TIMESTAMPTZ`,
	}

	for _, sql := range migrations {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("Warning: Could not apply column migration: %v", err)
		}
	}

	return nil
}

// createAdditionalConstraints создает дополнительные ограничения и индексы
func createAdditionalConstraints(db *gorm.DB) error {
	log.Println("Creating additional constraints and indexes...")

	indexes := []string{
		// Индекс для поиска по email
		`CREATE INDEX IF NOT EXISTS idx_user_email ON users (email)`,

		// Индекс для поиска активных чатов
		`CREATE INDEX IF NOT EXISTS idx_chat_active ON chats (is_active)`,

		// Индекс для link_tokens по user_id
		`CREATE INDEX IF NOT EXISTS idx_link_tokens_user ON link_tokens (user_id)`,

		// Индекс для messages по chat_id
		`CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages (chat_id)`,

		// Индексы для relations
		`CREATE INDEX IF NOT EXISTS idx_relations_first_user ON relations (first_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relations_second_user ON relations (second_user_id)`,

		// Составной уникальный индекс для relations
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_relation
		 ON relations (LEAST(first_user_id, second_user_id), GREATEST(first_user_id, second_user_id))`,

		// Индекс для sessions по user_id
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id)`,

		// Индекс для поиска по expiration_datetime (для очистки просроченных сессий)
		`CREATE INDEX IF NOT EXISTS idx_sessions_expiration ON sessions (expiration_datetime)`,

		// Индекс для daily_surveys по user_id
		`CREATE INDEX IF NOT EXISTS idx_daily_surveys_user ON daily_surveys (user_id)`,

		// Индекс и уникальное ограничение для survey_recommendations
		`CREATE INDEX IF NOT EXISTS idx_survey_recommendations_user ON survey_recommendations (user_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_survey_recommendations_user_date ON survey_recommendations (user_id, date)`,

		// Индекс для client_surveys по user_id
		`CREATE INDEX IF NOT EXISTS idx_client_surveys_user ON client_surveys (user_id)`,
	}

	for _, indexSQL := range indexes {
		if err := db.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Could not create index: %v", err)
		}
	}

	log.Println("Additional constraints created successfully")
	return nil
}

// DropTables удаляет все таблицы (используется для тестирования)
func DropTables(db *gorm.DB) error {
	log.Println("Dropping all tables...")

	// Удаляем таблицы в обратном порядке зависимостей
	tables := []string{
		"client_surveys",
		"survey_recommendations",
		"daily_surveys",
		"messages",
		"relations",
		"sessions",
		"chats",
		"link_tokens",
		"users",
	}

	for _, table := range tables {
		if err := db.Exec("DROP TABLE IF EXISTS " + table + " CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}
