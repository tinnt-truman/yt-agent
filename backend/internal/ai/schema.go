package ai

// strategySchema mirrors models.StrategyOutput as a JSON-Schema-shaped
// description. DeepSeek's JSON mode has no schema-enforcement parameter, so
// this is embedded as text in the prompt (see client.go) to tell the model
// the exact shape to produce; the result is still validated by unmarshalling
// into models.StrategyOutput after the call.
var strategySchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"positioning": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"niche":            map[string]any{"type": "string"},
				"targetAudience":   map[string]any{"type": "string"},
				"uniqueAngle":      map[string]any{"type": "string"},
				"channelNameIdeas": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required":             []string{"niche", "targetAudience", "uniqueAngle", "channelNameIdeas"},
			"additionalProperties": false,
		},
		"contentPillars": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":        map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"percentage":  map[string]any{"type": "integer"},
				},
				"required":             []string{"name", "description", "percentage"},
				"additionalProperties": false,
			},
		},
		"contentIdeas": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":           map[string]any{"type": "string"},
					"format":          map[string]any{"type": "string", "enum": []string{"Shorts", "Long-form"}},
					"hook":            map[string]any{"type": "string"},
					"description":     map[string]any{"type": "string"},
					"estimatedLength": map[string]any{"type": "string"},
				},
				"required":             []string{"title", "format", "hook", "description", "estimatedLength"},
				"additionalProperties": false,
			},
		},
		"postingSchedule": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"frequencyPerWeek": map[string]any{"type": "integer"},
				"bestDays":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"bestTimeOfDay":    map[string]any{"type": "string"},
			},
			"required":             []string{"frequencyPerWeek", "bestDays", "bestTimeOfDay"},
			"additionalProperties": false,
		},
		"titleTemplates": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"seoKeywords":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"hashtagSets": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"theme":    map[string]any{"type": "string"},
					"hashtags": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required":             []string{"theme", "hashtags"},
				"additionalProperties": false,
			},
		},
		"growthTactics": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"risks":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required": []string{
		"positioning", "contentPillars", "contentIdeas", "postingSchedule",
		"titleTemplates", "seoKeywords", "hashtagSets", "growthTactics", "risks",
	},
	"additionalProperties": false,
}
