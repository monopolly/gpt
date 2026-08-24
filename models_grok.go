package gpt

// pre-models
var (
	// =========================
	// Grok — VERIFIED MODELS
	// =========================

	// latest
	Model_Grok = Model{
		Provider: ProviderGrok,

		ID: "grok-4.3",

		Input:  1.25,
		Output: 2.50,
		Cached: 0.20,

		Context: 1_000_000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Grok_Images = Model{
		Provider: ProviderGrok,

		ID: "grok-imagine-image-quality",

		Input:  1.25,
		Output: 3.00,
		Cached: 0.20,

		Context: 1_000_000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}
)
