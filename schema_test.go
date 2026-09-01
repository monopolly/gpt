package gpt

import (
	"encoding/json"
	"reflect"
	"testing"
)

type schemaInner struct {
	A string `json:"a"`
	B int    `json:"b,omitempty"`
	C bool   `json:"-"`
	D float64
}

type schemaEmbedded struct {
	EmbField string `json:"emb_field"`
}

type schemaNode struct {
	Name     string        `json:"name"`
	Children []*schemaNode `json:"children"`
}

func TestGenerateSchemaFromType(t *testing.T) {
	type target struct {
		schemaEmbedded
		S     string            `json:"s"`
		Ptr   *string           `json:"ptr"`
		Sl    []string          `json:"sl"`
		Bytes []byte            `json:"bytes"`
		Mp    map[string]int    `json:"mp"`
		Any   any               `json:"any"`
		In    schemaInner       `json:"in"`
		InSl  []*schemaInner    `json:"in_sl"`
		Nest  map[string]*schemaInner `json:"-"`
		NoTag string
		priv  string
	}

	got := GenerateSchemaFromType(reflect.TypeOf(target{}))

	if got["$schema"] != jsonSchemaDialect {
		t.Fatalf("$schema = %v", got["$schema"])
	}
	if got["additionalProperties"] != false || got["type"] != "object" {
		t.Fatalf("root = %v", got)
	}

	props := got["properties"].(map[string]any)
	// embedded развёрнут, "-" и неэкспортируемые выкинуты, поле без тега по имени
	for _, name := range []string{"emb_field", "s", "ptr", "sl", "bytes", "mp", "any", "in", "in_sl", "NoTag"} {
		if _, ok := props[name]; !ok {
			t.Errorf("нет свойства %q", name)
		}
	}
	for _, name := range []string{"Nest", "priv", "-", "schemaEmbedded"} {
		if _, ok := props[name]; ok {
			t.Errorf("лишнее свойство %q", name)
		}
	}

	if want := (map[string]any{"type": "string", "contentEncoding": "base64"}); !reflect.DeepEqual(props["bytes"], want) {
		t.Errorf("bytes = %v", props["bytes"])
	}
	if props["any"] != true {
		t.Errorf("any = %v", props["any"])
	}
	// указатель разыменован
	if want := (map[string]any{"type": "string"}); !reflect.DeepEqual(props["ptr"], want) {
		t.Errorf("ptr = %v", props["ptr"])
	}
	// map: ключи описаны через additionalProperties
	if mp := props["mp"].(map[string]any); mp["type"] != "object" ||
		!reflect.DeepEqual(mp["additionalProperties"], map[string]any{"type": "integer"}) {
		t.Errorf("mp = %v", mp)
	}
	// слайс структур инлайнится без $ref
	if sl := props["in_sl"].(map[string]any); sl["type"] != "array" ||
		sl["items"].(map[string]any)["additionalProperties"] != false {
		t.Errorf("in_sl = %v", sl)
	}

	// omitempty не попадает в required, остальное попадает в порядке объявления
	in := props["in"].(map[string]any)
	if want := []string{"a", "D"}; !reflect.DeepEqual(toStrings(in["required"]), want) {
		t.Errorf("in.required = %v", in["required"])
	}

	want := []string{"emb_field", "s", "ptr", "sl", "bytes", "mp", "any", "in", "in_sl", "NoTag"}
	if !reflect.DeepEqual(toStrings(got["required"]), want) {
		t.Errorf("required = %v, want %v", got["required"], want)
	}

	if _, err := json.Marshal(got); err != nil {
		t.Fatalf("схема не маршалится: %v", err)
	}
}

// рекурсивный тип не должен уходить в бесконечную рекурсию
func TestGenerateSchemaRecursive(t *testing.T) {
	got := GenerateSchemaFromType(reflect.TypeOf(schemaNode{}))
	items := got["properties"].(map[string]any)["children"].(map[string]any)["items"].(map[string]any)
	if !reflect.DeepEqual(items, map[string]any{"type": "object"}) {
		t.Fatalf("children.items = %v", items)
	}
}

func TestGenerateSchemaNil(t *testing.T) {
	if got := GenerateSchema(nil); got["$schema"] != jsonSchemaDialect || len(got) != 1 {
		t.Fatalf("GenerateSchema(nil) = %v", got)
	}
}

func toStrings(v any) (res []string) {
	for _, x := range v.([]string) {
		res = append(res, x)
	}
	return
}
