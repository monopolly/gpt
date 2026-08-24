package gpt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/openai/openai-go/v3/responses"
)

const openAIBatchResponsesEndpoint = "/v1/responses"

func newGPTBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	return &gptBatch{
		batchBase: batchBase{
			name:    responseFormatName(name),
			handler: handler,
		},
		engine:       engine,
		timeout:      24 * time.Hour,
		pollInterval: time.Minute,
		token:        engine.getToken(),
	}
}

type gptBatch struct {
	batchBase

	engine Engine
	token  string

	timeout      time.Duration
	pollInterval time.Duration
}

func (a *gptBatch) Push() error {
	results, err := a.pushResults(context.Background())
	if err != nil {
		return err
	}

	if a.handler != nil && len(results) > 0 {
		a.handler(results)
	}

	return nil
}

func (a *gptBatch) pushResults(ctx context.Context) (results []BatchResult, err error) {
	list := a.takeMessages()
	if len(list) == 0 {
		return nil, nil
	}
	defer func() {
		if err != nil {
			a.returnMessages(list)
		}
	}()

	jsonl, requests, err := a.renderJSONL(ctx, list)
	if err != nil {
		return nil, err
	}

	fileID, err := a.batchUploadFile(ctx, jsonl)
	if err != nil {
		return nil, err
	}

	batchID, err := a.batchCreate(ctx, fileID)
	if err != nil {
		return nil, err
	}

	outputFileID, errorFileID, err := a.batchWait(ctx, batchID, a.timeout)
	if err != nil {
		return nil, err
	}

	var rows []batchResponseRow

	if outputFileID != "" {
		data, err := a.batchDownloadFile(ctx, outputFileID)
		if err != nil {
			return nil, err
		}

		parsed, err := parseBatchResponseRows(data)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed...)
	}

	if errorFileID != "" {
		data, err := a.batchDownloadFile(ctx, errorFileID)
		if err != nil {
			return nil, err
		}

		parsed, err := parseBatchResponseRows(data)
		if err != nil {
			return nil, err
		}
		rows = append(rows, parsed...)
	}

	return a.batchResults(rows, requests)
}

func (a *gptBatch) renderJSONL(ctx context.Context, messages []*Message) ([]byte, map[string]*Message, error) {
	model := a.engine.Model()
	if model == nil || model.ID == "" {
		return nil, nil, errors.New("openai batch: empty model")
	}

	var jsonl bytes.Buffer
	requests := make(map[string]*Message, len(messages))

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
		if err := a.ensureConversation(ctx, m); err != nil {
			return nil, nil, err
		}

		customID := a.customID(i, m)
		body := a.messageBody(model, m)
		req := map[string]any{
			"custom_id": customID,
			"method":    "POST",
			"url":       openAIBatchResponsesEndpoint,
			"body":      body,
		}

		b, err := jsoniter.Marshal(req)
		if err != nil {
			return nil, nil, fmt.Errorf("openai batch marshal %s: %w", customID, err)
		}

		jsonl.Write(b)
		jsonl.WriteByte('\n')
		requests[customID] = m
	}

	if len(requests) == 0 {
		return nil, nil, errors.New("openai batch: no valid messages")
	}

	return jsonl.Bytes(), requests, nil
}

func (a *gptBatch) messageBody(model *Model, m *Message) map[string]any {
	body := map[string]any{
		"model": model.ID,
		"input": batchInput(m),
	}

	if m.store {
		body["store"] = true
	}

	if system := m.RenderSystemPromt(); system != "" {
		body["instructions"] = system
	}

	if m.chat != "" {
		body["conversation"] = m.chat
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

	if model.WebSearch {
		if tools := batchWebSearchTools(m); len(tools) > 0 {
			body["tools"] = tools
		}
	}

	return body
}

func (a *gptBatch) ensureConversation(ctx context.Context, m *Message) error {
	if m.chat != "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/conversations", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("openai conversation create failed: %s", string(raw))
	}

	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return err
	}
	if out.ID == "" {
		return errors.New("openai conversation create: empty id")
	}

	m.chat = out.ID
	return nil
}

func batchInput(m *Message) []map[string]any {
	content := make([]map[string]any, 0, len(m.files)+len(m.imagefiles)+len(m.images)+1)

	// documents uploaded before (UploadFile)
	for _, id := range m.files {
		if id == "" {
			continue
		}

		content = append(content, map[string]any{
			"type":    "input_file",
			"file_id": id,
		})
	}

	// images uploaded before (UploadFile)
	for _, id := range m.imagefiles {
		if id == "" {
			continue
		}

		content = append(content, map[string]any{
			"type":    "input_image",
			"file_id": id,
			"detail":  "auto",
		})
	}

	for _, img := range m.images {
		if img == nil {
			continue
		}

		content = append(content, map[string]any{
			"type":      "input_image",
			"image_url": img.JPGBase64HTML(80),
			"detail":    "auto",
		})
	}

	if promt := m.RenderPromt(); promt != "" || len(content) == 0 {
		content = append(content, map[string]any{
			"type": "input_text",
			"text": promt,
		})
	}

	return []map[string]any{
		{
			"role":    "user",
			"content": content,
		},
	}
}

func batchWebSearchTools(m *Message) []map[string]any {
	if !m.websearch {
		return nil
	}

	tool := map[string]any{
		"type":                "web_search",
		"search_context_size": "medium",
	}

	if len(m.domains) > 0 {
		tool["filters"] = map[string]any{
			"allowed_domains": m.domains,
		}
	}

	location := map[string]any{}
	if m.country != "" {
		location["country"] = m.country
	}
	if m.city != "" {
		location["city"] = m.city
	}
	if m.region != "" {
		location["region"] = m.region
	}
	if len(location) > 0 {
		location["type"] = "approximate"
		tool["user_location"] = location
	}

	return []map[string]any{tool}
}

func (a *gptBatch) customID(index int, m *Message) string {
	id := strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(fmt.Sprint(m.id))
	if id == "" {
		id = fmt.Sprintf("%d", index+1)
	}

	return fmt.Sprintf("%s-%06d-%s", a.name, index+1, id)
}

func responseFormatName(v string) string {
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	res := b.String()
	if len(res) > 64 {
		res = res[:64]
	}

	return res
}

func (a *gptBatch) batchUploadFile(ctx context.Context, data []byte) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("purpose", "batch"); err != nil {
		return "", err
	}

	fw, err := w.CreateFormFile("file", "batch.jsonl")
	if err != nil {
		return "", err
	}

	if _, err = fw.Write(data); err != nil {
		return "", err
	}

	if err = w.Close(); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/files", &body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai batch upload failed: %s", string(raw))
	}

	var out struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}

	if out.ID == "" {
		return "", errors.New("openai batch upload: empty file id")
	}

	return out.ID, nil
}

func (a *gptBatch) batchCreate(ctx context.Context, fileID string) (string, error) {
	payload := map[string]any{
		"input_file_id":     fileID,
		"endpoint":          openAIBatchResponsesEndpoint,
		"completion_window": "24h",
	}

	b, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/batches", bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai batch create failed: %s", string(raw))
	}

	var out struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return "", err
	}

	if out.ID == "" {
		return "", errors.New("openai batch create: empty batch id")
	}

	return out.ID, nil
}

func (a *gptBatch) batchWait(ctx context.Context, batchID string, timeout time.Duration) (outputFileID string, errorFileID string, err error) {
	if timeout <= 0 {
		timeout = 24 * time.Hour
	}
	poll := a.pollInterval
	if poll <= 0 {
		poll = 10 * time.Second
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		outputFileID, errorFileID, status, err := a.batchStatus(ctx, batchID)
		if err != nil {
			return "", "", err
		}

		switch status {
		case "completed":
			if outputFileID == "" && errorFileID == "" {
				return "", "", errors.New("openai batch completed without output_file_id or error_file_id")
			}
			return outputFileID, errorFileID, nil

		case "failed", "expired", "cancelled":
			return "", errorFileID, fmt.Errorf("openai batch %s, error_file_id: %s", status, errorFileID)
		}

		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		case <-timer.C:
			return "", "", errors.New("openai batch wait timeout")
		case <-ticker.C:
		}
	}
}

func (a *gptBatch) batchStatus(ctx context.Context, batchID string) (outputFileID string, errorFileID string, status string, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/batches/"+batchID, nil)
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return "", "", "", fmt.Errorf("openai batch status failed: %s", string(raw))
	}

	var out struct {
		Status       string `json:"status"`
		OutputFileID string `json:"output_file_id"`
		ErrorFileID  string `json:"error_file_id"`
	}

	if err := json.Unmarshal(raw, &out); err != nil {
		return "", "", "", err
	}

	return out.OutputFileID, out.ErrorFileID, out.Status, nil
}

func (a *gptBatch) batchDownloadFile(ctx context.Context, fileID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/files/"+fileID+"/content", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai batch download failed: %s", string(raw))
	}

	return raw, nil
}

type batchResponseRow struct {
	CustomID string `json:"custom_id"`
	Response *struct {
		StatusCode int             `json:"status_code"`
		RequestID  string          `json:"request_id"`
		Body       json.RawMessage `json:"body"`
	} `json:"response"`
	Error *batchResponseError `json:"error"`
}

type batchResponseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func parseBatchResponseRows(data []byte) ([]batchResponseRow, error) {
	var result []batchResponseRow

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024), 1024*1024*20)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var row batchResponseRow
		if err := json.Unmarshal(line, &row); err != nil {
			return nil, err
		}

		if row.CustomID == "" {
			return nil, errors.New("openai batch: empty custom_id in output")
		}

		result = append(result, row)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *gptBatch) batchResults(rows []batchResponseRow, requests map[string]*Message) ([]BatchResult, error) {
	results := make([]BatchResult, 0, len(requests))
	seen := make(map[string]bool, len(requests))
	model := a.engine.Model()

	for _, row := range rows {
		m := requests[row.CustomID]
		if m == nil {
			return nil, fmt.Errorf("openai batch item %s has no matching message", row.CustomID)
		}
		if seen[row.CustomID] {
			return nil, fmt.Errorf("openai batch item %s returned twice", row.CustomID)
		}
		seen[row.CustomID] = true

		if row.Error != nil {
			return nil, fmt.Errorf("openai batch item %s error: %s %s", row.CustomID, row.Error.Code, row.Error.Message)
		}
		if row.Response == nil {
			return nil, fmt.Errorf("openai batch item %s empty response", row.CustomID)
		}
		if row.Response.StatusCode >= 300 {
			return nil, fmt.Errorf("openai batch item %s status %d: %s", row.CustomID, row.Response.StatusCode, string(row.Response.Body))
		}

		var resp responses.Response
		if err := json.Unmarshal(row.Response.Body, &resp); err != nil {
			return nil, fmt.Errorf("openai batch item %s response parse: %w", row.CustomID, err)
		}

		text := resp.OutputText()
		if text == "" {
			text = outputTextFromRawResponse(row.Response.Body)
		}
		if text == "" {
			return nil, fmt.Errorf("openai batch item %s empty output_text", row.CustomID)
		}

		clean := []byte(CleanMarkdown(text))
		if len(m.schema) > 0 || m.result != nil {
			if !json.Valid(clean) {
				return nil, fmt.Errorf("openai batch item %s invalid JSON: %s", row.CustomID, string(clean))
			}
			if m.result != nil {
				if err := jsoniter.Unmarshal(clean, m.result); err != nil {
					return nil, fmt.Errorf("openai batch item %s result parse: %w", row.CustomID, err)
				}
			}
		}

		m.resp = &resp
		if resp.Conversation.ID != "" {
			m.chat = resp.Conversation.ID
		}
		m.raw = text
		m.clean = string(clean)
		m.summary = Summary{
			Chat:       m.chat,
			Model:      model,
			Promt:      len(m.RenderPromt()),
			System:     len(m.RenderSystemPromt()),
			Images:     m.imagesSize(),
			Input:      int(resp.Usage.InputTokens),
			Cached:     int(resp.Usage.InputTokensDetails.CachedTokens),
			Output:     int(resp.Usage.OutputTokens),
			Total:      int(resp.Usage.TotalTokens),
			Resoning:   int(resp.Usage.OutputTokensDetails.ReasoningTokens),
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
			return nil, fmt.Errorf("openai batch item %s missing from output", customID)
		}
	}

	return results, nil
}

func outputTextFromRawResponse(data []byte) string {
	var body struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}

	if err := json.Unmarshal(data, &body); err != nil {
		return ""
	}

	if body.OutputText != "" {
		return body.OutputText
	}

	for _, out := range body.Output {
		for _, c := range out.Content {
			if c.Text != "" {
				return c.Text
			}
		}
	}

	return ""
}
