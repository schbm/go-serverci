package internal

import "time"

const APP_NAME = "go-serverci"

type CLI struct {
	Serve    bool          `help:"Start HTTP server mode. Mutually exclusive with file-based mode."`
	YAML     string        `name:"yaml"     help:"Path to input YAML file."`
	Template string        `name:"template" help:"Path to LaTeX template file (.tex)."`
	TexOut   string        `name:"texout"   help:"(Optional) Path to output .tex file (file mode only)."`
	PDFOut   string        `name:"pdfout"   help:"(Optional) Directory for compiled PDF (file mode only)."`
	Strict   bool          `help:"(Optional) Fail on missing template keys." default:"True"`
	Timeout  time.Duration `name:"timeout" help:"(Optional) Timeout for TeX compilation." default:"2m"`
}
