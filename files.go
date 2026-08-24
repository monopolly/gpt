package gpt

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"unicode/utf8"
)

// detect file extension from magic bytes for a proper upload filename.
func fileName(body []byte) string {
	return "file." + fileExt(body)
}

// detect file extension from magic bytes.
func fileExt(body []byte) string {
	switch {
	case len(body) >= 4 && bytes.Equal(body[:4], []byte("%PDF")):
		return "pdf"
	case len(body) >= 3 && bytes.Equal(body[:3], []byte{0xFF, 0xD8, 0xFF}):
		return "jpg"
	case len(body) >= 8 && bytes.Equal(body[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "png"
	case len(body) >= 6 && (bytes.Equal(body[:6], []byte("GIF87a")) || bytes.Equal(body[:6], []byte("GIF89a"))):
		return "gif"
	case len(body) >= 12 && bytes.Equal(body[:4], []byte("RIFF")) && bytes.Equal(body[8:12], []byte("WEBP")):
		return "webp"
	case isJSON(body):
		return "json"
	case isText(body):
		return "txt"
	}
	return "bin"
}

// mime type by body magic bytes, filename extension wins if known.
func fileMime(body []byte, filename ...string) string {
	if len(filename) > 0 {
		if i := strings.LastIndex(filename[0], "."); i >= 0 {
			if mime := extMime(strings.ToLower(filename[0][i+1:])); mime != "" {
				return mime
			}
		}
	}
	if mime := extMime(fileExt(body)); mime != "" {
		return mime
	}
	return "application/octet-stream"
}

func extMime(ext string) string {
	switch ext {
	case "pdf":
		return "application/pdf"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "json":
		return "application/json"
	case "txt", "md":
		return "text/plain"
	case "csv":
		return "text/csv"
	case "html", "htm":
		return "text/html"
	case "xml":
		return "text/xml"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	}
	return ""
}

// isJSON reports whether body looks like a json document.
func isJSON(body []byte) bool {
	t := bytes.TrimSpace(body)
	return len(t) > 0 && (t[0] == '{' || t[0] == '[') && json.Valid(t)
}

// isText reports whether body is valid utf-8 without control/null bytes.
func isText(body []byte) bool {
	if !utf8.Valid(body) {
		return false
	}
	for _, r := range string(body) {
		if r == 0 || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') {
			return false
		}
	}
	return true
}

// multipart upload body with an explicit filename and content type
type uploadFile struct {
	io.Reader
	filename string
	mime     string
}

func newUploadFile(body []byte, filename ...string) uploadFile {
	name := fileName(body)
	if len(filename) > 0 && filename[0] != "" {
		name = filename[0]
	}

	return uploadFile{
		Reader:   bytes.NewReader(body),
		filename: name,
		mime:     fileMime(body, name),
	}
}

func (a uploadFile) Filename() string    { return a.filename }
func (a uploadFile) Name() string        { return a.filename }
func (a uploadFile) ContentType() string { return a.mime }
