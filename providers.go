package gpt

/*
Constants Generator
Help to auto-generated correct constants for lists
Martin Prestone (c) 2024
github.com/monopolly

Struct options
#up            Uppercase for golang vars
#simple        Const only
name=Status    Named var. Ex name=Category
type=string    Type of const list. Ex type=string, int by default
#swift         Create swift enum
#index         Create int index anyway

Fields options
go{}           Add new golang var name for fields. Ex go{ArtList}
title{}        Add title for fields. Ex title{News}
#              Add lists for fields. Ex: id int //#readonly #must...
@              Custom values. Ex: id int //@code=401 @some=ErrSome
iota=100       Add iota counter
name{}         Add new name for fields. Ex name{newjsonname}
desc{}         Add desc for fields. Ex desc{New movie is good}
list{}         Create a.List() for field. Ex: list{colors, text}
*/
type Provider int

const (
	ProviderGPT      = Provider(iota) //go{ProviderGPT} title{OpenAI} name{gpt}
	ProviderGemini                    //go{ProviderGemini} title{Google}  name{gemini}
	ProviderGrok                      //go{ProviderGrok} title{Grok}  name{grok}
	ProviderDeepseek                  //go{ProviderDeepseek} title{Deepseek}  name{deepseek}
	ProviderClaude                    //go{ProviderClaude} title{Anthropic}  name{claude}
)

const (
	ProviderMistral    = Provider(iota) + 100 //go{ProviderMistral} title{Mistral}  name{mistral} iota=100
	ProviderCohere                            //go{ProviderCohere} title{Cohere}  name{cohere}
	ProviderMeta                              //go{ProviderMeta} title{Meta}  name{meta}
	ProviderAlibaba                           //go{ProviderAlibaba} title{Alibaba}  name{alibaba}
	ProviderKimi                              //go{ProviderKimi} title{Kimi}  name{kimi}
	ProviderPerplexity                        //go{ProviderPerplexity} title{Perplexity}
)

func (a Provider) String() string {
	switch a {
	case ProviderGPT:
		return "gpt"
	case ProviderGemini:
		return "gemini"
	case ProviderGrok:
		return "grok"
	case ProviderDeepseek:
		return "deepseek"
	case ProviderClaude:
		return "claude"
	case ProviderMistral:
		return "mistral"
	case ProviderCohere:
		return "cohere"
	case ProviderMeta:
		return "meta"
	case ProviderAlibaba:
		return "alibaba"
	case ProviderKimi:
		return "kimi"
	case ProviderPerplexity:
		return "perplexity"
	default:
		return ""
	}
}

func (a Provider) Title() string {
	switch a {
	case ProviderGPT:
		return "OpenAI"
	case ProviderGemini:
		return "Google"
	case ProviderGrok:
		return "Grok"
	case ProviderDeepseek:
		return "Deepseek"
	case ProviderClaude:
		return "Anthropic"
	case ProviderMistral:
		return "Mistral"
	case ProviderCohere:
		return "Cohere"
	case ProviderMeta:
		return "Meta"
	case ProviderAlibaba:
		return "Alibaba"
	case ProviderKimi:
		return "Kimi"
	case ProviderPerplexity:
		return "Perplexity"
	default:
		return ""
	}
}

func (a Provider) Desc() string {
	switch a {
	default:
		return ""
	}
}

func (a Provider) Int() int {
	return int(a)
}

func ValidIndexType(v Provider) bool {
	switch v {
	case ProviderGPT, ProviderGemini, ProviderGrok, ProviderDeepseek, ProviderClaude, ProviderMistral, ProviderCohere, ProviderMeta, ProviderAlibaba, ProviderKimi, ProviderPerplexity:
		return true
	default:
		return false
	}
}

func (a Provider) Valid() bool {
	return ValidIndexType(a)
}

func Valid(v int) bool {
	switch Provider(v) {
	case ProviderGPT, ProviderGemini, ProviderGrok, ProviderDeepseek, ProviderClaude, ProviderMistral, ProviderCohere, ProviderMeta, ProviderAlibaba, ProviderKimi, ProviderPerplexity:
		return true
	default:
		return false
	}
}

func ProviderIndexes() []Provider {
	return []Provider{ProviderGPT, ProviderGemini, ProviderGrok, ProviderDeepseek, ProviderClaude, ProviderMistral, ProviderCohere, ProviderMeta, ProviderAlibaba, ProviderKimi, ProviderPerplexity}
}

func Index(v string) int {
	switch v {
	case "gpt":
		return 1
	case "gemini":
		return 2
	case "grok":
		return 3
	case "deepseek":
		return 4
	case "claude":
		return 5
	case "mistral":
		return 100
	case "cohere":
		return 101
	case "meta":
		return 102
	case "alibaba":
		return 103
	case "kimi":
		return 104
	case "perplexity":
		return 105
	default:
		return 0
	}
}
