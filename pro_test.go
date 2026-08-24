package gpt

//testing
import (
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"

	//testing
	//go test -bench=.
	//go test --timeout 9999999999999s

	jsoniter "github.com/json-iterator/go"
	"github.com/monopolly/file"
)

var creds = struct {
	ChatGPT  string `json:"openai"`
	Gemini   string `json:"gemini"`
	Grok     string `json:"grok"`
	Deepseek string `json:"deepseek"`
	Claude   string `json:"claude"`
}{}

func sendTest(p Engine) {

	return
	log.Printf("%s send test", p.Provider())

	type Field struct {
		Updated    bool     `json:"is_updated"`
		Name       string   `json:"name"`
		Confidence int      `json:"confidence"`
		String     string   `json:"string"`
		Strings    []string `json:"strings_array"`
		Bool       bool     `json:"bool"`
		Int        int      `json:"int"`
		Ints       []int    `json:"ints_array"`
	}

	var res struct {
		Brand  []*Field `json:"brand_fields_array"`
		Fields []*Field `json:"general_fields_array"`
		Attr   []*Field `json:"attribute_fields_array"`
	}

	m := NewMessage("name").
		AddImage("https://images.watchfinder.co.uk/imgv3/stock/371294/TagHeuer-Monaco-CAW211R.FC6401-371294-1-250827-082859420.jpg;quality=55;h=1280").
		Promt("You are watches expert. User upload image. You have to recognize image watch and fill all fields with your confidence (in percents) for each field (0 - you dont know or no value, 100 - no mistake)").
		Line().
		Promt("Before we upload images to Google Lens and it found this links:").
		Promt("Tag Heuer Monaco CAW211R.FC6401 - Blue Dial &amp; Leather Strap").
		Promt("https://www.watchfinder.com/Tag%20Heuer/Monaco/CAW211R.FC6401/34765/item/371294").
		Promt("TAG Heuer Monaco Calibre 11 - GULF EDITION - 39 mm - TOP... for $7,178 for sale from a Trusted Seller on Chrono24").
		Promt("https://www.chrono24.com/tagheuer/tag-heuer-monaco-calibre-11---gulf-edition---39-mm---top-conditions-2019--id42751955.htm?searchHash=2aa36af1_SBox59&pos=1").
		Promt("You can use it in your results (if it correct)").
		Line().
		Promt("Fill or update (if applicapble/needs) brand_fields_array with new data (or skip/leave empty field):").
		Promt("- brand_id: int, current value: 99, (select from list: {Casio:1,Rolex:2,IWC:3,Tag Heuer:99} if any)").
		Promt("- name: string (brand name)").
		Promt("- model: string").
		Promt("- ref: string (reference number / unique identifier)").
		Promt("- sub_model: string").
		Line().
		Promt("Fill or update (if applicapble/needs) general_fields_array with new data (or skip/leave empty field):").
		Promt("- title: string").
		Promt("- price: int (in usd)").
		Promt("- min_price: int (in usd)").
		Promt("- max_price: int (in usd)").
		Promt("- year: int").
		Line().
		Promt("Fill or update (if applicapble/needs) attribute_fields_array with new data (or skip/leave empty field):").
		Promt("- diameter: int (mm), current value: 80, current confidence: 40%").
		Promt(`- material: strings_array, current value: ["stainless steel", "fabric"], current confidence: 90%`).
		Promt("- colors: strings_array (hex colors)").
		Promt("- colornames: strings_array (colors names: blue, yellow etc)").
		Struct(res)
	m.Result(&res)

	err := p.Send(m)
	if err != nil {
		log.Printf("%s send err: %s", p.Provider(), err)
		return
	}

	file.Save(fmt.Sprintf("logs/%s_promt.txt", p.Provider()), []byte(m.RenderPromt()))
	file.Save(fmt.Sprintf("logs/%s_system.txt", p.Provider()), []byte(m.RenderSystemPromt()))
	file.Json(fmt.Sprintf("logs/%s_res.json", p.Provider()), res)
	file.Json(fmt.Sprintf("logs/%s_sum.json", p.Provider()), m.Summary())

}

func chatTest(p Engine) {
	return
	log.Printf("chat test: %s", p.Provider())

	m := NewMessage().Promt("What is the capital of UK")
	res, err := p.Chat(m)
	if err != nil {
		log.Printf("%s chat err: %s", p.Provider(), err)
		return
	}

	// file.Save(fmt.Sprintf("logs/%s_chat_promt.txt", p.Provider()), []byte(m.RenderPromt()))
	file.Save(fmt.Sprintf("logs/%s_chat_res.txt", p.Provider()), []byte(res))
	file.Json(fmt.Sprintf("logs/%s_chat_sum.json", p.Provider()), m.Summary())

}

func imageTest(p Engine) {
	return
	log.Printf("image test: %s", p.Provider())
	var res struct {
		Tags []string `json:"tags"`
		Name string   `json:"name_title"`
		Desc string   `json:"description"`
	}

	img := file.OpenE("art/img1.jpg")
	m := NewMessage()
	m.Promt("Describe the image")
	m.AddImage(img)
	m.Struct(res)
	m.Result(&res)

	err := p.Send(m)
	if err != nil {
		log.Printf("%s image err: %s", p.Provider(), err)
		return
	}

	file.Save(fmt.Sprintf("logs/%s_img_promt.txt", p.Provider()), []byte(m.RenderPromt()))
	file.Json(fmt.Sprintf("logs/%s_img_res.json", p.Provider()), res)
	file.Json(fmt.Sprintf("logs/%s_img_sum.json", p.Provider()), m.Summary())

}

func genTest(p Engine) {
	return
	log.Printf("image gen test: %s", p.Provider())

	// img := file.OpenE("art/img1.jpg")

	m := NewImageReq().
		Promt("Create a clean photo of Rolex Daytona, Yellow Gold Green Dial, high details, cinematic. Small grass photo background. Pro-photo").
		Quality("high").
		Format("png")

	resp, err := p.Image(m)
	if err != nil {
		log.Printf("%s image gen err: %s", p.Provider(), err)
		return
	}

	for i, x := range resp.Images {
		file.Save(fmt.Sprintf("logs/%s_img_gen_%d.png", p.Provider(), i+1), x)
	}

	// file.Save(fmt.Sprintf("logs/%s_img_gen.txt", p.Provider()), []byte(m.RenderPromt()))
	// file.Json(fmt.Sprintf("logs/%s_img_res.json", p.Provider()), res)
	file.Json(fmt.Sprintf("logs/%s_img_gen_sum.json", p.Provider()), m.Summary())

}

func convTest(p Engine) {
	return
	log.Printf("conversation id test: %s", p.Provider())
	var res struct {
		Tags []string `json:"tags"`
		Name string   `json:"name_title"`
		Desc string   `json:"description"`
	}

	// first
	m1 := NewStoreMessage().Promt("Describe the image").AddImage(file.OpenE("art/img1.jpg")).Struct(res).Result(&res)
	err := p.Send(m1)
	if err != nil {
		log.Printf("%s image err: %s", p.Provider(), err)
		return
	}

	log.Printf("%s chat id: %s", p.Provider(), m1.ChatID())
	file.Json(fmt.Sprintf("logs/%s_conv_res_a.json", p.Provider()), res)
	file.Json(fmt.Sprintf("logs/%s_conv_sum_a.json", p.Provider()), m1.Summary())

	var res2 struct {
		Tags        []string `json:"tags"`
		MainColor   string   `json:"main_color"`
		FrameColor  string   `json:"frame_color"`
		Subject     string   `json:"subject"`
		Description string   `json:"description"`
		Brand       string   `json:"guess_brand_artist"`
		Price       int      `json:"guess_market_price_usd"`
	}

	// next
	m2 := NewStoreMessage().
		SetChatID(m1.ChatID()).
		Promt("Use the image from the previous message in this conversation. Do not ask for a new image. Add more visual details about that same image.").
		Struct(res2).
		Result(&res2)

	err = p.Send(m2)
	if err != nil {
		log.Printf("%s image err: %s", p.Provider(), err)
		return
	}
	log.Printf("%s chat id next: %s", p.Provider(), m2.ChatID())
	file.Json(fmt.Sprintf("logs/%s_conv_res_b.json", p.Provider()), res2)
	file.Json(fmt.Sprintf("logs/%s_conv_sum_b.json", p.Provider()), m2.Summary())

}

func batchTest(p Engine) {

	return
	log.Printf("batch test: %s", p.Provider())

	type ResStruct struct {
		Tags []string `json:"tags"`
		Name string   `json:"name_title"`
		Desc string   `json:"description"`
	}

	b := NewBatch("img_test", p, func(resp []BatchResult) {
		for i, x := range resp {
			var res ResStruct
			jsoniter.Unmarshal(x.Result, &res)
			x.Message.id = x.ID
			file.Json(fmt.Sprintf("logs/%s_batch_res_%d.json", p.Provider(), i+1), res)
			file.Json(fmt.Sprintf("logs/%s_batch_sum_%d.json", p.Provider(), i+1), x.Message.Summary())
		}
	})

	var res ResStruct
	b.Add(NewMessage().SetID(1).Promt("Describe the image").AddImage(file.OpenE("art/img1.jpg")).Struct(res).Result(&res))
	b.Add(NewMessage().SetID(2).Promt("Describe the image").AddImage(file.OpenE("art/img2.jpg")).Struct(res).Result(&res))
	b.Add(NewMessage().SetID(3).Promt("Describe the image").AddImage(file.OpenE("art/img3.jpg")).Struct(res).Result(&res))

	err := b.Push()
	if err != nil {
		log.Printf("%s: batch error: %s", p.Provider().Title(), err)
	}

}

func TestSetup(u *testing.T) {
	__(u)

	if file.Exists("_creds.json") {
		jsoniter.Unmarshal(file.OpenE("_creds.json"), &creds)
	} else {
		panic("creds")
	}

	var wg sync.WaitGroup

	wg.Go(func() { testGPT() })
	wg.Go(func() { testClaude() })
	wg.Go(func() { testDeepseek() })
	wg.Go(func() { testGemini() })
	wg.Go(func() { testGrok() })

	wg.Wait()

}

// GPT
func testGPT() {
	fmt.Println("gpt")
	p := GPT(creds.ChatGPT)
	var wg sync.WaitGroup
	wg.Go(func() { sendTest(p) })
	wg.Go(func() { chatTest(p) })
	wg.Go(func() { imageTest(p) })
	wg.Go(func() { batchTest(p) })
	wg.Go(func() { convTest(p) })
	wg.Go(func() { genTest(p) })
	wg.Wait()
}

// Claude
func testClaude() {
	fmt.Println("claude")
	p := Claude(creds.Claude)
	var wg sync.WaitGroup
	wg.Go(func() { sendTest(p) })
	wg.Go(func() { chatTest(p) })
	wg.Go(func() { imageTest(p) })
	wg.Go(func() { batchTest(p) })
	wg.Wait()
}

// deepseek
func testDeepseek() {
	fmt.Println("deepseek")
	p := Deepseek(creds.Deepseek)
	var wg sync.WaitGroup
	wg.Go(func() { sendTest(p) })
	wg.Go(func() { chatTest(p) })
	wg.Go(func() { imageTest(p) })
	wg.Wait()
}

func testGemini() {
	fmt.Println("gemini")
	p := Gemini(creds.Gemini)
	var wg sync.WaitGroup
	wg.Go(func() { sendTest(p) })
	wg.Go(func() { chatTest(p) })
	wg.Go(func() { imageTest(p) })
	wg.Go(func() { batchTest(p) })
	wg.Wait()
}

// grok
func testGrok() {
	fmt.Println("grok")
	p := Grok(creds.Grok)
	var wg sync.WaitGroup
	wg.Go(func() { sendTest(p) })
	wg.Go(func() { chatTest(p) })
	wg.Go(func() { imageTest(p) })
	wg.Go(func() { batchTest(p) })
	wg.Wait()
}

func __(u *testing.T) {
	fmt.Printf("\033[1;32m%s\033[0m\n", strings.ReplaceAll(u.Name(), "Test", ""))
}
