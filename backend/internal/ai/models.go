package ai

const (
	ProviderDeepSeek = "deepseek"
	ProviderZen      = "zen"

	EndpointChat      = "chat"
	EndpointResponses = "responses"
)

type ModelInfo struct {
	ID       string
	Provider string
	Endpoint string
}

// Catalog lists every model selectable from the config page. Endpoint follows
// https://opencode.ai/docs/zen: most Zen models speak OpenAI-compatible
// chat/completions, while Muse Spark Contributor Free is Responses-API only
// (chat/completions on it returns HTTP 500).
var Catalog = []ModelInfo{
	{ID: "deepseek-v4-pro", Provider: ProviderDeepSeek, Endpoint: EndpointChat},
	{ID: "deepseek-v4-flash", Provider: ProviderDeepSeek, Endpoint: EndpointChat},
	{ID: "big-pickle", Provider: ProviderZen, Endpoint: EndpointChat},
	{ID: "mimo-v2.5-free", Provider: ProviderZen, Endpoint: EndpointChat},
	{ID: "ling-3.0-flash-fin-free", Provider: ProviderZen, Endpoint: EndpointChat},
	{ID: "nemotron-3-ultra-free", Provider: ProviderZen, Endpoint: EndpointChat},
	{ID: "nemotron-3.5-lightning-free", Provider: ProviderZen, Endpoint: EndpointChat},
	{ID: "muse-spark-1.2-contributor-free", Provider: ProviderZen, Endpoint: EndpointResponses},
	{ID: "muse-spark-1.3-contributor-free", Provider: ProviderZen, Endpoint: EndpointResponses},
}

func lookupModel(id string) ModelInfo {
	for _, m := range Catalog {
		if m.ID == id {
			return m
		}
	}
	return ModelInfo{ID: id, Provider: ProviderDeepSeek, Endpoint: EndpointChat}
}

func IsZenModel(id string) bool {
	return lookupModel(id).Provider == ProviderZen
}

func IsResponsesModel(id string) bool {
	return lookupModel(id).Endpoint == EndpointResponses
}
