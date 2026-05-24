package models

import (
	"time"
	"gorm.io/gorm"
)

// PaymentApplication - запись о применении оплаты к студенту
type PaymentApplication struct {
	gorm.Model
	FullName        string    `gorm:"not null" json:"full_name"`
	Group           string    `gorm:"not null" json:"group"`
	AcademicYearID  uint      `gorm:"not null" json:"academic_year_id"`
	AppliedAmount   float64   `gorm:"not null" json:"applied_amount"` // Примененная сумма оплаты
	AppliedDate     time.Time `gorm:"not null" json:"applied_date"`   // Дата применения оплаты
	Notes           string    `json:"notes"`                           // Примечания
	UserID          uint      `json:"user_id"`                         // Кто применил оплату
	AcademicYear    AcademicYear `gorm:"foreignKey:AcademicYearID" json:"academic_year,omitempty"`
	User            User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

