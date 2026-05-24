package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"receipt-processor/backend/models"
	"receipt-processor/backend/utils"

	"gorm.io/gorm"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

func Login(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Printf("Login: Invalid request body: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		log.Printf("Login attempt for username: %s", req.Username)

		var user models.User
		if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
			log.Printf("Login: User not found: %s, error: %v", req.Username, err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			log.Printf("Login: Invalid password for user: %s", req.Username)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		token, err := utils.GenerateToken(user.Username, user.Role, user.ID)
		if err != nil {
			log.Printf("Login: Failed to generate token: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		log.Printf("Login: Success for user: %s, role: %s", req.Username, user.Role)
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			Token:    token,
			Username: user.Username,
			Role:     user.Role,
		})
	}
}

func Register(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterRequest
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

		user := models.User{
			Username: req.Username,
			Password: hashedPassword,
			Role:     "user",
		}

		if err := db.Create(&user).Error; err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		token, err := utils.GenerateToken(user.Username, user.Role, user.ID)
		if err != nil {
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(AuthResponse{
			Token:    token,
			Username: user.Username,
			Role:     user.Role,
		})
	}
}

