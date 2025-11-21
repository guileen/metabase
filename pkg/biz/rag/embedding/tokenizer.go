package embedding

import ("encoding/json"
	"fmt"
	"os"
	"sort"
	"strings")

// SentencePieceTokenizer implements tokenization for ONNX models
type SentencePieceTokenizer struct {
	vocab       map[string]int
	invVocab    map[int]string
	unkToken    string
	clsToken    string
	sepToken    string
	padToken    string
	maskToken   string
	maxLength   int
}

// TokenizerConfig holds tokenizer configuration
type TokenizerConfig struct {
	VocabFile    string `json:"vocab_file"`
	UnkToken     string `json:"unk_token"`
	ClsToken     string `json:"cls_token"`
	SepToken     string `json:"sep_token"`
	PadToken     string `json:"pad_token"`
	MaskToken    string `json:"mask_token"`
	MaxLength    int    `json:"max_length"`
}

// NewSentencePieceTokenizer creates a new tokenizer from JSON file
func NewSentencePieceTokenizer(modelPath string) (*SentencePieceTokenizer, error) {
	// For simplicity, we'll use a basic WordPiece tokenizer
	// In production, this would load actual SentencePiece or BERT tokenizer files

	tokenizer := &SentencePieceTokenizer{
		vocab:     make(map[string]int),
		invVocab:  make(map[int]string),
		unkToken:  "[UNK]",
		clsToken:  "[CLS]",
		sepToken:  "[SEP]",
		padToken:  "[PAD]",
		maskToken: "[MASK]",
		maxLength: 512,
	}

	// Initialize basic vocabulary (this would be loaded from model files in production)
	if err := tokenizer.loadBasicVocabulary(); err != nil {
		return nil, fmt.Errorf("failed to load vocabulary: %w", err)
	}

	return tokenizer, nil
}

// loadBasicVocabulary creates a basic vocabulary for demonstration
func (st *SentencePieceTokenizer) loadBasicVocabulary() error {
	// This is a simplified vocabulary - in production, load from actual model files
	basicVocab := []string{
		"[PAD]", "[UNK]", "[CLS]", "[SEP]", "[MASK]",
		"the", "of", "and", "in", "to", "a", "for", "on", "with", "as", "by", "at",
		"from", "that", "this", "it", "not", "or", "be", "are", "was", "were", "been",
		"have", "has", "had", "do", "does", "did", "will", "would", "could", "should",
		"can", "may", "might", "must", "shall",
		// Add some programming terms
		"database", "api", "server", "client", "user", "data", "code", "function",
		"method", "class", "object", "variable", "string", "number", "boolean",
		"array", "list", "map", "dictionary", "json", "xml", "html", "css", "js",
		"python", "java", "javascript", "go", "rust", "cpp", "c", "sql", "nosql",
		"http", "https", "rest", "graphql", "websocket", "tcp", "udp", "ip",
	}

	for i, token := range basicVocab {
		st.vocab[token] = i
		st.invVocab[i] = token
	}

	// Add common subword units (## prefix for BERT-style)
	subwords := []string{
		"##ing", "##ed", "##er", "##est", "##ly", "##tion", "##s", "##es", "##'s",
		"##able", "##ible", "##al", "##ial", "##ic", "##ical", "##ive", "##ous",
		"##ful", "##less", "##ness", "##ment", "##ship", "##hood", "##dom",
		"##ize", "##fy", "##ate", "##en", "##ify", "##ise",
	}

	for _, subword := range subwords {
		st.vocab[subword] = len(st.vocab)
		st.invVocab[len(st.invVocab)] = subword
	}

	return nil
}

// EncodeBatch tokenizes multiple texts and returns input IDs and attention mask
func (st *SentencePieceTokenizer) EncodeBatch(texts []string, maxLength int) ([][]int64, [][]int64, error) {
	if maxLength <= 0 {
		maxLength = st.maxLength
	}

	inputIDs := make([][]int64, len(texts))
	attentionMask := make([][]int64, len(texts))

	for i, text := range texts {
		tokens := st.tokenize(text)

		// Truncate or pad to maxLength
		if len(tokens) > maxLength {
			tokens = tokens[:maxLength]
		} else {
			// Pad with [PAD] tokens
			for len(tokens) < maxLength {
				tokens = append(tokens, st.padToken)
			}
		}

		// Convert to IDs
		ids := make([]int64, len(tokens))
		mask := make([]int64, len(tokens))

		for j, token := range tokens {
			if id, exists := st.vocab[token]; exists {
				ids[j] = int64(id)
			} else {
				ids[j] = int64(st.vocab[st.unkToken])
			}
			mask[j] = 1
			if token == st.padToken {
				mask[j] = 0
			}
		}

		inputIDs[i] = ids
		attentionMask[i] = mask
	}

	return inputIDs, attentionMask, nil
}

// tokenize converts text to tokens using WordPiece-style tokenization
func (st *SentencePieceTokenizer) tokenize(text string) []string {
	// Basic preprocessing
	text = strings.ToLower(text)
	text = strings.TrimSpace(text)

	// Add special tokens
	tokens := []string{st.clsToken}

	// Split into words and tokenize each
	words := strings.Fields(text)
	for _, word := range words {
		wordTokens := st.tokenizeWord(word)
		tokens = append(tokens, wordTokens...)
	}

	// Add separator token
	tokens = append(tokens, st.sepToken)

	return tokens
}

// tokenizeWord implements WordPiece tokenization for a single word
func (st *SentencePieceTokenizer) tokenizeWord(word string) []string {
	// Remove punctuation and clean the word
	word = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, word)

	if word == "" {
		return []string{st.unkToken}
	}

	// First, try to find the whole word
	if _, exists := st.vocab[word]; exists {
		return []string{word}
	}

	// WordPiece algorithm: longest match first
	var tokens []string
	start := 0

	for start < len(word) {
		end := len(word)
		found := false

		// Find longest matching subword
		for end > start {
			subword := word[start:end]
			if start > 0 {
				subword = "##" + subword
			}

			if _, exists := st.vocab[subword]; exists {
				tokens = append(tokens, subword)
				start = end
				found = true
				break
			}
			end--
		}

		if !found {
			// No subword found, use UNK
			tokens = append(tokens, st.unkToken)
			break
		}
	}

	return tokens
}

// Decode converts token IDs back to text
func (st *SentencePieceTokenizer) Decode(tokenIDs []int64) string {
	var tokens []string

	for _, id := range tokenIDs {
		if token, exists := st.invVocab[int(id)]; exists {
			// Skip special tokens except SEP
			if token == st.clsToken || token == st.padToken {
				continue
			}
			if token == st.sepToken {
				tokens = append(tokens, ". ")
				continue
			}
			tokens = append(tokens, token)
		}
	}

	return strings.Join(tokens, " ")
}

// GetVocabSize returns the vocabulary size
func (st *SentencePieceTokenizer) GetVocabSize() int {
	return len(st.vocab)
}

// Save saves the tokenizer to a file
func (st *SentencePieceTokenizer) Save(filepath string) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tokenizer: %w", err)
	}

	return os.WriteFile(filepath, data, 0644)
}

// LoadTokenizer loads a tokenizer from a file
func LoadTokenizer(filepath string) (*SentencePieceTokenizer, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tokenizer file: %w", err)
	}

	var tokenizer SentencePieceTokenizer
	if err := json.Unmarshal(data, &tokenizer); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tokenizer: %w", err)
	}

	return &tokenizer, nil
}

// GetTopTokens returns the most frequent tokens from a batch
func (st *SentencePieceTokenizer) GetTopTokens(texts []string, topK int) []TokenFrequency {
	freq := make(map[string]int)

	for _, text := range texts {
		tokens := st.tokenize(text)
		for _, token := range tokens {
			// Skip special tokens
			if token == st.clsToken || token == st.sepToken || token == st.padToken {
				continue
			}
			freq[token]++
		}
	}

	// Convert to slice and sort
	tokenFreqs := make([]TokenFrequency, 0, len(freq))
	for token, count := range freq {
		tokenFreqs = append(tokenFreqs, TokenFrequency{
			Token:     token,
			Frequency: count,
		})
	}

	sort.Slice(tokenFreqs, func(i, j int) bool {
		return tokenFreqs[i].Frequency > tokenFreqs[j].Frequency
	})

	if topK > 0 && topK < len(tokenFreqs) {
		tokenFreqs = tokenFreqs[:topK]
	}

	return tokenFreqs
}

// TokenFrequency represents a token and its frequency
type TokenFrequency struct {
	Token     string `json:"token"`
	Frequency int    `json:"frequency"`
}

// GetStats returns tokenizer statistics
func (st *SentencePieceTokenizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"vocab_size":  len(st.vocab),
		"max_length":  st.maxLength,
		"unk_token":   st.unkToken,
		"cls_token":   st.clsToken,
		"sep_token":   st.sepToken,
		"pad_token":   st.padToken,
		"mask_token":  st.maskToken,
	}
}