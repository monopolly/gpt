package gpt

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/genai"
)

// x.ai
func Gemini(token string, model ...*Model) (a Engine) {
	p := &gemini{
		model: &Model{},
		token: token,
	}

	switch len(model) > 0 {
	case true:
		p.model = model[0]
	default:
		p.model = &Model_Gemini_2_5_Flash_x0_2
	}

	p.provider = ProviderGemini
	conn, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  token,
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		a = p
		log.Println("gemini error", err)
		return
	}

	p.conn = conn
	return p
}

type gemini struct {
	conn     *genai.Client
	provider Provider
	model    *Model
	token    string
}

func (a *gemini) Provider() Provider {
	return a.provider
}

func (a *gemini) getToken() string {
	return a.token
}

func (a *gemini) Model(v ...*Model) *Model {
	switch v {
	case nil:
		return a.model
	default:
		a.model = v[0]
		return nil
	}
}

func (a *gemini) Chat(m *Message) (res string, err error) {
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

func (a *gemini) Send(m *Message) (err error) {
	if m.name == "" {
		m.name = "request"
	}
	if m.result != nil {
		applySystemPromtStruct(m)
	}
	t1 := time.Now()
	m.provider = a.provider

	var config genai.GenerateContentConfig

	// temperature
	if m.temperature > 0 {
		temperature := float32(m.temperature)
		config.Temperature = &temperature
	}

	// schema
	if m.schema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseJsonSchema = m.schema
	}

	// system prompt
	if systemPrompt := strings.TrimSpace(m.RenderSystemPromt()); systemPrompt != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{
				{Text: systemPrompt},
			},
		}
	}

	// user parts
	userparts := make([]*genai.Part, 0)

	if prompt := strings.TrimSpace(m.RenderPromt()); prompt != "" {
		userparts = append(userparts, &genai.Part{
			Text: prompt,
		})
	}

	for _, part := range a.images(m) {
		if part == nil {
			continue
		}

		// Не добавляем полностью пустой Part,
		// иначе Gemini вернёт:
		// required oneof field 'data' must have one initialized field
		if strings.TrimSpace(part.Text) == "" &&
			part.InlineData == nil &&
			part.FileData == nil &&
			part.FunctionCall == nil &&
			part.FunctionResponse == nil {
			continue
		}

		userparts = append(userparts, part)
	}

	if len(userparts) == 0 {
		return fmt.Errorf("gemini request is empty: prompt and images are empty")
	}

	resp, err := a.conn.Models.GenerateContent(
		context.Background(),
		a.model.Name,
		[]*genai.Content{
			{
				Parts: userparts,
				Role:  genai.RoleUser,
			},
		},
		&config,
	)
	if err != nil {
		return err
	}

	m.raw = CleanMarkdown(resp.Text())

	if m.schema != nil {
		_ = json.Unmarshal([]byte(m.raw), &m.result)
	}

	m.summary = Summary{
		Chat:   resp.ResponseID,
		Model:  a.model,
		Promt:  len(m.RenderPromt()),
		System: len(m.RenderSystemPromt()),
		Images: m.imagesSize(),
		Input:  int(resp.UsageMetadata.PromptTokenCount),
		Cached: int(resp.UsageMetadata.CachedContentTokenCount),
		Output: int(resp.UsageMetadata.ThoughtsTokenCount),
		Total:  int(resp.UsageMetadata.TotalTokenCount),
	}

	m.summary.calc()
	m.summary.Time = time.Since(t1)
	m.summary.Times = m.summary.Time.String()

	return nil
}

// convert to gemini image blobs
func (a *gemini) images(m *Message) (parts []*genai.Part) {

	// images
	for _, x := range m.images {
		p := genai.Part{InlineData: &genai.Blob{
			Data:     x.JPG(80).Bytes(),
			MIMEType: "image/jpeg",
		}}
		parts = append(parts, &p)
	}
	return
}
