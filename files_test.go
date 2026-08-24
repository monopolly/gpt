package gpt

// files api tests
//
//	go test -run TestFileURLDetect -v   // offline: link parsing and routing
//	go test -run TestUploadFile -v      // upload/delete in the provider storage
//	go test -run TestMessageFileID -v   // uploaded file ids in a message
//	go test -run TestMessageFileURL -v  // public links in a message
//
// keys come from _creds.json, a provider without a key is skipped.

import (
	"slices"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/monopolly/file"
)

// public links for TestMessageFileURL
const (
	// berkshire hathaway 2024 shareholder letter, the number is inside the pdf only
	testDocURL    = "https://www.berkshirehathaway.com/letters/2024ltr.pdf"
	testDocPromt  = "The attached document is a shareholder letter. How many times does the author say he used the words mistake or error in his letters during the 2019-23 period? Answer with the number only."
	testDocAnswer = "16"

	// flat illustration: sunset over mountains and sea
	testImageURL    = "https://platform.claude.com/docs/images/vision-example.jpg"
	testImagePromt  = "What is the main subject of the attached image? Answer with one word: mountains, city or animals."
	testImageAnswer = "mountain"
)

// local files for TestUploadFile and TestMessageFileID
const (
	testArtImage       = "art/img1.jpg"
	testArtImagePromt  = "What is the dominant color of the artwork in the attached image? Answer with one word."
	testArtImageAnswer = "green"

	// the model can answer only if it really got the file
	testSecret      = "zorfaline-7742"
	testSecretPromt = "The attached file contains one secret code. Answer with that code only."
)

// a text document the model has never seen before
func secretDoc() []byte {
	return []byte("internal note\n\nsecret code: " + testSecret + "\n\nkeep it out of the logs.\n")
}

func testCreds(t *testing.T) {
	t.Helper()

	if creds.ChatGPT != "" || creds.Claude != "" || creds.Gemini != "" || creds.Grok != "" {
		return
	}
	if !file.Exists("_creds.json") {
		t.Skip("no _creds.json")
	}
	if err := jsoniter.Unmarshal(file.OpenE("_creds.json"), &creds); err != nil {
		t.Fatalf("creds: %s", err)
	}
}

// providers with a files api: upload, file ids, file links
func fileEngines(t *testing.T) map[string]Engine {
	t.Helper()
	testCreds(t)

	res := make(map[string]Engine)
	if creds.ChatGPT != "" {
		res["gpt"] = GPT(creds.ChatGPT)
	}
	if creds.Claude != "" {
		res["claude"] = Claude(creds.Claude)
	}
	if creds.Gemini != "" {
		res["gemini"] = Gemini(creds.Gemini)
	}

	if len(res) == 0 {
		t.Skip("no keys for providers with a files api")
	}
	return res
}

// send and return the answer
func ask(t *testing.T, p Engine, m *Message) string {
	t.Helper()

	res, err := p.Chat(m)
	if err != nil {
		t.Fatalf("%s: %s", p.Provider(), err)
	}

	res = strings.TrimSpace(res)
	if res == "" {
		t.Fatalf("%s: empty answer", p.Provider())
	}

	t.Logf("%s: %s", p.Provider(), res)
	return res
}

func wantAnswer(t *testing.T, p Engine, got, want string) {
	t.Helper()

	if !strings.Contains(strings.ToLower(got), want) {
		t.Errorf("%s: answer %q has no %q, the file was probably not read", p.Provider(), got, want)
	}
}

// AddFileURL splits links into documents and images and drops the bad ones
func TestFileURLDetect(t *testing.T) {
	__(t)

	m := NewMessage()
	m.AddFileURL(
		"https://example.com/doc.pdf",
		"https://example.com/doc.pdf", // dup
		"  https://example.com/notes.txt  ",
		"https://example.com/report", // no extension: document
		"https://example.com/photo.JPG;quality=55;h=1280",
		"https://cdn.example.com/a/b/pic.webp?v=2",
		"ftp://example.com/x.pdf", // wrong scheme
		"art/img1.jpg",            // local path, not a link
		"",
	)
	m.AddImageURL("https://example.com/render?id=5") // no extension, forced as image

	docs := []string{
		"https://example.com/doc.pdf",
		"https://example.com/notes.txt",
		"https://example.com/report",
	}
	images := []string{
		"https://example.com/photo.JPG;quality=55;h=1280",
		"https://cdn.example.com/a/b/pic.webp?v=2",
		"https://example.com/render?id=5",
	}

	if !slices.Equal(m.FileURLs(), docs) {
		t.Errorf("document links: %v, want %v", m.FileURLs(), docs)
	}
	if !slices.Equal(m.ImageURLs(), images) {
		t.Errorf("image links: %v, want %v", m.ImageURLs(), images)
	}

	if m.ClearFiles(); len(m.FileURLs()) != 0 || len(m.ImageURLs()) != 0 {
		t.Errorf("ClearFiles left links: %v %v", m.FileURLs(), m.ImageURLs())
	}
}

// upload a document and an image into the provider storage, then delete both
func TestUploadFile(t *testing.T) {
	__(t)

	for name, p := range fileEngines(t) {
		t.Run(name, func(t *testing.T) {
			for _, x := range []struct {
				name string
				body []byte
			}{
				{"secret.txt", secretDoc()},
				{"img1.jpg", file.OpenE(testArtImage)},
			} {
				if len(x.body) == 0 {
					t.Fatalf("empty test file: %s", x.name)
				}

				id, err := p.UploadFile(x.body, x.name)
				if err != nil {
					t.Fatalf("upload %s: %s", x.name, err)
				}
				if id == "" {
					t.Fatalf("upload %s: empty file id", x.name)
				}
				t.Logf("%s uploaded: %s", x.name, id)

				if err := p.DeleteFiles(id); err != nil {
					t.Errorf("delete %s (%s): %s", x.name, id, err)
				}
			}
		})
	}
}

// attach files uploaded before by their file id
func TestMessageFileID(t *testing.T) {
	__(t)

	for name, p := range fileEngines(t) {
		t.Run(name, func(t *testing.T) {
			doc, err := p.UploadFile(secretDoc(), "secret.txt")
			if err != nil {
				t.Fatalf("upload document: %s", err)
			}

			img, err := p.UploadFile(file.OpenE(testArtImage), "img1.jpg")
			if err != nil {
				t.Fatalf("upload image: %s", err)
			}

			t.Cleanup(func() {
				if err := p.DeleteFiles(doc, img); err != nil {
					t.Logf("delete: %s", err)
				}
			})

			t.Run("doc", func(t *testing.T) {
				m := NewMessage("fileid_doc").PlainText().
					AddFileIDs(doc).
					Promt(testSecretPromt)

				if len(m.Files()) != 1 {
					t.Fatalf("document file id not attached: %v", m.Files())
				}
				wantAnswer(t, p, ask(t, p, m), testSecret)
			})

			t.Run("image", func(t *testing.T) {
				m := NewMessage("fileid_image").PlainText().
					AddImageFiles(img).
					Promt(testArtImagePromt)

				if len(m.ImageFiles()) != 1 {
					t.Fatalf("image file id not attached: %v", m.ImageFiles())
				}
				wantAnswer(t, p, ask(t, p, m), testArtImageAnswer)
			})
		})
	}
}

// attach files by a public link, no upload
func TestMessageFileURL(t *testing.T) {
	__(t)

	// documents: openai sends the link as is, claude accepts pdf links,
	// gemini has no link input so the file is downloaded and sent inline
	t.Run("doc", func(t *testing.T) {
		for name, p := range fileEngines(t) {
			t.Run(name, func(t *testing.T) {
				m := NewMessage("fileurl_doc").PlainText().
					AddFileURL(testDocURL).
					Promt(testDocPromt)

				if len(m.FileURLs()) != 1 {
					t.Fatalf("document link not attached: %v", m.FileURLs())
				}
				wantAnswer(t, p, ask(t, p, m), testDocAnswer)
			})
		}
	})

	// images: x.ai takes image links too, only document links are unsupported there
	t.Run("image", func(t *testing.T) {
		engines := fileEngines(t)
		if creds.Grok != "" {
			// x.ai accepts input_image.image_url, but gpt.Send always creates an
			// openai conversation first and x.ai has no /v1/conversations endpoint,
			// so every grok request fails before the image link is even sent
			t.Log("grok skipped: gpt.Send needs /v1/conversations, x.ai has no such endpoint")
		}

		for name, p := range engines {
			t.Run(name, func(t *testing.T) {
				m := NewMessage("fileurl_image").PlainText().
					AddFileURL(testImageURL).
					Promt(testImagePromt)

				if len(m.ImageURLs()) != 1 {
					t.Fatalf("image link went to documents: %v", m.FileURLs())
				}
				wantAnswer(t, p, ask(t, p, m), testImageAnswer)
			})
		}
	})
}
