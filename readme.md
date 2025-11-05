# go-serverci
go-serverci is a simple CLI application that generates ITIL configuration item documents for servers out.
Since the final needed structure of the document is highly dependent on an individual basis the code must be tweaked manually.
I do not provide a simple generic way to do this.

# Usage
```
Usage: go-serverci [flags]

Render LaTeX from YAML + template, or run an HTTP server.

Flags:
  -h, --help               Show context-sensitive help.
      --serve              Start HTTP server mode. Mutually exclusive with file-based mode.
      --yaml=STRING        Path to input YAML file.
      --template=STRING    Path to LaTeX template file (.tex).
      --texout=STRING      (Optional) Path to output .tex file (file mode only).
      --pdfout=STRING      (Optional) Directory for compiled PDF (file mode only).
      --strict             (Optional) Fail on missing template keys.
      --timeout=2m         (Optional) Timeout for TeX compilation.
```
Generate your CIs either via file or using HTTP mode:
```sh
# run file mode
# file mode needs the path to your yaml manifest containing your data
# and the path to your template file
go-serverci --yaml ../test.yaml --template ../template.tex

# run http server
# the http mode support either data supplied via yaml file or via JSON
# you must always supply a template file
go-serverci --serve
curl -X POST http://localhost:8080/process -F 'ci_yaml=@test.yaml' -F 'template=@template.tex' -o out.pdf
# or 
curl -X POST https://your.api/render \
  -H "Content-Type: multipart/form-data" \
  -F "template=@./template.tex" \
  -F 'ci={(...)}' \
  -o output.pdf
```

# Docker Usage
> **Note**
> The docker image will contain a full latex installation which can be huge!

Build the image:
```sh
docker build . -t go-serverci
```
Run the container:
```
docker run -p 8080:8080 go-serverci
```


## Data and Output Modification
Two adjustments would suffice most needs:
- Adjust the available data within the file: `pkg/data.go`. If needed also adjust the validation logic.
The program will inject the `Root` data into the template. 
- The final layout tweaks and template control can be done by editing the `template.tex` file.

# Possible Future Additions
- [ ] Support for easy layout and data modification
- [ ] Escape special Tex characters