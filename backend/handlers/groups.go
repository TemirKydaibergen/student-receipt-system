package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"receipt-processor/backend/models"
	"strconv"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func GetGroups(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groups []models.Group
		db.Order("name ASC").Find(&groups)
		
		// Если групп нет, создаем дефолтные
		if len(groups) == 0 {
			defaultGroups := []string{
				"ИТ-21-1", "ИТ-21-2", "ИТ-22-1", "ИТ-22-2", "ИТ-23-1", "ИТ-23-2",
				"ЭК-21-1", "ЭК-22-1", "ЭК-23-1",
				"МЕН-21-1", "МЕН-22-1", "МЕН-23-1",
			}
			for _, name := range defaultGroups {
				group := models.Group{Name: name}
				db.Create(&group)
			}
			db.Order("name ASC").Find(&groups)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(groups)
	}
}

func CreateGroup(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Group name is required", http.StatusBadRequest)
			return
		}

		// Проверка существования
		var existing models.Group
		if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
			http.Error(w, "Group already exists", http.StatusConflict)
			return
		}

		group := models.Group{Name: req.Name}
		if err := db.Create(&group).Error; err != nil {
			http.Error(w, "Failed to create group", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(group)
	}
}

func UpdateGroup(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var group models.Group
		if err := db.First(&group, id).Error; err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		group.Name = req.Name
		if err := db.Save(&group).Error; err != nil {
			http.Error(w, "Failed to update group", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(group)
	}
}

func DeleteGroup(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var group models.Group
		if err := db.First(&group, id).Error; err != nil {
			http.Error(w, "Group not found", http.StatusNotFound)
			return
		}

		db.Delete(&group)
		w.WriteHeader(http.StatusNoContent)
	}
}

func GetBanks(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var banks []models.Bank
		db.Order("name ASC").Find(&banks)
		
		// Если банков нет, создаем дефолтные (с проверкой существования)
		if len(banks) == 0 {
			defaultBanks := []string{"Halyk Bank", "Kaspi Bank"}
			for _, name := range defaultBanks {
				// Проверяем, существует ли банк перед созданием
				var existingBank models.Bank
				if err := db.Where("name = ?", name).First(&existingBank).Error; err != nil {
					// Банк не существует, создаем
					bank := models.Bank{Name: name}
					if err := db.Create(&bank).Error; err != nil {
						// Игнорируем ошибки создания (возможно, другой запрос уже создал)
						log.Printf("Failed to create bank %s: %v", name, err)
					}
				}
			}
			// Повторно загружаем список банков
			db.Order("name ASC").Find(&banks)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(banks)
	}
}

func CreateBank(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.Name == "" {
			http.Error(w, "Bank name is required", http.StatusBadRequest)
			return
		}

		// Проверка существования
		var existing models.Bank
		if err := db.Where("name = ?", req.Name).First(&existing).Error; err == nil {
			http.Error(w, "Bank already exists", http.StatusConflict)
			return
		}

		bank := models.Bank{Name: req.Name}
		if err := db.Create(&bank).Error; err != nil {
			http.Error(w, "Failed to create bank", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bank)
	}
}

func UpdateBank(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var req struct {
			Name string `json:"name"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		var bank models.Bank
		if err := db.First(&bank, id).Error; err != nil {
			http.Error(w, "Bank not found", http.StatusNotFound)
			return
		}

		bank.Name = req.Name
		if err := db.Save(&bank).Error; err != nil {
			http.Error(w, "Failed to update bank", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(bank)
	}
}

func DeleteBank(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, _ := strconv.ParseUint(vars["id"], 10, 32)

		var bank models.Bank
		if err := db.First(&bank, id).Error; err != nil {
			http.Error(w, "Bank not found", http.StatusNotFound)
			return
		}

		db.Delete(&bank)
		w.WriteHeader(http.StatusNoContent)
	}
}
