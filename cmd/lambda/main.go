package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

type Response struct {
	StatusCode int               `json:"statusCode"`
	Body       string            `json:"body"`
	Headers    map[string]string `json:"headers"`
}

func handler(ctx context.Context) (Response, error) {
	// Get configuration from environment
	apiEndpoint := os.Getenv("API_ENDPOINT")
	apiKey := os.Getenv("ADMIN_API_KEY")

	if apiEndpoint == "" {
		return Response{
			StatusCode: 500,
			Body:       `{"error": "API_ENDPOINT not configured"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	if apiKey == "" {
		return Response{
			StatusCode: 500,
			Body:       `{"error": "ADMIN_API_KEY not configured"}`,
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	// Call admin rotate endpoint
	requestBody := map[string]interface{}{}
	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return Response{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to create request: %v"}`, err),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiEndpoint, bytes.NewBuffer(requestBytes))
	if err != nil {
		return Response{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to create HTTP request: %v"}`, err),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Response{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to call API: %v"}`, err),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{
			StatusCode: 500,
			Body:       fmt.Sprintf(`{"error": "Failed to read response: %v"}`, err),
			Headers:    map[string]string{"Content-Type": "application/json"},
		}, nil
	}

	return Response{
		StatusCode: resp.StatusCode,
		Body:       string(bodyBytes),
		Headers:    map[string]string{"Content-Type": "application/json"},
	}, nil
}

func main() {
	lambda.Start(handler)
}
