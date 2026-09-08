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

const trendingInsightSystemPrompt = `Bạn là chuyên gia phân tích xu hướng YouTube. Bạn sẽ nhận danh sách các kênh đang có video trending (JSON), gồm tên kênh, số video đang trending, tổng lượt xem, subscriber. Hãy tóm tắt ngắn gọn các chủ đề/ngách đang nổi bật trong danh sách này, và gợi ý 3-5 cơ hội nội dung cụ thể mà một kênh mới có thể khai thác dựa trên xu hướng này.

Trả lời bằng tiếng Việt. Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "summary": "string, 3-5 câu tóm tắt xu hướng nổi bật",
  "opportunities": ["string", "..."]
}`

// GenerateTrendingInsight summarizes a trending-channels report into a short
// written commentary. Optional and on-demand from the frontend — unlike
// GenerateStrategy this isn't part of the core pipeline.
func (c *Client) GenerateTrendingInsight(ctx context.Context, report models.TrendingReport) (*models.TrendingInsight, error) {
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: trendingInsightSystemPrompt},
			{Role: "user", Content: fmt.Sprintf("Danh sách kênh trending (JSON):\n%s", string(reportJSON))},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      2000,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var out models.TrendingInsight
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse trending insight JSON: %w", err)
	}
	return &out, nil
}

const videoPromptSystemPrompt = `Bạn là chuyên gia viết prompt cho công cụ AI tạo video từ văn bản (text-to-video) như Kling, Runway, Sora. Bạn sẽ nhận thông tin (tiêu đề, mô tả, tag) của MỘT video YouTube tham khảo. Nhiệm vụ: viết một prompt MỚI để tạo ra một video khác, cùng chủ đề/không khí với video tham khảo nhưng KHÔNG sao chép cảnh quay hay chi tiết cụ thể của video gốc — chỉ lấy cảm hứng về chủ đề, không khí, phong cách hình ảnh.

Trường "prompt" và "negativePrompt" PHẢI viết bằng tiếng Anh (mô hình text-to-video hiện tại hiểu và bám sát prompt tiếng Anh tốt hơn hẳn tiếng Việt). Prompt cần mô tả cụ thể: chủ thể, bối cảnh, ánh sáng, chuyển động máy quay, phong cách hình ảnh — đủ chi tiết để AI tạo ra một cảnh quay rõ ràng, nhưng không quá 80 từ.

Trường "style" và "durationHint" viết bằng tiếng Việt.

Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "prompt": "string, tiếng Anh, mô tả cảnh quay chi tiết cho AI text-to-video",
  "negativePrompt": "string, tiếng Anh, những gì cần tránh (vd: text, watermark, blurry, distorted faces)",
  "style": "string, tiếng Việt, phong cách hình ảnh gợi ý (vd: 'điện ảnh, tông màu ấm, quay chậm')",
  "durationHint": "string, tiếng Việt, độ dài clip gợi ý (vd: '5-10 giây mỗi cảnh')"
}`

// GenerateVideoPrompt writes a text-to-video generation prompt inspired by
// one reference video's topic/mood — not a reproduction of it. Standalone
// and on-demand (like GenerateTrendingInsight): the caller already has the
// video's metadata (from an analysis result or a trending report), so this
// needs no extra YouTube API call.
func (c *Client) GenerateVideoPrompt(ctx context.Context, req models.VideoPromptRequest) (*models.VideoPrompt, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal video: %w", err)
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: videoPromptSystemPrompt},
			{Role: "user", Content: fmt.Sprintf("Video tham khảo (JSON):\n%s", string(reqJSON))},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      1000,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var out models.VideoPrompt
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse video prompt JSON: %w", err)
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
