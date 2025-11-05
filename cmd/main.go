package main

import (
	"go-serverci/internal"
	"log/slog"
	"os"
	"time"

	"github.com/alecthomas/kong"
)

func main() {
	var cli internal.CLI
	ctx := kong.Parse(&cli,
		kong.Name(internal.APP_NAME),
		kong.Description("Render LaTeX from YAML + template, or run an HTTP server."),
	)

	fileMode := cli.YAML != "" || cli.Template != ""
	if cli.Serve && fileMode {
		ctx.Errorf("'-serve' cannot be used together with -yaml/-template flags")
		os.Exit(2)
	}
	if !cli.Serve && !fileMode {
		ctx.Errorf("Either use '-serve' OR provide both -yaml and -template")
		ctx.PrintUsage(false)
		os.Exit(2)
	}

	if !cli.Serve {
		if cli.YAML == "" || cli.Template == "" {
			ctx.Errorf("File mode requires both -yaml and -template")
			os.Exit(2)
		}
		if err := internal.RunFileMode(cli); err != nil {
			slog.Error(
				"error running file mode",
				"error", err,
			)
			os.Exit(1)
		}
		return
	}

	if cli.TexOut != "" || cli.PDFOut != "" {
		ctx.Errorf("-texout and -pdfout are only valid in file mode (omit them with -serve)")
		os.Exit(2)
	}
	if err := internal.Serve(cli.Strict, cli.Timeout, 5*time.Second); err != nil {
		slog.Error(
			"server error",
			"error", err,
		)
		os.Exit(1)
	}
}
