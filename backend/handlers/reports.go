package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"receipt-processor/backend/models"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// GetGroupReport - получение отчета по группе
func GetGroupReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Query().Get("group")
		yearIDStr := r.URL.Query().Get("academic_year_id")

		if groupName == "" {
			http.Error(w, "Group is required", http.StatusBadRequest)
			return
		}

		var academicYearID uint
		if yearIDStr != "" {
			id, _ := strconv.ParseUint(yearIDStr, 10, 32)
			academicYearID = uint(id)
		} else {
			// Используем активный учебный год
			var activeYear models.AcademicYear
			if err := db.Where("is_active = ?", true).First(&activeYear).Error; err != nil {
				http.Error(w, "No active academic year found", http.StatusNotFound)
				return
			}
			academicYearID = activeYear.ID
		}

		// Получаем требуемую сумму для группы
		var group models.Group
		if err := db.Where("name = ?", groupName).First(&group).Error; err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		var groupYear models.GroupAcademicYear
		var requiredAmount float64 = 0
		if err := db.Where("group_id = ? AND academic_year_id = ?", group.ID, academicYearID).First(&groupYear).Error; err == nil {
			requiredAmount = groupYear.RequiredAmount
		}

		// Получаем все квитанции для группы в этом учебном году
		var receipts []models.Receipt
		db.Where("group = ? AND academic_year_id = ? AND status = ?", groupName, academicYearID, "processed").Find(&receipts)

		// Получаем примененные оплаты
		var appliedPayments []models.PaymentApplication
		db.Where("group = ? AND academic_year_id = ?", groupName, academicYearID).Find(&appliedPayments)

		// Группируем по ФИО
		type StudentReport struct {
			FullName        string    `json:"full_name"`
			TotalPaid       float64   `json:"total_paid"`       // Сумма из квитанций
			AppliedAmount   float64   `json:"applied_amount"`   // Примененная оплата
			TotalAmount     float64   `json:"total_amount"`     // Итого оплачено (квитанции + примененная)
			RequiredAmount  float64   `json:"required_amount"`
			RemainingDebt   float64   `json:"remaining_debt"`   // Остаток долга
			PaymentPercent  float64   `json:"payment_percent"`
			PaymentStatus   string    `json:"payment_status"`
			ReceiptCount    int       `json:"receipt_count"`
			AppliedDate     *time.Time `json:"applied_date"`
			AppliedBy       string    `json:"applied_by"`
			Notes           string    `json:"notes"`
		}

		studentMap := make(map[string]*StudentReport)
		
		// Обрабатываем квитанции
		for _, receipt := range receipts {
			if receipt.Amount == nil {
				continue
			}

			if stats, exists := studentMap[receipt.FullName]; exists {
				stats.TotalPaid += *receipt.Amount
				stats.ReceiptCount++
			} else {
				studentMap[receipt.FullName] = &StudentReport{
					FullName:       receipt.FullName,
					TotalPaid:      *receipt.Amount,
					RequiredAmount: requiredAmount,
					ReceiptCount:   1,
				}
			}
		}

		// Обрабатываем примененные оплаты
		for _, payment := range appliedPayments {
			if stats, exists := studentMap[payment.FullName]; exists {
				stats.AppliedAmount += payment.AppliedAmount
				if payment.AppliedDate.After(*stats.AppliedDate) || stats.AppliedDate == nil {
					stats.AppliedDate = &payment.AppliedDate
					var user models.User
					if err := db.First(&user, payment.UserID).Error; err == nil {
						stats.AppliedBy = user.Username
					}
					stats.Notes = payment.Notes
				}
			} else {
				studentMap[payment.FullName] = &StudentReport{
					FullName:       payment.FullName,
					AppliedAmount:  payment.AppliedAmount,
					RequiredAmount: requiredAmount,
					AppliedDate:    &payment.AppliedDate,
					ReceiptCount:   0,
				}
				var user models.User
				if err := db.First(&user, payment.UserID).Error; err == nil {
					studentMap[payment.FullName].AppliedBy = user.Username
				}
				studentMap[payment.FullName].Notes = payment.Notes
			}
		}

		// Вычисляем итоговые суммы и проценты
		var students []StudentReport
		for _, stats := range studentMap {
			stats.TotalAmount = stats.TotalPaid + stats.AppliedAmount
			if stats.RequiredAmount > 0 {
				stats.RemainingDebt = stats.RequiredAmount - stats.TotalAmount
				if stats.RemainingDebt < 0 {
					stats.RemainingDebt = 0
				}
				stats.PaymentPercent = (stats.TotalAmount / stats.RequiredAmount) * 100
				
				if stats.PaymentPercent >= 100 {
					if stats.TotalAmount > stats.RequiredAmount {
						stats.PaymentStatus = "overpaid"
					} else {
						stats.PaymentStatus = "paid"
					}
				} else if stats.PaymentPercent > 0 {
					stats.PaymentStatus = "partial"
				} else {
					stats.PaymentStatus = "unpaid"
				}
			} else {
				stats.RemainingDebt = 0
				stats.PaymentPercent = 0
				stats.PaymentStatus = "unpaid"
			}
			students = append(students, *stats)
		}

		response := map[string]interface{}{
			"group":           groupName,
			"academic_year_id": academicYearID,
			"required_amount": requiredAmount,
			"students":        students,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// ApplyPayment - применение оплаты к студенту
func ApplyPayment(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FullName       string  `json:"full_name"`
			Group          string  `json:"group"`
			AcademicYearID uint    `json:"academic_year_id"`
			AppliedAmount  float64 `json:"applied_amount"`
			Notes          string  `json:"notes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.FullName == "" || req.Group == "" || req.AppliedAmount <= 0 {
			http.Error(w, "FullName, Group and AppliedAmount are required", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)

		payment := models.PaymentApplication{
			FullName:       req.FullName,
			Group:          req.Group,
			AcademicYearID: req.AcademicYearID,
			AppliedAmount:  req.AppliedAmount,
			AppliedDate:    time.Now(),
			Notes:          req.Notes,
			UserID:         userID,
		}

		if err := db.Create(&payment).Error; err != nil {
			log.Printf("Error creating payment application: %v", err)
			http.Error(w, "Failed to apply payment", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payment)
	}
}

// DeletePaymentApplication - удаление примененной оплаты
func DeletePaymentApplication(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var payment models.PaymentApplication
		if err := db.First(&payment, id).Error; err != nil {
			http.Error(w, "Payment application not found", http.StatusNotFound)
			return
		}

		db.Delete(&payment)
		w.WriteHeader(http.StatusNoContent)
	}
}

// ExportGroupReport - экспорт отчета по группе в Excel
func ExportGroupReport(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupName := r.URL.Query().Get("group")
		yearIDStr := r.URL.Query().Get("academic_year_id")

		if groupName == "" {
			http.Error(w, "Group is required", http.StatusBadRequest)
			return
		}

		var academicYearID uint
		if yearIDStr != "" {
			id, _ := strconv.ParseUint(yearIDStr, 10, 32)
			academicYearID = uint(id)
		} else {
			var activeYear models.AcademicYear
			if err := db.Where("is_active = ?", true).First(&activeYear).Error; err != nil {
				http.Error(w, "No active academic year found", http.StatusNotFound)
				return
			}
			academicYearID = activeYear.ID
		}

		// Получаем данные отчета (используем ту же логику, что и GetGroupReport)
		var group models.Group
		if err := db.Where("name = ?", groupName).First(&group).Error; err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		var groupYear models.GroupAcademicYear
		var requiredAmount float64 = 0
		if err := db.Where("group_id = ? AND academic_year_id = ?", group.ID, academicYearID).First(&groupYear).Error; err == nil {
			requiredAmount = groupYear.RequiredAmount
		}

		var receipts []models.Receipt
		db.Where("group = ? AND academic_year_id = ? AND status = ?", groupName, academicYearID, "processed").Find(&receipts)

		var appliedPayments []models.PaymentApplication
		db.Where("group = ? AND academic_year_id = ?", groupName, academicYearID).Preload("User").Find(&appliedPayments)

		// Формируем данные для экспорта
		type ExportStudent struct {
			FullName       string    `json:"full_name"`
			TotalPaid      float64   `json:"total_paid"`
			AppliedAmount  float64   `json:"applied_amount"`
			TotalAmount    float64   `json:"total_amount"`
			RequiredAmount float64   `json:"required_amount"`
			RemainingDebt  float64   `json:"remaining_debt"`
			PaymentPercent float64   `json:"payment_percent"`
			PaymentStatus  string    `json:"payment_status"`
			ReceiptCount   int       `json:"receipt_count"`
			AppliedDate    *time.Time `json:"applied_date"`
			AppliedBy      string    `json:"applied_by"`
			Notes          string    `json:"notes"`
		}

		studentMap := make(map[string]*ExportStudent)

		for _, receipt := range receipts {
			if receipt.Amount == nil {
				continue
			}
			if stats, exists := studentMap[receipt.FullName]; exists {
				stats.TotalPaid += *receipt.Amount
				stats.ReceiptCount++
			} else {
				studentMap[receipt.FullName] = &ExportStudent{
					FullName:       receipt.FullName,
					TotalPaid:      *receipt.Amount,
					RequiredAmount: requiredAmount,
					ReceiptCount:   1,
				}
			}
		}

		for _, payment := range appliedPayments {
			if stats, exists := studentMap[payment.FullName]; exists {
				stats.AppliedAmount += payment.AppliedAmount
				if stats.AppliedDate == nil || payment.AppliedDate.After(*stats.AppliedDate) {
					stats.AppliedDate = &payment.AppliedDate
					stats.AppliedBy = payment.User.Username
					stats.Notes = payment.Notes
				}
			} else {
				studentMap[payment.FullName] = &ExportStudent{
					FullName:       payment.FullName,
					AppliedAmount:  payment.AppliedAmount,
					RequiredAmount: requiredAmount,
					AppliedDate:    &payment.AppliedDate,
					ReceiptCount:   0,
					AppliedBy:      payment.User.Username,
					Notes:          payment.Notes,
				}
			}
		}

		var students []ExportStudent
		for _, stats := range studentMap {
			stats.TotalAmount = stats.TotalPaid + stats.AppliedAmount
			if stats.RequiredAmount > 0 {
				stats.RemainingDebt = stats.RequiredAmount - stats.TotalAmount
				if stats.RemainingDebt < 0 {
					stats.RemainingDebt = 0
				}
				stats.PaymentPercent = (stats.TotalAmount / stats.RequiredAmount) * 100
				
				if stats.PaymentPercent >= 100 {
					if stats.TotalAmount > stats.RequiredAmount {
						stats.PaymentStatus = "Переплата"
					} else {
						stats.PaymentStatus = "Оплачено"
					}
				} else if stats.PaymentPercent > 0 {
					stats.PaymentStatus = "Частично"
				} else {
					stats.PaymentStatus = "Не оплачено"
				}
			} else {
				stats.RemainingDebt = 0
				stats.PaymentPercent = 0
				stats.PaymentStatus = "Не оплачено"
			}
			students = append(students, *stats)
		}

		// Вызов Python сервиса для генерации Excel
		pythonURL := os.Getenv("PYTHON_SERVICE_URL")
		if pythonURL == "" {
			pythonURL = "http://localhost:5000"
		}

		exportData := map[string]interface{}{
			"group":           groupName,
			"academic_year_id": academicYearID,
			"required_amount": requiredAmount,
			"students":        students,
		}

		jsonData, _ := json.Marshal(exportData)
		req, _ := http.NewRequest("POST", pythonURL+"/export-report", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to generate Excel", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, "Failed to generate Excel", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		filename := fmt.Sprintf("report_%s_%d.xlsx", strings.ReplaceAll(groupName, " ", "_"), academicYearID)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		io.Copy(w, resp.Body)
	}
}

