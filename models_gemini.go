package gpt

var (

	// =========================
	// Gemini — VERIFIED MODELS
	// =========================

	Model_Gemini_Pro = Model{
		Provider: ProviderGemini,

		ID: "gemini-3.1-pro-preview",

		Input:  2,
		Output: 12,
		Cached: 0.2,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    true,
		Batch:        true,
	}

	Model_Gemini_Medium = Model{
		Provider: ProviderGemini,

		ID: "gemini-3.5-flash",

		Input:  1.50,
		Output: 9,
		Cached: 0.15,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    true,
		Batch:        true,
	}

	Model_Gemini_Mini = Model{
		Provider: ProviderGemini,

		ID: "gemini-3.1-flash-lite",

		Input:  0.25,
		Output: 1.50,
		Cached: 0.025,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    true,
		Batch:        true,
	}
)
