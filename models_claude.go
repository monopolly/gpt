package gpt

// pre-models
var (

	// =========================
	// Claude — VERIFIED MODELS
	// =========================

	Model_Claude_Mini = Model{
		ID:       "claude-haiku-4-5",
		Provider: ProviderClaude,
		Year:     2025,

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

	Model_Claude_Medium = Model{
		ID:       "claude-sonnet-4-6",
		Provider: ProviderClaude,
		Year:     2025,

		Input:  3.00,
		Output: 15.00,
		Cached: 0.10,

		Context: 1_000_000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    false,
	}

	Model_Claude_Pro = Model{
		ID:       "claude-opus-4-8",
		Provider: ProviderClaude,
		Year:     2026,

		Input:  5.00,
		Output: 25.00,
		Cached: 0.10,

		Context: 1_000_000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    false,
	}
)
