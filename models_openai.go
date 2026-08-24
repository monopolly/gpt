package gpt

// pre-models
var (

	// =========================
	// OpenAI — VERIFIED MODELS
	// =========================

	Model_GPT_5_5_x5_30 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.5",
		Name:      "gpt-5.5",
		Input:     5.00,
		Output:    30.00,
		Cached:    0.50,
		Reasoning: 30.00,
		Context:   1050000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	Model_GPT_5_5_Pro_x30_180 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.5 Pro",
		Name:      "gpt-5.5-pro",
		Input:     30.00,
		Output:    180.00,
		Cached:    0.00,
		Reasoning: 180.00,
		Context:   1050000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	Model_GPT_5_4_x3_15 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.4",
		Name:      "gpt-5.4",
		Input:     2.50,
		Output:    15.00,
		Cached:    0.25,
		Reasoning: 15.00,
		Context:   1050000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	Model_GPT_5_4_Mini_x1_5 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.4 Mini",
		Name:      "gpt-5.4-mini",
		Input:     0.75,
		Output:    4.50,
		Cached:    0.075,
		Reasoning: 4.50,
		Context:   400000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	Model_GPT_5_4_Nano_x1_2 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.4 Nano",
		Name:      "gpt-5.4-nano",
		Input:     0.20,
		Output:    1.25,
		Cached:    0.02,
		Reasoning: 1.25,
		Context:   400000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       false,
	}

	Model_GPT_5_4_Pro_x30_180 = Model{
		Provider:  ProviderGPT,
		Title:     "GPT-5.4 Pro",
		Name:      "gpt-5.4-pro",
		Input:     30.00,
		Output:    180.00,
		Cached:    0.00,
		Reasoning: 180.00,
		Context:   1050000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	Model_GPT_Image_v1_x5 = Model{
		Title:    "GPT Image 1",
		Name:     "gpt-image-1",
		Date:     "2025-04-01",
		Provider: ProviderGPT,

		Input:     5.00,
		Output:    0.00,
		Reasoning: 0.00,
		Cached:    0.00,

		Context: 0,

		ImageSupport:    false,
		ImageFiles:      false,
		ImageGeneration: true,

		IsMultimodal: false,
		IsReasoning:  false,

		WebSearch: false,
	}
)
