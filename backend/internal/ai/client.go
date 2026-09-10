// Package ai turns a deterministic channel analysis into an AI-generated
// content strategy for a new, "biến tấu" (variation) channel. It speaks
// three OpenAI-compatible backends, selected per job from the settings row:
//   - DeepSeek (https://api.deepseek.com/chat/completions) with JSON mode.
//   - OpenRouter (https://openrouter.ai/api/v1/chat/completions), whose :free
//     model variants cost nothing via API (no credit card required, ~20
//     req/min and ~200 req/day limits). OpenCode Zen's free tier is NOT used:
//     it rejects API calls made outside the OpenCode client.
//   - 9Router (https://github.com/decolua/9router), a self-hosted router
//     (npm install -g 9router) that fronts 40+ upstream providers behind one
//     OpenAI-compatible endpoint. Defaults to http://localhost:20128/v1
//     (same machine as this backend) but Settings.NineRouterBaseURL can
//     point at a different host.
package ai

import (
	"bufio"
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
	// nineRouterDefaultBaseURL is used when Settings.NineRouterBaseURL is
	// blank — 9Router has no public hosted URL, it's a process the user runs
	// themselves, most commonly on the same machine as this backend at its
	// default port. A Settings-level override (see resolveNineRouterBaseURL)
	// covers running it on a different host.
	nineRouterDefaultBaseURL = "http://localhost:20128/v1"
)

// resolveNineRouterBaseURL falls back to the same-machine default when the
// user hasn't set one, and trims a trailing slash from a custom one so it
// composes cleanly with doPost's "/chat/completions" suffix.
func resolveNineRouterBaseURL(configured string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(configured), "/")
	if trimmed == "" {
		return nineRouterDefaultBaseURL
	}
	return trimmed
}

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
		httpClient: &http.Client{Timeout: httpClientTimeout},
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
	provider := s.AIProviderResolved()
	switch {
	case InCatalog(model):
		// A catalog preset carries its own fixed backend signal.
		provider = ProviderForModel(model)
	case provider == ProviderDeepSeek && (containsSlash(model) || endsWithFree(model)):
		// A custom "author/slug" or "...:free" ID pasted in while the
		// provider setting is still the (deepseek) default is assumed to be
		// an OpenRouter model — keeps older setups (env-seeded, or pasted
		// before an explicit provider toggle existed) working without
		// requiring the provider field too. An explicitly chosen provider
		// (including 9router, whose custom IDs look the same shape — e.g.
		// "cc/claude-opus-4-7" — but aren't OpenRouter) always wins over
		// this guess, since it no longer resolves to the deepseek default.
		provider = ProviderOpenRouter
	}

	var key, baseURL string
	switch provider {
	case ProviderOpenRouter:
		key = s.OpenRouterAPIKey
		baseURL = openRouterBaseURL
	case Provider9Router:
		key = s.NineRouterAPIKey
		baseURL = resolveNineRouterBaseURL(s.NineRouterBaseURL)
	default:
		key = s.DeepSeekAPIKey
		baseURL = deepseekBaseURL
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
		httpClient: &http.Client{Timeout: httpClientTimeout},
	}
}

// httpClientTimeout bounds a single AI call. Was 100s, raised to 280s after
// a real "context deadline exceeded" on a 5-episode video prompt series
// routed through 9Router: a slow/free upstream model plus this backend's own
// SSE reassembly (chatCompletionsStream reads the whole stream before
// returning) can take longer than 100s for a large request. Still 20s under
// the 300s Vercel ceiling referenced above.
const httpClientTimeout = 280 * time.Second

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
	// Stream is always sent explicitly false (no omitempty — Go's zero value
	// for bool already is false, and it must actually appear in the JSON).
	// DeepSeek and OpenRouter both treat an omitted "stream" as false, same
	// as the OpenAI spec, but 9Router does not: leaving it out gets back a
	// "text/event-stream" SSE response instead of one JSON object, which
	// chatCompletions below can't parse — confirmed against a real running
	// 9Router instance.
	Stream bool `json:"stream"`
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
		// Same rationale as scriptMaxTokens/videoPromptSeriesMaxTokens: the
		// catalog's current models support up to a 384K-token max output, and
		// max_tokens only sets a ceiling (billing is by tokens actually
		// generated), so there's no cost to sizing this well above what a
		// strategy JSON should ever need.
		MaxTokens: 128000,
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

// videoPromptSeriesSystemPromptTemplate follows the same "story bible first,
// then episodes" writing method a human screenwriter would use for a
// vi-kịch (short serialized drama): chốt thể loại/đối tượng/logline/nhân vật
// trước, rồi viết từng tập bám theo khuôn của Tập 1, kết mỗi tập giữa bằng
// cliffhanger, và chốt trọn vẹn ở tập cuối — all inside this one JSON call,
// with episodeCount always the caller's requested %d (never chosen by the
// model), so there's no separate copy-paste-between-prompts step for a
// human to do.
const videoPromptSeriesSystemPromptTemplate = `Bạn là biên kịch kiêm đạo diễn cho phim ngắn nhiều tập tạo bằng AI text-to-video (Kling, Google Flow, Runway, Sora...). Bạn sẽ nhận thông tin (tiêu đề, mô tả, tag) của MỘT video YouTube tham khảo. Nhiệm vụ: sáng tác một câu chuyện MỚI, có cốt truyện rõ ràng, chia thành đúng %d tập — LẤY CẢM HỨNG từ chủ đề/không khí của video tham khảo nhưng KHÔNG sao chép cốt truyện, nhân vật hay chi tiết cụ thể của video/bộ phim gốc.

Trước khi viết tập nào, hãy chốt "story bible":
- "genre": 2-3 thể loại ghép lại, mỗi thể loại một cụm ngắn (vd: "giả tưởng cổ đại + phản thần thoại + thế giới phản địa ngục")
- "targetAudience": đối tượng khán giả hướng đến (vd: "Hướng nam giới, dòng chính thống")
- "logline": một câu duy nhất — nêu bối cảnh, nhân vật chính, xung đột/lời nguyền hoặc tiên tri cốt lõi, hành trình phải đi, và ngụ ý về cái kết
- "synopsis": một đoạn 150-250 từ, kể liền mạch toàn bộ câu chuyện xuyên suốt %d tập: bối cảnh mở đầu → nhân vật chính và sứ mệnh/lời nguyền của họ → các mốc gặp gỡ đồng minh chính (gọi đúng tên nhân vật đã liệt kê ở dưới) → xung đột/âm mưu lớn được hé lộ → cách giải quyết và ý nghĩa của cái kết

Các công cụ AI tạo video hiện tại chỉ tạo được clip ngắn mỗi lần (vài giây đến khoảng một phút), nên câu chuyện dài được kể bằng cách tạo NHIỀU video riêng biệt (Tập 1, Tập 2, ...). Bên trong mỗi tập, hãy viết theo phong cách kịch bản quay chuyên nghiệp (shooting script): chia tập đó thành đúng %d CẢNH nối tiếp nhau — mỗi cảnh là một đơn vị bối cảnh/thời gian riêng và sẽ được tạo thành một clip video riêng.

Hãy xây dựng 2-4 NHÂN VẬT xuyên suốt câu chuyện (nhân vật chính, phản diện, phụ nếu cần), mỗi nhân vật gồm:
- "name": tên nhân vật
- "role": loại nhân vật (vd: "Nhân vật chính diện", "Phản diện", "Nhân vật phụ")
- "appearance": mô tả ngoại hình/trang phục bằng TIẾNG ANH, đủ chi tiết để dán vào prompt text-to-video ở mỗi cảnh nhằm giữ hình ảnh nhân vật nhất quán giữa các tập
- "coreTags": 3-5 từ khoá ngắn gọn (tiếng Việt) mô tả cốt lõi nhân vật (vd: "kiêu ngạo", "trung thành", "bí ẩn")
- "personalInfo": thông tin cá nhân ngắn gọn (tiếng Việt): tuổi, thân phận, nghề nghiệp/vai trò trong câu chuyện
- "personality": đặc điểm tính cách (tiếng Việt, 1-2 câu)

Mỗi tập cần: tiêu đề ngắn, tóm tắt cốt truyện của tập đó (tiếng Việt, 1-2 câu, nối tiếp mạch truyện xuyên suốt, KHÔNG lặp lại tình tiết đã xảy ra ở tập trước), một "negativePrompt" dùng chung cho cả tập (tiếng Anh, những gì cần tránh — vd: text, watermark, blurry, distorted faces), và đúng %d cảnh.

Quy tắc bắt buộc theo vị trí từng tập trong mạch truyện:
- Tập 1: thiết lập "khuôn mẫu" — cách chia cảnh, đặt tên, độ dài mô tả của tập này là chuẩn mọi tập sau phải bám theo. Phải giới thiệu bối cảnh, nhân vật chính, và kết ở một biến cố/nút thắt bất ngờ (cliffhanger) dẫn vào tập kế tiếp.
- Các tập giữa: bám sát văn phong/cấu trúc của Tập 1, nối tiếp logic mạch truyện đã xảy ra ở các tập trước, và kết ở một nút thắt/cliffhanger MỚI dẫn sang tập sau.
- Tập cuối (tập số %d): giải quyết trọn vẹn xung đột chính, có cảnh cao trào (hi sinh/lật kèo/đối đầu cuối) rồi hạ nhiệt bằng 1-2 cảnh kết thúc yên bình, khép vòng lặp cảm xúc bằng cách nhắc lại một motif/đồ vật/câu thoại đã gài từ Tập 1. KHÔNG kết bằng cliffhanger ở tập này.

Mỗi cảnh gồm:
- "sceneNumber": số thứ tự cảnh trong tập, bắt đầu từ 1
- "setting": tiếng Việt, thời điểm + nội/ngoại cảnh + địa điểm, ngắn gọn (vd: "Sáng · Ngoại cảnh · Cổng làng biên giới")
- "characters": mảng tên nhân vật (khớp với "name" trong danh sách nhân vật) xuất hiện trong cảnh
- "shotType": chọn MỘT trong: "Toàn cảnh", "Trung cảnh", "Cận cảnh", "Hành động", "Không gian trống" — loại khung hình chủ đạo của cảnh (cảnh cuối một tập giữa mạch nên ưu tiên "Không gian trống": một khung hình khí quyển không thoại, dùng để chuyển cảnh/kết tập gây tò mò)
- "action": tiếng Việt, mô tả ngắn gọn hành động chính và/hoặc câu thoại quan trọng nhất diễn ra trong cảnh (1 câu)
- "prompt": tiếng Anh, mô tả cụ thể chủ thể/hành động/bối cảnh/ánh sáng/chuyển động máy quay cho khung hình quan trọng nhất của cảnh, không quá 45 từ — nhắc tên/ngoại hình nhân vật xuất hiện trong cảnh để khớp với "appearance" đã mô tả

Các cảnh trong một tập phải nối tiếp nhau mạch lạc và dựng lên cao trào của tập đó; ưu tiên đặt các cỡ cảnh tương phản liên tiếp nhau (đặc tả/cận cảnh sau toàn cảnh, hoặc ngược lại) để tạo nhịp điện ảnh. Giữ ngoại hình/trang phục/khí chất nhân vật nhất quán xuyên suốt mọi tập — không tự đổi màu tóc, trang phục, vũ khí... trừ khi cốt truyện có lý do rõ ràng.

Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "genre": "string",
  "targetAudience": "string",
  "logline": "string",
  "synopsis": "string, tiếng Việt, tóm tắt cốt truyện tổng thể xuyên suốt các tập",
  "characters": [
    {"name": "string", "role": "string", "appearance": "string (tiếng Anh)", "coreTags": ["string", "..."], "personalInfo": "string", "personality": "string"}
  ],
  "style": "string, tiếng Việt, phong cách hình ảnh chung cho cả series",
  "durationHint": "string, tiếng Việt, độ dài gợi ý cho mỗi cảnh",
  "episodes": [
    {
      "episodeNumber": 1,
      "title": "string",
      "plotSummary": "string",
      "negativePrompt": "string (tiếng Anh)",
      "scenes": [
        {"sceneNumber": 1, "setting": "string", "characters": ["string", "..."], "shotType": "string", "action": "string", "prompt": "string (tiếng Anh)"}
      ]
    }
  ]
}`

// ClampVideoPromptSeriesDimensions bounds the requested episode count and
// scenes-per-episode to keep the generated JSON within the completion-token
// budget (see videoPromptSeriesMaxTokens): each dimension is bounded
// individually, and their product (total scenes, each with its own
// setting/characters/shotType/action/prompt) is capped separately since the
// two multiply together in output size. Exported so the API handler can
// clamp before persisting the job, keeping the stored ScenesPerEpisode
// consistent with what generation actually used (so a retry reproduces the
// same shape).
func ClampVideoPromptSeriesDimensions(episodeCount, scenesPerEpisode int) (int, int) {
	if episodeCount <= 0 {
		episodeCount = 5
	}
	if episodeCount > 10 {
		episodeCount = 10
	}
	if scenesPerEpisode <= 0 {
		scenesPerEpisode = 3
	}
	if scenesPerEpisode > 5 {
		scenesPerEpisode = 5
	}
	const maxTotalScenes = 30
	if episodeCount*scenesPerEpisode > maxTotalScenes {
		scenesPerEpisode = maxTotalScenes / episodeCount
		if scenesPerEpisode < 2 {
			scenesPerEpisode = 2
		}
	}
	return episodeCount, scenesPerEpisode
}

// GenerateVideoPromptSeries writes an ORIGINAL multi-episode story inspired
// by one reference video's topic/mood, broken into per-episode text-to-video
// prompts (see VideoPromptSeries) — unlike GenerateVideoPrompt, which writes
// a single prompt for one short clip, this is for a longer serialized story
// told across several separately-generated episode videos.
func (c *Client) GenerateVideoPromptSeries(ctx context.Context, req models.VideoPromptSeriesRequest) (*models.VideoPromptSeries, error) {
	episodeCount, scenesPerEpisode := ClampVideoPromptSeriesDimensions(req.EpisodeCount, req.ScenesPerEpisode)
	req.EpisodeCount = episodeCount
	req.ScenesPerEpisode = scenesPerEpisode

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal video series request: %w", err)
	}

	systemPrompt := fmt.Sprintf(videoPromptSeriesSystemPromptTemplate,
		episodeCount, episodeCount, scenesPerEpisode, scenesPerEpisode, episodeCount)
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Video tham khảo (JSON):\n%s", string(reqJSON))},
		},
		ResponseFormat: c.jsonMode(),
		MaxTokens:      videoPromptSeriesMaxTokens(episodeCount, scenesPerEpisode),
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
// series: a 2-4 character cast (~150-250 tokens each) plus, per episode, a
// small title/plotSummary/negativePrompt and, per scene (bounded to at most
// 30 total across the series — see ClampVideoPromptSeriesDimensions), a
// setting/characters/shotType/action/prompt (the English prompt is capped at
// 45 words, ~60 tokens, specifically so more scenes fit this budget — see
// the system prompt template). Margined generously because Vietnamese text
// tokenizes less efficiently than English (diacritics often split into
// multiple subword tokens) and DeepSeek tends to run verbose.
//
// The cap used to be pinned at 8000, then 32000 — both guesses anchored to
// deepseek-chat's old 8192-token max_tokens ceiling, which no longer applies:
// the catalog now defaults to deepseek-v4-pro/deepseek-v4-flash, whose real
// limits (per DeepSeek's docs) are a 1M-token context window and a 384K-token
// max output. Both prior caps still truncated real generations (confirmed via
// repeated "ai output was truncated" errors even at 32000), which means the
// per-scene formula is underestimating actual usage — Vietnamese output,
// verbose models, and JSON structure overhead all add up faster than a tight
// per-field token estimate accounts for. Since max_tokens is only a ceiling
// (billing is by tokens actually generated, not by this cap) there's no cost
// to sizing it far above what any single job should need: 128000 leaves
// enormous headroom under the real 384K ceiling for even the largest allowed
// request (10 episodes x 3 scenes, the max under the 30-scene total cap).
func videoPromptSeriesMaxTokens(episodeCount, scenesPerEpisode int) int {
	totalScenes := episodeCount * scenesPerEpisode
	tokens := 2000 + episodeCount*150 + totalScenes*300
	if tokens > 128000 {
		tokens = 128000
	}
	if tokens < 32000 {
		tokens = 32000
	}
	return tokens
}

const scriptSystemPrompt = `Bạn là biên kịch kiêm đạo diễn video YouTube chuyên nghiệp. Bạn sẽ nhận một ý tưởng nội dung (tiêu đề, mô tả, hook) và định dạng thời lượng mong muốn. Nhiệm vụ: viết kịch bản chi tiết theo từng cảnh, đúng phong cách kịch bản quay chuyên nghiệp (shooting script) — không phải một danh sách hình ảnh + lời thoại đơn giản.

Nếu ý tưởng có nhân vật cụ thể (phim ngắn, hoạt hình, tiểu phẩm, review có host xuyên suốt...), hãy liệt kê nhân vật trong mảng "characters", mỗi nhân vật gồm:
- "name": tên nhân vật
- "role": loại nhân vật (vd: "Nhân vật chính diện", "Phản diện", "Nhân vật phụ")
- "appearance": mô tả ngoại hình/trang phục, đủ chi tiết để giữ hình ảnh nhất quán xuyên suốt kịch bản
- "coreTags": 3-5 từ khoá ngắn gọn mô tả cốt lõi nhân vật (vd: "tinh quái", "tốt bụng")
- "personalInfo": tuổi, thân phận, nghề nghiệp/vai trò
- "personality": đặc điểm tính cách (1-2 câu)
Nếu ý tưởng KHÔNG có nhân vật cụ thể (vd: video giải thích, liệt kê, không thoại), để mảng "characters" rỗng.

Nếu durationFormat là "long" (video dài 5-10 phút): viết khoảng 8-12 cảnh.
Nếu durationFormat là "short" (Short 60 giây): viết khoảng 4-6 cảnh ngắn.

Mỗi cảnh gồm:
- "sceneNumber": số thứ tự cảnh, bắt đầu từ 1
- "timecode": khoảng thời gian trong video (vd "0:00-0:30" cho video dài, "0-8s" cho Short)
- "setting": thời điểm + nội/ngoại cảnh + địa điểm, ngắn gọn (vd: "Sáng · Ngoại cảnh · Quán cà phê")
- "characters": mảng tên nhân vật (khớp "name" ở trên) xuất hiện trong cảnh — rỗng nếu cảnh không có nhân vật cụ thể
- "shots": 1-3 khung hình trong cảnh, mỗi khung gồm "shotType" (chọn: "Toàn cảnh", "Trung cảnh", "Cận cảnh", "Động tác") và "description" (mô tả cụ thể hành động/hình ảnh trong khung đó) — chia nhỏ theo khung hình thay vì viết chung một đoạn
- "dialogue": lời thoại/voice-over trong cảnh (mảng, để rỗng nếu cảnh chỉ có hình ảnh), mỗi dòng gồm "character" (tên người nói, hoặc "Voice-over" nếu không phải nhân vật cụ thể), "direction" (chỉ dẫn diễn xuất ngắn gọn — giọng điệu, cảm xúc, hành động khi nói; KHÔNG tự thêm dấu ngoặc, chỉ viết phần chữ, vd "giọng trầm khàn, tự mãn" chứ không phải "(giọng trầm khàn, tự mãn)"; để trống nếu không cần), "line" (câu thoại)
- "cutaway": (tuỳ chọn) một khung hình khí quyển không thoại để kết cảnh/chuyển cảnh, tạo nhịp cho video — không phải cảnh nào cũng cần, ưu tiên dùng ở cảnh gây tò mò hoặc cảnh cuối; để trống nếu không cần

Viết bằng tiếng Việt, giọng văn tự nhiên, phù hợp để người dùng đọc trực tiếp hoặc lồng tiếng.

Trả về DUY NHẤT một object JSON hợp lệ theo cấu trúc sau, không thêm markdown hay giải thích ngoài JSON:

{
  "hook": "string, câu mở đầu gây chú ý trong 3-5 giây đầu",
  "characters": [
    {"name": "string", "role": "string", "appearance": "string", "coreTags": ["string"], "personalInfo": "string", "personality": "string"}
  ],
  "scenes": [
    {
      "sceneNumber": 1,
      "timecode": "string",
      "setting": "string",
      "characters": ["string"],
      "shots": [{"shotType": "string", "description": "string"}],
      "dialogue": [{"character": "string", "direction": "string", "line": "string"}],
      "cutaway": "string"
    }
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
		MaxTokens:      scriptMaxTokens(req.DurationFormat),
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

// scriptMaxTokens sizes the output budget by duration format instead of one
// flat number for both: "long" asks for 8-12 scenes, each now a richer
// shooting-script beat (optional character cast, a setting line, 1-3 shots,
// a dialogue array with acting directions, an optional cutaway) rather than
// a flat timecode/visual/voiceover triple, so it needs real headroom.
//
// This has been raised twice already (6000 -> 8000 -> 16000), each time
// anchored to deepseek-chat's old 8192-token ceiling, and each time still
// truncating real "long" scripts — confirming the bottleneck isn't the
// model's real limit but this budget itself being sized too tightly. The
// catalog's current models (deepseek-v4-pro/deepseek-v4-flash) support up to
// a 1M-token context and 384K-token max output per DeepSeek's docs, and
// max_tokens is only a ceiling (billing is by tokens actually generated, not
// by this cap), so there's no cost to sizing it generously: 128000/64000
// leave enormous headroom under the real model ceiling.
func scriptMaxTokens(durationFormat string) int {
	if durationFormat == "long" {
		return 128000
	}
	return 64000
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
	if c.provider == Provider9Router {
		// Confirmed against a real running 9Router instance: at least its
		// OpenCode Free models reliably return HTTP 200 with an empty
		// message.content when stream is false — content-less "success" that
		// chatCompletions can't tell apart from a genuinely empty answer.
		// The same models answer correctly when streamed, so 9Router always
		// streams and reassembles the full text from the SSE chunks instead.
		// See https://github.com/decolua/9router/issues/1025 for the same
		// class of bug reported upstream for other providers/endpoints.
		reqBody.Stream = true
		return c.chatCompletionsStream(ctx, reqBody)
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

// chatStreamChunk is one "data: {...}" line of an OpenAI-style SSE chat
// completion stream — the delta variant of chatResponse's Choices.
type chatStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// chatCompletionsStream is chatCompletions' 9Router-only counterpart: it
// sends the same request (with Stream forced true by the caller) and
// reassembles the full answer from the SSE response instead of expecting one
// JSON object. Falls back to parsing the body as a plain (non-streamed)
// response if it turns out not to be SSE at all — some models routed
// through 9Router may still ignore "stream": true.
func (c *Client) chatCompletionsStream(ctx context.Context, reqBody chatRequest) (string, error) {
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ai api error (status %d): %s", resp.StatusCode, string(raw))
	}

	if content, truncated, ok := parseSSEContent(raw); ok {
		if truncated {
			return "", fmt.Errorf("ai output was truncated (hit max_tokens); increase MaxTokens and retry")
		}
		return extractJSON(content)
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("parse ai response (status %d): %w", resp.StatusCode, err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai returned no choices")
	}
	return extractJSON(parsed.Choices[0].Message.Content)
}

// parseSSEContent reassembles the full message text from an OpenAI-style SSE
// response ("data: {...}" lines), concatenating each chunk's delta.content
// in order — streaming only ever splits the text into pieces, never
// reorders it, so simple concatenation reconstructs it exactly, JSON-mode
// output included. truncated mirrors chatCompletions' FinishReason ==
// "length" check, so a stream cut off by max_tokens surfaces the same clear
// error instead of a confusing downstream JSON-parse failure. ok is false
// when raw contains no "data:" line at all, telling the caller to try
// parsing it as a single JSON object instead.
func parseSSEContent(raw []byte) (content string, truncated bool, ok bool) {
	var sb strings.Builder
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	sawData := false
	for scanner.Scan() {
		data, isData := strings.CutPrefix(scanner.Text(), "data:")
		if !isData {
			continue
		}
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			continue
		}
		sawData = true
		var chunk chatStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			sb.WriteString(choice.Delta.Content)
			if choice.FinishReason == "length" {
				truncated = true
			}
		}
	}
	if !sawData {
		return "", false, false
	}
	return sb.String(), truncated, true
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
