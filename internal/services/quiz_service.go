package services

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/kkomantl/internal/models"
	"github.com/yourusername/kkomantl/pkg/bedrock"
	"github.com/yourusername/kkomantl/pkg/utils"
)

// QuizService manages daily quizzes and word similarity calculations
type QuizService struct {
	embeddingService *bedrock.EmbeddingService
	currentQuiz      *models.Quiz
	topWords         []WordSimilarity
	mu               sync.RWMutex
	wordList         []string
}

// WordSimilarity represents a word and its similarity to the answer
type WordSimilarity struct {
	Word       string
	Similarity float64
	Rank       int
}

// NewQuizService creates a new quiz service
func NewQuizService(embeddingService *bedrock.EmbeddingService) *QuizService {
	qs := &QuizService{
		embeddingService: embeddingService,
		wordList:         loadWordList(),
	}

	fmt.Printf("Loaded %d words from word list\n", len(qs.wordList))

	// Initialize with today's quiz
	fmt.Println("Initializing today's quiz...")
	if err := qs.initializeTodayQuiz(); err != nil {
		// If initialization fails, create a default quiz
		fmt.Printf("Failed to initialize today's quiz: %v\n", err)
	} else {
		fmt.Printf("Quiz initialized successfully! Answer: %s\n", qs.currentQuiz.Answer)
	}

	return qs
}

// GetCurrentQuiz returns the current quiz
func (qs *QuizService) GetCurrentQuiz() *models.Quiz {
	qs.mu.RLock()
	defer qs.mu.RUnlock()
	return qs.currentQuiz
}

// GetTop100Hint returns the similarity of the 100th ranked word
func (qs *QuizService) GetTop100Hint() float64 {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if len(qs.topWords) >= 100 {
		return qs.topWords[99].Similarity  // 100th word (0-indexed)
	}
	return 0.0
}

// GetTop200Hint returns the similarity of the 200th ranked word
func (qs *QuizService) GetTop200Hint() float64 {
	qs.mu.RLock()
	defer qs.mu.RUnlock()

	if len(qs.topWords) >= 200 {
		return qs.topWords[199].Similarity  // 200th word (0-indexed)
	}
	return 0.0
}

// Guess processes a word guess and returns similarity and rank
func (qs *QuizService) Guess(ctx context.Context, word string) (*models.GuessResponse, error) {
	qs.mu.RLock()
	quiz := qs.currentQuiz
	topWords := qs.topWords
	qs.mu.RUnlock()

	if quiz == nil {
		return nil, fmt.Errorf("no active quiz")
	}

	// Check if correct answer
	isCorrect := word == quiz.Answer

	// Get embedding for the guessed word
	wordEmbedding, err := qs.embeddingService.GetEmbedding(ctx, word)
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding: %w", err)
	}

	// Calculate similarity
	similarity := utils.CosineSimilarity(wordEmbedding, quiz.AnswerEmbedding)

	// Determine rank and insertion position
	var rank *int
	insertIndex := -1

	for i, ws := range topWords {
		if similarity >= ws.Similarity {
			r := i + 1
			rank = &r
			insertIndex = i
			break
		}
	}

	// Update topWords if this word qualifies for top 200
	if insertIndex >= 0 {
		// Check if word already exists in topWords
		wordExists := false
		for _, ws := range topWords {
			if ws.Word == word {
				wordExists = true
				break
			}
		}

		// Only insert if word doesn't already exist
		if !wordExists {
			newWord := WordSimilarity{
				Word:       word,
				Similarity: similarity,
				Rank:       insertIndex + 1,
			}

			// Create new slice with inserted word
			newTopWords := make([]WordSimilarity, 0, 200)

			// Insert at correct position
			newTopWords = append(newTopWords, topWords[:insertIndex]...)
			newTopWords = append(newTopWords, newWord)
			newTopWords = append(newTopWords, topWords[insertIndex:]...)

			// Keep only top 200
			if len(newTopWords) > 200 {
				newTopWords = newTopWords[:200]
			}

			// Reassign ranks
			for i := range newTopWords {
				newTopWords[i].Rank = i + 1
			}

			// Update in-memory topWords with write lock
			qs.mu.Lock()
			qs.topWords = newTopWords
			qs.mu.Unlock()

			// Save to JSON file
			if err := qs.saveTopWords(quiz.Date, newTopWords); err != nil {
				fmt.Printf("Failed to save updated top words: %v\n", err)
			} else {
				fmt.Printf("✓ Updated topWords: '%s' inserted at rank %d\n", word, insertIndex+1)
			}
		}
	}

	return &models.GuessResponse{
		Word:       word,
		Similarity: similarity,
		Rank:       rank,
		IsCorrect:  isCorrect,
	}, nil
}

// RotateQuiz creates a new quiz with a given or random answer
func (qs *QuizService) RotateQuiz(ctx context.Context, answer string) (*models.Quiz, error) {
	// If no answer provided, select a random one
	if answer == "" {
		answer = qs.selectRandomWord()
	}

	// Get embedding for the answer
	answerEmbedding, err := qs.embeddingService.GetEmbedding(ctx, answer)
	if err != nil {
		return nil, fmt.Errorf("failed to get answer embedding: %w", err)
	}

	// Create new quiz
	quiz := &models.Quiz{
		ID:              int(time.Now().Unix()),
		Date:            time.Now().Format("2006-01-02"),
		Answer:          answer,
		AnswerEmbedding: answerEmbedding,
		CreatedAt:       time.Now(),
	}

	// Calculate top similar words (prototype: store top 200)
	topWords := qs.calculateTopWords(ctx, quiz, 200)

	// Update current quiz
	qs.mu.Lock()
	qs.currentQuiz = quiz
	qs.topWords = topWords
	qs.mu.Unlock()

	// Save quiz to file
	if err := qs.saveQuiz(quiz); err != nil {
		fmt.Printf("Failed to save quiz: %v\n", err)
	}

	// Save top words to file
	if err := qs.saveTopWords(quiz.Date, topWords); err != nil {
		fmt.Printf("Failed to save top words: %v\n", err)
	}

	return quiz, nil
}

// initializeTodayQuiz initializes the quiz for today
func (qs *QuizService) initializeTodayQuiz() error {
	today := time.Now().Format("2006-01-02")

	// Try to load existing quiz for today
	quiz, err := qs.loadQuiz(today)
	if err == nil && quiz != nil {
		qs.mu.Lock()
		qs.currentQuiz = quiz
		qs.mu.Unlock()

		// Load or calculate top words (prototype: store top 200)
		topWords, err := qs.loadTopWords(today)
		if err != nil {
			// Calculate if not found
			topWords = qs.calculateTopWords(context.Background(), quiz, 200)
			qs.saveTopWords(today, topWords)
		}

		qs.mu.Lock()
		qs.topWords = topWords
		qs.mu.Unlock()

		return nil
	}

	// Create new quiz if not found
	_, err = qs.RotateQuiz(context.Background(), "")
	return err
}

// calculateTopWords calculates top N most similar words to the answer
func (qs *QuizService) calculateTopWords(ctx context.Context, quiz *models.Quiz, topN int) []WordSimilarity {
	// Use all words from words.txt for accurate ranking
	sampleSize := len(qs.wordList)

	similarities := make([]WordSimilarity, 0, sampleSize)

	// Calculate similarities for sample words
	fmt.Printf("Calculating top %d words from %d samples...\n", topN, sampleSize)
	fmt.Println("Note: Processing with rate limiting (50 requests/batch with 15s pause)")

	for i, word := range qs.wordList[:sampleSize] {
		if word == quiz.Answer {
			continue
		}

		// Aggressive rate limiting to avoid AWS Bedrock throttling
		// Cohere v4 has ~200-400 requests/minute limit
		// Strategy: 50 requests per batch, then 15 second pause
		if i > 0 && i%50 == 0 {
			fmt.Printf("⏸️  Rate limit pause... (%d/%d processed)\n", i, sampleSize)
			time.Sleep(15 * time.Second) // Pause every 50 requests
		}

		// Small delay between each request to spread load
		if i > 0 {
			time.Sleep(200 * time.Millisecond)
		}

		embedding, err := qs.embeddingService.GetEmbedding(ctx, word)
		if err != nil {
			fmt.Printf("❌ Error getting embedding for '%s': %v\n", word, err)
			// Longer delay before continuing after error
			time.Sleep(3 * time.Second)
			continue
		}

		similarity := utils.CosineSimilarity(embedding, quiz.AnswerEmbedding)
		similarities = append(similarities, WordSimilarity{
			Word:       word,
			Similarity: similarity,
		})

		if (i+1)%10 == 0 {
			fmt.Printf("✓ Processed %d/%d words...\n", i+1, sampleSize)
		}
	}
	fmt.Printf("Calculated %d similarities\n", len(similarities))

	// Sort by similarity (descending)
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Similarity > similarities[j].Similarity
	})

	// Assign ranks and limit to topN
	result := make([]WordSimilarity, 0, topN)
	for i := 0; i < len(similarities) && i < topN; i++ {
		similarities[i].Rank = i + 1
		result = append(result, similarities[i])
	}

	return result
}

// selectRandomWord selects a random word from the word list
func (qs *QuizService) selectRandomWord() string {
	if len(qs.wordList) == 0 {
		return "사랑"
	}
	return qs.wordList[rand.Intn(len(qs.wordList))]
}

// saveQuiz saves a quiz to file with proper UTF-8 encoding
func (qs *QuizService) saveQuiz(quiz *models.Quiz) error {
	os.MkdirAll("data", 0755)

	// Use encoder to preserve UTF-8 characters (not escaped)
	file, err := os.Create(fmt.Sprintf("data/quiz_%s.json", quiz.Date))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(quiz)
}

// loadQuiz loads a quiz from file
func (qs *QuizService) loadQuiz(date string) (*models.Quiz, error) {
	filename := fmt.Sprintf("data/quiz_%s.json", date)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var quiz models.Quiz
	if err := json.Unmarshal(data, &quiz); err != nil {
		return nil, err
	}

	return &quiz, nil
}

// saveTopWords saves top words to file with proper UTF-8 encoding
func (qs *QuizService) saveTopWords(date string, topWords []WordSimilarity) error {
	os.MkdirAll("data", 0755)

	// Use encoder to preserve UTF-8 characters (not escaped)
	file, err := os.Create(fmt.Sprintf("data/topwords_%s.json", date))
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(topWords)
}

// loadTopWords loads top words from file
func (qs *QuizService) loadTopWords(date string) ([]WordSimilarity, error) {
	filename := fmt.Sprintf("data/topwords_%s.json", date)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var topWords []WordSimilarity
	if err := json.Unmarshal(data, &topWords); err != nil {
		return nil, err
	}

	return topWords, nil
}

// loadWordList loads a list of Korean words with proper UTF-8 encoding support
func loadWordList() []string {
	// Try to load from file
	file, err := os.Open("data/words.txt")
	if err == nil {
		defer file.Close()

		words := []string{}
		scanner := bufio.NewScanner(file)

		// Read line by line with UTF-8 support
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			// Skip empty lines and comment lines
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			words = append(words, line)
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading words.txt: %v\n", err)
		} else if len(words) > 0 {
			fmt.Printf("Successfully loaded %d words from words.txt (UTF-8)\n", len(words))
			return words
		}
	} else {
		fmt.Printf("Could not open words.txt: %v\n", err)
	}

	// Default word list for demo
	fmt.Println("Using default word list")
	return []string{
		"사랑", "행복", "희망", "꿈", "열정",
		"도전", "성공", "실패", "노력", "인내",
		"친구", "가족", "부모", "자식", "형제",
		"학교", "회사", "집", "도시", "나라",
		"봄", "여름", "가을", "겨울", "날씨",
		"음악", "영화", "책", "그림", "사진",
		"컴퓨터", "휴대폰", "인터넷", "소프트웨어", "프로그래밍",
		"음식", "밥", "빵", "과일", "채소",
		"동물", "고양이", "강아지", "새", "물고기",
		"자동차", "버스", "기차", "비행기", "배",
	}
}
