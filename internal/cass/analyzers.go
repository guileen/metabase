package analysis

import (
	"context"
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/slices"
)

// BaseAnalyzer provides common functionality for all analyzers
type BaseAnalyzer struct {
	id           string
	name         string
	version      string
	capabilities AnalyzerCapability
	languages    []string
	types        []ArtifactType
	rules        []Rule
	mu           sync.RWMutex
}

// Rule represents an analysis rule
type Rule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Severity    string                 `json:"severity"`
	Pattern     string                 `json:"pattern"`
	Analyzer    string                 `json:"analyzer"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config"`
}

// NewBaseAnalyzer creates a new base analyzer
func NewBaseAnalyzer(id, name, version string, capabilities AnalyzerCapability) *BaseAnalyzer {
	return &BaseAnalyzer{
		id:           id,
		name:         name,
		version:      version,
		capabilities: capabilities,
		languages:    []string{"*"},
		types:        []ArtifactType{ArtifactTypeSource},
		rules:        make([]Rule, 0),
	}
}

// ID returns analyzer ID
func (b *BaseAnalyzer) ID() string {
	return b.id
}

// Name returns analyzer name
func (b *BaseAnalyzer) Name() string {
	return b.name
}

// Version returns analyzer version
func (b *BaseAnalyzer) Version() string {
	return b.version
}

// Capabilities returns analyzer capabilities
func (b *BaseAnalyzer) Capabilities() AnalyzerCapability {
	return b.capabilities
}

// SupportedLanguages returns supported languages
func (b *BaseAnalyzer) SupportedLanguages() []string {
	return b.languages
}

// SupportedTypes returns supported artifact types
func (b *BaseAnalyzer) SupportedTypes() []ArtifactType {
	return b.types
}

// Initialize initializes the analyzer
func (b *BaseAnalyzer) Initialize(ctx context.Context) error {
	return nil
}

// Cleanup cleans up analyzer resources
func (b *BaseAnalyzer) Cleanup() error {
	return nil
}

// AddRule adds a rule to the analyzer
func (b *BaseAnalyzer) AddRule(rule Rule) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.rules = append(b.rules, rule)
}

// GetRules returns all rules
func (b *BaseAnalyzer) GetRules() []Rule {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return slices.Clone(b.rules)
}

// DuplicateDetector implements duplicate detection
type DuplicateDetector struct {
	*BaseAnalyzer
	minLines       int
	ignoreWS       bool
	ignoreComments bool
	threshold      float64
	fingerprints   map[string]*Fingerprint
}

// Fingerprint represents code fingerprint for duplicate detection
type Fingerprint struct {
	ArtifactID string             `json:"artifact_id"`
	Hash       string             `json:"hash"`
	Tokens     []string           `json:"tokens"`
	Structure  string             `json:"structure"`
	Metrics    map[string]float64 `json:"metrics"`
}

// NewDuplicateDetector creates a new duplicate detector
func NewDuplicateDetector() *DuplicateDetector {
	detector := &DuplicateDetector{
		BaseAnalyzer: NewBaseAnalyzer(
			"duplicate-detector",
			"Code Duplicate Detector",
			"1.0.0",
			CapabilityAnalyze|CapabilityCompare|CapabilitySearch,
		),
		minLines:       5,
		ignoreWS:       true,
		ignoreComments: true,
		threshold:      0.8,
		fingerprints:   make(map[string]*Fingerprint),
	}

	// Set supported languages
	detector.languages = []string{"go", "java", "javascript", "typescript", "python", "c", "cpp"}

	// Add detection rules
	detector.AddRule(Rule{
		ID:          "DUPLICATE-001",
		Name:        "Exact Duplicate",
		Description: "Detects exact duplicate code blocks",
		Type:        "duplicate",
		Severity:    "medium",
		Pattern:     ".*",
		Enabled:     true,
	})

	detector.AddRule(Rule{
		ID:          "DUPLICATE-002",
		Name:        "Near Duplicate",
		Description: "Detects near-duplicate code with similarity threshold",
		Type:        "duplicate",
		Severity:    "low",
		Pattern:     ".*",
		Enabled:     true,
		Config: map[string]interface{}{
			"threshold": 0.8,
		},
	})

	return detector
}

// Analyze analyzes for duplicates
func (d *DuplicateDetector) Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error) {
	start := time.Now()
	result := &AnalysisResult{
		ArtifactID:  artifact.ID,
		AnalyzerID:  d.ID(),
		Type:        "duplicate",
		Findings:    make([]Finding, 0),
		Metrics:     make(map[string]float64),
		ProcessedAt: time.Now(),
	}

	// Extract content as string
	content := string(artifact.Content)
	if len(content) == 0 {
		return result, nil
	}

	// Preprocess content
	if d.ignoreComments {
		content = d.removeComments(content, artifact.Language)
	}

	// Split into lines
	lines := strings.Split(content, "\n")
	if len(lines) < d.minLines {
		return result, nil
	}

	// Generate fingerprint
	fingerprint := d.generateFingerprint(artifact, content)
	d.fingerprints[artifact.ID] = fingerprint

	// Find similar fingerprints
	similars := d.findSimilarFingerprints(fingerprint, d.threshold)

	// Create findings
	for _, similar := range similars {
		if similar.ArtifactID != artifact.ID {
			result.Findings = append(result.Findings, Finding{
				ID:         generateID(),
				Type:       "duplicate",
				Severity:   "medium",
				Message:    fmt.Sprintf("Code duplicate found with %s (similarity: %.2f%%)", similar.ArtifactID, similar.Metrics["similarity"]*100),
				Rule:       "DUPLICATE-001",
				Category:   "duplication",
				Confidence: similar.Metrics["similarity"],
				Metadata: map[string]interface{}{
					"similar_artifact": similar.ArtifactID,
					"similarity":       similar.Metrics["similarity"],
					"method":           "fingerprint",
				},
			})
		}
	}

	// Calculate metrics
	result.Metrics["lines"] = float64(len(lines))
	result.Metrics["unique_tokens"] = float64(len(fingerprint.Tokens))
	result.Metrics["duplication_score"] = d.calculateDuplicationScore(lines)

	result.Duration = time.Since(start)
	result.Score = float64(len(result.Findings))
	result.Confidence = 1.0

	return result, nil
}

// Compare compares two artifacts for similarity
func (d *DuplicateDetector) Compare(ctx context.Context, artifact1, artifact2 *Artifact) (*SimilarityResult, error) {
	fp1 := d.fingerprints[artifact1.ID]
	fp2 := d.fingerprints[artifact2.ID]

	if fp1 == nil {
		fp1 = d.generateFingerprint(artifact1, string(artifact1.Content))
		d.fingerprints[artifact1.ID] = fp1
	}

	if fp2 == nil {
		fp2 = d.generateFingerprint(artifact2, string(artifact2.Content))
		d.fingerprints[artifact2.ID] = fp2
	}

	// Calculate similarity
	similarity := d.calculateFingerprintSimilarity(fp1, fp2)

	// Determine match type
	var matchType string
	switch {
	case similarity >= 0.95:
		matchType = "exact"
	case similarity >= 0.8:
		matchType = "near"
	case similarity >= 0.5:
		matchType = "partial"
	default:
		matchType = "none"
	}

	// Find shared tokens
	sharedTokens := d.findSharedTokens(fp1.Tokens, fp2.Tokens)

	result := &SimilarityResult{
		ArtifactID1:  artifact1.ID,
		ArtifactID2:  artifact2.ID,
		Score:        similarity,
		Method:       "fingerprint",
		MatchType:    matchType,
		SharedTokens: sharedTokens,
		ComputedAt:   time.Now(),
	}

	return result, nil
}

// ExtractFeatures extracts features for indexing
func (d *DuplicateDetector) ExtractFeatures(ctx context.Context, artifact *Artifact) ([]*FeatureVector, error) {
	content := string(artifact.Content)
	fingerprint := d.generateFingerprint(artifact, content)

	vectors := make([]*FeatureVector, 0)

	// Token frequency vector
	tokenFreq := make(map[string]float64)
	for _, token := range fingerprint.Tokens {
		tokenFreq[token]++
	}

	// Normalize
	total := float64(len(fingerprint.Tokens))
	if total > 0 {
		for token, count := range tokenFreq {
			tokenFreq[token] = count / total
		}
	}

	// Create feature vector
	vector := make([]float64, 256) // Fixed size vector
	for i, token := range fingerprint.Tokens[:min(256, len(fingerprint.Tokens))] {
		h := sha256.Sum256([]byte(token))
		for j, b := range h[:8] {
			vector[i*8+j] = float64(b) / 255.0
		}
	}

	vectors = append(vectors, &FeatureVector{
		ArtifactID: artifact.ID,
		Type:       FeatureLexical,
		Vector:     vector,
		Metadata: map[string]string{
			"analyzer": "duplicate-detector",
			"method":   "fingerprint",
		},
		Confidence:  1.0,
		GeneratedAt: time.Now(),
	})

	return vectors, nil
}

// generateFingerprint generates code fingerprint
func (d *DuplicateDetector) generateFingerprint(artifact *Artifact, content string) *Fingerprint {
	// Tokenize content
	tokens := d.tokenize(content, artifact.Language)

	// Generate structure
	structure := d.generateStructure(content, artifact.Language)

	// Calculate metrics
	metrics := map[string]float64{
		"token_count":   float64(len(tokens)),
		"unique_tokens": float64(len(unique(tokens))),
		"line_count":    float64(strings.Count(content, "\n") + 1),
		"char_count":    float64(len(content)),
	}

	return &Fingerprint{
		ArtifactID: artifact.ID,
		Hash:       d.calculateHash(content),
		Tokens:     tokens,
		Structure:  structure,
		Metrics:    metrics,
	}
}

// tokenize tokenizes code content
func (d *DuplicateDetector) tokenize(content, language string) []string {
	var tokens []string

	switch language {
	case "go":
		tokens = d.tokenizeGo(content)
	case "javascript", "typescript":
		tokens = d.tokenizeJS(content)
	default:
		// Generic tokenization
		re := regexp.MustCompile(`\w+|[^\s\w]`)
		matches := re.FindAllString(content, -1)
		tokens = matches
	}

	return tokens
}

// tokenizeGo tokenizes Go code
func (d *DuplicateDetector) tokenizeGo(content string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		// Fallback to generic tokenization
		re := regexp.MustCompile(`\w+|[^\s\w]`)
		return re.FindAllString(content, -1)
	}

	var tokens []string
	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.Ident:
			tokens = append(tokens, t.Name)
		case *ast.BasicLit:
			tokens = append(tokens, t.Value)
		}
		return true
	})

	return tokens
}

// tokenizeJS tokenizes JavaScript/TypeScript code
func (d *DuplicateDetector) tokenizeJS(content string) []string {
	// Simple JS tokenization - in production, use proper JS parser
	re := regexp.MustCompile(`\b(function|var|let|const|if|else|for|while|return|class|extends|import|export|from|default)\b|\w+|[^\s\w]`)
	return re.FindAllString(content, -1)
}

// generateStructure generates code structure
func (d *DuplicateDetector) generateStructure(content, language string) string {
	// Simplified structure generation
	lines := strings.Split(content, "\n")
	var structure []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		// Normalize for structure
		normalized := regexp.MustCompile(`\b\w+\b`).ReplaceAllString(trimmed, "X")
		normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
		structure = append(structure, normalized)
	}

	return strings.Join(structure, "\n")
}

// findSimilarFingerprints finds similar fingerprints
func (d *DuplicateDetector) findSimilarFingerprints(fp *Fingerprint, threshold float64) []*Fingerprint {
	var similars []*Fingerprint

	for _, candidate := range d.fingerprints {
		if candidate.ArtifactID == fp.ArtifactID {
			continue
		}

		similarity := d.calculateFingerprintSimilarity(fp, candidate)
		if similarity >= threshold {
			similars = append(similars, candidate)
		}
	}

	return similars
}

// calculateFingerprintSimilarity calculates similarity between fingerprints
func (d *DuplicateDetector) calculateFingerprintSimilarity(fp1, fp2 *Fingerprint) float64 {
	// Token similarity
	tokenSim := d.calculateJaccardSimilarity(fp1.Tokens, fp2.Tokens)

	// Structure similarity
	structSim := d.calculateJaccardSimilarity(
		strings.Split(fp1.Structure, "\n"),
		strings.Split(fp2.Structure, "\n"),
	)

	// Weighted combination
	return tokenSim*0.7 + structSim*0.3
}

// calculateJaccardSimilarity calculates Jaccard similarity
func (d *DuplicateDetector) calculateJaccardSimilarity(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1.0
	}

	setA := make(map[string]bool)
	setB := make(map[string]bool)

	for _, item := range a {
		setA[item] = true
	}
	for _, item := range b {
		setB[item] = true
	}

	intersection := 0
	union := 0

	for item := range setA {
		if setB[item] {
			intersection++
		}
		union++
	}

	for item := range setB {
		if !setA[item] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// removeComments removes comments from code
func (d *DuplicateDetector) removeComments(content, language string) string {
	switch language {
	case "go":
		// Remove // comments
		re := regexp.MustCompile(`//.*`)
		content = re.ReplaceAllString(content, "")
		// Remove /* */ comments
		re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
		content = re.ReplaceAllString(content, "")
	case "javascript", "typescript", "java", "c", "cpp":
		// Similar comment removal
		re := regexp.MustCompile(`//.*`)
		content = re.ReplaceAllString(content, "")
		re = regexp.MustCompile(`/\*[\s\S]*?\*/`)
		content = re.ReplaceAllString(content, "")
	case "python":
		re := regexp.MustCompile(`#.*`)
		content = re.ReplaceAllString(content, "")
		re = regexp.MustCompile(`'''[\s\S]*?'''`)
		content = re.ReplaceAllString(content, "")
		re = regexp.MustCompile(`"""[\s\S]*?"""`)
		content = re.ReplaceAllString(content, "")
	}

	return content
}

// calculateDuplicationScore calculates duplication score
func (d *DuplicateDetector) calculateDuplicationScore(lines []string) float64 {
	// Simplified duplication calculation
	// In practice, this would be more sophisticated
	totalLines := len(lines)
	if totalLines < 10 {
		return 0.0
	}

	duplicateLines := 0
	lineCounts := make(map[string]int)

	for _, line := range lines {
		normalized := strings.TrimSpace(strings.ToLower(line))
		if len(normalized) > 10 { // Only consider meaningful lines
			lineCounts[normalized]++
			if lineCounts[normalized] > 1 {
				duplicateLines++
			}
		}
	}

	return float64(duplicateLines) / float64(totalLines)
}

// calculateHash calculates content hash
func (d *DuplicateDetector) calculateHash(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h)
}

// findSharedTokens finds shared tokens between two token lists
func (d *DuplicateDetector) findSharedTokens(tokens1, tokens2 []string) []string {
	set1 := make(map[string]bool)
	for _, token := range tokens1 {
		set1[token] = true
	}

	var shared []string
	for _, token := range tokens2 {
		if set1[token] {
			shared = append(shared, token)
		}
	}

	return shared
}

// SecurityScanner implements security vulnerability scanning
type SecurityScanner struct {
	*BaseAnalyzer
	rules    []SecurityRule
	patterns map[string]*regexp.Regexp
	sinks    map[string][]string
	sources  map[string][]string
}

// SecurityRule represents a security rule
type SecurityRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	CWE         string   `json:"cwe"`
	OWASP       string   `json:"owasp"`
	Severity    string   `json:"severity"`
	Patterns    []string `json:"patterns"`
	Sinks       []string `json:"sinks"`
	Sources     []string `json:"sources"`
	Language    string   `json:"language"`
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner() *SecurityScanner {
	scanner := &SecurityScanner{
		BaseAnalyzer: NewBaseAnalyzer(
			"security-scanner",
			"Security Vulnerability Scanner",
			"1.0.0",
			CapabilityAnalyze|CapabilitySearch|CapabilityValidate,
		),
		rules:    make([]SecurityRule, 0),
		patterns: make(map[string]*regexp.Regexp),
		sinks: map[string][]string{
			"sql":     {"query", "exec", "execute"},
			"command": {"exec", "spawn", "run"},
			"file":    {"writeFile", "createWriteStream", "open"},
			"crypto":  {"decrypt", "verify", "sign"},
		},
		sources: map[string][]string{
			"sql":     {"SELECT", "INSERT", "UPDATE", "DELETE"},
			"command": {"exec", "system", "cmd"},
			"file":    {"file", "path", "directory"},
			"crypto":  {"key", "password", "secret"},
		},
	}

	// Set supported languages
	scanner.languages = []string{"go", "javascript", "typescript", "python", "java", "c", "cpp"}

	// Load security rules
	scanner.loadSecurityRules()

	return scanner
}

// loadSecurityRules loads built-in security rules
func (s *SecurityScanner) loadSecurityRules() {
	rules := []SecurityRule{
		{
			ID:          "SEC-001",
			Name:        "SQL Injection",
			Description: "Potential SQL injection vulnerability",
			CWE:         "CWE-89",
			OWASP:       "A03:2021-Injection",
			Severity:    "critical",
			Patterns:    []string{`(?i)(query|exec|execute)\s*\(\s*[^)]*\+`},
			Sinks:       []string{"query", "exec", "execute"},
			Sources:     []string{"request", "param", "input"},
		},
		{
			ID:          "SEC-002",
			Name:        "Hard-coded Credentials",
			Description: "Hard-coded passwords or secrets",
			CWE:         "CWE-798",
			OWASP:       "A07:2021-Identification and Authentication Failures",
			Severity:    "high",
			Patterns: []string{
				`(?i)password\s*=\s*["'][^"']+["']`,
				`(?i)secret\s*=\s*["'][^"']+["']`,
				`(?i)api_key\s*=\s*["'][^"']+["']`,
			},
		},
		{
			ID:          "SEC-003",
			Name:        "Insecure Random Number Generation",
			Description: "Use of cryptographically insecure random number generator",
			CWE:         "CWE-338",
			OWASP:       "A02:2021-Cryptographic Failures",
			Severity:    "medium",
			Patterns: []string{
				`Math\.random\(\)`,
				`rand\.Seed\(\)`,
				`srand\(\)`,
			},
		},
		{
			ID:          "SEC-004",
			Name:        "Cross-Site Scripting (XSS)",
			Description: "Potential XSS vulnerability",
			CWE:         "CWE-79",
			OWASP:       "A03:2021-Injection",
			Severity:    "high",
			Patterns: []string{
				`innerHTML\s*=\s*[^;]*[^)`,
				`document\.write\s*\(`,
				`eval\s*\(`,
			},
		},
		{
			ID:          "SEC-005",
			Name:        "Path Traversal",
			Description: "Potential path traversal vulnerability",
			CWE:         "CWE-22",
			OWASP:       "A01:2021-Broken Access Control",
			Severity:    "high",
			Patterns: []string{
				`\.\./.*\.\./`,
				`%2e%2e%2f`,
				`\.\.\\`,
			},
		},
	}

	for _, rule := range rules {
		s.rules = append(s.rules, rule)

		// Compile patterns
		for _, pattern := range rule.Patterns {
			if re, err := regexp.Compile(pattern); err == nil {
				s.patterns[rule.ID+":"+pattern] = re
			}
		}
	}
}

// Analyze analyzes for security vulnerabilities
func (s *SecurityScanner) Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error) {
	start := time.Now()
	result := &AnalysisResult{
		ArtifactID:  artifact.ID,
		AnalyzerID:  s.ID(),
		Type:        "security",
		Findings:    make([]Finding, 0),
		Metrics:     make(map[string]float64),
		ProcessedAt: time.Now(),
	}

	content := string(artifact.Content)
	lines := strings.Split(content, "\n")

	for _, rule := range s.rules {
		// Check language
		if rule.Language != "" && rule.Language != artifact.Language {
			continue
		}

		// Check patterns
		for _, pattern := range rule.Patterns {
			if re, exists := s.patterns[rule.ID+":"+pattern]; exists {
				matches := re.FindAllStringSubmatchIndex(content, -1)
				for _, match := range matches {
					if len(match) >= 2 {
						// Find line and column
						offset := match[0]
						line, col := s.findPosition(content, offset)

						result.Findings = append(result.Findings, Finding{
							ID:         generateID(),
							Type:       "vulnerability",
							Severity:   rule.Severity,
							Line:       line,
							Column:     col,
							Message:    fmt.Sprintf("%s: %s", rule.Name, rule.Description),
							Rule:       rule.ID,
							Category:   "security",
							Context:    s.extractContext(lines, line, 3),
							Suggestion: s.getSuggestion(rule.ID),
							Metadata: map[string]interface{}{
								"cwe":     rule.CWE,
								"owasp":   rule.OWASP,
								"pattern": pattern,
							},
							Confidence: 0.8,
						})
					}
				}
			}
		}
	}

	// Calculate security score
	result.Score = s.calculateSecurityScore(result.Findings)
	result.Duration = time.Since(start)
	result.Metrics["vulnerabilities"] = float64(len(result.Findings))
	result.Metrics["security_score"] = result.Score

	return result, nil
}

// findPosition finds line and column from byte offset
func (s *SecurityScanner) findPosition(content string, offset int) (line, col int) {
	line = 1
	col = 1

	for i, c := range content {
		if i >= offset {
			break
		}
		if c == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}

	return line, col
}

// extractContext extracts context around a line
func (s *SecurityScanner) extractContext(lines []string, line, contextLines int) string {
	start := max(0, line-contextLines-1)
	end := min(len(lines), line+contextLines)

	contextLinesSlice := lines[start:end]
	return strings.Join(contextLinesSlice, "\n")
}

// getSuggestion returns a suggestion for a security rule
func (s *SecurityScanner) getSuggestion(ruleID string) string {
	suggestions := map[string]string{
		"SEC-001": "Use parameterized queries or prepared statements to prevent SQL injection",
		"SEC-002": "Remove hard-coded credentials and use environment variables or secure vault",
		"SEC-003": "Use cryptographically secure random number generators (crypto/rand in Go, secrets in Python)",
		"SEC-004": "Sanitize user input before rendering and use textContent instead of innerHTML",
		"SEC-005": "Validate and sanitize file paths, use absolute paths, and check for directory traversal",
	}

	if suggestion, exists := suggestions[ruleID]; exists {
		return suggestion
	}
	return "Review and fix the security vulnerability"
}

// calculateSecurityScore calculates security score
func (s *SecurityScanner) calculateSecurityScore(findings []Finding) float64 {
	if len(findings) == 0 {
		return 100.0
	}

	score := 100.0
	for _, finding := range findings {
		switch finding.Severity {
		case "critical":
			score -= 25
		case "high":
			score -= 15
		case "medium":
			score -= 10
		case "low":
			score -= 5
		}
	}

	return math.Max(0, score)
}

// ExtractFeatures extracts security features
func (s *SecurityScanner) ExtractFeatures(ctx context.Context, artifact *Artifact) ([]*FeatureVector, error) {
	vectors := make([]*FeatureVector, 0)

	// Create security feature vector
	vector := make([]float64, 64)

	// Extract security-related features
	content := string(artifact.Content)

	// Check for security keywords
	securityKeywords := []string{
		"password", "secret", "token", "key", "auth",
		"crypto", "encrypt", "decrypt", "hash", "salt",
		"sql", "query", "exec", "inject",
		"xss", "csrf", "sanitize", "validate",
	}

	for i, keyword := range securityKeywords {
		if strings.Contains(strings.ToLower(content), keyword) {
			vector[i] = 1.0
		}
	}

	vectors = append(vectors, &FeatureVector{
		ArtifactID: artifact.ID,
		Type:       FeatureSecurity,
		Vector:     vector,
		Metadata: map[string]string{
			"analyzer": "security-scanner",
			"feature":  "security_keywords",
		},
		Confidence:  0.7,
		GeneratedAt: time.Now(),
	})

	return vectors, nil
}

// QualityAnalyzer implements code quality analysis
type QualityAnalyzer struct {
	*BaseAnalyzer
	metrics map[string]func(*Artifact) float64
}

// NewQualityAnalyzer creates a new quality analyzer
func NewQualityAnalyzer() *QualityAnalyzer {
	analyzer := &QualityAnalyzer{
		BaseAnalyzer: NewBaseAnalyzer(
			"quality-analyzer",
			"Code Quality Analyzer",
			"1.0.0",
			CapabilityAnalyze|CapabilityCompare|CapabilityRecommend,
		),
		metrics: make(map[string]func(*Artifact) float64),
	}

	// Set supported languages
	analyzer.languages = []string{"go", "javascript", "typescript", "python", "java", "c", "cpp"}

	// Register metrics
	analyzer.registerMetrics()

	return analyzer
}

// registerMetrics registers quality metrics
func (q *QualityAnalyzer) registerMetrics() {
	q.metrics["complexity"] = q.calculateComplexity
	q.metrics["maintainability"] = q.calculateMaintainability
	q.metrics["test_coverage"] = q.estimateTestCoverage
	q.metrics["documentation"] = q.calculateDocumentationRatio
	q.metrics["duplication"] = q.calculateDuplicationRatio
}

// Analyze analyzes code quality
func (q *QualityAnalyzer) Analyze(ctx context.Context, artifact *Artifact) (*AnalysisResult, error) {
	start := time.Now()
	result := &AnalysisResult{
		ArtifactID:  artifact.ID,
		AnalyzerID:  q.ID(),
		Type:        "quality",
		Findings:    make([]Finding, 0),
		Metrics:     make(map[string]float64),
		ProcessedAt: time.Now(),
	}

	// Calculate all metrics
	for name, metric := range q.metrics {
		value := metric(artifact)
		result.Metrics[name] = value

		// Create findings for poor metrics
		if q.isPoorMetric(name, value) {
			result.Findings = append(result.Findings, Finding{
				ID:         generateID(),
				Type:       "quality",
				Severity:   q.getMetricSeverity(name, value),
				Message:    q.getMetricMessage(name, value),
				Rule:       fmt.Sprintf("QUALITY-%s", strings.ToUpper(name)),
				Category:   "quality",
				Suggestion: q.getMetricSuggestion(name),
				Confidence: 0.9,
				Metadata: map[string]interface{}{
					"metric_name":  name,
					"metric_value": value,
				},
			})
		}
	}

	// Calculate overall quality score
	result.Score = q.calculateQualityScore(result.Metrics)
	result.Duration = time.Since(start)
	result.Confidence = 0.85

	return result, nil
}

// calculateComplexity calculates cyclomatic complexity
func (q *QualityAnalyzer) calculateComplexity(artifact *Artifact) float64 {
	content := string(artifact.Content)
	complexity := 1.0 // Base complexity

	// Count complexity drivers
	drivers := []string{
		"if", "else", "elif", "for", "while", "switch", "case",
		"catch", "try", "throw", "&&", "||", "?", "goto",
	}

	for _, driver := range drivers {
		complexity += float64(strings.Count(content, driver))
	}

	return complexity
}

// calculateMaintainability calculates maintainability index
func (q *QualityAnalyzer) calculateMaintainability(artifact *Artifact) float64 {
	content := string(artifact.Content)

	// Simplified maintainability index calculation
	lines := float64(strings.Count(content, "\n") + 1)
	complexity := q.calculateComplexity(artifact)

	// Maintainability decreases with lines and complexity
	maintainability := 171.0 - 5.2*math.Log(complexity) - 0.23*complexity - 16.2*math.Log(lines)

	return math.Max(0, maintainability)
}

// estimateTestCoverage estimates test coverage
func (q *QualityAnalyzer) estimateTestCoverage(artifact *Artifact) float64 {
	// Check if it's a test file
	isTest := strings.Contains(strings.ToLower(artifact.Path), "test") ||
		strings.HasSuffix(artifact.Path, "_test.go") ||
		strings.Contains(artifact.Name, "test")

	if isTest {
		return 100.0
	}

	// Estimate based on assertions and test patterns
	content := strings.ToLower(string(artifact.Content))
	indicators := []string{"assert", "expect", "should", "test", "spec", "describe"}
	score := 0.0

	for _, indicator := range indicators {
		if strings.Contains(content, indicator) {
			score += 20.0
		}
	}

	return math.Min(100.0, score)
}

// calculateDocumentationRatio calculates documentation ratio
func (q *QualityAnalyzer) calculateDocumentationRatio(artifact *Artifact) float64 {
	content := string(artifact.Content)
	lines := strings.Split(content, "\n")

	docLines := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") ||
			strings.HasPrefix(trimmed, "*") {
			docLines++
		}
	}

	if len(lines) == 0 {
		return 0.0
	}

	return float64(docLines) / float64(len(lines)) * 100.0
}

// calculateDuplicationRatio calculates duplication ratio
func (q *QualityAnalyzer) calculateDuplicationRatio(artifact *Artifact) float64 {
	content := string(artifact.Content)
	lines := strings.Split(content, "\n")

	// Simple duplication detection
	seen := make(map[string]int)
	duplicateLines := 0

	for _, line := range lines {
		normalized := strings.TrimSpace(strings.ToLower(line))
		if len(normalized) > 10 {
			seen[normalized]++
			if seen[normalized] > 1 {
				duplicateLines++
			}
		}
	}

	if len(lines) == 0 {
		return 0.0
	}

	return float64(duplicateLines) / float64(len(lines)) * 100.0
}

// isPoorMetric checks if metric value is poor
func (q *QualityAnalyzer) isPoorMetric(name string, value float64) bool {
	thresholds := map[string]float64{
		"complexity":      10.0,
		"maintainability": 50.0,
		"test_coverage":   80.0,
		"documentation":   20.0,
		"duplication":     5.0,
	}

	switch name {
	case "maintainability", "test_coverage", "documentation":
		return value < thresholds[name]
	case "complexity", "duplication":
		return value > thresholds[name]
	}

	return false
}

// getMetricSeverity returns severity for metric
func (q *QualityAnalyzer) getMetricSeverity(name string, value float64) string {
	switch name {
	case "complexity":
		if value > 20 {
			return "high"
		}
		return "medium"
	case "maintainability":
		if value < 30 {
			return "high"
		}
		return "medium"
	case "test_coverage":
		if value < 50 {
			return "high"
		}
		return "medium"
	default:
		return "low"
	}
}

// getMetricMessage returns message for metric
func (q *QualityAnalyzer) getMetricMessage(name string, value float64) string {
	return fmt.Sprintf("Poor %s metric: %.2f", name, value)
}

// getMetricSuggestion returns suggestion for metric
func (q *QualityAnalyzer) getMetricSuggestion(name string) string {
	suggestions := map[string]string{
		"complexity":      "Consider refactoring to reduce complexity",
		"maintainability": "Improve code structure and add documentation",
		"test_coverage":   "Add more unit tests to increase coverage",
		"documentation":   "Add comments and documentation",
		"duplication":     "Extract duplicate code into reusable functions",
	}

	if suggestion, exists := suggestions[name]; exists {
		return suggestion
	}
	return "Review and improve code quality"
}

// calculateQualityScore calculates overall quality score
func (q *QualityAnalyzer) calculateQualityScore(metrics map[string]float64) float64 {
	scores := map[string]float64{
		"complexity":      math.Max(0, 100-metrics["complexity"]),
		"maintainability": metrics["maintainability"],
		"test_coverage":   metrics["test_coverage"],
		"documentation":   metrics["documentation"],
		"duplication":     math.Max(0, 100-metrics["duplication"]),
	}

	total := 0.0
	count := 0.0

	for _, score := range scores {
		total += score
		count++
	}

	if count == 0 {
		return 0.0
	}

	return total / count
}

// ExtractFeatures extracts quality features
func (q *QualityAnalyzer) ExtractFeatures(ctx context.Context, artifact *Artifact) ([]*FeatureVector, error) {
	vectors := make([]*FeatureVector, 0)

	// Create quality feature vector
	vector := make([]float64, 32)

	// Extract metrics as features
	i := 0
	for _, metric := range []string{"complexity", "maintainability", "test_coverage", "documentation", "duplication"} {
		if fn, exists := q.metrics[metric]; exists {
			vector[i] = fn(artifact) / 100.0 // Normalize to 0-1
			i++
		}
	}

	vectors = append(vectors, &FeatureVector{
		ArtifactID: artifact.ID,
		Type:       FeatureQuality,
		Vector:     vector,
		Metadata: map[string]string{
			"analyzer": "quality-analyzer",
			"feature":  "quality_metrics",
		},
		Confidence:  0.8,
		GeneratedAt: time.Now(),
	})

	return vectors, nil
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func unique[T comparable](slice []T) []T {
	seen := make(map[T]bool)
	result := make([]T, 0)

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
