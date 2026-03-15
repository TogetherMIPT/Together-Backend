package models

import (
	"time"
	// "gorm.io/gorm"
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
