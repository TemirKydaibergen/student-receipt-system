package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"receipt-processor/backend/models"
	"time"

	"gorm.io/gorm"
)

// TransferOverpayment переносит переплату с одного учебного года на другой
func TransferOverpayment(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := getUserID(r)

		var req struct {
			FullName          string  `json:"full_name"`
			Group             string  `json:"group"`
			FromAcademicYearID uint   `json:"from_academic_year_id"`
			ToAcademicYearID   uint   `json:"to_academic_year_id"`
			Amount            float64 `json:"amount"`
			Notes             string  `json:"notes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Валидация
		if req.FullName == "" || req.Group == "" {
			http.Error(w, "Full name and group are required", http.StatusBadRequest)
			return
		}
		if req.Amount <= 0 {
			http.Error(w, "Amount must be greater than 0", http.StatusBadRequest)
			return
		}
		if req.FromAcademicYearID == 0 || req.ToAcademicYearID == 0 {
			http.Error(w, "Academic year IDs are required", http.StatusBadRequest)
			return
		}
		if req.FromAcademicYearID == req.ToAcademicYearID {
			http.Error(w, "From and to academic years must be different", http.StatusBadRequest)
			return
		}

		// Проверяем, что учебные года существуют
		var fromYear, toYear models.AcademicYear
		if err := db.First(&fromYear, req.FromAcademicYearID).Error; err != nil {
			http.Error(w, "From academic year not found", http.StatusNotFound)
			return
		}
		if err := db.First(&toYear, req.ToAcademicYearID).Error; err != nil {
			http.Error(w, "To academic year not found", http.StatusNotFound)
			return
		}

		// Создаем запись о переносе переплаты
		transfer := models.OverpaymentTransfer{
			FullName:          req.FullName,
			Group:             req.Group,
			FromAcademicYearID: req.FromAcademicYearID,
			ToAcademicYearID:   req.ToAcademicYearID,
			Amount:            req.Amount,
			Notes:             req.Notes,
			TransferDate:      time.Now(),
			UserID:            userID,
		}

		if err := db.Create(&transfer).Error; err != nil {
			log.Printf("Error creating overpayment transfer: %v", err)
			http.Error(w, "Failed to create transfer", http.StatusInternalServerError)
			return
		}

		// Загружаем связанные данные для ответа
		db.Preload("FromAcademicYear").Preload("ToAcademicYear").Preload("User").First(&transfer, transfer.ID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(transfer)
	}
}

// GetOverpaymentTransfers получает все переносы переплаты для студента
func GetOverpaymentTransfers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fullName := r.URL.Query().Get("full_name")
		group := r.URL.Query().Get("group")
		academicYearID := r.URL.Query().Get("academic_year_id")

		query := db.Preload("FromAcademicYear").Preload("ToAcademicYear").Preload("User")

		if fullName != "" {
			query = query.Where("full_name = ?", fullName)
		}
		if group != "" {
			query = query.Where("group = ?", group)
		}
		if academicYearID != "" {
			// Получаем переносы, где целевой учебный год совпадает
			query = query.Where("to_academic_year_id = ?", academicYearID)
		}

		var transfers []models.OverpaymentTransfer
		query.Find(&transfers)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(transfers)
	}
}

// DeleteOverpaymentTransfer удаляет перенос переплаты
func DeleteOverpaymentTransfer(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID uint `json:"id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := db.Delete(&models.OverpaymentTransfer{}, req.ID).Error; err != nil {
			log.Printf("Error deleting overpayment transfer: %v", err)
			http.Error(w, "Failed to delete transfer", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}
}

