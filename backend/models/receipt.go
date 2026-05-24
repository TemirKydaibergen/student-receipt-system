package models

import (
	"time"
	"gorm.io/gorm"
)

type Receipt struct {
	gorm.Model
	FullName        string     `gorm:"not null" json:"FullName"`
	Group           string     `gorm:"not null" json:"Group"`
	BankType        string     `gorm:"not null" json:"BankType"` // "Halyk" or "Kaspi"
	Amount          *float64   `json:"Amount"`                    // nullable до обработки
	PaymentDate     *time.Time `json:"PaymentDate"`                // nullable до обработки
	UploadDate      time.Time  `gorm:"default:CURRENT_TIMESTAMP" json:"UploadDate"`
	FilePath        string     `gorm:"not null" json:"FilePath"`
	Status          string     `gorm:"default:'pending'" json:"Status"` // "pending", "processed", "error"
	UniversityName  string     `json:"UniversityName"`
	UserID          uint        `json:"UserID"`
	AcademicYearID  *uint       `json:"AcademicYearID"` // nullable
	User            User        `gorm:"foreignKey:UserID" json:"User,omitempty"`
	AcademicYear    *AcademicYear `gorm:"foreignKey:AcademicYearID" json:"AcademicYear,omitempty"`
}

