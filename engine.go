package gpt

type Engine interface {
	Send(*Message) error
	Chat(*Message) (string, error)
	Model(...*Model) *Model //change model
	Provider() Provider

	getToken() string
	Image(*ImageReq) (ImageResp, error)
	// Video(*ImageReq) error
	// Batch() *Batch

	// batch(list ...*Message)
}
