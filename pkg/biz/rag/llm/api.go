package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func toks(s string) []string {
	s = strings.ToLower(s)
	f := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return ' '
	}
	return strings.Fields(strings.Map(f, s))
}

func readAll(resp *http.Response) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}

func head(b []byte) string {
	s := string(b)
	if len(s) > 200 {
		return s[:200]
	}
	return s
}

func resolvePath(base, def string, tail string) string {
	if def != "" {
		return def
	}
	bt := strings.TrimRight(base, "/")
	if strings.HasSuffix(bt, "/v1") {
		return tail
	}
	return "/v1" + tail
}

// Config holds LLM configuration
type Config struct {
	BaseURL        string
	APIKey         string
	APIMode        string // "OpenAI", "Custom"
	Model          string
	EmbeddingModel string
	RerankModel    string
	Timeout        time.Duration
	RetryAttempts  int
	RetryDelay     time.Duration
}

// ModelInfo contains information about supported models
type ModelInfo struct {
	Name      string
	Type      string // "chat", "embedding", "rerank"
	MaxTokens int
	Provider  string
	Endpoint  string
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`
}

// ChatCompletionResponse represents a chat completion response
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// getDefaultConfig returns default LLM configuration
func getDefaultConfig() *Config {
	return &Config{
		APIMode:        os.Getenv("LLM_API_MODE"),
		BaseURL:        os.Getenv("LLM_BASE_URL"),
		APIKey:         os.Getenv("LLM_API_KEY"),
		Model:          os.Getenv("LLM_MODEL"),
		EmbeddingModel: os.Getenv("LLM_EMBEDDING_MODEL"),
		RerankModel:    os.Getenv("LLM_RERANK_MODEL"),
		Timeout:        60 * time.Second,
		RetryAttempts:  3,
		RetryDelay:     time.Second,
	}
}

// GetSupportedModels returns a list of supported models
func GetSupportedModels() []ModelInfo {
	return []ModelInfo{
		// Qwen Models via SiliconFlow
		{
			Name:      "Qwen/Qwen3-8B",
			Type:      "chat",
			Provider:  "SiliconFlow",
			MaxTokens: 8192,
		},
		{
			Name:      "Qwen/Qwen2.5-7B-Instruct",
			Type:      "chat",
			Provider:  "SiliconFlow",
			MaxTokens: 32768,
		},
		{
			Name:      "Qwen/Qwen2.5-14B-Instruct",
			Type:      "chat",
			Provider:  "SiliconFlow",
			MaxTokens: 32768,
		},

		// BGE Embedding Models
		{
			Name:     "BAAI/bge-m3",
			Type:     "embedding",
			Provider: "SiliconFlow",
		},
		{
			Name:     "BAAI/bge-large-zh-v1.5",
			Type:     "embedding",
			Provider: "SiliconFlow",
		},

		// BGE Rerank Models
		{
			Name:     "BAAI/bge-reranker-v2-m3",
			Type:     "rerank",
			Provider: "SiliconFlow",
		},

		// OpenAI Compatible Models
		{
			Name:     "text-embedding-3-small",
			Type:     "embedding",
			Provider: "OpenAI",
		},
		{
			Name:     "gpt-3.5-turbo",
			Type:     "chat",
			Provider: "OpenAI",
		},
		{
			Name:     "gpt-4",
			Type:     "chat",
			Provider: "OpenAI",
		},
	}
}

// getModelInfo returns information about a specific model
func getModelInfo(modelName string) *ModelInfo {
	models := GetSupportedModels()
	for _, model := range models {
		if model.Name == modelName {
			return &model
		}
	}
	return nil
}

// makeHTTPRequest makes an HTTP request with retry logic
func makeHTTPRequest(method, url string, headers map[string]string, body []byte) (*http.Response, error) {
	config := getDefaultConfig()

	client := &http.Client{
		Timeout: config.Timeout,
	}

	var lastErr error
	for attempt := 0; attempt < config.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(config.RetryDelay)
		}

		req, err := http.NewRequest(method, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}

		for key, value := range headers {
			req.Header.Set(key, value)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// EnhancedEmbeddings provides enhanced embedding functionality with retry and better error handling
func EnhancedEmbeddings(inputs []string, config *Config) ([][]float64, error) {
	if config == nil {
		config = getDefaultConfig()
	}

	if config.BaseURL == "" || config.APIKey == "" {
		return nil, fmt.Errorf("embedding not configured: missing BaseURL or APIKey")
	}

	model := config.EmbeddingModel
	if model == "" {
		model = "text-embedding-3-small"
	}

	// Validate model is supported
	if modelInfo := getModelInfo(model); modelInfo != nil && modelInfo.Type != "embedding" {
		return nil, fmt.Errorf("model %s is not an embedding model", model)
	}

	// Get token limit for the model (default 8192 for most models)
	maxTokens := getModelTokenLimit(model)

	// Use safety margin (60% of limit to be very conservative)
	safetyMaxTokens := int(float64(maxTokens) * 0.6)
	fmt.Printf("[LLM] Using safety token limit: %d (was %d)\n", safetyMaxTokens, maxTokens)

	// Estimate tokens and chunk if necessary
	chunks, err := chunkInputsByTokens(inputs, safetyMaxTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to chunk inputs: %w", err)
	}

	fmt.Printf("[LLM] Processing %d inputs in %d chunks (max tokens: %d)\n", len(inputs), len(chunks), maxTokens)

	var allEmbeddings [][]float64
	for i, chunk := range chunks {
		fmt.Printf("[LLM] Processing chunk %d/%d with %d inputs\n", i+1, len(chunks), len(chunk))

		embeddings, err := processEmbeddingChunk(chunk, model, config)
		if err != nil {
			return nil, fmt.Errorf("failed to process chunk %d: %w", i+1, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	fmt.Printf("[LLM] Successfully generated %d embeddings\n", len(allEmbeddings))
	return allEmbeddings, nil
}

// getModelTokenLimit returns the token limit for a given model
func getModelTokenLimit(model string) int {
	// Model-specific token limits
	limits := map[string]int{
		"BAAI/bge-m3":            8192,
		"BAAI/bge-large-zh-v1.5": 8192,
		"text-embedding-3-small": 8192,
		"text-embedding-3-large": 8192,
		"text-embedding-ada-002": 8192,
	}

	if limit, exists := limits[model]; exists {
		return limit
	}

	// Default limit
	return 8192
}

// estimateTokens estimates the number of tokens in a string
func estimateTokens(text string) int {
	// Very conservative estimation: ~2.5 characters per token to account for different languages
	// This is extremely conservative to avoid hitting token limits with code files
	return (len(text) + 2) / 2
}

// chunkInputsByTokens splits inputs into chunks that don't exceed token limits
func chunkInputsByTokens(inputs []string, maxTokens int) ([][]string, error) {
	if len(inputs) == 0 {
		return [][]string{}, nil
	}

	var chunks [][]string
	var currentChunk []string
	var currentTokens int

	for _, input := range inputs {
		inputTokens := estimateTokens(input)

		// If single input exceeds limit, truncate it extremely aggressively
		if inputTokens > maxTokens {
			fmt.Printf("[LLM] Warning: Single input exceeds token limit (%d > %d), truncating extremely aggressively\n", inputTokens, maxTokens)
			// Start with extremely conservative estimation (1.5 chars per token)
			maxChars := int(float64(maxTokens) * 1.5)
			if len(input) > maxChars {
				input = input[:maxChars]
			}
			inputTokens = estimateTokens(input)

			// If still too large, truncate even more aggressively
			iterations := 0
			for inputTokens > maxTokens && len(input) > 20 && iterations < 10 {
				iterations++
				// Reduce by larger amounts each iteration
				reductionFactor := 0.5 // Cut in half
				if inputTokens > maxTokens*3 {
					reductionFactor = 0.3 // Cut to 30% if way over limit
				}
				newLen := int(float64(len(input)) * reductionFactor)
				if newLen < 20 {
					newLen = 20
				}
				input = input[:newLen]
				inputTokens = estimateTokens(input)
				fmt.Printf("[LLM] Truncation %d: %d chars, estimated %d tokens\n", iterations, len(input), inputTokens)
			}

			// Final safety check - if still too large, use only first line or minimal snippet
			if inputTokens > maxTokens {
				lines := strings.Split(input, "\n")
				if len(lines) > 1 {
					input = lines[0] // Use only first line
				} else {
					// Use only first 50 characters as last resort
					if len(input) > 50 {
						input = input[:50]
					}
				}
				inputTokens = estimateTokens(input)
				fmt.Printf("[LLM] Emergency truncation to %d chars, estimated %d tokens\n", len(input), inputTokens)
			}
		}

		// Check if adding this input would exceed the limit
		if currentTokens+inputTokens > maxTokens && len(currentChunk) > 0 {
			chunks = append(chunks, currentChunk)
			currentChunk = []string{input}
			currentTokens = inputTokens
		} else {
			currentChunk = append(currentChunk, input)
			currentTokens += inputTokens
		}
	}

	// Add the last chunk if it's not empty
	if len(currentChunk) > 0 {
		chunks = append(chunks, currentChunk)
	}

	return chunks, nil
}

// processEmbeddingChunk processes a single chunk of inputs
func processEmbeddingChunk(chunk []string, model string, config *Config) ([][]float64, error) {
	path := resolvePath(config.BaseURL, os.Getenv("LLM_EMBEDDING_PATH"), "/embeddings")
	url := strings.TrimRight(config.BaseURL, "/") + path

	var input any
	if len(chunk) == 1 {
		input = chunk[0]
	} else {
		input = chunk
	}

	// Log chunk details
	totalChars := 0
	for _, text := range chunk {
		totalChars += len(text)
	}
	estimatedTokens := estimateTokens(strings.Join(chunk, " "))
	fmt.Printf("[LLM] Chunk details: %d inputs, %d chars, ~%d tokens\n", len(chunk), totalChars, estimatedTokens)

	body := map[string]interface{}{
		"model": model,
		"input": input,
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}

	resp, err := makeHTTPRequest("POST", url, headers, buf)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := readAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		// Enhanced error handling with specific token limit error
		if resp.StatusCode == 413 {
			return nil, fmt.Errorf("token limit exceeded: %s (estimated tokens: %d, chunk size: %d inputs, %d chars)",
				head(b), estimatedTokens, len(chunk), totalChars)
		}
		return nil, fmt.Errorf("embedding HTTP error: %d %s", resp.StatusCode, head(b))
	}

	var response struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Model string `json:"model"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, head(b))
	}

	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	result := make([][]float64, len(response.Data))
	for i, item := range response.Data {
		result[i] = item.Embedding
	}

	// Log success with actual token usage
	fmt.Printf("[LLM] Chunk processed successfully: %d embeddings, %d prompt tokens, %d total tokens\n",
		len(result), response.Usage.PromptTokens, response.Usage.TotalTokens)

	return result, nil
}

// Embeddings provides backward compatibility
func Embeddings(inputs []string) ([][]float64, error) {
	return EnhancedEmbeddings(inputs, nil)
}

// EnhancedRerank provides enhanced reranking functionality
func EnhancedRerank(query string, docs []string, config *Config) ([]float64, error) {
	if config == nil {
		config = getDefaultConfig()
	}

	if config.BaseURL == "" || config.APIKey == "" || config.RerankModel == "" {
		return nil, fmt.Errorf("rerank not configured: missing BaseURL, APIKey, or RerankModel")
	}

	// Validate model is supported
	if modelInfo := getModelInfo(config.RerankModel); modelInfo != nil && modelInfo.Type != "rerank" {
		return nil, fmt.Errorf("model %s is not a rerank model", config.RerankModel)
	}

	path := resolvePath(config.BaseURL, os.Getenv("LLM_RERANK_PATH"), "/rerank")
	url := strings.TrimRight(config.BaseURL, "/") + path

	body := map[string]interface{}{
		"model":     config.RerankModel,
		"query":     query,
		"documents": docs,
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}

	resp, err := makeHTTPRequest("POST", url, headers, buf)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := readAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("rerank HTTP error: %d %s", resp.StatusCode, head(b))
	}

	var response struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
			Document       string  `json:"document"`
		} `json:"results"`
	}

	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, head(b))
	}

	// Handle different response formats
	scores := make([]float64, len(docs))
	for i, result := range response.Results {
		if i < len(scores) {
			scores[result.Index] = result.RelevanceScore
		}
	}

	return scores, nil
}

// Rerank provides backward compatibility
func Rerank(query string, docs []string) ([]float64, error) {
	return EnhancedRerank(query, docs, nil)
}

// ChatCompletion provides enhanced chat completion functionality
func ChatCompletion(messages []ChatMessage, config *Config) (*ChatCompletionResponse, error) {
	if config == nil {
		config = getDefaultConfig()
	}

	if config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		return nil, fmt.Errorf("chat completion not configured: missing BaseURL, APIKey, or Model")
	}

	// Validate model is supported
	if modelInfo := getModelInfo(config.Model); modelInfo != nil && modelInfo.Type != "chat" {
		return nil, fmt.Errorf("model %s is not a chat model", config.Model)
	}

	path := resolvePath(config.BaseURL, os.Getenv("LLM_COMPLETIONS_PATH"), "/chat/completions")
	url := strings.TrimRight(config.BaseURL, "/") + path

	request := ChatCompletionRequest{
		Model:       config.Model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   1000,
		TopP:        0.9,
	}

	buf, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + config.APIKey,
		"Content-Type":  "application/json",
	}

	resp, err := makeHTTPRequest("POST", url, headers, buf)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	b, err := readAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("chat completion HTTP error: %d %s", resp.StatusCode, head(b))
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w, body: %s", err, head(b))
	}

	return &response, nil
}

// EnhancedExpandKeywords provides enhanced keyword expansion with prompt templates
func EnhancedExpandKeywords(query string, config *Config, promptTemplate string) ([]string, error) {
	if config == nil {
		config = getDefaultConfig()
	}

	if config.BaseURL == "" || config.APIKey == "" || config.Model == "" {
		return nil, nil // Return empty if not configured
	}

	// Default prompt template for keyword expansion
	if promptTemplate == "" {
		promptTemplate = "你是代码检索助手。根据用户需求输出10个英文与中文关键词、相关文件类型/目录名，用逗号分隔，尽量简短。"
	}

	messages := []ChatMessage{
		{Role: "system", Content: promptTemplate},
		{Role: "user", Content: query},
	}

	response, err := ChatCompletion(messages, config)
	if err != nil {
		return nil, err
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	content := response.Choices[0].Message.Content
	return toks(content), nil
}

// ExpandKeywords provides backward compatibility
func ExpandKeywords(query string) []string {
	keywords, err := EnhancedExpandKeywords(query, nil, "")
	if err != nil {
		return nil
	}
	return keywords
}
