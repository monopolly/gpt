package gpt

import "fmt"

func (a *claude) Image(m *ImageReq) (res ImageResp, err error) {
	return res, fmt.Errorf("%s image generation is not supported", a.provider.Title())
}

func (a *gemini) Image(m *ImageReq) (res ImageResp, err error) {
	return res, fmt.Errorf("%s image generation is not supported", a.provider.Title())
}

func (a *deep) Image(m *ImageReq) (res ImageResp, err error) {
	return res, fmt.Errorf("%s image generation is not supported", a.provider.Title())
}
