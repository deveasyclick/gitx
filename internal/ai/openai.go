package ai

type openAIConfig struct {
	apiKey string
	model  string
}

const openAIBaseURL = "https://api.openai.com"

func newOpenAI(cfg openAIConfig) Provider {
	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: openAIBaseURL,
		apiPath: "/v1/chat/completions",
		apiKey:  cfg.apiKey,
		model:   cfg.model,
	})
	return &namedProvider{
		name:     "openai",
		delegate: client,
	}
}
