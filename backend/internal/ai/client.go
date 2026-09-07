// Package ai turns a deterministic channel analysis into an AI-generated
// content strategy for a new, "biến tấu" (variation) channel, using the
// Claude API with structured JSON output.
package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"ytagent/backend/internal/models"
)

type Client struct {
	client *anthropic.Client
	model  string
}

// NewClient builds a Claude client for one job run. apiKey and model come
// from the current settings row, fetched fresh at the start of each job so
// changes made on the config page take effect without a restart.
func NewClient(apiKey, model string) *Client {
	c := anthropic.NewClient(option.WithAPIKey(apiKey))
	if model == "" {
		model = "claude-opus-5"
	}
	return &Client{client: &c, model: model}
}

const systemPrompt = `Bạn là chuyên gia chiến lược nội dung YouTube. Bạn sẽ nhận một bản phân tích số liệu (JSON) của MỘT kênh YouTube đã tồn tại (kênh tham khảo). Nhiệm vụ của bạn KHÔNG PHẢI là sao chép kênh đó, mà là "biến tấu" (variation): dùng nó làm nguồn cảm hứng để thiết kế một chiến lược cho một KÊNH MỚI, khác biệt, không vi phạm bản quyền, nhắm vào cùng ngách hoặc ngách liền kề nhưng có góc nhìn/định vị riêng.

Trả lời bằng tiếng Việt. Chỉ trả về JSON đúng theo schema được cung cấp, không thêm giải thích ngoài JSON.`

// GenerateStrategy calls Claude with structured output constrained to the
// StrategyOutput JSON schema.
func (c *Client) GenerateStrategy(ctx context.Context, analysis models.AnalysisResult) (*models.StrategyOutput, error) {
	analysisJSON, err := json.Marshal(analysis)
	if err != nil {
		return nil, fmt.Errorf("marshal analysis: %w", err)
	}

	userContent := fmt.Sprintf(
		"Dữ liệu phân tích kênh tham khảo (JSON):\n%s\n\nHãy tạo chiến lược cho kênh mới dựa trên gợi ý phía trên.",
		string(analysisJSON),
	)

	resp, err := c.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: 16000,
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userContent)),
		},
		OutputConfig: anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: strategySchema,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude request failed: %w", err)
	}

	if resp.StopReason == anthropic.StopReasonRefusal {
		return nil, fmt.Errorf("claude refused the request (category: %s)", resp.StopDetails.Category)
	}
	if resp.StopReason == anthropic.StopReasonMaxTokens {
		return nil, fmt.Errorf("claude output was truncated (hit max_tokens); increase MaxTokens and retry")
	}

	var text string
	for _, block := range resp.Content {
		if b, ok := block.AsAny().(anthropic.TextBlock); ok {
			text += b.Text
		}
	}
	if text == "" {
		return nil, fmt.Errorf("claude returned no text content")
	}

	var out models.StrategyOutput
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse strategy JSON: %w", err)
	}
	return &out, nil
}
