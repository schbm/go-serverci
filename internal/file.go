package internal

import (
	"context"
	"fmt"
	"go-serverci/pkg"
	"os"
	"time"
)

func RunFileMode(c CLI) error {
	in, err := os.Open(c.YAML)
	if err != nil {
		return fmt.Errorf("open yaml error: %w", err)
	}
	defer in.Close()

	root, err := pkg.DecodeYaml(in)
	if err != nil {
		return fmt.Errorf("open yaml decode error: %w", err)
	}

	if err := root.Validate(); err != nil {
		return fmt.Errorf("yaml validation error: %w", err)
	}

	tmplReader, err := os.Open(c.Template)
	if err != nil {
		return fmt.Errorf("template open error: %w", err)
	}
	defer tmplReader.Close()

	processedTmplBytes, err := pkg.ParseTempl(tmplReader, *root, c.Strict)
	if err != nil {
		return fmt.Errorf("template parsing error: %w", err)
	}

	if c.TexOut != "" {
		if err := os.WriteFile(c.TexOut, processedTmplBytes, 0o644); err != nil {
			return fmt.Errorf("tex output writing error: %w", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	pdfFilePath := ""
	if c.PDFOut != "" {
		pdfFilePath = c.PDFOut
	} else {
		timestamp := time.Now().Format("20060102_150405")
		pdfFilePath = "doc_" + timestamp
	}

	err = pkg.CompileTeX(ctx, processedTmplBytes, pdfFilePath)
	if err != nil {
		return fmt.Errorf("tex compilation error: %w", err)
	}

	return nil
}
