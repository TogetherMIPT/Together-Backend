package models

import (
	"log"
	"myapp/crypto"
	"time"

	"gorm.io/gorm"
)

// User представляет пользователя системы
type User struct {
	UserID           uint      `gorm:"primaryKey;column:user_id;autoIncrement"`
	Name             string    `gorm:"column:name;type:varchar(255);not null"`
	Email            string    `gorm:"column:email;type:varchar(255);not null"`
	Login            string    `gorm:"column:login;type:varchar(100);uniqueIndex;not null"`
	Country          string    `gorm:"column:country;type:varchar(100)"`
	City             string    `gorm:"column:city;type:varchar(100)"`
	Birthdate        time.Time `gorm:"column:birthdate;type:date"`
	Gender           string    `gorm:"column:gender;type:varchar(10)"`
	CreationDatetime    time.Time  `gorm:"column:creation_datetime;autoCreateTime"`
	Password            string     `gorm:"column:password;type:varchar(255);not null"`
	LastPaymentDatetime *time.Time `gorm:"column:last_payment_datetime"`

	// Связи
	LinkTokens []LinkToken `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Chats      []Chat      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Sessions   []Session   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	// Для связей в таблице Relation
	FirstUserRelations  []Relation `gorm:"foreignKey:FirstUserID;constraint:OnDelete:CASCADE"`
	SecondUserRelations []Relation `gorm:"foreignKey:SecondUserID;constraint:OnDelete:CASCADE"`
}

// LinkToken представляет токен для ссылок/авторизации
type LinkToken struct {
	Token              string    `gorm:"primaryKey;column:token;type:varchar(255)"`
	UserID             uint      `gorm:"column:user_id;not null;index"`
	CreationDatetime   time.Time `gorm:"column:creation_datetime;autoCreateTime"`
	ExpirationDatetime time.Time `gorm:"column:expiration_datetime;not null"`

	// Связи
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// Chat представляет чат
type Chat struct {
	ChatID           uint      `gorm:"primaryKey;column:chat_id;autoIncrement"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`
	UpdatedDatetime  time.Time `gorm:"column:updated_datetime;autoUpdateTime"`
	UserID           uint      `gorm:"column:user_id;not null;index"`
	IsActive         bool      `gorm:"column:is_active;default:true"`
	ChatName         string    `gorm:"column:chat_name;type:varchar(255)"`

	// Связи
	User     User      `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
	Messages []Message `gorm:"foreignKey:ChatID;constraint:OnDelete:CASCADE"`
}

// Message представляет сообщение в чате
type Message struct {
	MessageID        uint      `gorm:"primaryKey;column:message_id;autoIncrement"`
	ChatID           uint      `gorm:"column:chat_id;not null;index"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`
	MessageText      string    `gorm:"column:message_text;type:text"`
	IsFromUser       bool      `gorm:"column:is_from_user;not null;default:false"`

	// Связи
	Chat Chat `gorm:"foreignKey:ChatID;references:ChatID;constraint:OnDelete:CASCADE"`
}

// Relation представляет связь между двумя пользователями
type Relation struct {
	RelationID       uint      `gorm:"primaryKey;column:relation_id;autoIncrement"`
	FirstUserID      uint      `gorm:"column:first_user_id;not null;index"`
	SecondUserID     uint      `gorm:"column:second_user_id;not null;index"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`

	// Связи
	FirstUser  User `gorm:"foreignKey:FirstUserID;references:UserID;constraint:OnDelete:CASCADE"`
	SecondUser User `gorm:"foreignKey:SecondUserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// Session представляет сессию аутентификации пользователя
type Session struct {
	Token              string    `gorm:"primaryKey;column:token;type:varchar(255)"`
	UserID             uint      `gorm:"column:user_id;not null;index"`
	CreationDatetime   time.Time `gorm:"column:creation_datetime;autoCreateTime"`
	ExpirationDatetime time.Time `gorm:"column:expiration_datetime;not null"`

	// Связи
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// DailySurvey представляет ежедневный опрос состояния клиента
type DailySurvey struct {
	SurveyID         uint      `gorm:"primaryKey;column:survey_id;autoIncrement"`
	UserID           uint      `gorm:"column:user_id;not null;index"`
	MoodAnswer       int       `gorm:"column:mood_answer;not null"`
	AnxietyAnswer    int       `gorm:"column:anxiety_answer;not null"`
	ControlAnswer    int       `gorm:"column:control_answer;not null"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`

	// Связи
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// SurveyRecommendation хранит кэшированные рекомендации LLM для пользователя на конкретную дату
type SurveyRecommendation struct {
	RecommendationID uint      `gorm:"primaryKey;column:recommendation_id;autoIncrement"`
	UserID           uint      `gorm:"column:user_id;not null;index"`
	Date             time.Time `gorm:"column:date;type:date;not null"`
	Summary          string    `gorm:"column:summary;type:text"`
	Recommendations  string    `gorm:"column:recommendations;type:text"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`

	// Связи
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// ClientSurvey хранит результат первичного опроса клиента (один на пользователя)
type ClientSurvey struct {
	ClientSurveyID   uint      `gorm:"primaryKey;column:client_survey_id;autoIncrement"`
	UserID           uint      `gorm:"column:user_id;not null;uniqueIndex"`
	WithPsychologist bool      `gorm:"column:with_psychologist;not null"`
	TherapyRequest   string    `gorm:"column:therapy_request;type:text"`
	TherapyApproach  string    `gorm:"column:therapy_approach;type:varchar(255)"`
	WeeklyMeetings   int       `gorm:"column:weekly_meetings;not null;default:0"`
	CreationDatetime time.Time `gorm:"column:creation_datetime;autoCreateTime"`

	// Связи
	User User `gorm:"foreignKey:UserID;references:UserID;constraint:OnDelete:CASCADE"`
}

// ===================== Хуки шифрования =====================

// encryptField шифрует поле и логирует ошибку, не прерывая операцию.
func encryptField(value string) string {
	if value == "" {
		return value
	}
	enc, err := crypto.Encrypt(value)
	if err != nil {
		log.Printf("encrypt error: %v", err)
		return value
	}
	return enc
}

// decryptField расшифровывает поле и логирует ошибку, возвращая оригинал при неудаче.
func decryptField(value string) string {
	if value == "" {
		return value
	}
	dec, err := crypto.Decrypt(value)
	if err != nil {
		log.Printf("decrypt error: %v", err)
		return value
	}
	return dec
}

// BeforeCreate шифрует чувствительные поля User перед вставкой в БД.
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.Name = encryptField(u.Name)
	u.Email = encryptField(u.Email)
	u.Country = encryptField(u.Country)
	u.City = encryptField(u.City)
	u.Gender = encryptField(u.Gender)
	return nil
}

// AfterFind расшифровывает чувствительные поля User после загрузки из БД.
func (u *User) AfterFind(tx *gorm.DB) error {
	u.Name = decryptField(u.Name)
	u.Email = decryptField(u.Email)
	u.Country = decryptField(u.Country)
	u.City = decryptField(u.City)
	u.Gender = decryptField(u.Gender)
	return nil
}

// BeforeCreate шифрует текст сообщения Message перед вставкой в БД.
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	m.MessageText = encryptField(m.MessageText)
	return nil
}

// AfterFind расшифровывает текст сообщения Message после загрузки из БД.
func (m *Message) AfterFind(tx *gorm.DB) error {
	m.MessageText = decryptField(m.MessageText)
	return nil
}

// ===================== TableName методы для явного указания имен таблиц =====================

// TableName методы для явного указания имен таблиц
func (User) TableName() string {
	return "users"
}

func (LinkToken) TableName() string {
	return "link_tokens"
}

func (Chat) TableName() string {
	return "chats"
}

func (Message) TableName() string {
	return "messages"
}

func (Relation) TableName() string {
	return "relations"
}

func (Session) TableName() string {
	return "sessions"
}

func (DailySurvey) TableName() string {
	return "daily_surveys"
}

func (SurveyRecommendation) TableName() string {
	return "survey_recommendations"
}

func (ClientSurvey) TableName() string {
	return "client_surveys"
}
