package gpt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/monopolly/file"
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
		p.model = &Model_Gemini_Medium
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

	fallback []Engine
}

func (a *gemini) Provider() Provider {
	return a.provider
}

func (a *gemini) Fallback(v ...Engine) Engine {
	a.fallback = append(a.fallback, v...)
	return a
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

// upload a file to the gemini files api.
// accepts any gemini-compatible format (pdf, images, audio, video, text, etc).
// an explicit filename (with extension) helps the api detect the format.
// returns the file resource name (files/xxx), it is the fileID for Message.AddFiles.
func (a *gemini) UploadFile(body []byte, filename ...string) (fileID string, err error) {
	if a.conn == nil {
		return "", errors.New("gemini upload: nil client")
	}
	if len(body) == 0 {
		return "", errors.New("gemini upload: empty body")
	}

	f := newUploadFile(body, filename...)

	ctx := context.Background()
	up, err := a.conn.Files.Upload(ctx, f, &genai.UploadFileConfig{
		MIMEType:    f.ContentType(),
		DisplayName: f.Filename(),
	})
	if err != nil {
		return "", err
	}
	if up == nil || up.Name == "" {
		return "", errors.New("gemini upload: empty file name")
	}

	// the file must be processed before it can be used in a request
	up, err = a.waitFile(ctx, up)
	if err != nil {
		return "", err
	}

	geminiFileStore(a.token, up)
	return up.Name, nil
}

// wait until the uploaded file is ACTIVE
func (a *gemini) waitFile(ctx context.Context, f *genai.File) (res *genai.File, err error) {
	deadline := time.Now().Add(geminiFileTimeout)

	for {
		switch f.State {
		case genai.FileStateActive:
			return f, nil
		case genai.FileStateFailed:
			return nil, fmt.Errorf("gemini upload: file %s processing failed", f.Name)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("gemini upload: file %s is still processing", f.Name)
		}

		time.Sleep(time.Second)

		f, err = a.conn.Files.Get(ctx, f.Name, nil)
		if err != nil {
			return nil, err
		}
	}
}

// delete stored files
func (a *gemini) DeleteFiles(filesID ...string) (err error) {
	if a.conn == nil {
		return errors.New("gemini delete: nil client")
	}

	ctx := context.Background()
	for _, id := range filesID {
		name := geminiFileName(id)
		if name == "" {
			continue
		}
		if _, e := a.conn.Files.Delete(ctx, name, nil); e != nil {
			err = e
		}
		geminiFileForget(a.token, name)
	}
	return
}

// gemini has no server side conversations, attach files to the message instead:
// Message.AddFiles / Message.AddImageFiles
// gemini has no server side conversations
func (a *gemini) NewConversation() (conversationID string, err error) {
	return "", errors.New("gemini has no conversations: use Message promts")
}

// gemini has no server side conversations
func (a *gemini) AddText(conversationID string, text ...string) (err error) {
	return errors.New("gemini has no conversations: use Message promts")
}

func (a *gemini) AddFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("gemini has no conversations: use Message.AddFiles")
}

// gemini has no server side conversations, attach files to the message instead:
// Message.AddFiles / Message.AddImageFiles
func (a *gemini) AddImageFiles(conversationID string, filesID ...string) (err error) {
	return errors.New("gemini has no conversations: use Message.AddImageFiles")
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

	fileparts, err := geminiFileParts(context.Background(), a.conn, a.token, m)
	if err != nil {
		return err
	}

	urlparts, err := geminiURLParts(m)
	if err != nil {
		return err
	}

	for _, part := range append(append(fileparts, urlparts...), a.images(m)...) {
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
		a.model.ID,
		[]*genai.Content{
			{
				Parts: userparts,
				Role:  genai.RoleUser,
			},
		},
		&config,
	)
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

// gemini refers to uploaded files by uri + mime type, both come from the files api.
// files are immutable, so the resolved data is cached per api key.
const geminiFileTimeout = 2 * time.Minute

type geminiFileData struct {
	uri  string
	mime string
}

var (
	geminiFilesMu sync.RWMutex
	geminiFiles   = map[string]geminiFileData{}
)

// files/xxx resource name from a name, id or uri
func geminiFileName(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	if i := strings.LastIndex(id, "files/"); i >= 0 {
		return id[i:]
	}
	return "files/" + id
}

func geminiFileKey(token, name string) string {
	return token + "|" + name
}

func geminiFileStore(token string, f *genai.File) {
	if f == nil || f.Name == "" || f.URI == "" {
		return
	}

	geminiFilesMu.Lock()
	geminiFiles[geminiFileKey(token, f.Name)] = geminiFileData{uri: f.URI, mime: f.MIMEType}
	geminiFilesMu.Unlock()
}

func geminiFileForget(token, name string) {
	geminiFilesMu.Lock()
	delete(geminiFiles, geminiFileKey(token, name))
	geminiFilesMu.Unlock()
}

func geminiFileGet(token, name string) (res geminiFileData, ok bool) {
	geminiFilesMu.RLock()
	res, ok = geminiFiles[geminiFileKey(token, name)]
	geminiFilesMu.RUnlock()
	return
}

// gemini has no remote link input: file_data.file_uri accepts only files api
// uris (and youtube links), so links from AddFileURL are downloaded and sent inline.
func geminiURLParts(m *Message) (parts []*genai.Part, err error) {
	urls := append(append([]string{}, m.fileurls...), m.imageurls...)
	if len(urls) == 0 {
		return nil, nil
	}

	for _, u := range urls {
		body := file.Download(u)
		if len(body) == 0 {
			return nil, fmt.Errorf("gemini download error: %s", u)
		}

		parts = append(parts, &genai.Part{InlineData: &genai.Blob{
			Data:     body,
			MIMEType: fileMime(body, urlFileName(u)),
		}})
	}

	return
}

// resolve uploaded file ids into gemini file parts
func geminiFileParts(ctx context.Context, conn *genai.Client, token string, m *Message) (parts []*genai.Part, err error) {
	ids := append(append([]string{}, m.files...), m.imagefiles...)
	if len(ids) == 0 {
		return nil, nil
	}
	if conn == nil {
		return nil, errors.New("gemini files: nil client")
	}

	for _, id := range ids {
		name := geminiFileName(id)
		if name == "" {
			continue
		}

		data, ok := geminiFileGet(token, name)
		if !ok {
			f, e := conn.Files.Get(ctx, name, nil)
			if e != nil {
				return nil, e
			}
			if f == nil || f.URI == "" {
				return nil, fmt.Errorf("gemini files: %s not found", name)
			}
			geminiFileStore(token, f)
			data = geminiFileData{uri: f.URI, mime: f.MIMEType}
		}

		parts = append(parts, genai.NewPartFromURI(data.uri, data.mime))
	}

	return
}
