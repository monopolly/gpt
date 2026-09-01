package gpt

import (
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// Собственный рефлектор схем: специально без invopop/jsonschema.
// Прямая зависимость на него делала версию ordered-map (wk8 в 0.13 vs pb33f в 0.14)
// частью публичного контракта библиотеки — MVS у потребителя брал максимум и ломал
// сборку anthropic-sdk-go. Теперь версию jsonschema выбирает только сам SDK.

// диалект, который ждут structured outputs у openai/claude/gemini
const jsonSchemaDialect = "https://json-schema.org/draft/2020-12/schema"

var (
	timeType   = reflect.TypeFor[time.Time]()
	rawMsgType = reflect.TypeFor[json.RawMessage]()
)

// GenerateSchemaFromType строит инлайновую (без $ref/$defs) JSON Schema по типу.
// Правила имён и required совпадают с encoding/json: json-тег задаёт имя,
// "-" пропускает поле, omitempty убирает его из required,
// встроенные (embedded) структуры разворачиваются в родителя.
func GenerateSchemaFromType(t reflect.Type) map[string]any {
	res, _ := schemaOf(t, map[reflect.Type]bool{}).(map[string]any)
	if res == nil {
		res = map[string]any{}
	}
	res["$schema"] = jsonSchemaDialect
	return res
}

// schema
func GenerateSchema(structure any) (res map[string]any) {
	return GenerateSchemaFromType(reflect.TypeOf(structure))
}

// schemaOf возвращает либо map[string]any, либо true (схема "что угодно").
// path хранит структуры текущей ветки, чтобы не уйти в бесконечную рекурсию.
func schemaOf(t reflect.Type, path map[reflect.Type]bool) any {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == nil {
		return true
	}

	switch t {
	case timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	case rawMsgType:
		return true
	}

	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return map[string]any{"type": "integer"}

	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}

	case reflect.String:
		return map[string]any{"type": "string"}

	case reflect.Slice:
		// []byte едет как base64-строка
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "contentEncoding": "base64"}
		}
		return map[string]any{"type": "array", "items": schemaOf(t.Elem(), path)}

	case reflect.Array:
		return map[string]any{
			"type":     "array",
			"items":    schemaOf(t.Elem(), path),
			"minItems": t.Len(),
			"maxItems": t.Len(),
		}

	case reflect.Map:
		res := map[string]any{"type": "object"}
		if v := schemaOf(t.Elem(), path); v != true {
			res["additionalProperties"] = v
		}
		return res

	case reflect.Struct:
		return schemaOfStruct(t, path)

	default:
		// interface, chan, func, complex, unsafe.Pointer
		return true
	}
}

func schemaOfStruct(t reflect.Type, path map[reflect.Type]bool) any {
	// рекурсивный тип: отдаём объект без ограничений вместо ухода в бесконечность
	if path[t] {
		return map[string]any{"type": "object"}
	}
	path[t] = true
	defer delete(path, t)

	props := map[string]any{}
	var required []string
	collectFields(t, props, &required, path)

	res := map[string]any{
		"type":                 "object",
		"properties":           props,
		"additionalProperties": false,
	}
	if len(required) > 0 {
		res["required"] = required
	}
	return res
}

func collectFields(t reflect.Type, props map[string]any, required *[]string, path map[reflect.Type]bool) {
	for f := range t.Fields() {
		f := f

		// неэкспортируемые поля не сериализуются; embedded пропускаем ниже,
		// т.к. у него могут быть экспортируемые поля
		if !f.IsExported() && !f.Anonymous {
			continue
		}

		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")

		ft := f.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}

		// embedded структура без своего имени — разворачиваем поля в родителя
		if f.Anonymous && name == "" && ft.Kind() == reflect.Struct && ft != timeType && !path[ft] {
			collectFields(ft, props, required, path)
			continue
		}

		if !f.IsExported() {
			continue
		}
		if name == "" {
			name = f.Name
		}

		props[name] = schemaOf(f.Type, path)
		if !hasOpt(opts, "omitempty") {
			*required = append(*required, name)
		}
	}
}

func hasOpt(opts, want string) bool {
	for opts != "" {
		var o string
		o, opts, _ = strings.Cut(opts, ",")
		if o == want {
			return true
		}
	}
	return false
}
