package gpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/openai/openai-go/v3/responses"
)

const xAIBatchAPIBase = "https://api.x.ai/v1"

func newGrokBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	return &grokBatch{
		batchBase: batchBase{
			name:    responseFormatName(name),
			handler: handler,
		},
		engine:       engine,
		token:        engine.getToken(),
		timeout:      24 * time.Hour,
		pollInterval: time.Minute,
	}
}

type grokBatch struct {
	batchBase

	engine Engine
	token  string

	timeout      time.Duration
	pollInterval time.Duration
}

func (a *grokBatch) Push() error {
	results, err := a.pushResults(context.Background())
	if err != nil {
		return err
	}

	if a.handler != nil && len(results) > 0 {
		a.handler(results)
	}

	return nil
}

func (a *grokBatch) pushResults(ctx context.Context) (results []BatchResult, err error) {
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

	batchID, err := a.batchCreate(ctx)
	if err != nil {
		return nil, err
	}

	if err = a.batchAddRequests(ctx, batchID, requests); err != nil {
		return nil, err
	}

	if err = a.batchWait(ctx, batchID, a.timeout); err != nil {
		return nil, err
	}

	rows, err := a.batchResultsRows(ctx, batchID)
	if err != nil {
		return nil, err
	}

	return a.batchResults(rows, lookup)
}

func (a *grokBatch) renderRequests(messages []*Message) ([]map[string]any, map[string]*Message, error) {
	model := a.engine.Model()
	if model == nil || model.Name == "" {
		return nil, nil, errors.New("grok batch: empty model")
	}

	requests := make([]map[string]any, 0, len(messages))
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
		requests = append(requests, map[string]any{
			"batch_request_id": customID,
			"batch_request": map[string]any{
				"responses": a.messageBody(model, m),
			},
		})
		lookup[customID] = m
	}

	if len(requests) == 0 {
		return nil, nil, errors.New("grok batch: no valid messages")
	}

	return requests, lookup, nil
}

func (a *grokBatch) messageBody(model *Model, m *Message) map[string]any {
	body := map[string]any{
		"model": model.Name,
		"input": batchInput(m),
	}

	if system := m.RenderSystemPromt(); system != "" {
		body["instructions"] = system
	}

	if len(m.schema) > 0 {
		body["text"] = map[string]any{
			"format": map[string]any{
				"type":        "json_schema",
				"name":        responseFormatName(m.name),
				"schema":      m.schema,
				"strict":      true,
				"description": "Return only this JSON object.",
			},
		}
	}

	return body
}

func (a *grokBatch) customID(index int, m *Message) string {
	id := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(fmt.Sprint(m.id))
	if id == "" {
		id = fmt.Sprintf("%d", index+1)
	}

	return fmt.Sprintf("%s-%06d-%s", a.name, index+1, id)
}

func (a *grokBatch) batchCreate(ctx context.Context) (string, error) {
	var out struct {
		BatchID string `json:"batch_id"`
		ID      string `json:"id"`
	}

	err := a.doJSON(ctx, http.MethodPost, xAIBatchAPIBase+"/batches", map[string]any{
		"name": a.name,
	}, &out)
	if err != nil {
		return "", err
	}

	if out.BatchID != "" {
		return out.BatchID, nil
	}
	if out.ID != "" {
		return out.ID, nil
	}

	return "", errors.New("grok batch create: empty batch id")
}

func (a *grokBatch) batchAddRequests(ctx context.Context, batchID string, requests []map[string]any) error {
	return a.doJSON(ctx, http.MethodPost, xAIBatchAPIBase+"/batches/"+batchID+"/requests", map[string]any{
		"batch_requests": requests,
	}, nil)
}

func (a *grokBatch) batchWait(ctx context.Context, batchID string, timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(a.pollInterval)
	defer ticker.Stop()

	for {
		state, err := a.batchState(ctx, batchID)
		if err != nil {
			return err
		}

		if state.NumPending == 0 {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return errors.New("grok batch wait timeout")
		case <-ticker.C:
		}
	}
}

func (a *grokBatch) batchState(ctx context.Context, batchID string) (grokBatchState, error) {
	var out struct {
		State grokBatchState `json:"state"`
	}

	err := a.doJSON(ctx, http.MethodGet, xAIBatchAPIBase+"/batches/"+batchID, nil, &out)
	return out.State, err
}

type grokBatchState struct {
	NumPending int `json:"num_pending"`
}

func (a *grokBatch) batchResultsRows(ctx context.Context, batchID string) ([]grokBatchResultRow, error) {
	var rows []grokBatchResultRow
	next := ""

	for {
		u := xAIBatchAPIBase + "/batches/" + batchID + "/results?limit=100"
		if next != "" {
			u += "&pagination_token=" + url.QueryEscape(next)
		}

		var out struct {
			Results              []grokBatchResultRow `json:"results"`
			NextPaginationToken  string               `json:"next_pagination_token"`
			NextPageToken        string               `json:"next_page_token"`
			PaginationToken      string               `json:"pagination_token"`
			NextCursor           string               `json:"next_cursor"`
			HasMore              bool                 `json:"has_more"`
			NextPaginationCursor string               `json:"next_pagination_cursor"`
		}

		if err := a.doJSON(ctx, http.MethodGet, u, nil, &out); err != nil {
			return nil, err
		}

		rows = append(rows, out.Results...)
		next = firstNonEmpty(out.NextPaginationToken, out.NextPageToken, out.PaginationToken, out.NextCursor, out.NextPaginationCursor)
		if next == "" || !out.HasMore && out.NextPaginationToken == "" && out.NextPageToken == "" && out.NextCursor == "" && out.NextPaginationCursor == "" {
			break
		}
	}

	return rows, nil
}

type grokBatchResultRow struct {
	BatchRequestID string `json:"batch_request_id"`
	ErrorMessage   string `json:"error_message"`
	Error          any    `json:"error"`
	BatchResult    struct {
		Response json.RawMessage `json:"response"`
	} `json:"batch_result"`
}

func (a *grokBatch) batchResults(rows []grokBatchResultRow, requests map[string]*Message) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(requests))
	seen := make(map[string]bool, len(requests))
	model := a.engine.Model()

	for _, row := range rows {
		m := requests[row.BatchRequestID]
		if m == nil {
			return nil, fmt.Errorf("grok batch item %s has no matching message", row.BatchRequestID)
		}
		if seen[row.BatchRequestID] {
			return nil, fmt.Errorf("grok batch item %s returned twice", row.BatchRequestID)
		}
		seen[row.BatchRequestID] = true

		if row.ErrorMessage != "" {
			return nil, fmt.Errorf("grok batch item %s error: %s", row.BatchRequestID, row.ErrorMessage)
		}
		if row.Error != nil {
			return nil, fmt.Errorf("grok batch item %s error: %v", row.BatchRequestID, row.Error)
		}
		if len(row.BatchResult.Response) == 0 {
			return nil, fmt.Errorf("grok batch item %s empty response", row.BatchRequestID)
		}

		text, resp, usage, err := grokBatchText(row.BatchResult.Response)
		if err != nil {
			return nil, fmt.Errorf("grok batch item %s response parse: %w", row.BatchRequestID, err)
		}
		if text == "" {
			return nil, fmt.Errorf("grok batch item %s empty text", row.BatchRequestID)
		}

		clean := json.RawMessage(CleanMarkdown(text))
		if len(m.schema) > 0 || m.result != nil {
			if !json.Valid(clean) {
				return nil, fmt.Errorf("grok batch item %s invalid JSON: %s", row.BatchRequestID, string(clean))
			}
			if m.result != nil {
				if err := jsoniter.Unmarshal(clean, m.result); err != nil {
					return nil, fmt.Errorf("grok batch item %s result parse: %w", row.BatchRequestID, err)
				}
			}
		}

		m.resp = resp
		m.raw = text
		m.clean = string(clean)
		m.provider = ProviderGrok
		m.summary = Summary{
			Chat:       grokResponseID(row.BatchResult.Response),
			Model:      model,
			Promt:      len(m.RenderPromt()),
			System:     len(m.RenderSystemPromt()),
			Images:     m.imagesSize(),
			Input:      usage.Input,
			Cached:     usage.Cached,
			Output:     usage.Output,
			Total:      usage.Total,
			Resoning:   usage.Reasoning,
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
			CustomID: row.BatchRequestID,
			Result:   clean,
			Message:  m,
		})
	}

	for customID := range requests {
		if !seen[customID] {
			return nil, fmt.Errorf("grok batch item %s missing from output", customID)
		}
	}

	return results, nil
}

type grokUsage struct {
	Input     int
	Output    int
	Reasoning int
	Cached    int
	Total     int
}

func grokBatchText(raw json.RawMessage) (string, any, grokUsage, error) {
	var wrapper struct {
		ChatGetCompletion json.RawMessage `json:"chat_get_completion"`
		Responses         json.RawMessage `json:"responses"`
		Response          json.RawMessage `json:"response"`
	}
	_ = json.Unmarshal(raw, &wrapper)

	body := firstRaw(wrapper.Responses, wrapper.Response, wrapper.ChatGetCompletion, raw)

	var resp responses.Response
	if err := json.Unmarshal(body, &resp); err == nil {
		text := resp.OutputText()
		if text == "" {
			text = outputTextFromRawResponse(body)
		}
		if text != "" {
			return text, &resp, grokUsage{
				Input:     int(resp.Usage.InputTokens),
				Cached:    int(resp.Usage.InputTokensDetails.CachedTokens),
				Output:    int(resp.Usage.OutputTokens),
				Reasoning: int(resp.Usage.OutputTokensDetails.ReasoningTokens),
				Total:     int(resp.Usage.TotalTokens),
			}, nil
		}
	}

	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens        int `json:"prompt_tokens"`
			CompletionTokens    int `json:"completion_tokens"`
			TotalTokens         int `json:"total_tokens"`
			PromptTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &chat); err != nil {
		return "", nil, grokUsage{}, err
	}
	if len(chat.Choices) == 0 {
		return "", nil, grokUsage{}, nil
	}

	return chat.Choices[0].Message.Content, chat, grokUsage{
		Input:     chat.Usage.PromptTokens,
		Output:    chat.Usage.CompletionTokens,
		Reasoning: chat.Usage.CompletionTokensDetails.ReasoningTokens,
		Cached:    chat.Usage.PromptTokensDetails.CachedTokens,
		Total:     chat.Usage.TotalTokens,
	}, nil
}

func grokResponseID(raw json.RawMessage) string {
	var wrapper struct {
		ChatGetCompletion json.RawMessage `json:"chat_get_completion"`
		Responses         json.RawMessage `json:"responses"`
		Response          json.RawMessage `json:"response"`
	}
	_ = json.Unmarshal(raw, &wrapper)

	var body struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(firstRaw(wrapper.Responses, wrapper.Response, wrapper.ChatGetCompletion, raw), &body)
	return body.ID
}

func firstRaw(values ...json.RawMessage) json.RawMessage {
	for _, v := range values {
		if len(v) > 0 && string(v) != "null" {
			return v
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (a *grokBatch) doJSON(ctx context.Context, method string, endpoint string, payload any, out any) error {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("grok batch %s %s failed: %s", method, endpoint, string(raw))
	}

	if out == nil {
		return nil
	}

	return json.Unmarshal(raw, out)
}
