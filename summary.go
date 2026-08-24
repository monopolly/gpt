package gpt

import "time"

// openai, grok
type Summary struct {
	Chat string `json:"id,omitempty"` //conversation id when provider supports conversations
	// Model    string
	Model *Model `json:"model,omitempty"` //conversation
	// Provider string `json:"provider,omitempty"` //conversation

	Input    int `json:"input,omitempty"`
	Output   int `json:"output,omitempty"`
	Resoning int `json:"resoning,omitempty"`
	Cached   int `json:"cached,omitempty"`
	Total    int `json:"total,omitempty"`

	System int `json:"system,omitempty"` //system promt size bytes
	Promt  int `json:"promt,omitempty"`  //promt size bytes
	Images int `json:"images,omitempty"` //bytes eventually

	InputPrice  float64 `json:"input_price,omitempty"`  //if has *Price in Engine
	CachedPrice float64 `json:"cached_price,omitempty"` //if has *Price in Engine
	OutputPrice float64 `json:"output_price,omitempty"` //if has *Price in Engine
	Price       float64 `json:"price,omitempty"`        //if has *Price in Engine

	PromtText  string `json:"promt_text,omitempty"`  //
	SystemText string `json:"system_text,omitempty"` //
	RespText   string `json:"resp_text,omitempty"`   //

	Time  time.Duration `json:"time_duration,omitempty"` //
	Times string        `json:"time_string,omitempty"`   //
	// Raw   any           `json:"raw,omitempty"`           //raw origin resp &{}...
}

func (a *Summary) calc() {

	if a.Input > 0 && a.Model.Input > 0 {
		a.InputPrice = a.Model.Input * float64(a.Input) / 1_000_000
	}
	if a.Cached > 0 && a.Model.Cached > 0 {
		a.CachedPrice = a.Model.Cached * float64(a.Cached) / 1_000_000
	}
	if a.Output > 0 && a.Model.Output > 0 {
		a.OutputPrice = a.Model.Output * float64(a.Output) / 1_000_000
	}

	a.Price = a.InputPrice + a.OutputPrice + a.CachedPrice
}
