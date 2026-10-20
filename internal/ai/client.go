package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to any OpenAI-compatible local LLM server (Ollama, llama.cpp, vLLM, LM Studio).
type Client struct {
	Endpoint   string
	Model      string
	HTTPClient *http.Client
}

// NewClient initializes a local AI client with sensible defaults.
func NewClient(endpoint, model string) *Client {
	if endpoint == "" {
		endpoint = "http://localhost:11434/v1/chat/completions"
	}
	if model == "" {
		model = "gemma3:4b"
	}
	return &Client{
		Endpoint: endpoint,
		Model:    model,
		HTTPClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// Ask sends a system and user prompt to the local LLM and returns the text response.
func (c *Client) Ask(systemPrompt, userPrompt string) (string, error) {
	reqBody := chatRequest{
		Model: c.Model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.0,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode AI request: %w", err)
	}

	req, err := http.NewRequest("POST", c.Endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach local LLM at %s: %w", c.Endpoint, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM server returned HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var parsed chatResponse
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return "", fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("LLM returned zero choices")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// GenerateSQL prompts the LLM to translate a natural language question + schema into read-only SQL.
func (c *Client) GenerateSQL(schema, question string) (string, error) {
	sysPrompt := `You are a deterministic SQL generator.
Given a database/Parquet schema and a user question, output ONLY a single valid read-only SELECT SQL query.
Do NOT wrap the SQL in markdown backticks.
If the question cannot be answered using the provided columns, output exactly: UNSUPPORTED: <reason>`

	userPrompt := fmt.Sprintf("Schema:\n%s\n\nQuestion: %s", schema, question)

	raw, err := c.Ask(sysPrompt, userPrompt)
	if err != nil {
		return "", err
	}

	return cleanSQL(raw), nil
}

// cleanSQL strips accidental markdown code fences if a small local model includes them.
func cleanSQL(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```sql")
	cleaned = strings.TrimPrefix(cleaned, "```SQL")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	return strings.TrimSpace(cleaned)
}
