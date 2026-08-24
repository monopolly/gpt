package gpt

type Engine interface {
	Send(*Message) error
	Chat(*Message) (string, error)
	Model(...*Model) *Model //change model
	Provider() Provider
	Image(*ImageReq) (ImageResp, error)
	Fallback(...Engine) Engine
	UploadFile(body []byte, filename ...string) (fileID string, err error)
	DeleteFiles(filesID ...string) (err error)

	getToken() string
}
