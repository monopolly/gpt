package gpt

import (
	"encoding/json"
	"fmt"
	"sync"
)

type Batch interface {
	Add(...*Message)
	Push() error
	Waitlist() int
}

type BatchResult struct {
	ID       any // original message id
	CustomID string
	Result   json.RawMessage // clean JSON/text response body
	Message  *Message
}

func NewBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	switch engine.Provider() {
	case ProviderGPT:
		return newGPTBatch(name, engine, handler)
	case ProviderGrok:
		return newGrokBatch(name, engine, handler)
	case ProviderGemini:
		return newGeminiBatch(name, engine, handler)
	case ProviderClaude:
		return newClaudeBatch(name, engine, handler)
	default:
		return newUnsupportedBatch(name, engine, handler)
	}
}

type batchBase struct {
	name     string
	messages []*Message
	handler  func([]BatchResult)

	m sync.Mutex
}

func (a *batchBase) Add(v ...*Message) {
	a.m.Lock()
	defer a.m.Unlock()

	for _, m := range v {
		if m == nil || m.id == nil {
			continue
		}
		a.messages = append(a.messages, m)
	}
}

func (a *batchBase) Waitlist() int {
	a.m.Lock()
	defer a.m.Unlock()

	return len(a.messages)
}

func (a *batchBase) takeMessages() []*Message {
	a.m.Lock()
	defer a.m.Unlock()

	list := a.messages
	a.messages = nil
	return list
}

func (a *batchBase) returnMessages(list []*Message) {
	a.m.Lock()
	defer a.m.Unlock()

	a.messages = append(list, a.messages...)
}

type unsupportedBatch struct {
	batchBase
	provider Provider
}

func newUnsupportedBatch(name string, engine Engine, handler func([]BatchResult)) Batch {
	return &unsupportedBatch{
		batchBase: batchBase{
			name:    responseFormatName(name),
			handler: handler,
		},
		provider: engine.Provider(),
	}
}

func (a *unsupportedBatch) Push() error {
	return fmt.Errorf("%s batch is not supported", a.provider.Title())
}
