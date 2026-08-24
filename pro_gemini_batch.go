package gpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"google.golang.org/genai"
)

func newGeminiBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	a := &geminiBatch{
		batchBase: batchBase{
			name:    responseFormatName(name),
			handler: handler,
		},
		engine:       engine,
		timeout:      24 * time.Hour,
		pollInterval: time.Minute,
	}

	if g, ok := engine.(*gemini); ok && g != nil && g.conn != nil {
		a.conn = g.conn
	} else {
		conn, err := genai.NewClient(context.Background(), &genai.ClientConfig{
			APIKey:  engine.getToken(),
			Backend: genai.BackendGeminiAPI,
		})
		if err == nil {
			a.conn = conn
		}
	}

	return a
}

type geminiBatch struct {
	batchBase

	conn   *genai.Client
	engine Engine

	timeout      time.Duration
	pollInterval time.Duration
}

func (a *geminiBatch) Push() error {
	results, err := a.pushResults(context.Background())
	if err != nil {
		return err
	}

	if a.handler != nil && len(results) > 0 {
		a.handler(results)
	}

	return nil
}

func (a *geminiBatch) pushResults(ctx context.Context) (results []BatchResult, err error) {
	list := a.takeMessages()
	if len(list) == 0 {
		return nil, nil
	}
	defer func() {
		if err != nil {
			a.returnMessages(list)
		}
	}()

	if a.conn == nil {
		return nil, errors.New("gemini batch: nil client")
	}

	model := a.engine.Model()
	requests, customIDs, lookup, err := a.renderRequests(list, model)
	if err != nil {
		return nil, err
	}

	job, err := a.conn.Batches.Create(
		ctx,
		geminiBatchModel(model.ID),
		&genai.BatchJobSource{InlinedRequests: requests},
		&genai.CreateBatchJobConfig{DisplayName: a.name},
	)
	if err != nil {
		return nil, err
	}

	job, err = a.wait(ctx, job.Name)
	if err != nil {
		return nil, err
	}

	return a.batchResults(job, customIDs, lookup)
}

func (a *geminiBatch) renderRequests(messages []*Message, model *Model) ([]*genai.InlinedRequest, []string, map[string]*Message, error) {
	if model == nil || model.ID == "" {
		return nil, nil, nil, errors.New("gemini batch: empty model")
	}

	requests := make([]*genai.InlinedRequest, 0, len(messages))
	customIDs := make([]string, 0, len(messages))
	lookup := make(map[string]*Message, len(messages))

	for i, m := range messages {
		if m == nil || m.id == nil {
			continue
		}

		if m.name == "" {
			m.name = fmt.Sprintf("%s_%d", a.name, i+1)
		}
		if m.result != nil {
			applySystemPromtStruct(m)
		}

		customID := a.customID(i, m)
		parts, err := a.batchParts(m)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(parts) == 0 {
			return nil, nil, nil, fmt.Errorf("gemini batch item %s empty request", customID)
		}

		requests = append(requests, &genai.InlinedRequest{
			Contents: []*genai.Content{{
				Parts: parts,
				Role:  genai.RoleUser,
			}},
			Metadata: map[string]string{
				"custom_id": customID,
			},
			Config: geminiBatchConfig(m),
		})
		customIDs = append(customIDs, customID)
		lookup[customID] = m
	}

	if len(requests) == 0 {
		return nil, nil, nil, errors.New("gemini batch: no valid messages")
	}

	return requests, customIDs, lookup, nil
}

func (a *geminiBatch) batchParts(m *Message) ([]*genai.Part, error) {
	parts := make([]*genai.Part, 0, len(m.files)+len(m.imagefiles)+len(m.fileurls)+len(m.imageurls)+len(m.images)+1)

	if prompt := strings.TrimSpace(m.RenderPromt()); prompt != "" {
		parts = append(parts, &genai.Part{Text: prompt})
	}

	// files uploaded before (UploadFile)
	fileparts, err := geminiFileParts(context.Background(), a.conn, a.engine.getToken(), m)
	if err != nil {
		return nil, err
	}
	parts = append(parts, fileparts...)

	// links by AddFileURL are downloaded, gemini has no remote link input
	urlparts, err := geminiURLParts(m)
	if err != nil {
		return nil, err
	}
	parts = append(parts, urlparts...)

	for _, img := range m.images {
		if img == nil {
			continue
		}
		parts = append(parts, &genai.Part{InlineData: &genai.Blob{
			Data:     img.JPG(80).Bytes(),
			MIMEType: "image/jpeg",
		}})
	}

	return parts, nil
}

func geminiBatchConfig(m *Message) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if m.temperature > 0 {
		temperature := float32(m.temperature)
		config.Temperature = &temperature
	}

	if m.schema != nil {
		config.ResponseMIMEType = "application/json"
		config.ResponseJsonSchema = m.schema
	}

	if system := strings.TrimSpace(m.RenderSystemPromt()); system != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: system}},
		}
	}

	return config
}

func (a *geminiBatch) customID(index int, m *Message) string {
	id := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(fmt.Sprint(m.id))
	if id == "" {
		id = fmt.Sprintf("%d", index+1)
	}

	return fmt.Sprintf("%s-%06d-%s", a.name, index+1, id)
}

func geminiBatchModel(name string) string {
	if strings.HasPrefix(name, "models/") {
		return name
	}
	return "models/" + name
}

func (a *geminiBatch) wait(ctx context.Context, name string) (*genai.BatchJob, error) {
	timer := time.NewTimer(a.timeout)
	defer timer.Stop()

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		job, err := a.conn.Batches.Get(ctx, name, nil)
		if err != nil {
			return nil, err
		}

		switch job.State {
		case genai.JobStateSucceeded, genai.JobStatePartiallySucceeded:
			return job, nil
		case genai.JobStateFailed, genai.JobStateCancelled, genai.JobStateExpired:
			if job.Error != nil {
				return nil, fmt.Errorf("gemini batch %s: %s", job.State, job.Error.Message)
			}
			return nil, fmt.Errorf("gemini batch %s", job.State)
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, errors.New("gemini batch wait timeout")
		case <-ticker.C:
		}
	}
}

func (a *geminiBatch) batchResults(job *genai.BatchJob, customIDs []string, requests map[string]*Message) ([]BatchResult, error) {
	if job == nil || job.Dest == nil {
		return nil, errors.New("gemini batch: empty destination")
	}

	rows := job.Dest.InlinedResponses
	results := make([]BatchResult, 0, len(requests))
	seen := make(map[string]bool, len(requests))
	model := a.engine.Model()

	for i, row := range rows {
		customID := ""
		if row != nil && row.Metadata != nil {
			customID = row.Metadata["custom_id"]
		}
		if customID == "" && i < len(customIDs) {
			customID = customIDs[i]
		}

		m := requests[customID]
		if m == nil {
			return nil, fmt.Errorf("gemini batch item %s has no matching message", customID)
		}
		if seen[customID] {
			return nil, fmt.Errorf("gemini batch item %s returned twice", customID)
		}
		seen[customID] = true

		if row == nil {
			return nil, fmt.Errorf("gemini batch item %s empty row", customID)
		}
		if row.Error != nil {
			return nil, fmt.Errorf("gemini batch item %s error: %s", customID, row.Error.Message)
		}
		if row.Response == nil {
			return nil, fmt.Errorf("gemini batch item %s empty response", customID)
		}

		text := row.Response.Text()
		if text == "" {
			return nil, fmt.Errorf("gemini batch item %s empty text", customID)
		}

		clean := json.RawMessage(CleanMarkdown(text))
		if len(m.schema) > 0 || m.result != nil {
			if !json.Valid(clean) {
				return nil, fmt.Errorf("gemini batch item %s invalid JSON: %s", customID, string(clean))
			}
			if m.result != nil {
				if err := jsoniter.Unmarshal(clean, m.result); err != nil {
					return nil, fmt.Errorf("gemini batch item %s result parse: %w", customID, err)
				}
			}
		}

		usage := row.Response.UsageMetadata
		m.resp = row.Response
		m.raw = text
		m.clean = string(clean)
		m.provider = ProviderGemini
		m.summary = Summary{
			Chat:       row.Response.ResponseID,
			Model:      model,
			Promt:      len(m.RenderPromt()),
			System:     len(m.RenderSystemPromt()),
			Images:     m.imagesSize(),
			PromtText:  m.RenderPromt(),
			SystemText: m.RenderSystemPromt(),
			RespText:   text,
		}
		if usage != nil {
			m.summary.Input = int(usage.PromptTokenCount)
			m.summary.Cached = int(usage.CachedContentTokenCount)
			m.summary.Output = int(usage.CandidatesTokenCount)
			m.summary.Resoning = int(usage.ThoughtsTokenCount)
			m.summary.Total = int(usage.TotalTokenCount)
		}
		m.summary.calc()
		m.summary.InputPrice *= 0.5
		m.summary.CachedPrice *= 0.5
		m.summary.OutputPrice *= 0.5
		m.summary.Price *= 0.5
		m.price = m.summary.Price

		results = append(results, BatchResult{
			ID:       m.id,
			CustomID: customID,
			Result:   clean,
			Message:  m,
		})
	}

	for customID := range requests {
		if !seen[customID] {
			return nil, fmt.Errorf("gemini batch item %s missing from output", customID)
		}
	}

	return results, nil
}
