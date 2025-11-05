package pkg

import (
	"bytes"
	"io"
	"strings"
	"text/template"
)

func ParseTempl(texReader io.Reader, root Root, strict bool) ([]byte, error) {
	texBytes, err := io.ReadAll(texReader)
	if err != nil {
		return nil, err
	}
	tex := string(texBytes)

	funcMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}

	tmpl := template.New("latex").
		Delims("<<", ">>").
		Funcs(funcMap)

	if strict {
		tmpl = tmpl.Option("missingkey=error")
	} else {
		tmpl = tmpl.Option("missingkey=zero")
	}

	tmpl, err = tmpl.Parse(tex)
	if err != nil {
		return nil, err
	}

	var processedTmplBuff bytes.Buffer
	if err := tmpl.Execute(&processedTmplBuff, &root); err != nil {
		return nil, err
	}

	return processedTmplBuff.Bytes(), nil
}
