// internal/pkg/templates/templates.go
package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
)

// Cache for parsed templates
var templateCache = make(map[string]*template.Template)

// PageMetadata defines common metadata for all pages
type PageMetadata struct {
	Title string
	// Add other common metadata fields here
}

// TemplateEngine handles template operations
type TemplateEngine struct {
	templateFS embed.FS
	cache      map[string]*template.Template
	funcs      template.FuncMap
}

// New creates a new template engine
func New(templateFS embed.FS) *TemplateEngine {
	return &TemplateEngine{
		templateFS: templateFS,
		cache:      make(map[string]*template.Template),
		funcs:      template.FuncMap{},
	}
}

// AddFunc adds a custom template function
func (te *TemplateEngine) AddFunc(name string, fn interface{}) {
	te.funcs[name] = fn
}

// Render executes a template with the given data
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	tmpl, err := te.getTemplate(name)
	if err != nil {
		return fmt.Errorf("getting template %s: %w", name, err)
	}

	return tmpl.ExecuteTemplate(w, "base", data)
}

// getTemplate retrieves or creates a template
func (te *TemplateEngine) getTemplate(name string) (*template.Template, error) {
	// Check cache first
	if tmpl, ok := te.cache[name]; ok {
		return tmpl, nil
	}

	// Get base template
	baseContent, err := te.templateFS.ReadFile("src/templates/base.html")
	if err != nil {
		return nil, fmt.Errorf("reading base template: %w", err)
	}

	// Create new template with base
	tmpl := template.New("base").Funcs(te.funcs)
	tmpl, err = tmpl.Parse(string(baseContent))
	if err != nil {
		return nil, fmt.Errorf("parsing base template: %w", err)
	}

	// Get and parse the requested template
	content, err := te.templateFS.ReadFile(filepath.Join("src/templates", name))
	if err != nil {
		return nil, fmt.Errorf("reading template %s: %w", name, err)
	}

	tmpl, err = tmpl.Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing template %s: %w", name, err)
	}

	// Cache the template
	te.cache[name] = tmpl
	return tmpl, nil
}
