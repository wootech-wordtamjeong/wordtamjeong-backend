package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/yourusername/kkomantl/internal/models"
	"github.com/yourusername/kkomantl/internal/services"
)

// Handler manages HTTP requests
type Handler struct {
	quizService    *services.QuizService
	adminAPIKey    string
	allowedOrigins []string
}

// NewHandler creates a new handler
func NewHandler(quizService *services.QuizService) *Handler {
	// Get allowed origins from environment variable
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	var allowedOrigins []string
	if originsEnv != "" {
		allowedOrigins = strings.Split(originsEnv, ",")
		// Trim spaces
		for i, origin := range allowedOrigins {
			allowedOrigins[i] = strings.TrimSpace(origin)
		}
	} else {
		// Default to allow all if not specified
		allowedOrigins = []string{"*"}
	}

	return &Handler{
		quizService:    quizService,
		adminAPIKey:    os.Getenv("ADMIN_API_KEY"),
		allowedOrigins: allowedOrigins,
	}
}

// GuessHandler handles word guess requests
func (h *Handler) GuessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.GuessRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Word == "" {
		h.errorResponse(w, "Word is required", http.StatusBadRequest)
		return
	}

	// Trim and validate
	req.Word = strings.TrimSpace(req.Word)

	// Get guess result
	result, err := h.quizService.Guess(r.Context(), req.Word)
	if err != nil {
		log.Printf("Error processing guess: %v", err)
		h.errorResponse(w, "Failed to process guess", http.StatusInternalServerError)
		return
	}

	h.jsonResponse(w, result, http.StatusOK)
}

// StatusHandler handles status requests
func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	quiz := h.quizService.GetCurrentQuiz()
	if quiz == nil {
		h.errorResponse(w, "No active quiz", http.StatusInternalServerError)
		return
	}

	response := models.StatusResponse{
		QuizID:      quiz.ID,
		Date:        quiz.Date,
		Top100Hint:  h.quizService.GetTop100Hint(),
		TopRankHint: h.quizService.GetTop200Hint(),
	}

	h.jsonResponse(w, response, http.StatusOK)
}

// AdminRotateHandler handles admin quiz rotation requests
func (h *Handler) AdminRotateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check API key
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		apiKey = r.Header.Get("Authorization")
		if strings.HasPrefix(apiKey, "Bearer ") {
			apiKey = strings.TrimPrefix(apiKey, "Bearer ")
		}
	}

	if h.adminAPIKey == "" || apiKey != h.adminAPIKey {
		h.errorResponse(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.AdminRotateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is okay
		req.Answer = ""
	}

	// Rotate quiz
	quiz, err := h.quizService.RotateQuiz(r.Context(), req.Answer)
	if err != nil {
		log.Printf("Error rotating quiz: %v", err)
		h.errorResponse(w, "Failed to rotate quiz", http.StatusInternalServerError)
		return
	}

	response := models.AdminRotateResponse{
		QuizID: quiz.ID,
		Answer: quiz.Answer,
		Date:   quiz.Date,
	}

	h.jsonResponse(w, response, http.StatusOK)
}

// HealthHandler handles health check requests
func (h *Handler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, map[string]string{"status": "ok"}, http.StatusOK)
}

// CORSMiddleware adds CORS headers
func (h *Handler) CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is allowed
		if h.isOriginAllowed(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if len(h.allowedOrigins) == 1 && h.allowedOrigins[0] == "*" {
			// Fallback to wildcard if configured
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func (h *Handler) isOriginAllowed(origin string) bool {
	for _, allowed := range h.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// jsonResponse sends a JSON response
func (h *Handler) jsonResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// errorResponse sends an error response
func (h *Handler) errorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := models.ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	h.jsonResponse(w, response, statusCode)
}
