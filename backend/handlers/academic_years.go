package handlers

import (
	"encoding/json"
	"net/http"
	"receipt-processor/backend/models"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func GetAcademicYears(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var years []models.AcademicYear
		db.Order("year DESC").Find(&years)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(years)
	}
}

func GetActiveAcademicYear(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var year models.AcademicYear
		if err := db.Where("is_active = ?", true).Order("year DESC").First(&year).Error; err != nil {
			http.Error(w, "No active academic year found", http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(year)
	}
}

func CreateAcademicYear(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Year      string `json:"year"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Year == "" {
			http.Error(w, "Year is required", http.StatusBadRequest)
			return
		}

		// Проверка существования
		var existing models.AcademicYear
		if err := db.Where("year = ?", req.Year).First(&existing).Error; err == nil {
			http.Error(w, "Academic year already exists", http.StatusConflict)
			return
		}

		startDate, _ := time.Parse("2006-01-02", req.StartDate)
		endDate, _ := time.Parse("2006-01-02", req.EndDate)

		year := models.AcademicYear{
			Year:      req.Year,
			StartDate: startDate,
			EndDate:   endDate,
			IsActive:  false,
		}
		if err := db.Create(&year).Error; err != nil {
			http.Error(w, "Failed to create academic year", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(year)
	}
}

func UpdateAcademicYear(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var req struct {
			Year      string `json:"year"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
			IsActive  *bool  `json:"is_active"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var year models.AcademicYear
		if err := db.First(&year, id).Error; err != nil {
			http.Error(w, "Academic year not found", http.StatusNotFound)
			return
		}

		if req.Year != "" {
			year.Year = req.Year
		}
		if req.StartDate != "" {
			startDate, _ := time.Parse("2006-01-02", req.StartDate)
			year.StartDate = startDate
		}
		if req.EndDate != "" {
			endDate, _ := time.Parse("2006-01-02", req.EndDate)
			year.EndDate = endDate
		}
		if req.IsActive != nil {
			// Если устанавливаем активным, деактивируем остальные
			if *req.IsActive {
				db.Model(&models.AcademicYear{}).Where("id != ?", id).Update("is_active", false)
			}
			year.IsActive = *req.IsActive
		}

		if err := db.Save(&year).Error; err != nil {
			http.Error(w, "Failed to update academic year", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(year)
	}
}

func DeleteAcademicYear(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var year models.AcademicYear
		if err := db.First(&year, id).Error; err != nil {
			http.Error(w, "Academic year not found", http.StatusNotFound)
			return
		}

		db.Delete(&year)
		w.WriteHeader(http.StatusNoContent)
	}
}

func SetGroupRequiredAmount(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			GroupID        uint    `json:"group_id"`
			AcademicYearID uint    `json:"academic_year_id"`
			RequiredAmount float64 `json:"required_amount"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Находим или создаем запись
		var groupYear models.GroupAcademicYear
		result := db.Where("group_id = ? AND academic_year_id = ?", req.GroupID, req.AcademicYearID).First(&groupYear)
		
		if result.Error == gorm.ErrRecordNotFound {
			groupYear = models.GroupAcademicYear{
				GroupID:        req.GroupID,
				AcademicYearID: req.AcademicYearID,
				RequiredAmount: req.RequiredAmount,
			}
			if err := db.Create(&groupYear).Error; err != nil {
				http.Error(w, "Failed to create group academic year", http.StatusInternalServerError)
				return
			}
		} else {
			groupYear.RequiredAmount = req.RequiredAmount
			if err := db.Save(&groupYear).Error; err != nil {
				http.Error(w, "Failed to update group academic year", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groupYear)
	}
}

// GetAllGroupRequiredAmounts получает все установленные требуемые суммы для групп
func GetAllGroupRequiredAmounts(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groupYears []models.GroupAcademicYear
		db.Preload("Group").Preload("AcademicYear").Order("academic_year_id DESC, group_id ASC").Find(&groupYears)

		// Формируем ответ с удобной структурой
		type GroupAmountResponse struct {
			ID             uint    `json:"id"`
			GroupID        uint    `json:"group_id"`
			GroupName      string  `json:"group_name"`
			AcademicYearID uint    `json:"academic_year_id"`
			AcademicYear   string  `json:"academic_year"`
			RequiredAmount float64 `json:"required_amount"`
			CreatedAt      string  `json:"created_at"`
			UpdatedAt      string  `json:"updated_at"`
		}

		var response []GroupAmountResponse
		for _, gy := range groupYears {
			response = append(response, GroupAmountResponse{
				ID:             gy.ID,
				GroupID:        gy.GroupID,
				GroupName:      gy.Group.Name,
				AcademicYearID: gy.AcademicYearID,
				AcademicYear:   gy.AcademicYear.Year,
				RequiredAmount: gy.RequiredAmount,
				CreatedAt:      gy.CreatedAt.Format("2006-01-02 15:04:05"),
				UpdatedAt:      gy.UpdatedAt.Format("2006-01-02 15:04:05"),
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// DeleteGroupRequiredAmount удаляет установленную требуемую сумму
func DeleteGroupRequiredAmount(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		idStr := vars["id"]
		
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		if err := db.Delete(&models.GroupAcademicYear{}, id).Error; err != nil {
			http.Error(w, "Failed to delete", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
	}
}

func GetGroupStatistics(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Query().Get("group")
		yearID := r.URL.Query().Get("academic_year_id")

		if groupName == "" || yearID == "" {
			http.Error(w, "Group and academic_year_id are required", http.StatusBadRequest)
			return
		}

		yearIDUint, _ := strconv.ParseUint(yearID, 10, 32)

		// Находим группу по имени
		var group models.Group
		if err := db.Where("name = ?", groupName).First(&group).Error; err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		// Получаем требуемую сумму для группы
		var groupYear models.GroupAcademicYear
		var requiredAmount float64 = 0
		if err := db.Where("group_id = ? AND academic_year_id = ?", group.ID, yearIDUint).First(&groupYear).Error; err == nil {
			requiredAmount = groupYear.RequiredAmount
		}

		// Получаем все квитанции для группы в этом учебном году
		var receipts []models.Receipt
		db.Where("group = ? AND academic_year_id = ? AND status = ?", groupName, yearIDUint, "processed").Find(&receipts)

		// Группируем по ФИО
		type StudentStats struct {
			FullName       string           `json:"full_name"`
			TotalPaid      float64          `json:"total_paid"`
			RequiredAmount float64          `json:"required_amount"`
			PaymentPercent float64          `json:"payment_percent"`
			Receipts       []models.Receipt `json:"receipts"`
		}

		studentMap := make(map[string]*StudentStats)
		for _, receipt := range receipts {
			if receipt.Amount == nil {
				continue
			}

			if stats, exists := studentMap[receipt.FullName]; exists {
				stats.TotalPaid += *receipt.Amount
				stats.Receipts = append(stats.Receipts, receipt)
			} else {
				studentMap[receipt.FullName] = &StudentStats{
					FullName:       receipt.FullName,
					TotalPaid:      *receipt.Amount,
					RequiredAmount: requiredAmount,
					Receipts:       []models.Receipt{receipt},
				}
			}
		}

		// Вычисляем проценты
		var students []StudentStats
		for _, stats := range studentMap {
			if stats.RequiredAmount > 0 {
				stats.PaymentPercent = (stats.TotalPaid / stats.RequiredAmount) * 100
				if stats.PaymentPercent > 100 {
					stats.PaymentPercent = 100
				}
			}
			students = append(students, *stats)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(students)
	}
}

