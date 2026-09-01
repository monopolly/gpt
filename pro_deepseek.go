package gpt

import (
	"context"
	"errors"
	"time"

	"github.com/go-deepseek/deepseek"
	"github.com/go-deepseek/deepseek/request"
	jsoniter "github.com/json-iterator/go"
)

func Deepseek(token string, model ...*Model) (p Engine) {
	a := new(deep)
	a.conn, _ = deepseek.NewClient(token)

	switch len(model) > 0 {
	case true:
		a.model = model[0]
	default:
		a.model = &Model_DeepSeek_Chat
	}
	a.token = token
	a.provider = ProviderDeepseek
	return a
}

type deep struct {
	conn     deepseek.Client
	model    *Model
	provider Provider
	token    string

	fallback []Engine
}

func (a *deep) Provider() Provider {
	return a.provider
}

func (a *deep) Fallback(v ...Engine) Engine {
	a.fallback = append(a.fallback, v...)
	return a
}

// deepseek has no files api

func (a *deep) UploadFile(body []byte, filename ...string) (fileID string, err error) {
	return "", errors.New("deepseek files api is not supported")
}

// deepseek has no server side conversations
func (a *deep) NewConversation() (conversationID string, err error) {
	return "", errors.New("deepseek has no conversations api")
}

// deepseek has no server side conversations
func (a *deep) AddText(conversationID string, text ...string) (err error) {
	return errors.New("deepseek has no conversations api")
}

func (a *deep) AddFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("deepseek files api is not supported")
}

func (a *deep) AddImageFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("deepseek files api is not supported")
}

func (a *deep) DeleteFiles(filesID ...string) (err error) {
	return errors.New("deepseek files api is not supported")
}

func (a *deep) Model(v ...*Model) *Model {
	switch v {
	case nil:
		return a.model
	default:
		a.model = v[0]
		return nil
	}
}

func (a *deep) getToken() string {
	return a.token
}

func (a *deep) Chat(m *Message) (res string, err error) {
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

func (a *deep) Send(m *Message) (err error) {
	if m.name == "" {
		m.name = "request"
	}
	if m.result != nil {
		applySystemPromtStruct(m)
	}

	promt := m.RenderPromt()
	system := m.RenderSystemPromt()
	t1 := time.Now()

	req := &request.ChatCompletionsRequest{
		Model:    a.model.ID,
		Stream:   false,
		Messages: []*request.Message{{Role: "user", Content: promt}},
	}

	if m.result != nil {
		req.ResponseFormat = &request.ResponseFormat{Type: "json_object"}
	}

	if system != "" {
		req.Messages = append(req.Messages, &request.Message{Role: "system", Content: system})
	}

	if m.temperature > 0 {
		t := float32(m.temperature)
		req.Temperature = &t
	}

	resp, err := a.conn.CallChatCompletionsChat(context.Background(), req)
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

	m.raw = CompactText(resp.Choices[0].Message.Content)

	switch m.result {
	case nil:
	default:
		err = jsoniter.Unmarshal([]byte(CleanMarkdown(m.raw)), &m.result)
	}

	m.summary = Summary{
		Chat:       resp.Id,
		Model:      a.model,
		Promt:      len(promt),
		System:     len(system),
		Images:     m.imagesSize(),
		Input:      resp.Usage.PromptTokens,
		Cached:     resp.Usage.PromptCacheHitTokens,
		Output:     resp.Usage.CompletionTokens,
		Total:      resp.Usage.TotalTokens,
		Resoning:   resp.Usage.CompletionTokensDetails.ReasoningTokens,
		SystemText: system,
		PromtText:  promt,
		RespText:   m.raw,
	}
	m.summary.calc()
	m.summary.Time = time.Since(t1)
	m.summary.Times = m.summary.Time.String()
	m.price = m.summary.Price
	return
}
