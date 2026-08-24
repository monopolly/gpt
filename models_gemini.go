package gpt

var (

	// =========================
	// Gemini — VERIFIED MODELS
	// =========================

	Model_Gemini_2_5_Flash_Lite_x0_1 = Model{
		Provider: ProviderGemini,

		Title: "Gemini 2.5 Flash Lite",
		Name:  "gemini-2.5-flash-lite",

		Input:  0.10,
		Output: 0.40,
		Cached: 0.01,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  false,
		WebSearch:    true,
	}

	Model_Gemini_2_5_Flash_x0_2 = Model{
		Provider: ProviderGemini,

		Title: "Gemini 2.5 Flash",
		Name:  "gemini-2.5-flash",

		Input:  0.30,
		Output: 2.50,
		Cached: 0.03,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Gemini_2_5_Pro_x1_10 = Model{
		Provider: ProviderGemini,

		Title: "Gemini 2.5 Pro",
		Name:  "gemini-2.5-pro",

		Input:  1.25,
		Output: 10.00,
		Cached: 0.125,

		Context: 2000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}
)

// preview
var (
	// =========================
	// Gemini PREVIEW
	// =========================

	Model_Gemini_3_1_Flash_Lite_Preview_x0_2 = Model{
		Provider: ProviderGemini,

		Title: "Gemini 3.1 Flash Lite Preview",
		Name:  "gemini-3.1-flash-lite-preview",

		Input:  0.25,
		Output: 1.50,
		Cached: 0.025,

		Context: 1000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}

	Model_Gemini_3_Pro_Preview_x2_12 = Model{
		Provider: ProviderGemini,

		Title: "Gemini 3 Pro Preview",
		Name:  "gemini-3-pro-preview",

		Input:  2.00,
		Output: 12.00,
		Cached: 0.20,

		Context: 2000000,

		ImageSupport: true,
		ImageFiles:   true,

		IsMultimodal: true,
		IsReasoning:  true,
		WebSearch:    true,
	}
)
