package gpt

// pre-models
var (

	// DeepSeek

	Model_DeepSeek_Chat = Model{
		Provider: ProviderDeepseek,
		ID:       "deepseek-chat",
		Input:    0.27, Output: 1.10, Cached: 0.07, Context: 64000,
		ImageSupport: false, ImageFiles: false,
		IsMultimodal: false, IsReasoning: false, WebSearch: false,
	}

	Model_DeepSeek_Reasoner = Model{
		Provider: ProviderDeepseek,
		ID:       "deepseek-reasoner",
		Input:    0.55, Output: 2.19, Cached: 0.14, Context: 64000,
		ImageSupport: false, ImageFiles: false,
		IsMultimodal: false, IsReasoning: true, WebSearch: false,
	}
)
