package gpt

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/monopolly/file"
	"github.com/monopolly/images"
	"github.com/openai/openai-go/v3/responses"
)

func NewMessage(name ...string) (res *Message) {
	res = new(Message)
	res.created = time.Now().Unix()
	if name != nil {
		res.name = strings.Join(name, "_")
	}
	res.retry = 3
	return
}

func NewStoreMessage(name ...string) (res *Message) {
	res = new(Message)
	res.created = time.Now().Unix()
	if name != nil {
		res.name = strings.Join(name, "_")
	}
	res.retry = 3
	res.store = true
	return
}

type Message struct {
	id   any
	chat string //chat id

	created int64

	name   string
	promt  []string
	system []string

	provider     Provider
	temperature  float64
	images       []*images.Image //base64 or links
	files        []string        //uploaded file id (documents: pdf, txt, json...)
	imagefiles   []string        //uploaded file id (images)
	result       any             // &Res
	schema       map[string]any  //json schema
	schemarender string          //json string {"":1...}
	websearch    bool
	plaintext    bool
	store        bool
	retry        int
	timeout      time.Duration

	raw   string //raw text
	clean string //raw text
	resp  any    //raw {tokens etc}

	// websearch
	domains []string
	country string
	city    string
	region  string

	// prices  float64
	price   float64
	summary Summary
}

func (a *Message) AddImageLocalFile(path string) (m *Message) {
	body := file.OpenE(path)
	if body == nil {
		return
	}

	a.AddImage(body)
	return
}

// conversation id (chatgpt)
func (a *Message) SetChatID(v string) (m *Message) {
	a.chat = v
	return a
}

// batch props or other
func (a *Message) SetID(v any) (m *Message) {
	a.id = v
	return a
}

// batch props or other
func (a *Message) ID() any {
	return a.id
}

// no markdown
func (a *Message) PlainText() (m *Message) {
	a.plaintext = true
	return a
}

func (a *Message) Store() (m *Message) {
	a.store = true
	return a
}

// conversation id (chatgpt)
func (a *Message) ChatID() (id string) {
	return a.chat
}

func (a *Message) AddImages(v ...string) (m *Message) {
	for _, x := range v {
		a.AddImage(x)
	}
	return a
}

// url, body, base64, dataurl
func (a *Message) AddImage(v any) (m *Message) {

	switch body := v.(type) {
	case *images.Image:
		a.images = append(a.images, body)
	case []byte:
		img, err := images.New(body)
		if err != nil {
			log.Println("gpt image format error")
			return
		}
		a.images = append(a.images, img)
	case string:
		switch {
		case strings.Contains(body, "http"):
			temp := file.Download(body)
			if temp == nil {
				log.Println("gpt download image error")
				return
			}
			img, err := images.New(temp)
			if err != nil {
				log.Println("gpt image format error")
				return
			}
			a.images = append(a.images, img)
		default:
			switch {
			case images.IsBase64ImageDataURL(body):
				img, err := images.New([]byte(body))
				if err != nil {
					log.Println("gpt image format error")
					return
				}
				a.images = append(a.images, img)
			case images.IsBase64(body):
				body = images.CleanBase64(body)
				img, err := images.New([]byte(body))
				if err != nil {
					log.Println("gpt image format error")
					return
				}
				a.images = append(a.images, img)
			}

		}
	}

	return a
}

// max retry attemts
func (a *Message) Retry(max int) (m *Message) {
	a.retry = max
	return a
}

// max retry attemts
func (a *Message) Timeout(t time.Duration) (m *Message) {
	a.timeout = t
	return a
}

// add promt
func (a *Message) Promt(lines ...string) (m *Message) {
	a.promt = append(a.promt, lines...)
	return a
}

// IMPORTANT! Result must be strict valid JSON only! I need to parse (golang) it into this structure:
// must be structure - ResultStruct{}
func (a *Message) Struct(structure any, name ...string) (m *Message) {
	n := "Result"
	if name != nil {
		n = name[0]
	}
	a.Line()
	a.Promt(`IMPORTANT! Result must be strict valid JSON only! I need to parse (golang) it into this structure:`)
	a.Promt(StructToText(structure, n))
	return a
}

func (a *Message) StructOnly(structure any, name string) (m *Message) {
	a.Line()
	a.Promt(StructToText(structure, name))
	return a
}

// empty line
func (a *Message) Line() (m *Message) {
	a.promt = append(a.promt, "")
	return a
}

// add promtf
func (a *Message) Promtf(k string, v ...any) (m *Message) {
	a.promt = append(a.promt, fmt.Sprintf(k, v...))
	return a
}

// add system promt
func (a *Message) SystemPromt(lines ...string) (m *Message) {
	a.system = append(a.system, lines...)
	return a
}

// add system promtf
func (a *Message) SystemPromtf(k string, v ...any) (m *Message) {
	a.system = append(a.system, fmt.Sprintf(k, v...))
	return a
}

// render promt
func (a *Message) RenderPromt() string {
	return CompactText(strings.Join(a.promt, "\n"))
}

// render system promt
func (a *Message) RenderSystemPromt() string {
	return CompactText(strings.Join(a.system, "\n"))
}

// resp raw text
func (a *Message) Raw() string {
	return a.raw
}

// resp clean markdown raw text
func (a *Message) CleanText() string {
	return CleanMarkdown(a.raw)
}

// set model
func (a *Message) SetWebSearch(v bool) (m *Message) {
	a.websearch = v
	return a
}

// add promt
func (a *Message) WebSearchDomains(domains ...string) (m *Message) {
	if len(domains) == 0 {
		return a
	}
	a.websearch = true
	a.domains = append(a.domains, domains...)
	return a
}

func (a *Message) WebSearchCountry(v string) (m *Message) {
	a.websearch = true
	a.country = v
	return a
}

func (a *Message) WebSearchCity(v string) (m *Message) {
	a.websearch = true
	a.city = v
	return a
}

func (a *Message) WebSearchRegion(v string) (m *Message) {
	a.websearch = true
	a.region = v
	return a
}

// Deepseek: 0 math, 1 analys, 1.3 chat/translation, 1.5 creative
func (a *Message) Temperature(t float64) (m *Message) {
	a.temperature = t
	return a
}

// res must be pointer &Res{}... or panic
func (a *Message) Result(res any) (m *Message) {

	a.result = res

	t := reflect.TypeOf(res)
	if t == nil {
		panic("nil result")
	}
	if t.Kind() != reflect.Pointer {
		panic("Result expects pointer, e.g. &MyStruct{}")
	}
	a.schema = GenerateSchemaFromType(t.Elem())
	rr, _ := json.Marshal(a.schema)
	a.schemarender = CompactText(string(rr))
	return a
}

// chatgpt raw resp
func (a *Message) ChatGPTResp() (resp *responses.Response) {
	resp = new(responses.Response)
	resp, _ = a.resp.(*responses.Response)
	return
}

func (a *Message) Summary() (res *Summary) {
	return &a.summary
}

func (a *Message) imagesSize() (bytes int) {
	for _, x := range a.images {
		bytes += x.Size
	}
	return
}

// attach files uploaded before (fileID from Engine.UploadFile): pdf, text, json, csv...
func (a *Message) AddFiles(fileIDs ...string) (m *Message) {
	a.files = addFileIDs(a.files, fileIDs)
	return a
}

// attach images uploaded before (fileID from Engine.UploadFile)
func (a *Message) AddImageFiles(fileIDs ...string) (m *Message) {
	a.imagefiles = addFileIDs(a.imagefiles, fileIDs)
	return a
}

// attached document file ids
func (a *Message) Files() []string {
	return a.files
}

// attached image file ids
func (a *Message) ImageFiles() []string {
	return a.imagefiles
}

// drop all attached file ids
func (a *Message) ClearFiles() (m *Message) {
	a.files = nil
	a.imagefiles = nil
	return a
}

// clean, dedup and append file ids
func addFileIDs(list []string, add []string) []string {
	for _, id := range add {
		id = strings.TrimSpace(id)
		if id == "" || slices.Contains(list, id) {
			continue
		}
		list = append(list, id)
	}
	return list
}
