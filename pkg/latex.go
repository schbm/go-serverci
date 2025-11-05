package pkg

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Engine string

const (
	EnginePDFLaTeX Engine = "pdflatex"
	EngineXeLaTeX  Engine = "xelatex"
	EngineLuaLaTeX Engine = "lualatex"
)

func CompileTeX(ctx context.Context, tex []byte, outPath string) error {
	if outPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	outDir := filepath.Dir(outPath)
	jobName := strings.TrimSuffix(filepath.Base(outPath), filepath.Ext(outPath))

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

	workDir := outDir

	srcFile, err := os.CreateTemp(workDir, jobName+"-*.tex")
	if err != nil {
		return fmt.Errorf("creating temp tex file: %w", err)
	}
	srcPath := srcFile.Name()
	if _, err := srcFile.Write(tex); err != nil {
		_ = srcFile.Close()
		_ = os.Remove(srcPath)
		return fmt.Errorf("writing tex: %w", err)
	}
	if err := srcFile.Close(); err != nil {
		_ = os.Remove(srcPath)
		return fmt.Errorf("closing temp tex file: %w", err)
	}
	defer func() { _ = os.Remove(srcPath) }()

	engine := detectMagicEngine(tex)
	if engine == "" {
		engine = EnginePDFLaTeX
	}

	if hasBinary("latexmk") {
		if err := compileWithLatexmk(ctx, workDir, outDir, jobName, engine, srcPath); err != nil {
			return err
		}
		cleanupAuxFiles(outDir, jobName)
		return nil
	}

	if !hasBinary(string(engine)) {
		return fmt.Errorf("%s not found in PATH and latexmk is unavailable", engine)
	}
	if err := compileRawEngine(ctx, workDir, outDir, jobName, engine, srcPath); err != nil {
		return err
	}
	cleanupAuxFiles(outDir, jobName)
	return nil
}

func compileWithLatexmk(ctx context.Context, workDir, outDir, jobName string, engine Engine, mainTexPath string) error {
	mode := "-pdf"
	switch engine {
	case EngineXeLaTeX:
		mode = "-pdfxe"
	case EngineLuaLaTeX:
		mode = "-pdflua"
	}

	args := []string{
		mode,
		"-synctex=1",
		"-interaction=nonstopmode",
		"-file-line-error",
		"-halt-on-error",
		"-outdir=" + outDir,
		"-jobname=" + jobName,
		mainTexPath,
	}

	out, err := runCmd(ctx, workDir, "latexmk", args...)
	if err != nil {
		return fmt.Errorf("latexmk failed: %w\n%s", err, tail(out, 2000))
	}
	return nil
}

func compileRawEngine(ctx context.Context, workDir, outDir, jobName string, engine Engine, mainTexPath string) error {
	args := []string{
		"-synctex=1",
		"-interaction=nonstopmode",
		"-file-line-error",
		"-recorder",
		"-halt-on-error",
		"-jobname", jobName,
		"-output-directory", outDir,
		mainTexPath,
	}

	var combined string
	for i := 0; i < 3; i++ {
		out, err := runCmd(ctx, workDir, string(engine), args...)
		combined += out
		if err != nil {
			return fmt.Errorf("%s pass %d failed: %w\n%s", engine, i+1, err, tail(combined, 2000))
		}
	}
	return nil
}

func detectMagicEngine(tex []byte) Engine {
	s := string(tex)
	if len(s) > 8192 {
		s = s[:8192]
	}
	re := regexp.MustCompile(`(?mi)^\s*%!TEX\s+TS-program\s*=\s*(\S+)\s*$`)
	m := re.FindStringSubmatch(s)
	if len(m) < 2 {
		return ""
	}
	prog := strings.ToLower(m[1])
	switch prog {
	case "xelatex":
		return EngineXeLaTeX
	case "lualatex":
		return EngineLuaLaTeX
	case "pdflatex":
		return EnginePDFLaTeX
	default:
		return ""
	}
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func cleanupAuxFiles(dir, jobName string) {
	exts := []string{".aux", ".log", ".out", ".toc", ".synctex.gz", ".nav", ".snm", ".fls", ".fdb_latexmk", ".bbl", ".blg"}
	for _, ext := range exts {
		_ = os.Remove(filepath.Join(dir, jobName+ext))
	}
}

func runCmd(ctx context.Context, wd string, bin string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	if wd != "" {
		cmd.Dir = wd
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.String(), err
}

func tail(s string, n int) string {
	rs := []rune(s)
	if len(rs) <= n {
		return s
	}
	return string(rs[len(rs)-n:])
}
