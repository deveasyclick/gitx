package ai

type deepSeekConfig struct {
	apiKey string
	model  string
}

const deepSeekBaseURL = "https://api.deepseek.com"

func newDeepSeek(cfg deepSeekConfig) Provider {
	client := newOpenAICompatibleClient(openAICompatibleConfig{
		baseURL: deepSeekBaseURL,
		apiPath: "/chat/completions",
		apiKey:  cfg.apiKey,
		model:   cfg.model,
	})
	return &namedProvider{
		name:     "deepseek",
		delegate: client,
	}
}
