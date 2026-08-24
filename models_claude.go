package gpt

// pre-models
var (

	// =========================
	// Claude — VERIFIED MODELS
	// =========================

	Model_Claude_Haiku_4_x1_5 = Model{
		Provider: ProviderClaude,

		Title: "Claude Haiku 4.5",
		Name:  "claude-haiku-4-5",

		Input:  1.00,
		Output: 5.00,
		Cached: 0.10,

		Context: 200000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    false,
	}

	Model_Claude_Sonnet_4_x3_15 = Model{
		Provider: ProviderClaude,

		Title: "Claude Sonnet 4.6",
		Name:  "claude-sonnet-4-6",

		Input:  3.00,
		Output: 15.00,
		Cached: 0.30,

		Context: 200000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    false,
	}

	Model_Claude_Opus_4_x15_75 = Model{
		Provider: ProviderClaude,

		Title: "Claude Opus 4.7",
		Name:  "claude-opus-4-7",

		Input:  5.00,
		Output: 25.00,
		Cached: 0.50,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    false,
	}
)
