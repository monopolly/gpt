package gpt

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	anthropicoption "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	jsoniter "github.com/json-iterator/go"
	"github.com/monopolly/images"
)

type claudeBatch struct {
	batchBase

	conn   anthropic.Client
	engine Engine

	timeout      time.Duration
	pollInterval time.Duration
}

func newClaudeBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	a := &claudeBatch{
		batchBase: batchBase{
			name:    responseFormatName(name),
			handler: handler,
		},
		engine:       engine,
		timeout:      24 * time.Hour,
		pollInterval: time.Minute,
	}

	if c, ok := engine.(*claude); ok && c != nil {
		a.conn = c.conn
	} else {
		a.conn = anthropic.NewClient(anthropicoption.WithAPIKey(engine.getToken()))
	}

	return a
}

func (a *claudeBatch) Push() error {
	results, err := a.pushResults(context.Background())
	if err != nil {
		return err
	}

	if a.handler != nil && len(results) > 0 {
		a.handler(results)
	}

	return nil
}

func (a *claudeBatch) pushResults(ctx context.Context) (results []BatchResult, err error) {
	list := a.takeMessages()
	if len(list) == 0 {
		return nil, nil
	}
	defer func() {
		if err != nil {
			a.returnMessages(list)
		}
	}()

	requests, lookup, err := a.renderRequests(list)
	if err != nil {
		return nil, err
	}

	batch, err := a.conn.Messages.Batches.New(ctx, anthropic.MessageBatchNewParams{
		Requests: requests,
	}, claudeBatchOptions(list)...)
	if err != nil {
		return nil, err
	}

	batch, err = a.wait(ctx, batch.ID)
	if err != nil {
		return nil, err
	}

	rows, err := a.results(ctx, batch.ID)
	if err != nil {
		return nil, err
	}

	return a.batchResults(rows, lookup)
}

func (a *claudeBatch) renderRequests(messages []*Message) ([]anthropic.MessageBatchNewParamsRequest, map[string]*Message, error) {
	model := a.engine.Model()
	if model == nil || model.ID == "" {
		return nil, nil, fmt.Errorf("claude batch: empty model")
	}

	requests := make([]anthropic.MessageBatchNewParamsRequest, 0, len(messages))
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
		params := a.messageParams(model, m)

		requests = append(requests, anthropic.MessageBatchNewParamsRequest{
			CustomID: customID,
			Params:   params,
		})
		lookup[customID] = m
	}

	if len(requests) == 0 {
		return nil, nil, fmt.Errorf("claude batch: no valid messages")
	}

	return requests, lookup, nil
}

func (a *claudeBatch) messageParams(model *Model, m *Message) anthropic.MessageBatchNewParamsRequestParams {
	params := anthropic.MessageBatchNewParamsRequestParams{
		Model:     anthropic.Model(model.ID),
		MaxTokens: claudeMaxTokens(model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(claudeBlocks(m)...),
		},
	}

	if system := m.RenderSystemPromt(); system != "" {
		params.System = []anthropic.TextBlockParam{{Text: system}}
	}

	if len(m.schema) > 0 {
		params.OutputConfig = anthropic.OutputConfigParam{
			Format: anthropic.JSONOutputFormatParam{
				Schema: m.schema,
			},
		}
	}

	return params
}

// user content: uploaded files, uploaded images, inline images, promt
func claudeBlocks(m *Message) []anthropic.ContentBlockParamUnion {
	blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.files)+len(m.imagefiles)+len(m.fileurls)+len(m.imageurls)+len(m.images)+1)

	// documents uploaded before (UploadFile)
	for _, id := range m.files {
		if id == "" {
			continue
		}
		blocks = append(blocks, claudeFileBlock("document", id))
	}

	// images uploaded before (UploadFile)
	for _, id := range m.imagefiles {
		if id == "" {
			continue
		}
		blocks = append(blocks, claudeFileBlock("image", id))
	}

	// documents by a public link (AddFileURL), pdf only
	for _, u := range m.fileurls {
		if ext := urlExt(u); ext != "" && ext != "pdf" {
			log.Printf("claude: only pdf links are supported as documents, skip %s", u)
			continue
		}
		blocks = append(blocks, anthropic.NewDocumentBlock(anthropic.URLPDFSourceParam{URL: u}))
	}

	// images by a public link (AddFileURL, AddImageURL)
	for _, u := range m.imageurls {
		blocks = append(blocks, anthropic.NewImageBlock(anthropic.URLImageSourceParam{URL: u}))
	}

	for _, img := range m.images {
		if img == nil {
			continue
		}
		blocks = append(blocks, anthropic.NewImageBlockBase64("image/jpeg", claudeImageBase64JPEG(img)))
	}

	blocks = append(blocks, anthropic.NewTextBlock(m.RenderPromt()))
	return blocks
}

// file id blocks are files api (beta) only, the sdk has no typed source for them
func claudeFileBlock(kind, fileID string) anthropic.ContentBlockParamUnion {
	raw, _ := json.Marshal(map[string]any{
		"type": kind,
		"source": map[string]any{
			"type":    "file",
			"file_id": fileID,
		},
	})

	return param.Override[anthropic.ContentBlockParamUnion](json.RawMessage(raw))
}

// files api requires the beta header on the messages request
func claudeOptions(m *Message) (opts []anthropicoption.RequestOption) {
	if m == nil || (len(m.files) == 0 && len(m.imagefiles) == 0) {
		return
	}

	return []anthropicoption.RequestOption{
		anthropicoption.WithHeaderAdd("anthropic-beta", string(anthropic.AnthropicBetaFilesAPI2025_04_14)),
	}
}

// files api beta header for every message of the batch
func claudeBatchOptions(list []*Message) (opts []anthropicoption.RequestOption) {
	for _, m := range list {
		if o := claudeOptions(m); o != nil {
			return o
		}
	}
	return
}

func claudeImageBase64JPEG(img *images.Image) string {
	return base64.StdEncoding.EncodeToString(img.JPG(80).Bytes())
}

func claudeMaxTokens(model *Model) int64 {
	if model != nil && model.Output > 0 {
		return int64(model.Output * 1000)
	}
	return 4096
}

func (a *claudeBatch) customID(index int, m *Message) string {
	id := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(fmt.Sprint(m.id))
	if id == "" {
		id = fmt.Sprintf("%d", index+1)
	}

	return fmt.Sprintf("%s-%06d-%s", a.name, index+1, id)
}

func (a *claudeBatch) wait(ctx context.Context, batchID string) (*anthropic.MessageBatch, error) {
	timer := time.NewTimer(a.timeout)
	defer timer.Stop()

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		batch, err := a.conn.Messages.Batches.Get(ctx, batchID)
		if err != nil {
			return nil, err
		}

		switch batch.ProcessingStatus {
		case anthropic.MessageBatchProcessingStatusEnded:
			return batch, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
			return nil, fmt.Errorf("claude batch wait timeout")
		case <-ticker.C:
		}
	}
}

func (a *claudeBatch) results(ctx context.Context, batchID string) ([]anthropic.MessageBatchIndividualResponse, error) {
	stream := a.conn.Messages.Batches.ResultsStreaming(ctx, batchID)
	if err := stream.Err(); err != nil {
		return nil, err
	}
	defer stream.Close()

	var rows []anthropic.MessageBatchIndividualResponse
	for stream.Next() {
		rows = append(rows, stream.Current())
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}

	return rows, nil
}

func (a *claudeBatch) batchResults(rows []anthropic.MessageBatchIndividualResponse, requests map[string]*Message) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(requests))
	seen := make(map[string]bool, len(requests))
	model := a.engine.Model()

	for _, row := range rows {
		m := requests[row.CustomID]
		if m == nil {
			return nil, fmt.Errorf("claude batch item %s has no matching message", row.CustomID)
		}
		if seen[row.CustomID] {
			return nil, fmt.Errorf("claude batch item %s returned twice", row.CustomID)
		}
		seen[row.CustomID] = true

		if row.Result.Type != "succeeded" {
			return nil, fmt.Errorf("claude batch item %s %s: %s", row.CustomID, row.Result.Type, row.Result.Error.Error.Message)
		}

		resp := row.Result.Message
		text := claudeMessageText(resp)
		if text == "" {
			return nil, fmt.Errorf("claude batch item %s empty response", row.CustomID)
		}

		clean := []byte(CleanMarkdown(text))
		if len(m.schema) > 0 || m.result != nil {
			if !json.Valid(clean) {
				return nil, fmt.Errorf("claude batch item %s invalid JSON: %s", row.CustomID, string(clean))
			}
			if m.result != nil {
				if err := jsoniter.Unmarshal(clean, m.result); err != nil {
					return nil, fmt.Errorf("claude batch item %s result parse: %w", row.CustomID, err)
				}
			}
		}

		m.resp = &resp
		m.raw = text
		m.clean = string(clean)
		m.provider = ProviderClaude
		m.summary = Summary{
			Chat:       resp.ID,
			Model:      model,
			Promt:      len(m.RenderPromt()),
			System:     len(m.RenderSystemPromt()),
			Images:     m.imagesSize(),
			Input:      int(resp.Usage.InputTokens),
			Cached:     int(resp.Usage.CacheReadInputTokens),
			Output:     int(resp.Usage.OutputTokens),
			Total:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
			PromtText:  m.RenderPromt(),
			SystemText: m.RenderSystemPromt(),
			RespText:   text,
		}
		m.summary.calc()
		m.summary.InputPrice *= 0.5
		m.summary.CachedPrice *= 0.5
		m.summary.OutputPrice *= 0.5
		m.summary.Price *= 0.5
		m.price = m.summary.Price

		results = append(results, BatchResult{
			ID:       m.id,
			CustomID: row.CustomID,
			Result:   clean,
			Message:  m,
		})
	}

	for customID := range requests {
		if !seen[customID] {
			return nil, fmt.Errorf("claude batch item %s missing from output", customID)
		}
	}

	return results, nil
}

func claudeMessageText(resp anthropic.Message) string {
	var parts []string
	for _, c := range resp.Content {
		if c.Text != "" {
			parts = append(parts, c.Text)
		}
	}

	return strings.Join(parts, "\n")
}
