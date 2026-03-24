package builtins

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"zumbra/object"
)

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}

func objectToText(obj object.Object) (string, error) {
	switch v := obj.(type) {
	case *object.String:
		return v.Value, nil
	case *object.Integer, *object.Float, *object.Boolean:
		return v.Inspect(), nil
	case *object.Array:
		parts := make([]string, 0, len(v.Elements))
		for _, el := range v.Elements {
			txt, err := objectToText(el)
			if err != nil {
				return "", err
			}
			parts = append(parts, txt)
		}
		return strings.Join(parts, "\n"), nil
	case *object.Dict:
		lines := []string{}
		for _, pair := range v.Pairs {
			val, err := objectToText(pair.Value)
			if err != nil {
				return "", err
			}
			lines = append(lines, fmt.Sprintf("%s: %s", pair.Key.Inspect(), val))
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported content type %s", obj.Type())
	}
}

func objectToCSV(obj object.Object) (string, error) {
	if str, ok := obj.(*object.String); ok {
		return str.Value, nil
	}

	arr, ok := obj.(*object.Array)
	if !ok {
		return "", fmt.Errorf("argument 2 to `createCsv` must be STRING or ARRAY, got %s", obj.Type())
	}

	rows := []string{}
	for _, row := range arr.Elements {
		rowArr, ok := row.(*object.Array)
		if !ok {
			txt, err := objectToText(row)
			if err != nil {
				return "", err
			}
			rows = append(rows, escapeCSVValue(txt))
			continue
		}

		cols := []string{}
		for _, col := range rowArr.Elements {
			txt, err := objectToText(col)
			if err != nil {
				return "", err
			}
			cols = append(cols, escapeCSVValue(txt))
		}
		rows = append(rows, strings.Join(cols, ","))
	}

	return strings.Join(rows, "\n"), nil
}

func escapeCSVValue(value string) string {
	value = strings.ReplaceAll(value, `"`, `""`)
	if strings.ContainsAny(value, ",\n\"") {
		return `"` + value + `"`
	}
	return value
}

func buildDocContent(title, content string) string {
	escapedTitle := htmlEscape(title)
	escapedBody := strings.ReplaceAll(htmlEscape(content), "\n", "<br>")

	return fmt.Sprintf("<html><head><meta charset=\"utf-8\"><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>", escapedTitle, escapedTitle, escapedBody)
}

func htmlEscape(value string) string {
	value = strings.ReplaceAll(value, "&", "&amp;")
	value = strings.ReplaceAll(value, "<", "&lt;")
	value = strings.ReplaceAll(value, ">", "&gt;")
	value = strings.ReplaceAll(value, `"`, "&quot;")
	return value
}

func buildMinimalPDF(title, content string) []byte {
	lines := []string{title, ""}
	lines = append(lines, strings.Split(content, "\n")...)

	var stream strings.Builder
	stream.WriteString("BT\n/F1 18 Tf\n50 780 Td\n")
	for i, line := range lines {
		escaped := escapePDFText(line)
		if i == 0 {
			stream.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
			stream.WriteString("0 -28 Td\n/F1 12 Tf\n")
			continue
		}
		stream.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
		stream.WriteString("0 -18 Td\n")
	}
	stream.WriteString("ET")

	streamStr := stream.String()

	objects := []string{
		"1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n",
		"2 0 obj\n<< /Type /Pages /Count 1 /Kids [3 0 R] >>\nendobj\n",
		"3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n",
		fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(streamStr), streamStr),
		"5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")

	offsets := []int{0}
	for _, obj := range objects {
		offsets = append(offsets, buf.Len())
		buf.WriteString(obj)
	}

	xrefStart := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF", xrefStart))

	return buf.Bytes()
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}

func CreateFileBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wrong number of arguments. got=%d, want=2", len(args))
		}

		pathObj, ok := args[0].(*object.String)
		if !ok {
			return NewError("argument 1 to `createFile` must be STRING, got %s", args[0].Type())
		}

		content, err := objectToText(args[1])
		if err != nil {
			return NewError("%s", err.Error())
		}

		if err := ensureDir(pathObj.Value); err != nil {
			return NewError("failed to create directory for '%s'. got %s", pathObj.Value, err)
		}

		if err := os.WriteFile(pathObj.Value, []byte(content), 0644); err != nil {
			return NewError("failed to create file '%s'. got %s", pathObj.Value, err)
		}

		return NewString(pathObj.Value)
	}}
}

func CreateTxtBuiltin() *object.Builtin {
	return CreateFileBuiltin()
}

func CreateCsvBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 {
			return NewError("wrong number of arguments. got=%d, want=2", len(args))
		}

		pathObj, ok := args[0].(*object.String)
		if !ok {
			return NewError("argument 1 to `createCsv` must be STRING, got %s", args[0].Type())
		}

		content, err := objectToCSV(args[1])
		if err != nil {
			return NewError("%s", err.Error())
		}

		if err := ensureDir(pathObj.Value); err != nil {
			return NewError("failed to create directory for '%s'. got %s", pathObj.Value, err)
		}

		if err := os.WriteFile(pathObj.Value, []byte(content), 0644); err != nil {
			return NewError("failed to create csv '%s'. got %s", pathObj.Value, err)
		}

		return NewString(pathObj.Value)
	}}
}

func CreateDocBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 && len(args) != 3 {
			return NewError("wrong number of arguments. got=%d, want=2 or 3", len(args))
		}

		pathObj, ok := args[0].(*object.String)
		if !ok {
			return NewError("argument 1 to `createDoc` must be STRING, got %s", args[0].Type())
		}

		title := "Zumbra document"
		bodyArg := args[1]
		if len(args) == 3 {
			titleObj, ok := args[1].(*object.String)
			if !ok {
				return NewError("argument 2 to `createDoc` must be STRING, got %s", args[1].Type())
			}
			title = titleObj.Value
			bodyArg = args[2]
		}

		content, err := objectToText(bodyArg)
		if err != nil {
			return NewError("%s", err.Error())
		}

		doc := buildDocContent(title, content)
		if err := ensureDir(pathObj.Value); err != nil {
			return NewError("failed to create directory for '%s'. got %s", pathObj.Value, err)
		}

		if err := os.WriteFile(pathObj.Value, []byte(doc), 0644); err != nil {
			return NewError("failed to create doc '%s'. got %s", pathObj.Value, err)
		}

		return NewString(pathObj.Value)
	}}
}

func CreatePdfBuiltin() *object.Builtin {
	return &object.Builtin{Fn: func(args ...object.Object) object.Object {
		if len(args) != 2 && len(args) != 3 {
			return NewError("wrong number of arguments. got=%d, want=2 or 3", len(args))
		}

		pathObj, ok := args[0].(*object.String)
		if !ok {
			return NewError("argument 1 to `createPdf` must be STRING, got %s", args[0].Type())
		}

		title := "Zumbra PDF"
		bodyArg := args[1]
		if len(args) == 3 {
			titleObj, ok := args[1].(*object.String)
			if !ok {
				return NewError("argument 2 to `createPdf` must be STRING, got %s", args[1].Type())
			}
			title = titleObj.Value
			bodyArg = args[2]
		}

		content, err := objectToText(bodyArg)
		if err != nil {
			return NewError("%s", err.Error())
		}

		pdfBytes := buildMinimalPDF(title, content)
		if err := ensureDir(pathObj.Value); err != nil {
			return NewError("failed to create directory for '%s'. got %s", pathObj.Value, err)
		}

		if err := os.WriteFile(pathObj.Value, pdfBytes, 0644); err != nil {
			return NewError("failed to create pdf '%s'. got %s", pathObj.Value, err)
		}

		return NewString(pathObj.Value)
	}}
}
