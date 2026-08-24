package gpt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/conversations"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
)

// chat gpt
func GPT(token string, model ...*Model) (a Engine) {
	p := new(gpt)
	p.conn = openai.NewClient(option.WithAPIKey(token))
	p.token = token
	switch len(model) > 0 {
	case true:
		p.model = model[0]
	default:
		p.model = &Model_GPT_Mini
	}
	p.provider = ProviderGPT
	return p
}

type gpt struct {
	conn     openai.Client
	model    *Model
	provider Provider
	token    string

	fallback []Engine
}

func (a *gpt) Provider() Provider {
	return a.provider
}

func (a *gpt) getToken() string {
	return a.token
}

func (a *gpt) Model(v ...*Model) *Model {
	switch v {
	case nil:
		return a.model
	default:
		a.model = v[0]
		return nil
	}
}

func (a *gpt) Fallback(v ...Engine) Engine {
	a.fallback = append(a.fallback, v...)
	return a
}

func (a *gpt) Chat(m *Message) (res string, err error) {
	if m.name == "" {
		m.name = "chat"
	}
	if m.plaintext {
		m.SystemPromt("RESPONSE MUST BE PLAIN TEXT! STRICT")
	}
	err = a.Send(m)
	if err != nil {
		return
	}
	res = m.CleanText()
	return
}

// `json:"...,required"`

func (a *gpt) Send(m *Message) (err error) {
	if m.name == "" {
		m.name = "request"
	}

	if m.result != nil {
		applySystemPromtStruct(m)
	}

	t1 := time.Now()
	ctx, cancel := reqContext(m.timeout)
	defer cancel()
	promt := m.RenderPromt()
	system := m.RenderSystemPromt()

	// req
	req := responses.ResponseNewParams{
		Model: a.model.ID,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: buildUserInput(m, promt, a.provider),
		},
	}
	if m.store {
		req.Store = openai.Bool(true)
	}
	if err = a.ensureConversation(ctx, m); err != nil {
		return
	}
	req.Conversation = responses.ResponseNewParamsConversationUnion{
		OfString: openai.String(m.chat),
	}

	if system != "" {
		req.Instructions = openai.String(system)
	}

	// if m.temperature > 0 {
	// 	req.Temperature = openai.Float(m.temperature)
	// }

	if len(m.schema) > 0 {
		req.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:        m.name,
					Schema:      m.schema,
					Strict:      openai.Bool(true),
					Description: openai.String("Return only this JSON object."),
				},
			},
		}
	}

	// add websearch
	if a.model.WebSearch {
		a.applyWebSearch(&req, m)
	}

	resp, err := a.conn.Responses.New(ctx, req)
	if err != nil {
		for _, x := range a.fallback {
			if err = x.Send(m); err == nil {
				return nil
			}
		}
		return
	}

	m.resp = resp
	if resp.Conversation.ID != "" {
		m.chat = resp.Conversation.ID
	}
	m.raw = resp.OutputText()
	m.clean = CleanMarkdown(m.raw)

	if len(m.schema) > 0 {
		_ = json.Unmarshal([]byte(m.raw), &m.result)
	}

	m.summary = Summary{
		Chat:       m.chat,
		Model:      a.model,
		Promt:      len(promt),
		System:     len(system),
		Images:     m.imagesSize(),
		Input:      int(resp.Usage.InputTokens),
		Cached:     int(resp.Usage.InputTokensDetails.CachedTokens),
		Output:     int(resp.Usage.OutputTokens),
		Total:      int(resp.Usage.TotalTokens),
		Resoning:   int(resp.Usage.OutputTokensDetails.ReasoningTokens),
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

func (a *gpt) ensureConversation(ctx context.Context, m *Message) error {
	if m.chat != "" {
		return nil
	}

	conv, err := a.conn.Conversations.New(ctx, conversations.ConversationNewParams{})
	if err != nil {
		return err
	}
	if conv.ID == "" {
		return errors.New("openai conversation: empty id")
	}

	m.chat = conv.ID
	return nil
}

func (a *gpt) applyWebSearch(req *responses.ResponseNewParams, m *Message) {
	if !m.websearch {
		return
	}

	tool := responses.ToolParamOfWebSearch(
		responses.WebSearchToolTypeWebSearch,
	)

	tool.OfWebSearch.Filters = responses.WebSearchToolFiltersParam{
		AllowedDomains: m.domains,
	}

	tool.OfWebSearch.SearchContextSize =
		responses.WebSearchToolSearchContextSizeMedium

	if m.country != "" {
		tool.OfWebSearch.UserLocation.Country = openai.String(m.country)
	}
	if m.city != "" {
		tool.OfWebSearch.UserLocation.City = openai.String(m.city)
	}
	if m.region != "" {
		tool.OfWebSearch.UserLocation.Region = openai.String(m.region)
	}

	req.Tools = append(req.Tools, tool)
}

func (a *gpt) Image(m *ImageReq) (res ImageResp, err error) {
	if a.provider != ProviderGPT {
		return res, fmt.Errorf("%s image generation is not supported", a.provider.Title())
	}
	if m == nil {
		return res, errors.New("openai image: empty request")
	}
	if m.name == "" {
		m.name = "image"
	}

	t1 := time.Now()
	promt := m.RenderPromt()
	system := m.RenderSystemPromt()
	fullpromt := strings.TrimSpace(strings.Join([]string{system, promt}, "\n\n"))
	if fullpromt == "" {
		return res, errors.New("openai image: empty prompt")
	}

	model := m.Model()
	if model == nil || model.ID == "" {
		model = &Model_GPT_Image
	}

	ctx, cancel := reqContext(m.timeout)
	defer cancel()
	var resp *openai.ImagesResponse

	if len(m.images) > 0 {
		req := openai.ImageEditParams{
			Image:  gptImageEditInput(m),
			Model:  openai.ImageModel(model.ID),
			Prompt: fullpromt,
		}
		applyGPTImageEditParams(&req, m)
		resp, err = a.conn.Images.Edit(ctx, req)
	} else {
		req := openai.ImageGenerateParams{
			Model:  openai.ImageModel(model.ID),
			Prompt: fullpromt,
		}
		applyGPTImageGenerateParams(&req, m)
		resp, err = a.conn.Images.Generate(ctx, req)
	}
	if err != nil {
		return res, err
	}

	res, err = gptImageResp(resp)
	if err != nil {
		return res, err
	}

	m.provider = a.provider
	m.summary = Summary{
		Chat:       m.chat,
		Model:      model,
		Promt:      len(promt),
		System:     len(system),
		Images:     m.imagesSize(),
		Input:      int(resp.Usage.InputTokens),
		Output:     int(resp.Usage.OutputTokens),
		Total:      int(resp.Usage.TotalTokens),
		PromtText:  promt,
		SystemText: system,
	}
	m.summary.calc()
	m.summary.Time = time.Since(t1)
	m.summary.Times = m.summary.Time.String()
	m.price = m.summary.Price

	res.ChatID = m.chat
	res.Summary = &m.summary
	res.Raw = resp
	return res, nil
}

func gptImageEditInput(m *ImageReq) openai.ImageEditParamsImageUnion {
	readers := make([]io.Reader, 0, len(m.images))
	for _, img := range m.images {
		if img == nil {
			continue
		}
		readers = append(readers, bytes.NewReader(img.JPG(90).Bytes()))
	}

	return openai.ImageEditParamsImageUnion{OfFileArray: readers}
}

func applyGPTImageGenerateParams(req *openai.ImageGenerateParams, m *ImageReq) {
	if m.count > 0 {
		req.N = openai.Int(int64(m.count))
	}
	if m.compression > 0 {
		req.OutputCompression = openai.Int(int64(m.compression))
	}
	if m.uid != "" {
		req.User = openai.String(m.uid)
	}
	if m.background != "" {
		req.Background = openai.ImageGenerateParamsBackground(m.background)
	}
	if m.moderation != "" {
		req.Moderation = openai.ImageGenerateParamsModeration(m.moderation)
	}
	if m.format != "" {
		req.OutputFormat = openai.ImageGenerateParamsOutputFormat(m.format)
	}
	if m.link {
		req.ResponseFormat = openai.ImageGenerateParamsResponseFormatURL
	}
	if m.quality != "" {
		req.Quality = openai.ImageGenerateParamsQuality(m.quality)
	}
	if m.size != "" {
		req.Size = openai.ImageGenerateParamsSize(m.size)
	}
	if m.style != "" {
		req.Style = openai.ImageGenerateParamsStyle(m.style)
	}
}

func applyGPTImageEditParams(req *openai.ImageEditParams, m *ImageReq) {
	if m.count > 0 {
		req.N = openai.Int(int64(m.count))
	}
	if m.compression > 0 {
		req.OutputCompression = openai.Int(int64(m.compression))
	}
	if m.uid != "" {
		req.User = openai.String(m.uid)
	}
	if m.background != "" {
		req.Background = openai.ImageEditParamsBackground(m.background)
	}
	if m.format != "" {
		req.OutputFormat = openai.ImageEditParamsOutputFormat(m.format)
	}
	if m.link {
		req.ResponseFormat = openai.ImageEditParamsResponseFormatURL
	}
	if m.quality != "" {
		req.Quality = openai.ImageEditParamsQuality(m.quality)
	}
	if m.size != "" {
		req.Size = openai.ImageEditParamsSize(m.size)
	}
}

func gptImageResp(resp *openai.ImagesResponse) (res ImageResp, err error) {
	if resp == nil || len(resp.Data) == 0 {
		return res, errors.New("openai image: empty response")
	}

	for _, item := range resp.Data {
		if item.B64JSON != "" {
			img, err := base64.StdEncoding.DecodeString(item.B64JSON)
			if err != nil {
				return res, err
			}
			res.Images = append(res.Images, img)
		}
		if item.URL != "" {
			res.URLs = append(res.URLs, item.URL)
		}
	}

	if len(res.Images) == 0 && len(res.URLs) == 0 {
		return res, errors.New("openai image: empty image data")
	}

	return res, nil
}

// accepts any gpt-compatible format (pdf, images, text, etc).
// an explicit filename (with extension) helps the api detect the format.
func (a *gpt) UploadFile(body []byte, filename ...string) (fileID string, err error) {
	if err = a.filesSupported(); err != nil {
		return "", err
	}
	if len(body) == 0 {
		return "", errors.New("openai upload: empty body")
	}

	f := newUploadFile(body, filename...)

	ctx := context.Background()
	// images must be uploaded as vision to be usable as input_image
	purpose := openai.FilePurposeUserData
	if strings.HasPrefix(f.ContentType(), "image/") {
		purpose = openai.FilePurposeVision
	}

	resp, err := a.conn.Files.New(ctx, openai.FileNewParams{
		File:    openai.File(f, f.Filename(), f.ContentType()),
		Purpose: purpose,
	})
	if err != nil {
		return "", err
	}
	if resp.ID == "" {
		return "", errors.New("openai upload: empty file id")
	}

	return resp.ID, nil
}

// add files to an existing chat conversation as images (input_image)
func (a *gpt) AddImageFiles(conversationID string, filesID ...string) (err error) {
	if err = a.filesSupported(); err != nil {
		return err
	}
	if conversationID == "" {
		return errors.New("openai add image files: empty conversation id")
	}
	if len(filesID) == 0 {
		return nil
	}

	var content responses.ResponseInputMessageContentListParam
	for _, id := range filesID {
		if id == "" {
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				Detail: responses.ResponseInputImageDetailAuto,
				FileID: openai.String(id),
			},
		})
	}
	if len(content) == 0 {
		return nil
	}

	ctx := context.Background()
	_, err = a.conn.Conversations.Items.New(ctx, conversationID, conversations.ItemNewParams{
		Items: responses.ResponseInputParam{
			responses.ResponseInputItemParamOfMessage(
				content,
				responses.EasyInputMessageRoleUser,
			),
		},
	})
	return
}

// add files to an existing chat conversation
func (a *gpt) AddFiles(conversationID string, filesID ...string) (err error) {
	if err = a.filesSupported(); err != nil {
		return err
	}
	if conversationID == "" {
		return errors.New("openai add files: empty conversation id")
	}
	if len(filesID) == 0 {
		return nil
	}

	var content responses.ResponseInputMessageContentListParam
	for _, id := range filesID {
		if id == "" {
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputFile: &responses.ResponseInputFileParam{
				FileID: openai.String(id),
			},
		})
	}
	if len(content) == 0 {
		return nil
	}

	ctx := context.Background()
	_, err = a.conn.Conversations.Items.New(ctx, conversationID, conversations.ItemNewParams{
		Items: responses.ResponseInputParam{
			responses.ResponseInputItemParamOfMessage(
				content,
				responses.EasyInputMessageRoleUser,
			),
		},
	})
	return
}

// reqContext returns a context honoring the request timeout, if set.
func reqContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		return context.WithTimeout(context.Background(), timeout)
	}
	return context.Background(), func() {}
}

// files api is used only for openai, other providers on this client (x.ai) are not verified
func (a *gpt) filesSupported() error {
	if a.provider != ProviderGPT {
		return fmt.Errorf("%s files api is not supported", a.provider.Title())
	}
	return nil
}

// buildUserInput assembles a single user message from the request files, images and prompt.
func buildUserInput(m *Message, promt string, provider Provider) responses.ResponseInputParam {
	var content responses.ResponseInputMessageContentListParam

	// documents uploaded before (UploadFile)
	for _, id := range m.files {
		if id == "" {
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputFile: &responses.ResponseInputFileParam{
				FileID: openai.String(id),
			},
		})
	}

	// images uploaded before (UploadFile)
	for _, id := range m.imagefiles {
		if id == "" {
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				Detail: responses.ResponseInputImageDetailAuto,
				FileID: openai.String(id),
			},
		})
	}

	// documents by a public link (AddFileURL), openai only, x.ai takes images only
	for _, u := range m.fileurls {
		if provider != ProviderGPT {
			log.Printf("%s: document links are not supported, skip %s", provider.Title(), u)
			continue
		}
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputFile: &responses.ResponseInputFileParam{
				FileURL: openai.String(u),
			},
		})
	}

	// images by a public link (AddFileURL, AddImageURL)
	for _, u := range m.imageurls {
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				Detail:   responses.ResponseInputImageDetailAuto,
				ImageURL: openai.String(u),
			},
		})
	}

	for _, x := range m.images {
		content = append(content, responses.ResponseInputContentUnionParam{
			OfInputImage: &responses.ResponseInputImageParam{
				Detail:   responses.ResponseInputImageDetailAuto,
				ImageURL: openai.String(x.JPGBase64HTML(80)),
			},
		})
	}

	content = append(content, responses.ResponseInputContentUnionParam{
		OfInputText: &responses.ResponseInputTextParam{Text: promt},
	})

	return responses.ResponseInputParam{
		responses.ResponseInputItemParamOfMessage(content, responses.EasyInputMessageRoleUser),
	}
}

// delete stored files
func (a *gpt) DeleteFiles(filesID ...string) (err error) {
	if err = a.filesSupported(); err != nil {
		return err
	}

	ctx := context.Background()
	for _, id := range filesID {
		if id == "" {
			continue
		}
		if _, e := a.conn.Files.Delete(ctx, id); e != nil {
			err = e
		}
	}
	return
}
