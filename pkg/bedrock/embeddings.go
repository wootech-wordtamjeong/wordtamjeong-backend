package bedrock

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// EmbeddingService handles text embedding operations using AWS Bedrock
type EmbeddingService struct {
	client    *bedrockruntime.Client
	modelID   string
	modelType ModelType
}

// ModelType represents the type of embedding model
type ModelType string

const (
	ModelTypeTitan  ModelType = "titan"
	ModelTypeCohere ModelType = "cohere"
)

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(cfg aws.Config, modelID string) *EmbeddingService {
	client := bedrockruntime.NewFromConfig(cfg)

	// Determine model type from model ID
	var modelType ModelType
	if contains(modelID, "titan") {
		modelType = ModelTypeTitan
	} else if contains(modelID, "cohere") {
		modelType = ModelTypeCohere
	} else {
		modelType = ModelTypeTitan // default
	}

	return &EmbeddingService{
		client:    client,
		modelID:   modelID,
		modelType: modelType,
	}
}

// GetEmbedding returns the embedding vector for a given text
func (s *EmbeddingService) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	switch s.modelType {
	case ModelTypeTitan:
		return s.getTitanEmbedding(ctx, text)
	case ModelTypeCohere:
		return s.getCohereEmbedding(ctx, text)
	default:
		return nil, fmt.Errorf("unsupported model type: %s", s.modelType)
	}
}

// getTitanEmbedding gets embedding from Titan model
func (s *EmbeddingService) getTitanEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Titan Embedding request format
	requestBody := map[string]interface{}{
		"inputText": text,
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        requestBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke model: %w", err)
	}

	// Parse response
	var response struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(output.Body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Embedding, nil
}

// getCohereEmbedding gets embedding from Cohere model (supports v3 and v4)
func (s *EmbeddingService) getCohereEmbedding(ctx context.Context, text string) ([]float64, error) {
	// Cohere Embedding request format (works for both v3 and v4)
	requestBody := map[string]interface{}{
		"texts":      []string{text},
		"input_type": "search_document",
		"truncate":   "END",
	}

	// For v4, specify embedding types and output dimension
	if contains(s.modelID, "v4") {
		requestBody["embedding_types"] = []string{"float"}
	}

	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	output, err := s.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        requestBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to invoke model: %w", err)
	}

	// Try parsing as v4 response format first
	if contains(s.modelID, "v4") {
		var v4Response struct {
			ID           string `json:"id"`
			ResponseType string `json:"response_type"`
			Embeddings   struct {
				Float [][]float64 `json:"float"`
			} `json:"embeddings"`
		}
		if err := json.Unmarshal(output.Body, &v4Response); err == nil {
			if len(v4Response.Embeddings.Float) > 0 {
				return v4Response.Embeddings.Float[0], nil
			}
		}
	}

	// Fallback to v3 format or simple array format
	var v3Response struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.Unmarshal(output.Body, &v3Response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(v3Response.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return v3Response.Embeddings[0], nil
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
