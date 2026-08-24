package gpt

// pre-models
var (
	// =========================
	// Grok — VERIFIED MODELS
	// =========================

	Model_Grok_4_x3_15 = Model{
		Provider: ProviderGrok,

		Title: "Grok 4",
		Name:  "grok-4",

		Input:  3.00,
		Output: 15.00,
		Cached: 0.30,

		Context: 256000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Grok_4_Fast_x1_3 = Model{
		Provider: ProviderGrok,

		Title: "Grok 4 Fast",
		Name:  "grok-4-fast",

		Input:  0.75,
		Output: 3.00,
		Cached: 0.075,

		Context: 256000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Grok_3_x2_10 = Model{
		Provider: ProviderGrok,

		Title: "Grok 3",
		Name:  "grok-3",

		Input:  2.00,
		Output: 10.00,
		Cached: 0.20,

		Context: 131072,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Grok_3_Fast = Model{
		Provider: ProviderGrok,

		Title: "Grok 3 Fast",
		Name:  "grok-3-fast",

		Input:  0.50,
		Output: 2.00,
		Cached: 0.05,

		Context: 131072,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    true,
	}
)
