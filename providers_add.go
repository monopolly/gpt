package gpt

func (a Provider) Link() string {
	switch a {

	case ProviderGPT:
		return "https://platform.openai.com/docs/models"

	case ProviderGemini:
		return "https://ai.google.dev/gemini-api/docs/models"

	case ProviderGrok:
		return "https://docs.x.ai/docs/models"

	case ProviderDeepseek:
		return "https://api-docs.deepseek.com/quick_start/pricing"

	case ProviderClaude:
		return "https://docs.anthropic.com/en/docs/about-claude/models"

	case ProviderMistral:
		return "https://docs.mistral.ai/getting-started/models/models_overview/"

	case ProviderCohere:
		return "https://docs.cohere.com/docs/models"

	case ProviderMeta:
		return "https://llama.meta.com/docs/models/"

	case ProviderAlibaba:
		return "https://www.alibabacloud.com/help/en/model-studio/getting-started/models"

	case ProviderKimi:
		return "https://platform.moonshot.ai/docs/intro"

	case ProviderPerplexity:
		return "https://docs.perplexity.ai/docs/model-cards"

	default:
		return ""
	}
}

func (a Provider) SID() string {
	switch a {

	case ProviderGPT:
		return "gpt"

	case ProviderGemini:
		return "gemini"

	case ProviderGrok:
		return "grok"

	case ProviderDeepseek:
		return "deepseek"

	case ProviderClaude:
		return "claude"

	case ProviderMistral:
		return "mistral"

	case ProviderCohere:
		return "cohere"

	case ProviderMeta:
		return "meta"

	case ProviderAlibaba:
		return "alibaba"

	case ProviderKimi:
		return "kimi"

	case ProviderPerplexity:
		return "perplexity"

	default:
		return ""
	}
}
