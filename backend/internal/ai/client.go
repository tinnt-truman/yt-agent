// Package ai turns a deterministic channel analysis into an AI-generated
// content strategy for a new, "biến tấu" (variation) channel, using the
// DeepSeek chat completions API (OpenAI-compatible) with JSON mode.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ytagent/backend/internal/models"
)

const apiURL = "https://api.deepseek.com/chat/completions"

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewClient builds a DeepSeek client for one job run. apiKey and model come
// from the current settings row, fetched fresh at the start of each job so
// changes made on the config page take effect without a restart.
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = "deepseek-v4-pro"
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		// Comfortably under Vercel's function duration (300s on Hobby with
		// Fluid Compute, the platform default — see backend/vercel.json) so
		// a stuck request fails with a clear error instead of the platform
		// silently killing the function.
		httpClient: &http.Client{Timeout: 100 * time.Second},
	}
}

var strategySchemaJSON = mustIndentJSON(strategySchema)

func mustIndentJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

const systemPromptTemplate = `Bạn là chuyên gia chiến lược nội dung YouTube. Bạn sẽ nhận một bản phân tích số liệu (JSON) của MỘT kênh YouTube đã tồn tại (kênh tham khảo). Nhiệm vụ của bạn KHÔNG PHẢI là sao chép kênh đó, mà là "biến tấu" (variation): dùng nó làm nguồn cảm hứng để thiết kế một chiến lược cho một KÊNH MỚI, khác biệt, không vi phạm bản quyền, nhắm vào cùng ngách hoặc ngách liền kề nhưng có góc nhìn/định vị riêng.

Sinh đúng 8-10 ý tưởng nội dung, mỗi mục mô tả ngắn gọn (1-2 câu). Trả lời bằng tiếng Việt.

Trả về DUY NHẤT một object JSON hợp lệ, đúng theo cấu trúc sau (không thêm markdown, không thêm giải thích, không thêm text nào khác ngoài JSON):

%s`

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// GenerateStrategy calls DeepSeek with JSON mode enabled, describing the
// required shape in the prompt (DeepSeek's JSON mode guarantees syntactically
// valid JSON, not schema conformance — unlike Anthropic's structured
// outputs, so we validate the shape ourselves via json.Unmarshal below).
func (c *Client) GenerateStrategy(ctx context.Context, analysis models.AnalysisResult) (*models.StrategyOutput, error) {
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		return nil, fmt.Errorf("marshal analysis: %w", err)
	}

	systemPrompt := fmt.Sprintf(systemPromptTemplate, strategySchemaJSON)
	userContent := fmt.Sprintf(
		"Dữ liệu phân tích kênh tham khảo (JSON):\n%s\n\nHãy tạo chiến lược cho kênh mới dựa trên gợi ý phía trên.",
		string(analysisJSON),
	)

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      8000,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var out models.StrategyOutput
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse strategy JSON: %w", err)
	}
	return &out, nil
}

func (c *Client) chat(ctx context.Context, reqBody chatRequest) (string, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("deepseek request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var parsed chatResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("parse deepseek response (status %d): %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := string(respBytes)
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("deepseek api error (status %d): %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("deepseek returned no choices")
	}

	choice := parsed.Choices[0]
	if choice.FinishReason == "length" {
		return "", fmt.Errorf("deepseek output was truncated (hit max_tokens); increase MaxTokens and retry")
	}
	if choice.Message.Content == "" {
		return "", fmt.Errorf("deepseek returned empty content")
	}
	return choice.Message.Content, nil
}
