package gpt

type Engine interface {
	Send(*Message) error
	Chat(*Message) (string, error)
	Model(...*Model) *Model //change model
	Provider() Provider

	getToken() string
	Image(*ImageReq) (ImageResp, error)
	Fallback(...Engine) Engine

	UploadFile(body []byte, filename ...string) (fileID string, err error)
	// AddFiles(conversationID string, filesID ...string) (err error)
	// AddImageFiles(conversationID string, filesID ...string) (err error)
	DeleteFiles(filesID ...string) (err error)

	// Video(*ImageReq) error
	// Batch() *Batch

	// batch(list ...*Message)
}
