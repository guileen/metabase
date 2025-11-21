package skills

import ("fmt"
	"regexp"
	"strings"

	"github.com/guileen/metabase/pkg/biz/rag/llm")

// ExpandQuerySkill expands search queries with related keywords and concepts
type ExpandQuerySkill struct{}

func (s *ExpandQuerySkill) Name() string {
	return "expandQuery"
}

func (s *ExpandQuerySkill) Description() string {
	return "Expands user search queries with related keywords, concepts, and file types for better search results"
}

func (s *ExpandQuerySkill) Validate(input interface{}) error {
	skillInput, ok := input.(*SkillInput)
	if !ok {
		return fmt.Errorf("input must be SkillInput")
	}

	if skillInput.Query == "" {
		return fmt.Errorf("query is required")
	}

	return nil
}

func (s *ExpandQuerySkill) Execute(input interface{}, config *llm.Config) (interface{}, error) {
	skillInput := input.(*SkillInput)

	// Build prompt for query expansion
	prompt := fmt.Sprintf(`You are a code search assistant. Expand the user's search query with related keywords, concepts, and file types.

Original query: %s
Context: %s

Please provide:
1. 5-8 related keywords and concepts
2. 2-3 relevant file types or extensions
3. 2-3 relevant directory or module names

Output as a comma-separated list. Be concise and focus on technical terms.`,
		skillInput.Query,
		skillInput.Context)

	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a code search optimization expert."},
		{Role: "user", Content: prompt},
	}

	response, err := llm.ChatCompletion(messages, config)
	if err != nil {
		// Fallback to basic keyword extraction
		return s.basicKeywordExpansion(skillInput.Query), nil
	}

	if len(response.Choices) == 0 {
		return s.basicKeywordExpansion(skillInput.Query), nil
	}

	content := response.Choices[0].Message.Content

	// Parse and clean the expansion
	expanded := s.parseExpansionResult(content)

	result := map[string]interface{}{
		"original_query": skillInput.Query,
		"expanded_terms": expanded,
		"raw_response":   content,
		"method":         "llm",
	}

	return result, nil
}

func (s *ExpandQuerySkill) basicKeywordExpansion(query string) []string {
	// Simple keyword extraction as fallback
	words := strings.Fields(strings.ToLower(query))
	var expanded []string

	// Add common programming-related terms
	programmingTerms := map[string][]string{
		"function":    {"method", "procedure", "func", "def"},
		"class":       {"struct", "type", "interface", "object"},
		"database":    {"db", "sql", "table", "schema", "model"},
		"api":         {"endpoint", "route", "handler", "service"},
		"test":        {"spec", "unit", "integration", "mock"},
		"config":      {"settings", "env", "properties", "options"},
		"error":       {"exception", "failure", "bug", "issue"},
		"component":   {"module", "part", "element", "piece"},
	}

	for _, word := range words {
		expanded = append(expanded, word)
		if relatedTerms, exists := programmingTerms[word]; exists {
			expanded = append(expanded, relatedTerms...)
		}
	}

	// Add common file extensions
	extensions := []string{".go", ".js", ".ts", ".py", ".java", ".cpp", ".md", ".json", ".yaml"}
	expanded = append(expanded, extensions...)

	return expanded
}

func (s *ExpandQuerySkill) parseExpansionResult(content string) []string {
	// Clean and split the response
	content = strings.TrimSpace(content)
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")

	// Split by common separators
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})

	var expanded []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && len(part) > 1 {
			expanded = append(expanded, part)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	var unique []string
	for _, term := range expanded {
		if !seen[term] {
			seen[term] = true
			unique = append(unique, term)
		}
	}

	return unique
}

// SummarizeTextSkill summarizes text content
type SummarizeTextSkill struct{}

func (s *SummarizeTextSkill) Name() string {
	return "summarizeText"
}

func (s *SummarizeTextSkill) Description() string {
	return "Summarizes text content concisely"
}

func (s *SummarizeTextSkill) Validate(input interface{}) error {
	skillInput, ok := input.(*SkillInput)
	if !ok {
		return fmt.Errorf("input must be SkillInput")
	}

	if skillInput.Query == "" {
		return fmt.Errorf("text to summarize is required")
	}

	return nil
}

func (s *SummarizeTextSkill) Execute(input interface{}, config *llm.Config) (interface{}, error) {
	skillInput := input.(*SkillInput)

	maxLength := 200
	if ml, ok := skillInput.Parameters["max_length"].(float64); ok {
		maxLength = int(ml)
	}

	prompt := fmt.Sprintf(`Summarize the following text concisely:

Text: %s

Provide a summary of approximately %d words that captures the main points.`,
		skillInput.Query,
		maxLength/5) // Rough estimate of words

	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a text summarization expert. Create clear, concise summaries."},
		{Role: "user", Content: prompt},
	}

	response, err := llm.ChatCompletion(messages, config)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	result := map[string]interface{}{
		"summary":   response.Choices[0].Message.Content,
		"original":  skillInput.Query,
		"tokens":    response.Usage.TotalTokens,
	}

	return result, nil
}

// CodeAnalysisSkill analyzes code and provides insights
type CodeAnalysisSkill struct{}

func (s *CodeAnalysisSkill) Name() string {
	return "codeAnalysis"
}

func (s *CodeAnalysisSkill) Description() string {
	return "Analyzes code and provides insights about functionality, complexity, and improvements"
}

func (s *CodeAnalysisSkill) Validate(input interface{}) error {
	skillInput, ok := input.(*SkillInput)
	if !ok {
		return fmt.Errorf("input must be SkillInput")
	}

	if skillInput.Query == "" {
		return fmt.Errorf("code to analyze is required")
	}

	return nil
}

func (s *CodeAnalysisSkill) Execute(input interface{}, config *llm.Config) (interface{}, error) {
	skillInput := input.(*SkillInput)

	analysisType := "general"
	if at, ok := skillInput.Parameters["analysis_type"].(string); ok {
		analysisType = at
	}

	var prompt string
	switch analysisType {
	case "complexity":
		prompt = fmt.Sprintf(`Analyze the complexity of this code:

%s

Focus on:
1. Time complexity
2. Space complexity
3. Potential bottlenecks
4. Optimization opportunities`, skillInput.Query)
	case "security":
		prompt = fmt.Sprintf(`Perform a security analysis of this code:

%s

Identify:
1. Security vulnerabilities
2. Input validation issues
3. Authentication/authorization concerns
4. Data exposure risks`, skillInput.Query)
	case "best_practices":
		prompt = fmt.Sprintf(`Review this code for best practices:

%s

Evaluate:
1. Code style and conventions
2. Error handling
3. Documentation
4. Maintainability
5. Testability`, skillInput.Query)
	default:
		prompt = fmt.Sprintf(`Analyze this code and provide insights:

%s

Include:
1. What the code does
2. Key functions and their purposes
3. Data flow
4. Dependencies
5. Potential improvements`, skillInput.Query)
	}

	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a senior software engineer. Provide thorough code analysis with actionable insights."},
		{Role: "user", Content: prompt},
	}

	response, err := llm.ChatCompletion(messages, config)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	result := map[string]interface{}{
		"analysis":     response.Choices[0].Message.Content,
		"code":         skillInput.Query,
		"type":         analysisType,
		"tokens":       response.Usage.TotalTokens,
	}

	return result, nil
}

// GenerateTagsSkill generates relevant tags for content
type GenerateTagsSkill struct{}

func (s *GenerateTagsSkill) Name() string {
	return "generateTags"
}

func (s *GenerateTagsSkill) Description() string {
	return "Generates relevant tags for content categorization"
}

func (s *GenerateTagsSkill) Validate(input interface{}) error {
	skillInput, ok := input.(*SkillInput)
	if !ok {
		return fmt.Errorf("input must be SkillInput")
	}

	if skillInput.Query == "" {
		return fmt.Errorf("content is required")
	}

	return nil
}

func (s *GenerateTagsSkill) Execute(input interface{}, config *llm.Config) (interface{}, error) {
	skillInput := input.(*SkillInput)

	contentType := "general"
	if ct, ok := skillInput.Parameters["content_type"].(string); ok {
		contentType = ct
	}

	maxTags := 10
	if mt, ok := skillInput.Parameters["max_tags"].(float64); ok {
		maxTags = int(mt)
	}

	prompt := fmt.Sprintf(`Generate relevant tags for this content:

Content: %s
Type: %s

Generate %d relevant tags. Consider:
1. Main topics and themes
2. Technologies and frameworks
3. Programming languages
4. Concepts and methodologies
5. Use cases and applications

Output tags as a comma-separated list.`,
		skillInput.Query,
		contentType,
		maxTags)

	messages := []llm.ChatMessage{
		{Role: "system", Content: "You are a content categorization expert. Generate precise, relevant tags."},
		{Role: "user", Content: prompt},
	}

	response, err := llm.ChatCompletion(messages, config)
	if err != nil {
		return s.basicTagGeneration(skillInput.Query), nil
	}

	if len(response.Choices) == 0 {
		return s.basicTagGeneration(skillInput.Query), nil
	}

	content := response.Choices[0].Message.Content
	tags := s.parseTags(content)

	result := map[string]interface{}{
		"tags":        tags,
		"content":     skillInput.Query,
		"type":        contentType,
		"method":      "llm",
		"count":       len(tags),
	}

	return result, nil
}

func (s *GenerateTagsSkill) basicTagGeneration(content string) []string {
	content = strings.ToLower(content)

	// Common technology keywords
	techKeywords := []string{
		"api", "database", "frontend", "backend", "javascript", "python", "golang",
		"react", "vue", "angular", "nodejs", "docker", "kubernetes", "aws", "azure",
		"microservices", "rest", "graphql", "sql", "nosql", "cache", "security",
		"testing", "devops", "monitoring", "logging", "authentication", "authorization",
	}

	var tags []string
	for _, keyword := range techKeywords {
		if strings.Contains(content, keyword) {
			tags = append(tags, keyword)
		}
	}

	// Add content type tags
	if strings.Contains(content, "function") || strings.Contains(content, "method") {
		tags = append(tags, "function")
	}
	if strings.Contains(content, "class") || strings.Contains(content, "struct") {
		tags = append(tags, "class")
	}
	if strings.Contains(content, "test") || strings.Contains(content, "spec") {
		tags = append(tags, "testing")
	}

	// Limit to 10 tags
	if len(tags) > 10 {
		tags = tags[:10]
	}

	return tags
}

func (s *GenerateTagsSkill) parseTags(content string) []string {
	content = strings.TrimSpace(content)
	parts := strings.FieldsFunc(content, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n'
	})

	var tags []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" && len(part) > 1 {
			tags = append(tags, strings.ToLower(part))
		}
	}

	return tags
}

// TranslateTextSkill translates text between languages
type TranslateTextSkill struct{}

func (s *TranslateTextSkill) Name() string {
	return "translateText"
}

func (s *TranslateTextSkill) Description() string {
	return "Translates text between different languages"
}

func (s *TranslateTextSkill) Validate(input interface{}) error {
	skillInput, ok := input.(*SkillInput)
	if !ok {
		return fmt.Errorf("input must be SkillInput")
	}

	if skillInput.Query == "" {
		return fmt.Errorf("text to translate is required")
	}

	sourceLang, _ := skillInput.Parameters["source_lang"].(string)
	targetLang, _ := skillInput.Parameters["target_lang"].(string)

	if sourceLang == "" || targetLang == "" {
		return fmt.Errorf("both source_lang and target_lang are required")
	}

	return nil
}

func (s *TranslateTextSkill) Execute(input interface{}, config *llm.Config) (interface{}, error) {
	skillInput := input.(*SkillInput)

	sourceLang := skillInput.Parameters["source_lang"].(string)
	targetLang := skillInput.Parameters["target_lang"].(string)

	prompt := fmt.Sprintf(`Translate the following text from %s to %s:

%s

Provide a natural, accurate translation that preserves the original meaning and context.`,
		sourceLang,
		targetLang,
		skillInput.Query)

	messages := []llm.ChatMessage{
		{Role: "system", Content: fmt.Sprintf("You are a professional translator specializing in %s to %s translation.", sourceLang, targetLang)},
		{Role: "user", Content: prompt},
	}

	response, err := llm.ChatCompletion(messages, config)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	result := map[string]interface{}{
		"translated_text": response.Choices[0].Message.Content,
		"original_text":   skillInput.Query,
		"source_lang":     sourceLang,
		"target_lang":     targetLang,
		"tokens":          response.Usage.TotalTokens,
	}

	return result, nil
}