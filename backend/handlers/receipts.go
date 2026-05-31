package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"receipt-processor/backend/models"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Драйвер SQLite без CGO
)

func GetReceipts(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := getUserID(r)
		role := r.Header.Get("X-User-Role")

		log.Printf("GetReceipts called - UserID: %d, Role: %s", userID, role)
		log.Printf("Request URL: %s", r.URL.String())
		log.Printf("Query params: %v", r.URL.Query())

		var receipts []models.Receipt
		query := db.Preload("User")

		// Пользователь видит только свои квитанции, админ - все
		if role != "admin" {
			query = query.Where("user_id = ?", userID)
			log.Printf("Filtering by user_id: %d", userID)
		} else {
			log.Printf("Admin access - no user_id filter")
		}

		// Универсальный поиск (по фамилии, группе, ФИО)
		// Используем регистронезависимый поиск
		if search := r.URL.Query().Get("search"); search != "" {
			search = strings.TrimSpace(search)
			if search != "" {
				searchPattern := "%" + search + "%"
				// SQLite LIKE с COLLATE NOCASE для регистронезависимого поиска
				// Ищем по полному ФИО, группе и типу банка
				query = query.Where("full_name LIKE ? COLLATE NOCASE OR \"group\" LIKE ? COLLATE NOCASE OR bank_type LIKE ? COLLATE NOCASE",
					searchPattern, searchPattern, searchPattern)
				log.Printf("Search filter applied: '%s', pattern: '%s'", search, searchPattern)
			}
		}

		// Детальные фильтры
		groupParam := r.URL.Query().Get("group")
		log.Printf("Raw group parameter from URL: '%s'", groupParam)

		if groupParam != "" && groupParam != "all" {
			// Убираем пробелы и нормализуем значение
			group := strings.TrimSpace(groupParam)
			log.Printf("Group filter received: '%s' (length: %d, bytes: %v)", group, len(group), []byte(group))

			// Проверим все уникальные группы в БД для отладки ПЕРЕД применением фильтра
			var allGroups []string
			db.Model(&models.Receipt{}).Distinct("\"group\"").Pluck("\"group\"", &allGroups)
			log.Printf("All groups in DB (before filter): %v", allGroups)

			// Проверяем, есть ли такая группа в БД
			var testCount int64
			db.Model(&models.Receipt{}).Where("\"group\" = ? COLLATE NOCASE", group).Count(&testCount)
			log.Printf("Total receipts in DB with group '%s' (case-insensitive): %d", group, testCount)

			// Также проверим точное совпадение
			var exactCount int64
			db.Model(&models.Receipt{}).Where("\"group\" = ?", group).Count(&exactCount)
			log.Printf("Total receipts in DB with group '%s' (exact match): %d", group, exactCount)

			// Проверяем с учетом user_id фильтра (если не админ)
			if role != "admin" {
				var userFilteredCount int64
				db.Model(&models.Receipt{}).Where("user_id = ? AND \"group\" = ? COLLATE NOCASE", userID, group).Count(&userFilteredCount)
				log.Printf("Total receipts for user %d with group '%s': %d", userID, group, userFilteredCount)
			}

			// Используем точное сравнение (SQLite по умолчанию регистронезависим для ASCII)
			// Но для надежности используем COLLATE NOCASE
			query = query.Where("\"group\" = ? COLLATE NOCASE", group)
			log.Printf("Group filter applied to query: '%s' (trimmed, case-insensitive)", group)
		}
		if bankType := r.URL.Query().Get("bank_type"); bankType != "" {
			query = query.Where("bank_type = ?", bankType)
		}
		if status := r.URL.Query().Get("status"); status != "" {
			query = query.Where("status = ?", status)
		}
		if dateFrom := r.URL.Query().Get("date_from"); dateFrom != "" {
			// Фильтр по дате платежа или дате загрузки (для необработанных)
			query = query.Where("payment_date >= ? OR (payment_date IS NULL AND upload_date >= ?)", dateFrom, dateFrom)
		}
		if dateTo := r.URL.Query().Get("date_to"); dateTo != "" {
			// Фильтр по дате платежа или дате загрузки (для необработанных)
			query = query.Where("payment_date <= ? OR (payment_date IS NULL AND upload_date <= ?)", dateTo, dateTo)
		}

		// Подсчет общего количества записей ДО пагинации
		var total int64
		query.Model(&models.Receipt{}).Count(&total)
		log.Printf("Total receipts count (before pagination): %d", total)

		// Логируем фильтр по группе ДО пагинации
		if groupFilter := r.URL.Query().Get("group"); groupFilter != "" && groupFilter != "all" {
			var countBeforePagination int64
			query.Model(&models.Receipt{}).Count(&countBeforePagination)
			log.Printf("Group filter '%s' found %d receipts before pagination", groupFilter, countBeforePagination)
		}

		// Пагинация
		page := 1
		limit := 20
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
				limit = l
			}
		}

		offset := (page - 1) * limit
		result := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&receipts)
		if result.Error != nil {
			log.Printf("Error querying receipts: %v", result.Error)
			receipts = []models.Receipt{}
		}
		// Убеждаемся, что receipts не nil
		if receipts == nil {
			receipts = []models.Receipt{}
		}

		// Логируем для отладки
		if groupFilter := r.URL.Query().Get("group"); groupFilter != "" && groupFilter != "all" {
			log.Printf("Group filter '%s': found %d receipts after pagination (page %d, limit %d)", groupFilter, len(receipts), page, limit)
			if len(receipts) > 0 {
				log.Printf("First receipt after pagination: FullName='%s', Group='%s'", receipts[0].FullName, receipts[0].Group)
			} else {
				log.Printf("WARNING: No receipts found with group filter '%s' after pagination!", groupFilter)
			}
		}

		// Логируем для отладки поиска
		if search := r.URL.Query().Get("search"); search != "" {
			log.Printf("Search '%s' found %d receipts before grouping", search, len(receipts))
			if len(receipts) > 0 {
				log.Printf("First receipt: FullName='%s', Group='%s'", receipts[0].FullName, receipts[0].Group)
			}
		}

		// Группируем по ФИО и вычисляем статистику
		type ReceiptGroup struct {
			FullName          string           `json:"full_name"`
			Group             string           `json:"group"`
			TotalPaid         float64          `json:"total_paid"`
			RequiredAmount    float64          `json:"required_amount"`
			PaymentPercent    float64          `json:"payment_percent"`
			PaymentStatus     string           `json:"payment_status"` // "paid", "partial", "unpaid", "overpaid"
			Receipts          []models.Receipt `json:"receipts"`
			TransferredAmount float64          `json:"transferred_amount,omitempty"` // Перенесенная переплата
		}

		// Получаем все требуемые суммы для всех групп и учебных годов
		// Ключ: "groupName|academicYearID", значение: требуемая сумма
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

		// Получаем активный учебный год для случаев, когда у квитанции нет AcademicYearID
		var activeYear models.AcademicYear
		hasActiveYear := db.Where("is_active = ?", true).First(&activeYear).Error == nil

		// Получаем все переносы переплаты
		// Ключ: "fullName|group|toAcademicYearID", значение: сумма перенесенной переплаты
		overpaymentTransfers := make(map[string]float64)
		var allTransfers []models.OverpaymentTransfer
		db.Find(&allTransfers)
		for _, transfer := range allTransfers {
			key := transfer.FullName + "|" + transfer.Group + "|" + fmt.Sprintf("%d", transfer.ToAcademicYearID)
			overpaymentTransfers[key] += transfer.Amount
		}

		// Группируем квитанции по ФИО и группе
		// Сначала собираем необработанные квитанции отдельно
		var unprocessedReceipts []models.Receipt
		receiptGroups := make(map[string]*ReceiptGroup)

		// Логируем все квитанции перед группировкой для отладки
		if groupFilter := r.URL.Query().Get("group"); groupFilter != "" && groupFilter != "all" {
			log.Printf("Before grouping: processing %d receipts with group filter '%s'", len(receipts), groupFilter)
			for i, receipt := range receipts {
				log.Printf("Receipt %d: FullName='%s', Group='%s', Status='%s', Amount=%v",
					i+1, receipt.FullName, receipt.Group, receipt.Status, receipt.Amount)
			}
		}

		for _, receipt := range receipts {
			// Если квитанция не обработана или с ошибкой, добавляем в отдельный список
			if receipt.Status != "processed" || receipt.Status == "error" || receipt.Amount == nil || *receipt.Amount == 0 {
				unprocessedReceipts = append(unprocessedReceipts, receipt)
				continue
			}

			// Определяем учебный год для квитанции
			var academicYearID uint
			if receipt.AcademicYearID != nil {
				academicYearID = *receipt.AcademicYearID
			} else if hasActiveYear {
				academicYearID = activeYear.ID
			}

			// Получаем требуемую сумму для этой группы и учебного года
			key := receipt.Group + "|" + fmt.Sprintf("%d", academicYearID)
			requiredAmount := groupYearAmounts[key]

			// Если не найдено, пробуем найти по группе в активном учебном году
			if requiredAmount == 0 && hasActiveYear {
				activeKey := receipt.Group + "|" + fmt.Sprintf("%d", activeYear.ID)
				requiredAmount = groupYearAmounts[activeKey]
			}

			// Ключ для группировки: ФИО + группа (группируем все квитанции одного студента в одной группе)
			// Примечание: если у студента есть квитанции из разных учебных годов, используем максимальную требуемую сумму
			groupKey := receipt.FullName + "|" + receipt.Group
			if group, exists := receiptGroups[groupKey]; exists {
				group.TotalPaid += *receipt.Amount
				group.Receipts = append(group.Receipts, receipt)
				// Обновляем требуемую сумму, если она больше (на случай разных учебных годов с разными суммами)
				// Это гарантирует, что процент оплаты рассчитывается от максимальной требуемой суммы
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

		// Вычисляем проценты и статусы
		var groupedReceipts []ReceiptGroup
		for _, group := range receiptGroups {
			if group.RequiredAmount > 0 {
				group.PaymentPercent = (group.TotalPaid / group.RequiredAmount) * 100
				if group.PaymentPercent > 100 {
					group.PaymentPercent = 100
				}
			} else {
				group.PaymentPercent = 0
			}

			if group.RequiredAmount > 0 {
				if group.PaymentPercent >= 100 {
					group.PaymentStatus = "paid"
				} else if group.PaymentPercent > 0 {
					group.PaymentStatus = "partial"
				} else {
					group.PaymentStatus = "unpaid"
				}
			} else {
				// Если требуемая сумма не установлена, считаем как неоплаченное
				group.PaymentStatus = "unpaid"
			}

			groupedReceipts = append(groupedReceipts, *group)
		}

		// Добавляем необработанные квитанции как отдельные группы
		// Применяем фильтр по группе и к необработанным квитанциям
		groupFilterForUnprocessed := r.URL.Query().Get("group")
		if groupFilterForUnprocessed != "" && groupFilterForUnprocessed != "all" {
			groupFilterForUnprocessed = strings.TrimSpace(groupFilterForUnprocessed)
			filteredUnprocessed := []models.Receipt{}
			for _, receipt := range unprocessedReceipts {
				if strings.EqualFold(receipt.Group, groupFilterForUnprocessed) {
					filteredUnprocessed = append(filteredUnprocessed, receipt)
				}
			}
			unprocessedReceipts = filteredUnprocessed
		}

		for _, receipt := range unprocessedReceipts {
			// Определяем статус: error или pending
			receiptStatus := receipt.Status
			if receiptStatus == "" {
				receiptStatus = "pending"
			}

			groupKey := receipt.FullName + "|" + receipt.Group + "|" + receiptStatus
			receiptGroups[groupKey] = &ReceiptGroup{
				FullName:       receipt.FullName,
				Group:          receipt.Group,
				TotalPaid:      0,
				RequiredAmount: 0,
				PaymentPercent: 0,
				PaymentStatus:  receiptStatus,
				Receipts:       []models.Receipt{receipt},
			}
		}

		// Пересобираем groupedReceipts с учетом необработанных и перенесенной переплаты
		groupedReceipts = []ReceiptGroup{}
		for _, group := range receiptGroups {
			// Определяем учебный год для расчета (используем активный год или первый найденный)
			var targetYearID uint
			if hasActiveYear {
				targetYearID = activeYear.ID
			} else if len(group.Receipts) > 0 && group.Receipts[0].AcademicYearID != nil {
				targetYearID = *group.Receipts[0].AcademicYearID
			}

			// Получаем перенесенную переплату для этого студента и учебного года
			transferKey := group.FullName + "|" + group.Group + "|" + fmt.Sprintf("%d", targetYearID)
			transferredAmount := overpaymentTransfers[transferKey]

			// Сохраняем оригинальную сумму оплаты
			originalPaid := group.TotalPaid

			// Учитываем перенесенную переплату при расчете
			effectivePaid := originalPaid + transferredAmount

			if group.RequiredAmount > 0 {
				group.PaymentPercent = (effectivePaid / group.RequiredAmount) * 100
				// Не ограничиваем 100%, чтобы видеть переплату
			} else {
				group.PaymentPercent = 0
			}

			// Не перезаписываем статус "error" или "pending"
			if group.PaymentStatus != "pending" && group.PaymentStatus != "error" {
				if group.RequiredAmount > 0 {
					if group.PaymentPercent >= 100 {
						// Проверяем переплату
						if effectivePaid > group.RequiredAmount {
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
			}

			// Добавляем информацию о перенесенной переплате в группу
			group.TransferredAmount = transferredAmount
			// Обновляем TotalPaid с учетом переноса для отображения
			group.TotalPaid = effectivePaid

			groupedReceipts = append(groupedReceipts, *group)
		}

		// Применяем фильтр по группе ПОСЛЕ группировки, если он был указан
		groupFilter := r.URL.Query().Get("group")
		if groupFilter != "" && groupFilter != "all" {
			groupFilter = strings.TrimSpace(groupFilter)
			// Фильтруем сгруппированные результаты по группе (регистронезависимо)
			filteredGroupedReceipts := []ReceiptGroup{}
			for _, gr := range groupedReceipts {
				if strings.EqualFold(gr.Group, groupFilter) {
					filteredGroupedReceipts = append(filteredGroupedReceipts, gr)
				}
			}
			groupedReceipts = filteredGroupedReceipts

			log.Printf("After grouping and filtering by group '%s': found %d groups", groupFilter, len(groupedReceipts))
			if len(groupedReceipts) > 0 {
				log.Printf("First group: FullName='%s', Group='%s'", groupedReceipts[0].FullName, groupedReceipts[0].Group)
			} else {
				log.Printf("No groups found with filter '%s' after grouping. Total receipts before grouping: %d", groupFilter, len(receipts))
				if len(receipts) > 0 {
					log.Printf("Sample receipt: FullName='%s', Group='%s', Status='%s'", receipts[0].FullName, receipts[0].Group, receipts[0].Status)
				}
			}
		}

		// Если нет групп вообще, возвращаем обычный список
		if len(groupedReceipts) == 0 {
			log.Printf("No grouped receipts, returning raw receipts list (count: %d, total: %d)", len(receipts), total)
			// Убеждаемся, что receipts не nil
			if receipts == nil {
				receipts = []models.Receipt{}
			}
			// Если total = 0, возвращаем пустой массив групп для единообразия
			if total == 0 {
				log.Printf("Total is 0, returning empty grouped receipts array")
				response := map[string]interface{}{
					"receipts": []ReceiptGroup{},
					"total":    0,
					"page":     page,
					"limit":    limit,
					"pages":    0,
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(response); err != nil {
					log.Printf("Error encoding response: %v", err)
					http.Error(w, "Error encoding response", http.StatusInternalServerError)
				}
				return
			}
			response := map[string]interface{}{
				"receipts": receipts,
				"total":    total,
				"page":     page,
				"limit":    limit,
				"pages":    (int(total) + limit - 1) / limit,
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(response); err != nil {
				log.Printf("Error encoding response: %v", err)
				http.Error(w, "Error encoding response", http.StatusInternalServerError)
			}
			return
		}

		// Логируем для отладки
		if len(groupedReceipts) > 0 && len(groupedReceipts[0].Receipts) > 0 {
			log.Printf("Sample receipt ID: %d, FullName: %s", groupedReceipts[0].Receipts[0].ID, groupedReceipts[0].Receipts[0].FullName)
		}

		// Возвращаем результат с метаданными пагинации
		// Убеждаемся, что groupedReceipts не nil и является массивом
		if groupedReceipts == nil {
			groupedReceipts = []ReceiptGroup{}
		}

		// Если после фильтрации групп нет, но total > 0, это означает, что фильтр не нашел совпадений
		// В этом случае возвращаем пустой массив групп
		if len(groupedReceipts) == 0 && total > 0 {
			log.Printf("No groups after filtering, but total > 0. Returning empty groups array. Total: %d", total)
			groupedReceipts = []ReceiptGroup{}
		}

		log.Printf("Final response: groupedReceipts count=%d, total=%d, page=%d, limit=%d", len(groupedReceipts), total, page, limit)

		// Убеждаемся, что мы всегда возвращаем массив, а не nil
		receiptsValue := interface{}(groupedReceipts)
		if receiptsValue == nil {
			receiptsValue = []ReceiptGroup{}
		}

		response := map[string]interface{}{
			"receipts": receiptsValue,
			"total":    total,
			"page":     page,
			"limit":    limit,
			"pages":    (int(total) + limit - 1) / limit,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
		log.Printf("Response sent successfully")
	}
}

func GetReceipt(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var receipt models.Receipt
		if err := db.Preload("User").First(&receipt, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Receipt not found", http.StatusNotFound)
			} else {
				log.Printf("Error finding receipt: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Проверка прав доступа
		userID := getUserID(r)
		role := r.Header.Get("X-User-Role")
		if role != "admin" && receipt.UserID != userID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipt)
	}
}

func UploadReceipt(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Парсинг multipart form
		err := r.ParseMultipartForm(10 << 20) // 10 MB
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		fullName := r.FormValue("fullname")
		group := r.FormValue("group")
		bankType := r.FormValue("bank_type")
		autoExtractFullName := r.FormValue("auto_extract_fullname") == "true"
		autoExtractBank := r.FormValue("auto_extract_bank") == "true"

		// Если автоматическое извлечение включено, fullname не обязателен
		if !autoExtractFullName && fullName == "" {
			http.Error(w, "FullName is required when auto-extract is disabled", http.StatusBadRequest)
			return
		}

		// Если автоматическое извлечение банка включено, bankType не обязателен
		if !autoExtractBank && bankType == "" {
			http.Error(w, "Bank type is required when auto-extract is disabled", http.StatusBadRequest)
			return
		}

		if group == "" {
			http.Error(w, "Missing required field: group", http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Failed to get file", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Сохранение файла
		// Получаем имя файла без пути и расширение отдельно
		originalName := filepath.Base(handler.Filename)
		ext := filepath.Ext(originalName)
		nameWithoutExt := originalName[:len(originalName)-len(ext)]
		filename := fmt.Sprintf("%d_%s%s", time.Now().Unix(), nameWithoutExt, ext)
		filePath := filepath.Join("uploads", filename)
		// Нормализуем путь для правильного сохранения в БД
		filePath = filepath.Clean(filePath)
		// Используем прямые слеши для хранения в БД (кроссплатформенность)
		filepathForDB := strings.ReplaceAll(filePath, "\\", "/")

		dst, err := os.Create(filePath)
		if err != nil {
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		io.Copy(dst, file)

		// Получаем активный учебный год
		var activeYear models.AcademicYear
		var academicYearID *uint
		if err := db.Where("is_active = ?", true).First(&activeYear).Error; err == nil {
			academicYearID = &activeYear.ID
		}

		// Создание записи в БД
		userID := getUserID(r)
		zeroAmount := 0.0
		receipt := models.Receipt{
			FullName:       fullName, // Если autoExtractFullName=true, будет обновлено после OCR
			Group:          group,
			BankType:       bankType,      // Если autoExtractBank=true, будет обновлено после OCR
			FilePath:       filepathForDB, // Сохраняем нормализованный путь с прямыми слешами
			Status:         "pending",
			UserID:         userID,
			UploadDate:     time.Now(),
			Amount:         &zeroAmount, // временное значение
			AcademicYearID: academicYearID,
		}

		if err := db.Create(&receipt).Error; err != nil {
			http.Error(w, "Failed to create receipt", http.StatusInternalServerError)
			return
		}

		// Отправка файла в Python сервис для обработки
		go processReceiptWithPython(db, receipt.ID, filePath, bankType, autoExtractFullName, autoExtractBank)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipt)
	}
}

func GetReceiptFile(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var receipt models.Receipt
		if err := db.First(&receipt, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Receipt not found", http.StatusNotFound)
			} else {
				log.Printf("Error finding receipt: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Проверка прав доступа
		userID := getUserID(r)
		role := r.Header.Get("X-User-Role")
		if role != "admin" && receipt.UserID != userID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Нормализуем путь к файлу (исправляем возможные проблемы с разделителями)
		filePath := receipt.FilePath

		// Заменяем ~ на правильный разделитель пути
		if strings.Contains(filePath, "~") {
			filePath = strings.ReplaceAll(filePath, "~", string(filepath.Separator))
		}

		// Нормализуем путь
		filePath = filepath.FromSlash(filePath)
		filePath = filepath.Clean(filePath)

		// Если путь не начинается с "uploads", добавляем его
		if !filepath.IsAbs(filePath) && !strings.HasPrefix(filePath, "uploads") {
			filePath = filepath.Join("uploads", filepath.Base(filePath))
		}

		log.Printf("Attempting to serve file: '%s' (original path: '%s')", filePath, receipt.FilePath)

		// Проверяем существование файла
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("File not found at path: '%s', error: %v", filePath, err)

			// Список альтернативных путей для проверки
			alternativePaths := []string{
				filePath, // Текущий путь
				filepath.Join("uploads", filepath.Base(receipt.FilePath)), // Только имя файла в uploads
			}

			// Если в оригинальном пути есть ~, пробуем заменить на разделитель
			originalPath := receipt.FilePath
			if strings.Contains(originalPath, "~") {
				altPath := strings.ReplaceAll(originalPath, "~", string(filepath.Separator))
				altPath = filepath.Clean(altPath)
				alternativePaths = append(alternativePaths, altPath)
			}

			// Пробуем найти файл по части имени (без timestamp)
			baseName := filepath.Base(receipt.FilePath)
			if strings.Contains(baseName, "_") {
				parts := strings.SplitN(baseName, "_", 2)
				if len(parts) == 2 {
					// Ищем файлы, которые заканчиваются на вторую часть имени
					searchPattern := filepath.Join("uploads", "*"+parts[1])
					matches, _ := filepath.Glob(searchPattern)
					if len(matches) > 0 {
						alternativePaths = append(alternativePaths, matches[0])
						log.Printf("Found file by pattern search: '%s'", matches[0])
					}
				}
			}

			// Пробуем все альтернативные пути
			found := false
			for _, altPath := range alternativePaths {
				if altPath == "" {
					continue
				}
				log.Printf("Trying alternative path: '%s'", altPath)
				if _, err := os.Stat(altPath); err == nil {
					filePath = altPath
					found = true
					log.Printf("File found at: '%s'", filePath)
					break
				}
			}

			if !found {
				log.Printf("File not found after trying all alternatives. Original path: '%s'", receipt.FilePath)
				http.Error(w, fmt.Sprintf("File not found. Original path: %s", receipt.FilePath), http.StatusNotFound)
				return
			}
		}

		log.Printf("Serving file from path: '%s'", filePath)
		http.ServeFile(w, r, filePath)
	}
}

func processReceiptWithPython(
    db *gorm.DB,
    receiptID uint,
    filePath,
    bankType string,
    autoExtractFullName,
    autoExtractBank bool,
) {
	// Вызов Python API для обработки
	pythonURL := os.Getenv("PYTHON_SERVICE_URL")
	if pythonURL == "" {
		pythonURL = "http://localhost:5000"
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Failed to open file: %v", err)
		return
	}
	defer file.Close()

	// Отправка файла в Python сервис через multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Используем только имя файла для multipart, не полный путь
	fileNameOnly := filepath.Base(filePath)
	part, err := writer.CreateFormFile("file", fileNameOnly)
	if err != nil {
		log.Printf("Failed to create form file: %v", err)
		return
	}
	io.Copy(part, file)
	writer.WriteField("bank_type", bankType)
	writer.WriteField("receipt_id", fmt.Sprintf("%d", receiptID))
	writer.WriteField("auto_extract_fullname", fmt.Sprintf("%v", autoExtractFullName))
	writer.WriteField("auto_extract_bank", fmt.Sprintf("%v", autoExtractBank))
	writer.Close()

	req, _ := http.NewRequest("POST", pythonURL+"/process", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to call Python service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Читаем тело ответа с ошибкой
		errorBody, _ := io.ReadAll(resp.Body)
		log.Printf("Python service returned error: %d, body: %s", resp.StatusCode, string(errorBody))
		return
	}

	var result struct {
		FullName       string  `json:"full_name"`
		Amount         float64 `json:"amount"`
		PaymentDate    string  `json:"payment_date"`
		UniversityName string  `json:"university_name"`
		BankType       string  `json:"bank_type"`
		DetectedBank   string  `json:"detected_bank"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode response: %v", err)
		return
	}

	
	log.Printf("Received from Python service: amount=%.2f, date=%s, bank_type=%s, full_name=%s",
		result.Amount, result.PaymentDate, result.BankType, result.FullName)

	paymentDate, err := time.Parse("2006-01-02", result.PaymentDate)
	if err != nil {
		log.Printf("Failed to parse payment date '%s': %v, using current date", result.PaymentDate, err)
		paymentDate = time.Now()
	}
	amount := result.Amount

	// Проверяем, распознал ли OCR данные
	// Сумма (amount) - это критически важное поле, без него квитанция не может считаться обработанной
	// Если amount = 0, это всегда ошибка, независимо от других условий
	hasValidAmount := amount > 0

	// Проверяем, распознано ли ФИО (если включено автоматическое извлечение)
	hasValidFullName := true // По умолчанию true, если autoExtractFullName = false (пользователь ввел вручную)
	if autoExtractFullName {
		hasValidFullName = result.FullName != ""
	}

	updates := map[string]interface{}{
		"amount":          amount,
		"payment_date":    paymentDate,
		"university_name": result.UniversityName,
	}

	// Определяем статус: если сумма не распознана (amount = 0), это всегда ошибка
	// Также ошибка, если включено автоматическое извлечение ФИО, но оно не распознано
	if !hasValidAmount {
		updates["status"] = "error"
		log.Printf("OCR failed to extract amount for receipt %d: amount=%.2f, full_name='%s'", receiptID, amount, result.FullName)
	} else if !hasValidFullName {
		updates["status"] = "error"
		log.Printf("OCR failed to extract full_name for receipt %d: amount=%.2f, full_name='%s'", receiptID, amount, result.FullName)
	} else {
		updates["status"] = "processed"
		log.Printf("Receipt %d successfully processed: amount=%.2f, full_name='%s'", receiptID, amount, result.FullName)
	}

	// Обновляем ФИО только если автоматическое извлечение включено
	if autoExtractFullName && result.FullName != "" {
		updates["full_name"] = result.FullName
	}

	// Обновляем тип банка только если автоматическое извлечение включено
	if autoExtractBank && result.BankType != "" {
		updates["bank_type"] = result.BankType
	}

	statusStr := updates["status"].(string)
	log.Printf("Updating receipt %d with data: amount=%.2f, payment_date=%v, status=%s", receiptID, amount, paymentDate, statusStr)
	updateResult := db.Model(&models.Receipt{}).Where("id = ?", receiptID).Updates(updates)
	if updateResult.Error != nil {
		log.Printf("Failed to update receipt %d: %v", receiptID, updateResult.Error)
	} else if updateResult.RowsAffected == 0 {
		log.Printf("Receipt %d not found for update", receiptID)
	} else {
		log.Printf("Successfully updated receipt %d, rows affected: %d, status: %s", receiptID, updateResult.RowsAffected, statusStr)
	}
}

func DeleteReceipt(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var receipt models.Receipt
		if err := db.First(&receipt, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				http.Error(w, "Receipt not found", http.StatusNotFound)
			} else {
				log.Printf("Error finding receipt: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}

		// Проверка прав доступа
		userID := getUserID(r)
		role := r.Header.Get("X-User-Role")
		if role != "admin" && receipt.UserID != userID {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Удаление файла квитанции
		if receipt.FilePath != "" {
			if err := os.Remove(receipt.FilePath); err != nil {
				log.Printf("Failed to delete receipt file: %v", err)
				// Продолжаем удаление записи даже если файл не удален
			}
		}

		// Удаление записи из БД
		if err := db.Delete(&receipt).Error; err != nil {
			http.Error(w, "Failed to delete receipt", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getUserID(r *http.Request) uint {
	userIDStr := r.Header.Get("X-User-ID")
	userID, _ := strconv.ParseUint(userIDStr, 10, 32)
	return uint(userID)
}
