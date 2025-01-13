// internal/pkg/templates/templates.go
package templates

import (
	"embed"
	"html/template"
	"io"
)

// Cache for parsed templates
var templateCache = make(map[string]*template.Template)

// This will be filled in on initialization
var teFS embed.FS

// TemplateData wraps all template data with common metadata
type TemplateData struct {
	Title   string
	Content interface{}
}

// TemplateEngine handles template operations
type TemplateEngine struct {
	templates *template.Template
}

// New creates a new template engine with embedded templates
func New(fs embed.FS) *TemplateEngine {
	// Set the global fs
	teFS = fs

	// Parse all templates with common functions
	tmpl := template.New("").Funcs(template.FuncMap{
		"hasTitle": func(d TemplateData) bool {
			return d.Title != ""
		},
	})

	// Parse all templates from embedded filesystem
	tmpl = template.Must(tmpl.ParseFS(fs, "src/templates/*.html", "src/templates/**/*.html"))

	return &TemplateEngine{
		templates: tmpl,
	}
}

// Render executes a template with the given data
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	// Wrap the data in our TemplateData structure
	templateData := TemplateData{
		Content: data,
	}

	// If data implements a specific interface, we can get the title
	if titled, ok := data.(interface{ GetTitle() string }); ok {
		templateData.Title = titled.GetTitle()
	}

	tmpl := template.Must(te.templates.Clone())
	tmpl = template.Must(tmpl.ParseFS(teFS, "src/templates/*.html", "src/templates/**/*.html"))
	return tmpl.ExecuteTemplate(w, name, templateData)
}
