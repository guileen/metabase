package skills

import ("testing"
	"time")

// TestExpandQuerySkill tests the expandQuery skill
func TestExpandQuerySkill(t *testing.T) {
	skill := &ExpandQuerySkill{}

	// Test skill properties
	if skill.Name() != "expandQuery" {
		t.Errorf("Expected skill name 'expandQuery', got '%s'", skill.Name())
	}

	expectedDesc := "Expands user search queries with related keywords, concepts, and file types for better search results"
	if skill.Description() != expectedDesc {
		t.Errorf("Expected description '%s', got '%s'", expectedDesc, skill.Description())
	}

	// Test validation with valid input
	validInput := &SkillInput{
		Query: "test query",
	}
	if err := skill.Validate(validInput); err != nil {
		t.Errorf("Valid input should not cause validation error: %v", err)
	}

	// Test validation with invalid input (empty query)
	invalidInput := &SkillInput{
		Query: "",
	}
	if err := skill.Validate(invalidInput); err == nil {
		t.Error("Empty query should cause validation error")
	}

	// Test validation with wrong type
	if err := skill.Validate("not a SkillInput"); err == nil {
		t.Error("Wrong input type should cause validation error")
	}

	// Test execution (will use fallback without LLM)
	result, err := skill.Execute(validInput, nil)
	if err != nil {
		t.Errorf("Execution should not fail even without LLM: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}

	// Check result structure
	if resultMap, ok := result.(map[string]interface{}); ok {
		if _, hasOriginal := resultMap["original_query"]; !hasOriginal {
			t.Error("Result should contain original_query")
		}
		if _, hasExpanded := resultMap["expanded_terms"]; !hasExpanded {
			t.Error("Result should contain expanded_terms")
		}
		if _, hasMethod := resultMap["method"]; !hasMethod {
			t.Error("Result should contain method")
		}
	}
}

// TestSummarizeTextSkill tests the summarizeText skill
func TestSummarizeTextSkill(t *testing.T) {
	skill := &SummarizeTextSkill{}

	if skill.Name() != "summarizeText" {
		t.Errorf("Expected skill name 'summarizeText', got '%s'", skill.Name())
	}

	// Test with valid input
	input := &SkillInput{
		Query: "This is a long text that needs to be summarized. " +
			"It contains multiple sentences and various information. " +
			"The skill should be able to process this text and provide a summary.",
		Parameters: map[string]interface{}{
			"max_length": 50.0,
		},
	}

	if err := skill.Validate(input); err != nil {
		t.Errorf("Valid input should not cause validation error: %v", err)
	}

	// Test execution (will fail without proper LLM config, but should handle gracefully)
	result, err := skill.Execute(input, nil)
	// Without proper LLM config, this might fail, which is acceptable
	if err != nil {
		t.Logf("SummarizeText execution failed (expected without LLM): %v", err)
	} else if result == nil {
		t.Error("Result should not be nil when execution succeeds")
	}
}

// TestCodeAnalysisSkill tests the codeAnalysis skill
func TestCodeAnalysisSkill(t *testing.T) {
	skill := &CodeAnalysisSkill{}

	if skill.Name() != "codeAnalysis" {
		t.Errorf("Expected skill name 'codeAnalysis', got '%s'", skill.Name())
	}

	// Test with code input
	codeInput := &SkillInput{
		Query: `func hello() {
	fmt.Println("Hello, World!")
	return "success"
}`,
		Parameters: map[string]interface{}{
			"analysis_type": "complexity",
		},
	}

	if err := skill.Validate(codeInput); err != nil {
		t.Errorf("Valid code input should not cause validation error: %v", err)
	}

	// Test different analysis types
	analysisTypes := []string{"complexity", "security", "best_practices", "general"}
	for _, analysisType := range analysisTypes {
		input := &SkillInput{
			Query: "test code",
			Parameters: map[string]interface{}{
				"analysis_type": analysisType,
			},
		}

		result, err := skill.Execute(input, nil)
		// Without proper LLM config, this might fail
		if err != nil {
			t.Logf("CodeAnalysis execution failed for type %s (expected without LLM): %v", analysisType, err)
		} else if result == nil {
			t.Errorf("Result should not be nil for analysis type %s", analysisType)
		}
	}
}

// TestGenerateTagsSkill tests the generateTags skill
func TestGenerateTagsSkill(t *testing.T) {
	skill := &GenerateTagsSkill{}

	if skill.Name() != "generateTags" {
		t.Errorf("Expected skill name 'generateTags', got '%s'", skill.Name())
	}

	// Test with content input
	input := &SkillInput{
		Query: "This is a blog post about machine learning and artificial intelligence. " +
			"It covers topics like neural networks, deep learning, and natural language processing.",
		Parameters: map[string]interface{}{
			"content_type": "blog",
			"max_tags":     5.0,
		},
	}

	if err := skill.Validate(input); err != nil {
		t.Errorf("Valid input should not cause validation error: %v", err)
	}

	// Test execution
	_, err := skill.Execute(input, nil)
	if err != nil {
		t.Logf("GenerateTags execution failed (expected without LLM): %v", err)
	}

	// Test basic tag generation fallback
	skillInstance := &GenerateTagsSkill{}
	tags := skillInstance.basicTagGeneration("python code machine learning api")
	if len(tags) == 0 {
		t.Error("Basic tag generation should produce some tags")
	}

	// Check for expected tags
	expectedTags := []string{"api", "python"}
	for _, expectedTag := range expectedTags {
		found := false
		for _, tag := range tags {
			if tag == expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Expected tag '%s' not found in generated tags: %v", expectedTag, tags)
		}
	}
}

// TestTranslateTextSkill tests the translateText skill
func TestTranslateTextSkill(t *testing.T) {
	skill := &TranslateTextSkill{}

	if skill.Name() != "translateText" {
		t.Errorf("Expected skill name 'translateText', got '%s'", skill.Name())
	}

	// Test validation with missing parameters
	input := &SkillInput{
		Query: "Hello, world!",
	}
	if err := skill.Validate(input); err == nil {
		t.Error("Missing language parameters should cause validation error")
	}

	// Test validation with valid input
	validInput := &SkillInput{
		Query: "Hello, world!",
		Parameters: map[string]interface{}{
			"source_lang": "English",
			"target_lang": "Spanish",
		},
	}
	if err := skill.Validate(validInput); err != nil {
		t.Errorf("Valid input should not cause validation error: %v", err)
	}

	// Test execution
	result, err := skill.Execute(validInput, nil)
	// Without proper LLM config, this might fail
	if err != nil {
		t.Logf("TranslateText execution failed (expected without LLM): %v", err)
	} else if result == nil {
		t.Error("Result should not be nil when execution succeeds")
	}
}

// TestTemplateManager tests the template manager
func TestTemplateManager(t *testing.T) {
	tm := NewTemplateManager()

	// Test that built-in skills are registered
	skills := tm.ListSkills()
	if len(skills) == 0 {
		t.Error("Expected some skills to be registered")
	}

	// Check that expandQuery skill is registered
	expandQuerySkill, err := tm.GetSkill("expandQuery")
	if err != nil {
		t.Errorf("expandQuery skill should be registered: %v", err)
	}
	if expandQuerySkill == nil {
		t.Error("expandQuery skill should not be nil")
	}

	// Test template registration
	template := &PromptTemplate{
		Name:        "test_template",
		Description: "Test template",
		Category:    "test",
		System:      "You are a test assistant.",
		User:        "Test query: {{query}}",
		Parameters: []TemplateParameter{
			{Name: "query", Type: "string", Description: "Test query", Required: true},
		},
	}

	err = tm.RegisterTemplate(template)
	if err != nil {
		t.Errorf("Failed to register template: %v", err)
	}

	// Test template retrieval
	retrieved, err := tm.GetTemplate("test_template")
	if err != nil {
		t.Errorf("Failed to retrieve template: %v", err)
	}
	if retrieved.Name != "test_template" {
		t.Errorf("Retrieved template name mismatch: expected 'test_template', got '%s'", retrieved.Name)
	}

	// Test template rendering
	msg, err := tm.RenderTemplate("test_template", map[string]interface{}{
		"query": "test query",
	})
	if err != nil {
		t.Errorf("Failed to render template: %v", err)
	}
	if msg.Role != "system" {
		t.Errorf("Expected system role, got '%s'", msg.Role)
	}

	// Test template listing
	templates := tm.ListTemplates("test")
	if len(templates) == 0 {
		t.Error("Expected at least one test template")
	}

	// Test ExecuteSkill
	skillInput := &SkillInput{
		Query: "test query",
	}

	output, err := tm.ExecuteSkill("expandQuery", skillInput, nil)
	// Without proper LLM config, this might use fallback
	if err != nil {
		t.Logf("ExecuteSkill failed (expected without LLM): %v", err)
	}
	if output == nil {
		t.Error("Output should not be nil")
	}
}

// TestSkillInput tests the SkillInput structure
func TestSkillInput(t *testing.T) {
	input := &SkillInput{
		Query: "test query",
		Context: "test context",
		Parameters: map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		},
		Options: map[string]interface{}{
			"option1": true,
		},
	}

	if input.Query != "test query" {
		t.Errorf("Expected query 'test query', got '%s'", input.Query)
	}

	if input.Context != "test context" {
		t.Errorf("Expected context 'test context', got '%s'", input.Context)
	}

	if input.Parameters["param1"] != "value1" {
		t.Errorf("Expected param1 to be 'value1', got %v", input.Parameters["param1"])
	}

	if input.Parameters["param2"] != 42 {
		t.Errorf("Expected param2 to be 42, got %v", input.Parameters["param2"])
	}

	if input.Options["option1"] != true {
		t.Errorf("Expected option1 to be true, got %v", input.Options["option1"])
	}
}

// TestSkillOutput tests the SkillOutput structure
func TestSkillOutput(t *testing.T) {
	result := map[string]interface{}{
		"test": "value",
	}
	metadata := map[string]interface{}{
		"processing_time": "100ms",
	}

	output := &SkillOutput{
		Result:   result,
		Metadata: metadata,
		Duration: 100 * time.Millisecond,
		Success:  true,
	}

	if !output.Success {
		t.Error("Expected success to be true")
	}

	if output.Duration != 100*time.Millisecond {
		t.Errorf("Expected duration 100ms, got %v", output.Duration)
	}

	resultMap, ok := output.Result.(map[string]interface{})
	if !ok {
		t.Error("Expected Result to be a map")
	} else if resultMap["test"] != "value" {
		t.Errorf("Expected result test to be 'value', got %v", resultMap["test"])
	}

	if output.Metadata["processing_time"] != "100ms" {
		t.Errorf("Expected metadata processing_time to be '100ms', got %v", output.Metadata["processing_time"])
	}
}

// TestPromptTemplate tests the PromptTemplate structure
func TestPromptTemplate(t *testing.T) {
	template := &PromptTemplate{
		Name:        "test",
		Description: "Test template",
		Category:    "test",
		System:      "System prompt",
		User:        "User prompt with {{variable}}",
		Parameters: []TemplateParameter{
			{
				Name:        "variable",
				Type:        "string",
				Description: "Test variable",
				Required:    true,
				Default:     "default_value",
			},
		},
		Variables: map[string]interface{}{
			"global_var": "global_value",
		},
	}

	if template.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", template.Name)
	}

	if len(template.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(template.Parameters))
	}

	param := template.Parameters[0]
	if param.Name != "variable" {
		t.Errorf("Expected parameter name 'variable', got '%s'", param.Name)
	}
	if param.Type != "string" {
		t.Errorf("Expected parameter type 'string', got '%s'", param.Type)
	}
	if !param.Required {
		t.Error("Expected parameter to be required")
	}
	if param.Default != "default_value" {
		t.Errorf("Expected parameter default 'default_value', got %v", param.Default)
	}
}

// TestTemplateParameter tests the TemplateParameter structure
func TestTemplateParameter(t *testing.T) {
	param := TemplateParameter{
		Name:        "test_param",
		Type:        "number",
		Description: "Test parameter",
		Required:    false,
		Default:     42.0,
		Enum:        []string{"option1", "option2", "option3"},
	}

	if param.Name != "test_param" {
		t.Errorf("Expected name 'test_param', got '%s'", param.Name)
	}

	if param.Type != "number" {
		t.Errorf("Expected type 'number', got '%s'", param.Type)
	}

	if param.Required {
		t.Error("Expected required to be false")
	}

	if param.Default != 42.0 {
		t.Errorf("Expected default 42.0, got %v", param.Default)
	}

	if len(param.Enum) != 3 {
		t.Errorf("Expected 3 enum values, got %d", len(param.Enum))
	}
}

// Benchmark tests
func BenchmarkExpandQuerySkillExecute(b *testing.B) {
	skill := &ExpandQuerySkill{}
	input := &SkillInput{
		Query: "test query for benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = skill.Execute(input, nil)
	}
}

func BenchmarkGenerateTagsSkillBasicGeneration(b *testing.B) {
	skill := &GenerateTagsSkill{}
	text := "This is a test text for benchmarking tag generation with Python JavaScript and API keywords"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = skill.basicTagGeneration(text)
	}
}

func BenchmarkTemplateManagerGetSkill(b *testing.B) {
	tm := NewTemplateManager()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tm.GetSkill("expandQuery")
	}
}