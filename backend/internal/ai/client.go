// Package ai turns a deterministic channel analysis into an AI-generated
// content strategy for a new, "biến tấu" (variation) channel. It speaks two
// OpenAI-compatible backends, selected per job from the settings row:
//   - DeepSeek (https://api.deepseek.com/chat/completions) with JSON mode.
//   - OpenRouter (https://openrouter.ai/api/v1/chat/completions), whose :free
//     model variants cost nothing via API (no credit card required, ~20
//     req/min and ~200 req/day limits). OpenCode Zen's free tier is NOT used:
//     it rejects API calls made outside the OpenCode client.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ytagent/backend/internal/models"
)

const (
	deepseekBaseURL   = "https://api.deepseek.com"
	openRouterBaseURL = "https://openrouter.ai/api/v1"
)

type Client struct {
	apiKey     string
	model      string
	provider   string
	endpoint   string
	baseURL    string
	httpClient *http.Client
}

// NewClient builds an AI client for one job run. apiKey and model come
// from the current settings row, fetched fresh at the start of each job so
// changes made on the config page take effect without a restart. The model
// ID determines the backend (see Catalog and ProviderForModel); unknown IDs
// fall back to DeepSeek.
func NewClient(apiKey, model string) *Client {
	if model == "" {
		model = "deepseek-v4-pro"
	}
	provider := ProviderForModel(model)
	baseURL := deepseekBaseURL
	if provider == ProviderOpenRouter {
		baseURL = openRouterBaseURL
	}
	return &Client{
		apiKey:     apiKey,
		model:      model,
		provider:   provider,
		endpoint:   lookupModel(model).Endpoint,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 100 * time.Second},
	}
}

// NewClientFromSettings resolves the key for the selected provider and
// forces the model's home backend, so picking an OpenRouter model while the
// provider still says deepseek (or vice versa) still calls the right
// endpoint.
func NewClientFromSettings(s models.Settings) *Client {
	model := s.AIModel
	if model == "" {
		model = "deepseek-v4-pro"
	}
	// Preset models and OpenRouter-shaped custom IDs ("author/slug", ":free")
	// carry their own backend signal; a plain custom ID honors the explicit
	// provider setting instead.
	provider := ProviderForModel(model)
	if !InCatalog(model) && !containsSlash(model) && !endsWithFree(model) {
		provider = s.AIProviderResolved()
	}
	key := s.DeepSeekAPIKey
	baseURL := deepseekBaseURL
	if provider == ProviderOpenRouter {
		key = s.OpenRouterAPIKey
		baseURL = openRouterBaseURL
	}
	return &Client{
		apiKey:   key,
		model:    model,
		provider: provider,
		endpoint: lookupModel(model).Endpoint,
		baseURL:  baseURL,
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

// GenerateStrategy calls the configured AI backend with JSON mode enabled,
// describing the required shape in the prompt (JSON mode guarantees
// syntactically valid JSON, not schema conformance — unlike Anthropic's
// structured outputs, so we validate the shape ourselves via json.Unmarshal
// below). On Zen backends without JSON mode the same prompt plus extractJSON
// keeps the contract.
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
		ResponseFormat: c.jsonMode(),
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
		ResponseFormat: c.jsonMode(),
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
		ResponseFormat: c.jsonMode(),
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
		ResponseFormat: c.jsonMode(),
		MaxTokens:      videoPromptSeriesMaxTokens(episodeCount),
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

// videoPromptSeriesMaxTokens sizes the output budget for a multi-episode
// series: a 2-4 character cast (~150-250 tokens each) plus per-episode
// title/plotSummary/prompt/negativePrompt, generously margined because
// Vietnamese text tokenizes less efficiently than English (diacritics often
// split into multiple subword tokens) and DeepSeek tends to run verbose.
// Capped at 8000 to stay clear of common provider completion-token ceilings.
func videoPromptSeriesMaxTokens(episodeCount int) int {
	tokens := 3000 + episodeCount*600
	if tokens > 8000 {
		tokens = 8000
	}
	return tokens
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
		ResponseFormat: c.jsonMode(),
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

// jsonMode enables the chat API's JSON mode. Both DeepSeek and OpenRouter
// (which normalizes response_format across providers) honor it; extractJSON
// below still guards against models that wrap the answer in fences or prose.
func (c *Client) jsonMode() *responseFormat {
	return &responseFormat{Type: "json_object"}
}

func (c *Client) chat(ctx context.Context, reqBody chatRequest) (string, error) {
	if c.endpoint == EndpointResponses {
		return c.responses(ctx, reqBody)
	}
	return c.chatCompletions(ctx, reqBody)
}

func (c *Client) doPost(ctx context.Context, url string, reqBody any, out any) (int, []byte, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.provider == ProviderOpenRouter {
		req.Header.Set("HTTP-Referer", "https://github.com/tinnt-truman/yt-agent")
		req.Header.Set("X-Title", "YT-Agent")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	if err := json.Unmarshal(respBytes, out); err != nil {
		return resp.StatusCode, respBytes, fmt.Errorf("parse ai response (status %d): %w", resp.StatusCode, err)
	}
	return resp.StatusCode, respBytes, nil
}

func (c *Client) chatCompletions(ctx context.Context, reqBody chatRequest) (string, error) {
	var parsed chatResponse
	status, raw, err := c.doPost(ctx, c.baseURL+"/chat/completions", reqBody, &parsed)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		msg := string(raw)
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("ai api error (status %d): %s", status, msg)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai returned no choices")
	}

	choice := parsed.Choices[0]
	if choice.FinishReason == "length" {
		return "", fmt.Errorf("ai output was truncated (hit max_tokens); increase MaxTokens and retry")
	}
	return extractJSON(choice.Message.Content)
}

type responsesRequest struct {
	Model           string `json:"model"`
	Input           string `json:"input"`
	MaxOutputTokens int    `json:"max_output_tokens,omitempty"`
}

type responsesResponse struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// responses calls Zen's Responses API (Muse Spark Contributor Free models).
// The prompt already demands a single JSON object, so system+user text are
// concatenated into one input string.
func (c *Client) responses(ctx context.Context, reqBody chatRequest) (string, error) {
	var sb strings.Builder
	for i, m := range reqBody.Messages {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(m.Content)
	}
	in := responsesRequest{
		Model:           c.model,
		Input:           sb.String(),
		MaxOutputTokens: reqBody.MaxTokens,
	}

	var parsed responsesResponse
	status, raw, err := c.doPost(ctx, c.baseURL+"/responses", in, &parsed)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		msg := string(raw)
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return "", fmt.Errorf("ai api error (status %d): %s", status, msg)
	}
	text := parsed.OutputText
	if text == "" {
		for _, item := range parsed.Output {
			for _, part := range item.Content {
				text += part.Text
			}
		}
	}
	return extractJSON(text)
}

// extractJSON tolerates models that wrap the answer in markdown fences or add
// stray prose around it: strip ```json fences, then cut to the outermost
// {...} when present. Returns an error on empty output so callers fail
// loudly instead of storing garbage.
func extractJSON(text string) (string, error) {
	t := strings.TrimSpace(text)
	if t == "" {
		return "", fmt.Errorf("ai returned empty content")
	}
	t = strings.TrimPrefix(t, "```json")
	t = strings.TrimPrefix(t, "```")
	t = strings.TrimSuffix(t, "```")
	t = strings.TrimSpace(t)
	if start := strings.Index(t, "{"); start > 0 {
		t = t[start:]
	}
	if end := strings.LastIndex(t, "}"); end >= 0 {
		t = strings.TrimSpace(t[:end+1])
	}
	if t == "" {
		return "", fmt.Errorf("ai returned empty content")
	}
	return t, nil
}
