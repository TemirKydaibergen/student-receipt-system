package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"receipt-processor/backend/models"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportStudentRequest struct {
	FullName       string  `json:"fullName"`
	Group          string  `json:"group"`
	Price          float64 `json:"price"`
	AcademicYearID uint    `json:"academic_year_id"`
}

type ImportStudentsRequest struct {
	Students      []ImportStudentRequest `json:"students"`
	AcademicYearID uint                  `json:"academic_year_id"`
}

type ImportResponse struct {
	Imported int `json:"imported"`
	Errors   []string `json:"errors,omitempty"`
}

func ImportStudents(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ImportStudentsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if len(req.Students) == 0 {
			http.Error(w, "No students provided", http.StatusBadRequest)
			return
		}

		userID := getUserID(r)
		imported := 0
		errors := []string{}

		for _, student := range req.Students {
			if student.FullName == "" || student.Group == "" {
				errors = append(errors, "Пропущена запись: отсутствует ФИО или группа")
				continue
			}

			// Создаем квитанцию с нулевой суммой (или указанной ценой)
			amount := student.Price
			if amount == 0 {
				amount = 0.0
			}

			receipt := models.Receipt{
				FullName:       student.FullName,
				Group:          student.Group,
				BankType:       "Импорт", // Помечаем как импортированную
				Amount:         &amount,
				PaymentDate:     nil, // Дата платежа не указана
				UploadDate:     time.Now(),
				FilePath:       "", // Нет файла
				Status:         "imported", // Статус "импортировано"
				UserID:         userID,
				AcademicYearID: &req.AcademicYearID,
			}

			if err := db.Create(&receipt).Error; err != nil {
				log.Printf("Failed to import student %s: %v", student.FullName, err)
				errors = append(errors, "Ошибка импорта: "+student.FullName)
				continue
			}

			imported++
		}

		response := ImportResponse{
			Imported: imported,
			Errors:   errors,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// DownloadImportTemplate создает и возвращает шаблон Excel для импорта студентов
func DownloadImportTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Создаем новый Excel файл
		f := excelize.NewFile()
		defer func() {
			if err := f.Close(); err != nil {
				log.Printf("Error closing Excel file: %v", err)
			}
		}()

		// Устанавливаем название листа
		sheetName := "Студенты"
		f.SetSheetName("Sheet1", sheetName)

		// Заголовки колонок
		headers := []string{"ФИО", "ГРУППА", "ЦЕНА"}
		for i, header := range headers {
			cell := fmt.Sprintf("%c1", 'A'+i)
			f.SetCellValue(sheetName, cell, header)
			// Устанавливаем стиль для заголовков
			style, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
				Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
			})
			f.SetCellStyle(sheetName, cell, cell, style)
		}

		// Добавляем примеры данных
		examples := [][]interface{}{
			{"Иванов Иван Иванович", "ИТ-21-1", 50000},
			{"Петров Петр Петрович", "ИТ-21-2", 50000},
			{"Сидоров Сидор Сидорович", "ЭК-22-1", 45000},
		}
		for rowIdx, row := range examples {
			for colIdx, value := range row {
				cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+2)
				f.SetCellValue(sheetName, cell, value)
			}
		}

		// Устанавливаем ширину колонок
		f.SetColWidth(sheetName, "A", "A", 30)
		f.SetColWidth(sheetName, "B", "B", 15)
		f.SetColWidth(sheetName, "C", "C", 15)

		// Сохраняем файл во временный буфер
		var buf bytes.Buffer
		if err := f.Write(&buf); err != nil {
			http.Error(w, "Failed to generate template: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Устанавливаем заголовки для скачивания
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=шаблон_импорта_студентов.xlsx")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", buf.Len()))

		// Отправляем файл
		if _, err := w.Write(buf.Bytes()); err != nil {
			log.Printf("Error writing template to response: %v", err)
		}
	}
}

