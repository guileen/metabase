package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/biz/rag/llm"
)

// Skill represents a specific skill that can be executed
type Skill interface {
	Name() string
	Description() string
	Execute(input interface{}, config *llm.Config) (interface{}, error)
	Validate(input interface{}) error
}

// TemplateManager manages prompt templates and skills
type TemplateManager struct {
	templates map[string]*PromptTemplate
	skills    map[string]Skill
	mutex     sync.RWMutex
}

// PromptTemplate represents a reusable prompt template
type PromptTemplate struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	System      string                 `json:"system"`
	User        string                 `json:"user"`
	Parameters  []TemplateParameter    `json:"parameters"`
	Variables   map[string]interface{} `json:"variables"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TemplateParameter defines a parameter for the template
type TemplateParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"` // "string", "number", "boolean", "array"
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// SkillInput represents input to a skill
type SkillInput struct {
	Query      string                 `json:"query"`
	Context    string                 `json:"context,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	Options    map[string]interface{} `json:"options,omitempty"`
}

// SkillOutput represents output from a skill
type SkillOutput struct {
	Result   interface{}            `json:"result"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Duration time.Duration          `json:"duration"`
	Success  bool                   `json:"success"`
	Error    string                 `json:"error,omitempty"`
}

// NewTemplateManager creates a new template manager
func NewTemplateManager() *TemplateManager {
	tm := &TemplateManager{
		templates: make(map[string]*PromptTemplate),
		skills:    make(map[string]Skill),
	}

	// Register built-in skills
	tm.registerBuiltinSkills()

	// Load default templates
	tm.loadDefaultTemplates()

	return tm
}

// RegisterTemplate registers a new prompt template
func (tm *TemplateManager) RegisterTemplate(template *PromptTemplate) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	template.UpdatedAt = time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = time.Now()
	}

	tm.templates[template.Name] = template
	return nil
}

// GetTemplate retrieves a template by name
func (tm *TemplateManager) GetTemplate(name string) (*PromptTemplate, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	template, exists := tm.templates[name]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", name)
	}

	return template, nil
}

// ListTemplates returns all templates, optionally filtered by category
func (tm *TemplateManager) ListTemplates(category string) []*PromptTemplate {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var templates []*PromptTemplate
	for _, template := range tm.templates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}

	return templates
}

// RegisterSkill registers a new skill
func (tm *TemplateManager) RegisterSkill(skill Skill) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.skills[skill.Name()] = skill
	return nil
}

// GetSkill retrieves a skill by name
func (tm *TemplateManager) GetSkill(name string) (Skill, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	skill, exists := tm.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill '%s' not found", name)
	}

	return skill, nil
}

// ListSkills returns all registered skills
func (tm *TemplateManager) ListSkills() []Skill {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	skills := make([]Skill, 0, len(tm.skills))
	for _, skill := range tm.skills {
		skills = append(skills, skill)
	}

	return skills
}

// ExecuteSkill executes a skill with the given input
func (tm *TemplateManager) ExecuteSkill(skillName string, input *SkillInput, config *llm.Config) (*SkillOutput, error) {
	start := time.Now()

	skill, err := tm.GetSkill(skillName)
	if err != nil {
		return &SkillOutput{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, err
	}

	// Validate input
	if err := skill.Validate(input); err != nil {
		return &SkillOutput{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, err
	}

	// Execute skill
	result, err := skill.Execute(input, config)
	duration := time.Since(start)

	if err != nil {
		return &SkillOutput{
			Success:  false,
			Error:    err.Error(),
			Duration: duration,
		}, err
	}

	return &SkillOutput{
		Result:   result,
		Success:  true,
		Duration: duration,
	}, nil
}

// RenderTemplate renders a template with the given parameters
func (tm *TemplateManager) RenderTemplate(templateName string, params map[string]interface{}) (*llm.ChatMessage, error) {
	template, err := tm.GetTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Merge template variables with parameters
	variables := make(map[string]interface{})
	for k, v := range template.Variables {
		variables[k] = v
	}
	for k, v := range params {
		variables[k] = v
	}

	// Render system message
	systemMsg, err := tm.renderString(template.System, variables)
	if err != nil {
		return nil, fmt.Errorf("error rendering system message: %w", err)
	}

	return &llm.ChatMessage{
		Role:    "system",
		Content: systemMsg,
	}, nil
}

// renderString performs simple template variable substitution
func (tm *TemplateManager) renderString(template string, variables map[string]interface{}) (string, error) {
	result := template

	// Simple {{variable}} substitution
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result, nil
}

// registerBuiltinSkills registers built-in skills
func (tm *TemplateManager) registerBuiltinSkills() {
	// Register expandQuery skill
	tm.RegisterSkill(&ExpandQuerySkill{})

	// Register summarizeText skill
	tm.RegisterSkill(&SummarizeTextSkill{})

	// Register codeAnalysis skill
	tm.RegisterSkill(&CodeAnalysisSkill{})

	// Register generateTags skill
	tm.RegisterSkill(&GenerateTagsSkill{})

	// Register translateText skill
	tm.RegisterSkill(&TranslateTextSkill{})
}

// loadDefaultTemplates loads default prompt templates
func (tm *TemplateManager) loadDefaultTemplates() {
	templates := []*PromptTemplate{
		{
			Name:        "expand_query",
			Description: "Expand user query with related keywords and concepts",
			Category:    "search",
			System:      "You are a code search assistant. Help expand user queries with relevant keywords and concepts.",
			User:        "Expand this search query with related keywords, file types, and concepts: {{query}}\n\nContext: {{context}}",
			Parameters: []TemplateParameter{
				{Name: "query", Type: "string", Description: "The search query to expand", Required: true},
				{Name: "context", Type: "string", Description: "Additional context for the query", Required: false},
			},
		},
		{
			Name:        "summarize_code",
			Description: "Summarize code functionality",
			Category:    "code",
			System:      "You are a code analysis expert. Provide clear, concise summaries of code functionality.",
			User:        "Summarize this code:\n\n{{code}}\n\nFocus on: {{focus}}",
			Parameters: []TemplateParameter{
				{Name: "code", Type: "string", Description: "Code to summarize", Required: true},
				{Name: "focus", Type: "string", Description: "What to focus on in the summary", Required: false},
			},
		},
		{
			Name:        "generate_tags",
			Description: "Generate relevant tags for content",
			Category:    "content",
			System:      "You are a content tagging expert. Generate relevant tags for the given content.",
			User:        "Generate relevant tags for this content:\n\n{{content}}\n\nType: {{content_type}}",
			Parameters: []TemplateParameter{
				{Name: "content", Type: "string", Description: "Content to tag", Required: true},
				{Name: "content_type", Type: "string", Description: "Type of content", Required: false},
			},
		},
		{
			Name:        "translate_text",
			Description: "Translate text between languages",
			Category:    "text",
			System:      "You are a professional translator. Provide accurate translations while preserving meaning and context.",
			User:        "Translate this text from {{source_lang}} to {{target_lang}}:\n\n{{text}}",
			Parameters: []TemplateParameter{
				{Name: "text", Type: "string", Description: "Text to translate", Required: true},
				{Name: "source_lang", Type: "string", Description: "Source language", Required: true},
				{Name: "target_lang", Type: "string", Description: "Target language", Required: true},
			},
		},
	}

	for _, template := range templates {
		tm.RegisterTemplate(template)
	}
}

// SaveToFile saves templates to a file
func (tm *TemplateManager) SaveToFile(filename string) error {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	templates := make([]*PromptTemplate, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal templates: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// LoadFromFile loads templates from a file
func (tm *TemplateManager) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var templates []*PromptTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return fmt.Errorf("unmarshal templates: %w", err)
	}

	for _, template := range templates {
		tm.RegisterTemplate(template)
	}

	return nil
}
