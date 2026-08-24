package gpt

// pre-models
var (

	// =========================
	// OpenAI — VERIFIED MODELS
	// =========================

	// gpt-5.5-pro — топовая reasoning-модель API.
	// Прямого аналога в семействе 5.6 нет: "pro" у 5.6 существует
	// только как уровень reasoning в ChatGPT, но не как отдельная модель API.
	Model_GPT_Max = Model{
		Provider: ProviderGPT,

		ID:        "gpt-5.5-pro",
		Year:      2026,
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

	// gpt-5.6-sol — флагман семейства GPT-5.6 (GA 09.07.2026),
	// на него же указывает алиас gpt-5.6. Заменяет gpt-5.5 по той же цене.
	Model_GPT_Pro = Model{
		Provider: ProviderGPT,

		ID:        "gpt-5.6-sol",
		Year:      2026,
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

	// gpt-5.6-terra — сбалансированный тир, заменяет gpt-5.4 по той же цене.
	Model_GPT_Medium = Model{
		Provider: ProviderGPT,

		ID:        "gpt-5.6-terra",
		Year:      2026,
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

	// gpt-5.6-luna — дешёвый тир для высоких объёмов.
	// Дороже gpt-5.4-mini ($0.75/$4.50), но контекст 1.05M вместо 400K.
	Model_GPT_Mini = Model{
		Provider: ProviderGPT,

		ID:        "gpt-5.6-luna",
		Year:      2026,
		Input:     1.00,
		Output:    6.00,
		Cached:    0.10,
		Reasoning: 6.00,
		Context:   1050000,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,
		IsMultimodal:    true,
		IsReasoning:     true,
		WebSearch:       true,
	}

	// gpt-5.4-nano — самый дешёвый вариант, nano-тира в семействе 5.6 нет.
	Model_GPT_Nano = Model{
		Provider: ProviderGPT,

		ID:        "gpt-5.4-nano",
		Year:      2026,
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

	// gpt-image-2 — замена gpt-image-1, который выключают из API 01.12.2026.
	// Принимает текст и изображения (генерация + редактирование),
	// Output — цена за image-токены на выходе.
	Model_GPT_Image = Model{

		ID:       "gpt-image-2",
		Year:     2026,
		Provider: ProviderGPT,

		Input:     8.00,
		Output:    30.00,
		Reasoning: 0.00,
		Cached:    2.00,

		Context: 0,

		ImageSupport:    true,
		ImageFiles:      true,
		ImageGeneration: true,

		IsMultimodal: true,
		IsReasoning:  false,

		WebSearch: false,
	}
)
