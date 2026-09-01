package gpt

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func CleanMarkdown(body string) (res string) {
	res = body
	res = strings.ReplaceAll(res, "```json", "")
	res = strings.ReplaceAll(res, "```", "")
	res = strings.TrimSpace(res)
	return
}

// OptimizePrompt очищает и немного сжимает текст промпта
func CompactText(s string) string {
	if s == "" {
		return s
	}

	// 1. Убираем CR
	s = strings.ReplaceAll(s, "\r", "")

	// 2. Убираем trailing spaces
	lines := strings.Split(s, "\n")
	cleaned := make([]string, 0, len(lines))

	var prev string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// пропускаем повторяющиеся строки подряд
		if line == prev && line != "" {
			continue
		}
		prev = line
		cleaned = append(cleaned, line)
	}

	s = strings.Join(cleaned, "\n")

	// 3. Схлопываем 3+ пустых строк в одну
	reMultiNewlines := regexp.MustCompile(`\n{3,}`)
	s = reMultiNewlines.ReplaceAllString(s, "\n\n")

	// 4. Схлопываем множественные пробелы (но не переносы строк)
	reMultiSpaces := regexp.MustCompile(`[ \t]{2,}`)
	s = reMultiSpaces.ReplaceAllString(s, " ")

	// 5. Финальный trim
	s = strings.TrimSpace(s)
	return s
}

func StructToText(v any, structname ...string) (res string) {
	if v == nil {
		return
	}

	t := reflect.TypeOf(v)
	// Разыменуем указатели
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return
	}

	var b strings.Builder
	typeName := t.Name()
	if len(structname) > 0 {
		typeName = structname[0]
	}

	if typeName == "" {
		typeName = "Result"
	}

	b.WriteString("type ")
	b.WriteString(typeName)
	b.WriteString(" struct {\n")

	// Чтобы красивее выровнять колонки (имя / тип / тег)
	maxName := 0
	maxType := 0

	fields := make([]reflect.StructField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fields = append(fields, f)

		name := f.Name
		// embedded поле без имени в объявлении — печатаем тип
		if f.Anonymous && name == "" {
			name = f.Type.String()
		}
		if len(name) > maxName {
			maxName = len(name)
		}

		ts := typeString(f.Type)
		if len(ts) > maxType {
			maxType = len(ts)
		}
	}

	for _, f := range fields {
		name := f.Name
		if f.Anonymous && name == "" {
			name = typeString(f.Type)
		}

		typ := typeString(f.Type)

		// строка: \t<Name><pad> <Type><pad> <Tag>
		b.WriteString("\t")
		b.WriteString(name)
		if pad := maxName - len(name); pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}

		b.WriteString("  ")
		b.WriteString(typ)
		if pad := maxType - len(typ); pad > 0 {
			b.WriteString(strings.Repeat(" ", pad))
		}

		if tag := string(f.Tag); tag != "" {
			b.WriteString("  `")
			b.WriteString(tag)
			b.WriteString("`")
		}
		b.WriteString("\n")
	}

	b.WriteString("}\n")
	return b.String()
}

func typeString(t reflect.Type) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + typeString(t.Elem())

	case reflect.Slice:
		return "[]" + typeString(t.Elem())

	case reflect.Array:
		return fmt.Sprintf("[%d]%s", t.Len(), typeString(t.Elem()))

	case reflect.Map:
		return "map[" + typeString(t.Key()) + "]" + typeString(t.Elem())

	case reflect.Struct:
		// именованный struct → только имя
		if t.Name() != "" {
			return t.Name()
		}
		// анонимный struct (struct{...})
		return "struct"

	default:
		// базовые и именованные типы
		if t.Name() != "" {
			return t.Name()
		}
		return t.String()
	}
}

func GetModels(a Engine) (res []Model, err error) {

	var models struct {
		List []Model `json:"active_api_models"`
	}

	m := NewMessage("model_list")
	m.Result(&models)
	m.Promtf("Give me all new and active %s API GPT models. Use: %s", a.Provider(), a.Provider().Link())
	m.Promt("No deprecated and outdated models! Check models list twice! I need all models including new and current")
	m.Struct(models)
	m.Promt(StructToText(Model{}, "Model"))

	fmt.Println(m.RenderPromt())

	err = a.Send(m)
	if err != nil {
		return
	}

	for _, x := range models.List {
		x.Provider = a.Provider()
		res = append(res, x)
	}
	// res = models.List
	return
}

func applySystemPromtStruct(m *Message) {
	if m.schemarender == "" {
		return
	}
	m.SystemPromt("Output must be a valid JSON format!")
	m.SystemPromt("Strict Output JSON Schema:")
	m.SystemPromt(m.schemarender)
}
