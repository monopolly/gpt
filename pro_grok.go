package gpt

import (
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// x.ai
func Grok(token string, model ...*Model) (a Engine) {
	p := new(gpt)
	p.provider = ProviderGrok
	p.token = token

	switch len(model) > 0 {
	case true:
		p.model = model[0]
	default:
		p.model = &Model_Grok
	}

	p.conn = openai.NewClient(option.WithAPIKey(token), option.WithBaseURL("https://api.x.ai/v1"))
	return p
}
