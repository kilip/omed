package mailx

import (
	"bytes"
	"embed"
	"fmt"
	htmltmpl "html/template"
	texttmpl "text/template"
)

//go:embed templates/*
var defaultTemplateFS embed.FS

// TemplateEngine renders HTML and plain text email bodies from template files.
type TemplateEngine struct {
	fs embed.FS
}

// NewTemplateEngine creates an engine backed by the default embedded templates.
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		fs: defaultTemplateFS,
	}
}

// NewCustomTemplateEngine allows passing a custom embed.FS.
func NewCustomTemplateEngine(fs embed.FS) *TemplateEngine {
	return &TemplateEngine{fs: fs}
}

// Render executes both .html.tmpl and .txt.tmpl for the given template name prefix.
// If the text template is missing, textBody will be empty without failing.
func (e *TemplateEngine) Render(name string, data any) (htmlBody, textBody string, err error) {
	htmlPath := fmt.Sprintf("templates/%s.html.tmpl", name)
	txtPath := fmt.Sprintf("templates/%s.txt.tmpl", name)

	// HTML template
	htmlContent, err := e.fs.ReadFile(htmlPath)
	if err != nil {
		return "", "", fmt.Errorf("mailx: read html template %s: %w", htmlPath, err)
	}

	ht, err := htmltmpl.New(name).Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("mailx: parse html template %s: %w", htmlPath, err)
	}

	var htmlBuf bytes.Buffer
	if err := ht.Execute(&htmlBuf, data); err != nil {
		return "", "", fmt.Errorf("mailx: execute html template %s: %w", htmlPath, err)
	}
	htmlBody = htmlBuf.String()

	// Text template (optional fallback)
	txtContent, err := e.fs.ReadFile(txtPath)
	if err == nil {
		tt, err := texttmpl.New(name).Parse(string(txtContent))
		if err == nil {
			var txtBuf bytes.Buffer
			if err := tt.Execute(&txtBuf, data); err == nil {
				textBody = txtBuf.String()
			}
		}
	}

	return htmlBody, textBody, nil
}
