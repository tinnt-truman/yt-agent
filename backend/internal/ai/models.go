package ai

const (
	ProviderDeepSeek  = "deepseek"
	ProviderOpenRouter = "openrouter"

	EndpointChat      = "chat"
	EndpointResponses = "responses"
)

type ModelInfo struct {
	ID       string
	Provider string
	Endpoint string
}

// Catalog lists the selectable models. OpenRouter :free variants (9/2026) are
// callable via API — unlike OpenCode Zen's free tier, which rejects calls
// made outside the OpenCode client. The list rotates, so the config page also
// accepts a custom model ID; anything unknown containing "/" or ending in
// ":free" is assumed to be OpenRouter (see ProviderForModel).
var Catalog = []ModelInfo{
	{ID: "deepseek-v4-pro", Provider: ProviderDeepSeek, Endpoint: EndpointChat},
	{ID: "deepseek-v4-flash", Provider: ProviderDeepSeek, Endpoint: EndpointChat},
	{ID: "openrouter/free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "nvidia/nemotron-3-ultra-550b-a55b:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "nvidia/nemotron-3.5-lightning:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "inclusionai/ling-3.0-flash-fin:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "minimax/minimax-m3:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "minimax/minimax-m2.7:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
	{ID: "z-ai/glm-5.2:free", Provider: ProviderOpenRouter, Endpoint: EndpointChat},
}

func lookupModel(id string) ModelInfo {
	for _, m := range Catalog {
		if m.ID == id {
			return m
		}
	}
	return ModelInfo{ID: id, Provider: ProviderForModel(id), Endpoint: EndpointChat}
}

// InCatalog reports whether id is one of the preset models.
func InCatalog(id string) bool {
	for _, m := range Catalog {
		if m.ID == id {
			return true
		}
	}
	return false
}

// ProviderForModel infers the backend from a model ID, including custom IDs
// the UI lets the user paste in: OpenRouter IDs look like "author/slug" and
// its free variants end in ":free".
func ProviderForModel(id string) string {
	for _, m := range Catalog {
		if m.ID == id {
			return m.Provider
		}
	}
	if len(id) > 0 && (containsSlash(id) || endsWithFree(id)) {
		return ProviderOpenRouter
	}
	return ProviderDeepSeek
}

func containsSlash(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '/' {
			return true
		}
	}
	return false
}

func endsWithFree(s string) bool {
	const suffix = ":free"
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}

func IsResponsesModel(id string) bool {
	return lookupModel(id).Endpoint == EndpointResponses
}
