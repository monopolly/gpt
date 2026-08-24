package gpt

// per million
type Model struct {
	ID       string   `json:"model_name"`
	Year     int      `json:"model_year"`
	Provider Provider `json:"provider_name"`

	Input     float64 `json:"input_price_for_1m_usd"`
	Output    float64 `json:"output_price_for_1m_usd"`
	Reasoning float64 `json:"resoning_price_for_1m_usd"`
	Cached    float64 `json:"cached_price_for_1m_usd"`

	BatchInput     float64 `json:"batch_input_price_for_1m_usd"`
	BatchOutput    float64 `json:"batch_output_price_for_1m_usd"`
	BatchReasoning float64 `json:"batch_resoning_price_for_1m_usd"`
	BatchCached    float64 `json:"batch_cached_price_for_1m_usd"`

	Context int `json:"context_window_tokens"`

	ImageSupport    bool `json:"is_image_recognition_support"`
	ImageFiles      bool `json:"is_image_files_links_upload_support"`
	ImageGeneration bool `json:"is_image_generation_support"`
	IsMultimodal    bool `json:"is_multimodal_model"`
	IsReasoning     bool `json:"is_reasoning_model"`
	WebSearch       bool `json:"tool_web_search_support"`
	Batch           bool `json:"batch_support"`
}
