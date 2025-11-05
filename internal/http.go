package internal

import (
	"bytes"
	"context"
	"fmt"
	"go-serverci/pkg"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func Serve(strict bool, timeout time.Duration, shutdownTimeout time.Duration) error {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "multipart/form-data") {
			http.Error(w, "Content-Type must be multipart/form-data", http.StatusUnsupportedMediaType)
			return
		}

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, fmt.Sprintf("Error parsing form: %v", err), http.StatusBadRequest)
			return
		}

		var (
			root *pkg.Root
			err  error
		)

		if yamlFile, _, yfErr := r.FormFile("ci_yaml"); yfErr == nil {
			defer yamlFile.Close()
			root, err = pkg.DecodeYaml(yamlFile)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid YAML: %v", err), http.StatusBadRequest)
				return
			}
		} else if yfErr != http.ErrMissingFile {
			http.Error(w, fmt.Sprintf("Error reading YAML file: %v", yfErr), http.StatusBadRequest)
			return
		} else {
			ciJson := r.FormValue("ci")
			if ciJson == "" {
				http.Error(w, "Missing values: provide either YAML file 'ci_yaml' or JSON field 'ci'", http.StatusBadRequest)
				return
			}
			root, err = pkg.DecodeJson(bytes.NewReader([]byte(ciJson)))
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
				return
			}
		}

		if err := root.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("ci validation error:%v\n", err), http.StatusBadRequest)
			return
		}

		tmplFile, _, err := r.FormFile("template")
		if err != nil {
			http.Error(w, fmt.Sprintf("Error reading file: %v", err), http.StatusBadRequest)
			return
		}
		defer tmplFile.Close()

		processedTmplBytes, err := pkg.ParseTempl(tmplFile, *root, strict)
		if err != nil {
			http.Error(w, fmt.Sprintf("error parsing template file: %v\n", err), http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		timestamp := time.Now().Format("20060102_150405")
		pdfBase := "doc_" + timestamp

		if err := pkg.CompileTeX(ctx, processedTmplBytes, pdfBase); err != nil {
			http.Error(w, fmt.Sprintf("error compiling tex: %v", err), http.StatusInternalServerError)
			return
		}

		pdfPath := pdfBase + ".pdf"
		f, err := os.Open(pdfPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("error opening compiled pdf: %v", err), http.StatusInternalServerError)
			_ = os.Remove(pdfPath)
			return
		}
		defer func() {
			f.Close()
			_ = os.Remove(pdfPath)
		}()

		fi, err := f.Stat()
		if err != nil {
			http.Error(w, fmt.Sprintf("error stating compiled pdf: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, pdfBase+".pdf"))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fi.Size()))
		http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	})

	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}
	go func() {
		slog.Info("starting server", "address", listener.Addr().String())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("server error occured", "error", err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	slog.Info("shutting down server gracefully", "shutdownTimeout", shutdownTimeout)

	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	return nil
}
