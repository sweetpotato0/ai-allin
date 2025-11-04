package prompt

import (
	"fmt"
	"strings"
	"sync"
	"text/template"
)

// Template represents a prompt template with variables
type Template struct {
	Name     string
	Content  string
	template *template.Template
}

// NewTemplate creates a new prompt template
func NewTemplate(name, content string) (*Template, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	return &Template{
		Name:     name,
		Content:  content,
		template: tmpl,
	}, nil
}

// Render renders the template with given variables
func (t *Template) Render(vars map[string]interface{}) (string, error) {
	var buf strings.Builder
	if err := t.template.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return buf.String(), nil
}

// Manager manages prompt templates
// All operations are thread-safe using RWMutex protection
type Manager struct {
	mu        sync.RWMutex // Protects templates map
	templates map[string]*Template
}

// NewManager creates a new prompt manager
func NewManager() *Manager {
	return &Manager{
		templates: make(map[string]*Template),
	}
}

// Register adds a template to the manager
func (m *Manager) Register(tmpl *Template) error {
	if tmpl.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.templates[tmpl.Name]; exists {
		return fmt.Errorf("template %s already registered", tmpl.Name)
	}
	m.templates[tmpl.Name] = tmpl
	return nil
}

// RegisterString registers a template from string content
func (m *Manager) RegisterString(name, content string) error {
	tmpl, err := NewTemplate(name, content)
	if err != nil {
		return err
	}
	return m.Register(tmpl)
}

// Get retrieves a template by name
func (m *Manager) Get(name string) (*Template, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tmpl, ok := m.templates[name]
	if !ok {
		return nil, fmt.Errorf("template %s not found", name)
	}
	return tmpl, nil
}

// Render renders a template by name with given variables
func (m *Manager) Render(name string, vars map[string]interface{}) (string, error) {
	tmpl, err := m.Get(name)
	if err != nil {
		return "", err
	}
	return tmpl.Render(vars)
}

// List returns all registered template names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.templates))
	for name := range m.templates {
		names = append(names, name)
	}
	return names
}

// Builder helps build complex prompts
type Builder struct {
	parts []string
}

// NewBuilder creates a new prompt builder
func NewBuilder() *Builder {
	return &Builder{
		parts: make([]string, 0),
	}
}

// Add adds a part to the prompt
func (b *Builder) Add(part string) *Builder {
	b.parts = append(b.parts, part)
	return b
}

// AddFormat adds a formatted part to the prompt
func (b *Builder) AddFormat(format string, args ...interface{}) *Builder {
	b.parts = append(b.parts, fmt.Sprintf(format, args...))
	return b
}

// AddLine adds a part with a newline
func (b *Builder) AddLine(part string) *Builder {
	b.parts = append(b.parts, part+"\n")
	return b
}

// AddSection adds a section with title and content
func (b *Builder) AddSection(title, content string) *Builder {
	b.parts = append(b.parts, fmt.Sprintf("## %s\n%s\n", title, content))
	return b
}

// Build returns the final prompt string
func (b *Builder) Build() string {
	return strings.Join(b.parts, "")
}

// Reset clears all parts
func (b *Builder) Reset() *Builder {
	b.parts = make([]string, 0)
	return b
}

