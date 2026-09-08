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
		MaxTokens:      16000,
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
		MaxTokens:      4000,
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
		MaxTokens:      2000,
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

const videoPromptSeriesSystemPromptTemplate = `Bạn là biên kịch kiêm đạo diễn cho phim ngắn nhiều tập tạo bằng AI text-to-video (Kling, Google Flow, Runway, Sora...). Bạn sẽ nhận thông tin (tiêu đề, mô tả, tag) của MỘT video YouTube tham khảo. Nhiệm vụ: sáng tác một câu chuyện MỚI, có cốt truyện rõ ràng, chia thành đúng %d tập — LẤY CẢM HỨNG từ chủ đề/không khí của video tham khảo nhưng KHÔNG sao chép cốt truyện, nhân vật hay chi tiết cụ thể của video/bộ phim gốc.

Các công cụ AI tạo video hiện tại chỉ tạo được clip ngắn mỗi lần (vài giây đến khoảng một phút), nên câu chuyện dài được kể bằng cách tạo NHIỀU video riêng biệt (Tập 1, Tập 2, ...), mỗi tập một prompt riêng — không phải một prompt duy nhất cho cả bộ phim.

Hãy xây dựng 2-4 NHÂN VẬT xuyên suốt câu chuyện (nhân vật chính, phản diện, phụ nếu cần), mỗi nhân vật gồm:
- "name": tên nhân vật
- "role": loại nhân vật (vd: "Nhân vật chính diện", "Phản diện", "Nhân vật phụ")
- "appearance": mô tả ngoại hình/trang phục bằng TIẾNG ANH, đủ chi tiết để dán vào prompt text-to-video ở mỗi tập nhằm giữ hình ảnh nhân vật nhất quán giữa các tập
- "coreTags": 3-5 từ khoá ngắn gọn (tiếng Việt) mô tả cốt lõi nhân vật (vd: "kiêu ngạo", "trung thành", "bí ẩn")
- "personalInfo": thông tin cá nhân ngắn gọn (tiếng Việt): tuổi, thân phận, nghề nghiệp/vai trò trong câu chuyện
- "personality": đặc điểm tính cách (tiếng Việt, 1-2 câu)

Mỗi tập cần: tiêu đề ngắn, tóm tắt cốt truyện của tập đó (tiếng Việt, 1-2 câu, nối tiếp mạch truyện xuyên suốt), và một prompt text-to-video (tiếng Anh, mô tả cụ thể chủ thể/hành động/bối cảnh/ánh sáng/chuyển động máy quay cho cảnh quan trọng nhất của tập đó, không quá 80 từ — nhắc tên/ngoại hình nhân vật xuất hiện trong cảnh để khớp với "appearance" đã mô tả).

Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "synopsis": "string, tiếng Việt, tóm tắt cốt truyện tổng thể xuyên suốt các tập",
  "characters": [
    {"name": "string", "role": "string", "appearance": "string (tiếng Anh)", "coreTags": ["string", "..."], "personalInfo": "string", "personality": "string"}
  ],
  "style": "string, tiếng Việt, phong cách hình ảnh chung cho cả series",
  "durationHint": "string, tiếng Việt, độ dài gợi ý cho mỗi tập",
  "episodes": [
    {"episodeNumber": 1, "title": "string", "plotSummary": "string", "prompt": "string", "negativePrompt": "string"}
  ]
}`

// GenerateVideoPromptSeries writes an ORIGINAL multi-episode story inspired
// by one reference video's topic/mood, broken into per-episode text-to-video
// prompts (see VideoPromptSeries) — unlike GenerateVideoPrompt, which writes
// a single prompt for one short clip, this is for a longer serialized story
// told across several separately-generated episode videos.
func (c *Client) GenerateVideoPromptSeries(ctx context.Context, req models.VideoPromptSeriesRequest) (*models.VideoPromptSeries, error) {
	episodeCount := req.EpisodeCount
	if episodeCount <= 0 {
		episodeCount = 5
	}
	if episodeCount > 10 {
		episodeCount = 10
	}
	req.EpisodeCount = episodeCount

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal video series request: %w", err)
	}

	systemPrompt := fmt.Sprintf(videoPromptSeriesSystemPromptTemplate, episodeCount)
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Video tham khảo (JSON):\n%s", string(reqJSON))},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      1800 + episodeCount*350,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var out models.VideoPromptSeries
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse video prompt series JSON: %w", err)
	}
	return &out, nil
}

const scriptSystemPrompt = `Bạn là biên kịch video YouTube chuyên nghiệp. Bạn sẽ nhận một ý tưởng nội dung (tiêu đề, mô tả, hook) và định dạng thời lượng mong muốn. Nhiệm vụ: viết một kịch bản chi tiết theo từng cảnh cho video đó.

Nếu durationFormat là "long" (video dài 5-10 phút): viết khoảng 8-12 cảnh, mỗi cảnh có timecode dạng khoảng thời gian (vd "0:00-0:30").
Nếu durationFormat là "short" (Short 60 giây): viết khoảng 4-6 cảnh ngắn, mỗi cảnh có timecode dạng khoảng thời gian trong 60 giây (vd "0-8s").

Mỗi cảnh gồm: mô tả hình ảnh/hành động cụ thể (visual) và lời thoại/voice-over gợi ý (voiceover, có thể để trống nếu cảnh chỉ có hình ảnh). Viết bằng tiếng Việt, giọng văn tự nhiên, phù hợp để người dùng đọc trực tiếp hoặc lồng tiếng.

Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "hook": "string, câu mở đầu gây chú ý trong 3-5 giây đầu",
  "scenes": [
    {"timecode": "string", "visual": "string", "voiceover": "string"}
  ],
  "callToAction": "string, lời kêu gọi hành động ở cuối video (like, subscribe, comment, ...)",
  "durationFormat": "string, giữ nguyên giá trị durationFormat đã nhận"
}`

// GenerateScript writes a scene-by-scene video script for one content idea,
// tailored to a long-form (5-10 minute) or Short (60 second) duration.
// Standalone and on-demand, like GenerateVideoPrompt — the caller already
// has the idea's title/description/hook from a generated strategy.
func (c *Client) GenerateScript(ctx context.Context, req models.ScriptRequest) (*models.Script, error) {
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal script request: %w", err)
	}

	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: scriptSystemPrompt},
			{Role: "user", Content: fmt.Sprintf("Ý tưởng nội dung (JSON):\n%s", string(reqJSON))},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		MaxTokens:      6000,
	}

	text, err := c.chat(ctx, reqBody)
	if err != nil {
		return nil, err
	}

	var out models.Script
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("parse script JSON: %w", err)
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
