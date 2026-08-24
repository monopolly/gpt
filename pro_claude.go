package gpt

import (
	"context"
	"encoding/json"
	"errors"
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
		p.model = &Model_Claude_Mini
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

	fallback []Engine
}

func (a *claude) Provider() Provider {
	return a.provider
}

func (a *claude) Fallback(v ...Engine) Engine {
	a.fallback = append(a.fallback, v...)
	return a
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

// upload a file to the anthropic files api (beta).
// accepts any claude-compatible format (pdf, images, text, etc).
// an explicit filename (with extension) helps the api detect the format.
func (a *claude) UploadFile(body []byte, filename ...string) (fileID string, err error) {
	if len(body) == 0 {
		return "", errors.New("claude upload: empty body")
	}

	ctx := context.Background()
	resp, err := a.conn.Beta.Files.Upload(ctx, anthropic.BetaFileUploadParams{
		File:  newUploadFile(body, filename...),
		Betas: []anthropic.AnthropicBeta{anthropic.AnthropicBetaFilesAPI2025_04_14},
	})
	if err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", errors.New("claude upload: empty file id")
	}

	return resp.ID, nil
}

// delete stored files
func (a *claude) DeleteFiles(filesID ...string) (err error) {
	ctx := context.Background()
	for _, id := range filesID {
		if id == "" {
			continue
		}
		if _, e := a.conn.Beta.Files.Delete(ctx, id, anthropic.BetaFileDeleteParams{
			Betas: []anthropic.AnthropicBeta{anthropic.AnthropicBetaFilesAPI2025_04_14},
		}); e != nil {
			err = e
		}
	}
	return
}

// claude has no server side conversations, attach files to the message instead:
// Message.AddFiles / Message.AddImageFiles
func (a *claude) AddFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("claude has no conversations: use Message.AddFiles")
}

// claude has no server side conversations, attach files to the message instead:
// Message.AddFiles / Message.AddImageFiles
func (a *claude) AddImageFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("claude has no conversations: use Message.AddImageFiles")
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

	req := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model.ID),
		MaxTokens: int64(a.model.Output * 1000),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(claudeBlocks(m)...),
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

	resp, err := a.conn.Messages.New(context.Background(), req, claudeOptions(m)...)
	if err != nil {
		for _, x := range a.fallback {
			err = nil
			err = x.Send(m)
			if err != nil {
				continue
			}
		}
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
