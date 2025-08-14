// src/pkg/templates/templates.go
package templates

import (
	"html/template"
	"io"
	"path/filepath"
	"sync"
)

// TemplateEngine handles template operations with thread-safe caching
type TemplateEngine struct {
	rootDir string
	cache   map[string]*template.Template
	mutex   sync.RWMutex
	funcs   template.FuncMap
}

// New creates a new template engine
func New(rootDir string) *TemplateEngine {
	return &TemplateEngine{
		rootDir: rootDir,
		cache:   make(map[string]*template.Template),
		funcs:   make(template.FuncMap),
	}
}

// AddFunc adds a custom template function
func (te *TemplateEngine) AddFunc(name string, fn interface{}) {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	te.funcs[name] = fn
}

// Render executes a template with the given data
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	tmpl, err := te.getTemplate(name)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, data)
}

// getTemplate retrieves or parses a template with thread-safe caching
func (te *TemplateEngine) getTemplate(name string) (*template.Template, error) {
	te.mutex.RLock()
	tmpl, exists := te.cache[name]
	te.mutex.RUnlock()

	if exists {
		return tmpl, nil
	}

	te.mutex.Lock()
	defer te.mutex.Unlock()

	// Double-check after acquiring write lock
	if tmpl, exists = te.cache[name]; exists {
		return tmpl, nil
	}

	tmpl, err := template.New(filepath.Base(name)).
		Funcs(te.funcs).
		ParseFiles(filepath.Join(te.rootDir, name))

	if err != nil {
		return nil, err
	}

	te.cache[name] = tmpl
	return tmpl, nil
}
