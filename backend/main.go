package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	// "path/filepath"
	"receipt-processor/backend/handlers"
	"receipt-processor/backend/middleware"
	"receipt-processor/backend/models"
	"receipt-processor/backend/utils"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite" // Драйвер SQLite без CGO
)

func main() {
	// Инициализация базы данных
	// Используем modernc.org/sqlite напрямую через database/sql (без CGO)
	// Затем оборачиваем в GORM
	sqlDB, err := sql.Open("sqlite", "/tmp/receipts.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	// Проверяем соединение
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Оборачиваем в GORM используя Dialector с существующим соединением
	db, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Автомиграция
	db.AutoMigrate(&models.User{}, &models.Receipt{}, &models.Group{}, &models.Bank{}, &models.AcademicYear{}, &models.GroupAcademicYear{}, &models.PaymentApplication{}, &models.OverpaymentTransfer{})

	// Создание текущего учебного года по умолчанию
	createDefaultAcademicYear(db)

	// Создание администратора по умолчанию
	createDefaultAdmin(db)

	// Создание директорий для файлов
	os.MkdirAll("uploads", os.ModePerm)
	os.MkdirAll("exports", os.ModePerm)

	// Инициализация роутера
	r := mux.NewRouter()

	// Определяем путь к frontend относительно текущей рабочей директории
	// wd, _ := os.Getwd()
	// log.Printf("Current working directory: %s", wd)

	// // Пробуем разные варианты путей
	// var frontendPath string
	// possiblePaths := []string{
	// 	"./frontend",
	// 	"../frontend",
	// 	filepath.Join(wd, "frontend"),
	// 	filepath.Join(wd, "..", "frontend"),
	// }

	// for _, path := range possiblePaths {
	// 	if _, err := os.Stat(path); err == nil {
	// 		absPath, _ := filepath.Abs(path)
	// 		frontendPath = absPath
	// 		log.Printf("Found frontend at: %s", frontendPath)
	// 		break
	// 	}
	// }

	// if frontendPath == "" {
	// 	log.Fatal("Frontend directory not found! Please ensure frontend/ directory exists.")
	// }

	// // Статические файлы (CSS, JS, изображения)
	// r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(frontendPath))))

	// // Главная страница
	// r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	indexPath := filepath.Join(frontendPath, "index.html")
	// 	http.ServeFile(w, r, indexPath)
	// })

	// API маршруты
	api := r.PathPrefix("/api").Subrouter()

	// Публичные маршруты
	// Регистрация отключена - пользователи создаются только администратором
	api.HandleFunc("/login", handlers.Login(db)).Methods("POST")

	// Защищенные маршруты
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware)

	protected.HandleFunc("/receipts", handlers.GetReceipts(db)).Methods("GET")
	protected.HandleFunc("/receipts", handlers.UploadReceipt(db)).Methods("POST")
	protected.HandleFunc("/receipts/{id}", handlers.GetReceipt(db)).Methods("GET")
	protected.HandleFunc("/receipts/{id}", handlers.DeleteReceipt(db)).Methods("DELETE")
	protected.HandleFunc("/receipts/{id}/file", handlers.GetReceiptFile(db)).Methods("GET")

	// Административные маршруты
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.AdminMiddleware)

	admin.HandleFunc("/users", handlers.GetUsers(db)).Methods("GET")
	admin.HandleFunc("/users", handlers.CreateUser(db)).Methods("POST")
	admin.HandleFunc("/users/{id}", handlers.DeleteUser(db)).Methods("DELETE")
	admin.HandleFunc("/export", handlers.ExportExcel(db)).Methods("GET")

	// Группы образовательных программ (публичный доступ для загрузки списка)
	api.HandleFunc("/groups", handlers.GetGroups(db)).Methods("GET")

	// Банки (публичный доступ для загрузки списка)
	api.HandleFunc("/banks", handlers.GetBanks(db)).Methods("GET")

	// Административные маршруты для групп и банков
	admin.HandleFunc("/groups", handlers.CreateGroup(db)).Methods("POST")
	admin.HandleFunc("/groups/{id}", handlers.UpdateGroup(db)).Methods("PUT")
	admin.HandleFunc("/groups/{id}", handlers.DeleteGroup(db)).Methods("DELETE")

	admin.HandleFunc("/banks", handlers.CreateBank(db)).Methods("POST")
	admin.HandleFunc("/banks/{id}", handlers.UpdateBank(db)).Methods("PUT")
	admin.HandleFunc("/banks/{id}", handlers.DeleteBank(db)).Methods("DELETE")

	// Учебные годы
	api.HandleFunc("/academic-years", handlers.GetAcademicYears(db)).Methods("GET")
	api.HandleFunc("/academic-years/active", handlers.GetActiveAcademicYear(db)).Methods("GET")
	admin.HandleFunc("/academic-years", handlers.CreateAcademicYear(db)).Methods("POST")
	admin.HandleFunc("/academic-years/{id}", handlers.UpdateAcademicYear(db)).Methods("PUT")
	admin.HandleFunc("/academic-years/{id}", handlers.DeleteAcademicYear(db)).Methods("DELETE")
	admin.HandleFunc("/academic-years/group-amount", handlers.SetGroupRequiredAmount(db)).Methods("POST")
	admin.HandleFunc("/academic-years/group-amounts", handlers.GetAllGroupRequiredAmounts(db)).Methods("GET")
	admin.HandleFunc("/academic-years/group-amount/{id}", handlers.DeleteGroupRequiredAmount(db)).Methods("DELETE")
	api.HandleFunc("/statistics/group", handlers.GetGroupStatistics(db)).Methods("GET")

	// Отчеты
	admin.HandleFunc("/reports/group", handlers.GetGroupReport(db)).Methods("GET")
	admin.HandleFunc("/reports/group/export", handlers.ExportGroupReport(db)).Methods("GET")
	admin.HandleFunc("/reports/payment", handlers.ApplyPayment(db)).Methods("POST")
	admin.HandleFunc("/reports/payment/{id}", handlers.DeletePaymentApplication(db)).Methods("DELETE")

	// Перенос переплаты
	admin.HandleFunc("/overpayment/transfer", handlers.TransferOverpayment(db)).Methods("POST")
	api.HandleFunc("/overpayment/transfers", handlers.GetOverpaymentTransfers(db)).Methods("GET")
	admin.HandleFunc("/overpayment/transfer", handlers.DeleteOverpaymentTransfer(db)).Methods("DELETE")

	// Импорт студентов
	admin.HandleFunc("/import/students", handlers.ImportStudents(db)).Methods("POST")
	admin.HandleFunc("/import/template", handlers.DownloadImportTemplate()).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Добавляем CORS для всех запросов
	corsHandler := gorillaHandlers.CORS(
		gorillaHandlers.AllowedOrigins([]string{"*"}),
		gorillaHandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		gorillaHandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)(r)

	log.Printf("Server starting on port %s", port)
	log.Printf("Frontend available at: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsHandler))
}

func createDefaultAdmin(db *gorm.DB) {
	var admin models.User
	result := db.Where("username = ?", "admin").First(&admin)
	if result.Error == gorm.ErrRecordNotFound {
		hashedPassword, _ := utils.HashPassword("admin123")
		admin = models.User{
			Username: "admin",
			Password: hashedPassword,
			Role:     "admin",
		}
		db.Create(&admin)
		log.Println("Default admin created: admin/admin123")
	}
}

func createDefaultAcademicYear(db *gorm.DB) {
	// Создаем текущий учебный год, если его нет
	var currentYear models.AcademicYear
	now := time.Now()
	yearStr := fmt.Sprintf("%d-%d", now.Year(), now.Year()+1)

	result := db.Where("year = ?", yearStr).First(&currentYear)
	if result.Error == gorm.ErrRecordNotFound {
		startDate := time.Date(now.Year(), 9, 1, 0, 0, 0, 0, time.Local)     // 1 сентября
		endDate := time.Date(now.Year()+1, 6, 30, 23, 59, 59, 0, time.Local) // 30 июня следующего года

		// Если сейчас до сентября, берем предыдущий учебный год
		if now.Month() < 9 {
			startDate = time.Date(now.Year()-1, 9, 1, 0, 0, 0, 0, time.Local)
			endDate = time.Date(now.Year(), 6, 30, 23, 59, 59, 0, time.Local)
			yearStr = fmt.Sprintf("%d-%d", now.Year()-1, now.Year())
		}

		currentYear = models.AcademicYear{
			Year:      yearStr,
			StartDate: startDate,
			EndDate:   endDate,
			IsActive:  true,
		}
		db.Create(&currentYear)
		log.Printf("Default academic year created: %s", yearStr)
	}
}
