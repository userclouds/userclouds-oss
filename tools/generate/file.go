package generate

import (
	"bufio"
	"bytes"
	"context"
	"go/format"
	"os"
	"text/template"

	"userclouds.com/infra/uclog"
)

// WriteFileIfChanged writes the given template to the given filename, but only if the contents would change.
func WriteFileIfChanged(ctx context.Context, filename string, templateString string, data any) {
	bs, err := os.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		uclog.Fatalf(ctx, "error reading output file: %v", err)
	}

	var buf bytes.Buffer
	w := bufio.NewWriter(&buf)
	temp := template.Must(template.New("template").Delims("<<", ">>").Parse(templateString))
	if err := temp.Execute(w, data); err != nil {
		uclog.Fatalf(ctx, "error executing template: %v", err)
	}
	w.Flush()

	p, err := format.Source(buf.Bytes())
	if err != nil {
		uclog.Fatalf(ctx, "error formatting source: %v", err)
	}

	// don't write (and cause potential conflicts) if the file contents are the same
	if bytes.Equal(bs, p) {
		return
	}

	if err := os.WriteFile(filename, p, 0644); err != nil {
		uclog.Fatalf(ctx, "error writing output file: %v", err)
	}
}
