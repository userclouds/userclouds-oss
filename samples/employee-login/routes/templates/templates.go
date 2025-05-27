package templates

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"path/filepath"

	"userclouds.com/infra/uchttp"
)

// RenderTemplate renders an HTML template
func RenderTemplate(ctx context.Context, w http.ResponseWriter, tmpl string, data any) {
	cwd, _ := os.Getwd()
	t, err := template.ParseFiles(filepath.Join(cwd, "./routes/"+tmpl+"/"+tmpl+".html"))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	err = t.Execute(w, data)
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
}
