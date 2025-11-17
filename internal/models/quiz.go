package models

import "time"

// Quiz represents a daily quiz
type Quiz struct {
	ID              int       `json:"quizId"`
	Date            string    `json:"date"`
	Answer          string    `json:"answer"`
	AnswerEmbedding []float64 `json:"answerEmbedding"`
	CreatedAt       time.Time `json:"createdAt"`
}

// GuessRequest represents a word guess from the user
type GuessRequest struct {
	Word string `json:"word"`
}

// GuessResponse represents the response for a guess
type GuessResponse struct {
	Word       string  `json:"word"`
	Similarity float64 `json:"similarity"`
	Rank       *int    `json:"rank"`
	IsCorrect  bool    `json:"isCorrect"`
}

// StatusResponse represents the current quiz status
type StatusResponse struct {
	QuizID         int     `json:"quizId"`
	Date           string  `json:"date"`
	Top100Hint     float64 `json:"top100Hint"`     // Similarity of 100th ranked word (reference point)
	TopRankHint    float64 `json:"topRankHint"`    // Similarity of 200th ranked word
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// AdminRotateRequest represents a request to create a new quiz
type AdminRotateRequest struct {
	Answer string `json:"answer,omitempty"`
}

// AdminRotateResponse represents the response after creating a new quiz
type AdminRotateResponse struct {
	QuizID int    `json:"quizId"`
	Answer string `json:"answer"`
	Date   string `json:"date"`
}
