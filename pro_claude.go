package gpt

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func Claude(token string, model ...*Model) (a Engine) {
	p := new(claude)
	p.conn = anthropic.NewClient(option.WithAPIKey(token))

	if len(model) > 0 {
		p.model = model[0]
	} else {
		p.model = &Model_Claude_Sonnet_4_x3_15
	}

	p.token = token
	p.provider = ProviderClaude
	return p
}

type claude struct {
	conn     anthropic.Client
	model    *Model
	provider Provider
	token    string
}

func (a *claude) Provider() Provider {
	return a.provider
}

func (a *claude) Model(v ...*Model) *Model {
	switch v {
	case nil:
		return a.model
	default:
		a.model = v[0]
		return nil
	}
}

func (a *claude) Models() (res []Model, err error) {
	return GetModels(a)
}

func (a *claude) getToken() string {
	return a.token
}

func (a *claude) Chat(m *Message) (res string, err error) {
	if m.name == "" {
		m.name = "chat"
	}
	if m.plaintext {
		m.Promt("Answer must be plain text! No markdown!")
	}
	err = a.Send(m)
	if err != nil {
		return
	}
	res = m.CleanText()
	return
}

func (a *claude) Send(m *Message) (err error) {
	if m.name == "" {
		m.name = "request"
	}

	if m.result != nil {
		applySystemPromtStruct(m)
	}

	t1 := time.Now()
	promt := m.RenderPromt()
	system := m.RenderSystemPromt()

	var blocks []anthropic.ContentBlockParamUnion

	for _, x := range m.images {
		blocks = append(blocks, anthropic.NewImageBlockBase64("image/jpeg", claudeImageBase64JPEG(x)))
	}

	blocks = append(blocks, anthropic.NewTextBlock(promt))

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model.Name),
		MaxTokens: int64(a.model.Output * 1000),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(blocks...),
		},
	}

	if system != "" {
		req.System = []anthropic.TextBlockParam{{Text: system}}
	}

	if len(m.schema) > 0 {
		req.OutputConfig = anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: m.schema,
			},
		}
	}

	resp, err := a.conn.Messages.New(context.Background(), req)
	if err != nil {
		return
	}

	var parts []string
	for _, c := range resp.Content {
		if c.Text != "" {
			parts = append(parts, c.Text)
		}
	}

	m.resp = resp
	m.raw = strings.Join(parts, "\n")
	m.clean = CleanMarkdown(m.raw)

	if len(m.schema) > 0 {
		_ = json.Unmarshal([]byte(m.raw), &m.result)
	}

	m.provider = a.provider
	m.summary = Summary{
		Chat:       resp.ID,
		Model:      a.model,
		Promt:      len(promt),
		System:     len(system),
		Images:     m.imagesSize(),
		Input:      int(resp.Usage.InputTokens),
		Output:     int(resp.Usage.OutputTokens),
		Total:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		PromtText:  promt,
		SystemText: system,
		RespText:   m.raw,
	}

	m.summary.calc()
	m.summary.Time = time.Since(t1)
	m.summary.Times = m.summary.Time.String()
	m.price = m.summary.Price
	return
}
