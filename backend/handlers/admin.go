package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"receipt-processor/backend/models"
	"receipt-processor/backend/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func GetUsers(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []models.User
		db.Find(&users)

		// Не возвращаем пароли
		for i := range users {
			users[i].Password = ""
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func CreateUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Role     string `json:"role"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Проверка существования пользователя
		var existingUser models.User
		if err := db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}

		hashedPassword, err := utils.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		role := req.Role
		if role != "admin" && role != "user" {
			role = "user"
		}

		user := models.User{
			Username: req.Username,
			Password: hashedPassword,
			Role:     role,
		}

		if err := db.Create(&user).Error; err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		user.Password = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	}
}

func DeleteUser(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var user models.User
		if err := db.First(&user, id).Error; err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		db.Delete(&user)
		w.WriteHeader(http.StatusNoContent)
	}
}

func ExportExcel(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		role := r.Header.Get("X-User-Role")

		var receipts []models.Receipt
		query := db.Preload("User")

		// Пользователь видит только свои квитанции, админ - все
		if role != "admin" {
			query = query.Where("user_id = ?", userID)
		}

		// Универсальный поиск (по фамилии, группе, ФИО)
		if search := r.URL.Query().Get("search"); search != "" {
			search = strings.TrimSpace(search)
			if search != "" {
				searchPattern := "%" + search + "%"
				query = query.Where("full_name LIKE ? COLLATE NOCASE OR \"group\" LIKE ? COLLATE NOCASE OR bank_type LIKE ? COLLATE NOCASE",
					searchPattern, searchPattern, searchPattern)
			}
		}

		// Детальные фильтры
		if group := r.URL.Query().Get("group"); group != "" {
			query = query.Where("\"group\" = ?", group)
		}
		if bankType := r.URL.Query().Get("bank_type"); bankType != "" {
			query = query.Where("bank_type = ?", bankType)
		}
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
			query = query.Where("payment_date >= ? OR (payment_date IS NULL AND upload_date >= ?)", dateFrom, dateFrom)
		}
		if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
			query = query.Where("payment_date <= ? OR (payment_date IS NULL AND upload_date <= ?)", dateTo, dateTo)
		}

		// Получаем ВСЕ квитанции без пагинации для экспорта
		query.Order("created_at DESC").Find(&receipts)

		// Используем ту же логику группировки, что и в GetReceipts
		type ReceiptGroup struct {
			FullName       string           `json:"full_name"`
			Group          string           `json:"group"`
			TotalPaid      float64          `json:"total_paid"`
			RequiredAmount float64          `json:"required_amount"`
			PaymentPercent float64          `json:"payment_percent"`
			PaymentStatus  string           `json:"payment_status"`
			Receipts       []models.Receipt `json:"receipts"`
		}

		// Получаем все требуемые суммы для всех групп и учебных годов
		groupYearAmounts := make(map[string]float64)
		var allGroupYears []models.GroupAcademicYear
		db.Preload("Group").Preload("AcademicYear").Find(&allGroupYears)
		for _, gy := range allGroupYears {
			var group models.Group
			if err := db.First(&group, gy.GroupID).Error; err == nil {
				key := group.Name + "|" + fmt.Sprintf("%d", gy.AcademicYearID)
				groupYearAmounts[key] = gy.RequiredAmount
			}
		}

		var activeYear models.AcademicYear
		hasActiveYear := db.Where("is_active = ?", true).First(&activeYear).Error == nil

		var unprocessedReceipts []models.Receipt
		receiptGroups := make(map[string]*ReceiptGroup)
		for _, receipt := range receipts {
			if receipt.Status != "processed" || receipt.Amount == nil || *receipt.Amount == 0 {
				unprocessedReceipts = append(unprocessedReceipts, receipt)
				continue
			}

			var academicYearID uint
			if receipt.AcademicYearID != nil {
				academicYearID = *receipt.AcademicYearID
			} else if hasActiveYear {
				academicYearID = activeYear.ID
			}

			key := receipt.Group + "|" + fmt.Sprintf("%d", academicYearID)
			requiredAmount := groupYearAmounts[key]

			if requiredAmount == 0 && hasActiveYear {
				activeKey := receipt.Group + "|" + fmt.Sprintf("%d", activeYear.ID)
				requiredAmount = groupYearAmounts[activeKey]
			}

			groupKey := receipt.FullName + "|" + receipt.Group
			if group, exists := receiptGroups[groupKey]; exists {
				group.TotalPaid += *receipt.Amount
				group.Receipts = append(group.Receipts, receipt)
				if requiredAmount > group.RequiredAmount {
					group.RequiredAmount = requiredAmount
				}
			} else {
				receiptGroups[groupKey] = &ReceiptGroup{
					FullName:       receipt.FullName,
					Group:          receipt.Group,
					TotalPaid:      *receipt.Amount,
					RequiredAmount: requiredAmount,
					Receipts:       []models.Receipt{receipt},
				}
			}
		}

		var groupedReceipts []ReceiptGroup
		for _, group := range receiptGroups {
			if group.RequiredAmount > 0 {
				group.PaymentPercent = (group.TotalPaid / group.RequiredAmount) * 100
			} else {
				group.PaymentPercent = 0
			}

			if group.RequiredAmount > 0 {
				if group.PaymentPercent >= 100 {
					if group.TotalPaid > group.RequiredAmount {
						group.PaymentStatus = "overpaid"
					} else {
						group.PaymentStatus = "paid"
					}
				} else if group.PaymentPercent > 0 {
					group.PaymentStatus = "partial"
				} else {
					group.PaymentStatus = "unpaid"
				}
			} else {
				group.PaymentStatus = "unpaid"
			}

			groupedReceipts = append(groupedReceipts, *group)
		}

		for _, receipt := range unprocessedReceipts {
			groupedReceipts = append(groupedReceipts, ReceiptGroup{
				FullName:       receipt.FullName,
				Group:          receipt.Group,
				TotalPaid:      0,
				RequiredAmount: 0,
				PaymentPercent: 0,
				PaymentStatus:  "pending",
				Receipts:       []models.Receipt{receipt},
			})
		}

		// Вызов Python сервиса для генерации Excel
		pythonURL := os.Getenv("PYTHON_SERVICE_URL")
		if pythonURL == "" {
			pythonURL = "http://localhost:5000"
		}

		// Отправка группированных данных в Python сервис
		jsonData, _ := json.Marshal(groupedReceipts)
		req, _ := http.NewRequest("POST", pythonURL+"/export", bytes.NewBuffer(jsonData))
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

		// Получение файла Excel
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=receipts.xlsx")
		io.Copy(w, resp.Body)
	}
}

