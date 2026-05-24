package models

import (
	"time"
	"gorm.io/gorm"
)

type AcademicYear struct {
	gorm.Model
	Year          string  `gorm:"not null;unique" json:"Year"` // Например: "2024-2025"
	StartDate     time.Time `gorm:"not null" json:"StartDate"`
	EndDate       time.Time `gorm:"not null" json:"EndDate"`
	IsActive      bool    `gorm:"default:true" json:"IsActive"`
}

type GroupAcademicYear struct {
	gorm.Model
	GroupID       uint    `gorm:"not null" json:"GroupID"`
	AcademicYearID uint   `gorm:"not null" json:"AcademicYearID"`
	RequiredAmount float64 `gorm:"not null" json:"RequiredAmount"` // Требуемая сумма для группы в этом году
	
	Group         Group        `gorm:"foreignKey:GroupID"`
	AcademicYear  AcademicYear `gorm:"foreignKey:AcademicYearID"`
}

