package models

import (
	"time"
	"gorm.io/gorm"
)

// OverpaymentTransfer хранит информацию о переносе переплаты с одного учебного года на другой
type OverpaymentTransfer struct {
	gorm.Model
	FullName          string    `gorm:"not null" json:"full_name"`           // ФИО студента
	Group             string    `gorm:"not null" json:"group"`               // Группа студента
	FromAcademicYearID uint     `gorm:"not null" json:"from_academic_year_id"` // Учебный год, откуда переносится переплата
	ToAcademicYearID   uint     `gorm:"not null" json:"to_academic_year_id"`   // Учебный год, куда переносится переплата
	Amount            float64   `gorm:"not null" json:"amount"`               // Сумма переплаты
	TransferDate      time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"transfer_date"` // Дата переноса
	Notes             string    `json:"notes"`                                // Примечания
	UserID            uint      `json:"user_id"`                             // Кто выполнил перенос
	
	FromAcademicYear  AcademicYear `gorm:"foreignKey:FromAcademicYearID" json:"from_academic_year,omitempty"`
	ToAcademicYear    AcademicYear `gorm:"foreignKey:ToAcademicYearID" json:"to_academic_year,omitempty"`
	User              User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

