package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/infra/storage"
)

// CIContext represents the CI/CD context
type CIContext struct {
	Provider     string                 `json:"provider"` // "github", "gitlab", "jenkins", etc.
	BuildNumber  string                 `json:"build_number"`
	Branch       string                 `json:"branch"`
	Commit       string                 `json:"commit"`
	Tag          string                 `json:"tag"`
	PullRequest  string                 `json:"pull_request"`
	Actor        string                 `json:"actor"`
	Workflow     string                 `json:"workflow"`
	Repository   string                 `json:"repository"`
	BaseBranch   string                 `json:"base_branch"`   // For PRs
	ChangedFiles []string               `json:"changed_files"` // For incremental analysis
	Environment  map[string]string      `json:"environment"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CIConfig represents CI/CD analysis configuration
type CIConfig struct {
	// Analysis settings
	AnalyzeAllFiles bool   `yaml:"analyze_all_files"`
	IncrementalMode bool   `yaml:"incremental_mode"`
	FailOnNewIssues bool   `yaml:"fail_on_new_issues"`
	FailOnSeverity  string `yaml:"fail_on_severity"`
	Parallelism     int    `yaml:"parallelism"`
	Timeout         string `yaml:"timeout"`
	CacheResults    bool   `yaml:"cache_results"`

	// File patterns
	IncludePatterns []string `yaml:"include_patterns"`
	ExcludePatterns []string `yaml:"exclude_patterns"`

	// Analyzers
	EnabledAnalyzers []string `yaml:"enabled_analyzers"`

	// Thresholds
	Thresholds struct {
		QualityScore     float64 `yaml:"quality_score"`
		SecurityScore    float64 `yaml:"security_score"`
		DuplicationRatio float64 `yaml:"duplication_ratio"`
		TestCoverage     float64 `yaml:"test_coverage"`
		Complexity       float64 `yaml:"complexity"`
	} `yaml:"thresholds"`

	// Reporting
	ReportFormats   []string `yaml:"report_formats"`
	OutputDirectory string   `yaml:"output_directory"`
	Annotations     bool     `yaml:"annotations"`
	Gatekeeper      bool     `yaml:"gatekeeper"`

	// Search and indexing
	EnableSearchIndex bool `yaml:"enable_search_index"`
	UpdateBaseline    bool `yaml:"update_baseline"`

	// Advanced
	BaselineFile         string   `yaml:"baseline_file"`
	CustomRules          []string `yaml:"custom_rules"`
	EnvironmentVariables []string `yaml:"environment_variables"`
}

// CIRunner runs the CASS analysis in CI/CD environments
type CIRunner struct {
	engine    *Engine
	config    *CIConfig
	context   *CIContext
	storage   storage.Storage
	baseline  *CIBaseline
	reporters map[string]CIReporter
	startTime time.Time
}

// CIBaseline represents analysis baseline for comparison
type CIBaseline struct {
	Commit    string                     `json:"commit"`
	Branch    string                     `json:"branch"`
	Timestamp time.Time                  `json:"timestamp"`
	Metrics   map[string]float64         `json:"metrics"`
	Issues    map[string][]BaselineIssue `json:"issues"`
	Artifacts int                        `json:"artifacts"`
	Version   string                     `json:"version"`
}

// BaselineIssue represents a baseline issue for comparison
type BaselineIssue struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Severity string                 `json:"severity"`
	Rule     string                 `json:"rule"`
	File     string                 `json:"file"`
	Line     int                    `json:"line"`
	Hash     string                 `json:"hash"`
	Metadata map[string]interface{} `json:"metadata"`
}

// CIReporter interface for different output formats
type CIReporter interface {
	Generate(ctx context.Context, results *CIResults) error
	GetFormat() string
	GetExtension() string
}

// CIResults represents the complete CI analysis results
type CIResults struct {
	Context     *CIContext            `json:"context"`
	Config      *CIConfig             `json:"config"`
	Summary     *CISummary            `json:"summary"`
	Artifacts   []*CIArtifactResult   `json:"artifacts"`
	Metrics     map[string]float64    `json:"metrics"`
	Issues      map[string][]*CIIssue `json:"issues"`
	Duplicates  []*CIDuplicateResult  `json:"duplicates"`
	Trends      *CITrends             `json:"trends"`
	Baseline    *CIBaseline           `json:"baseline"`
	Duration    time.Duration         `json:"duration"`
	GeneratedAt time.Time             `json:"generated_at"`
}

// CIArtifactResult represents analysis result for a single artifact
type CIArtifactResult struct {
	ArtifactID string                 `json:"artifact_id"`
	Path       string                 `json:"path"`
	Type       string                 `json:"type"`
	Language   string                 `json:"language"`
	Size       int64                  `json:"size"`
	Hash       string                 `json:"hash"`
	Analyzers  []string               `json:"analyzers"`
	Duration   time.Duration          `json:"duration"`
	Results    []*AnalysisResult      `json:"results"`
	Score      float64                `json:"score"`
	Status     string                 `json:"status"` // "passed", "failed", "warning"
	Metadata   map[string]interface{} `json:"metadata"`
}

// CISummary represents high-level summary
type CISummary struct {
	TotalArtifacts    int      `json:"total_artifacts"`
	AnalyzedArtifacts int      `json:"analyzed_artifacts"`
	PassedArtifacts   int      `json:"passed_artifacts"`
	FailedArtifacts   int      `json:"failed_artifacts"`
	WarningArtifacts  int      `json:"warning_artifacts"`
	SkippedArtifacts  int      `json:"skipped_artifacts"`
	TotalIssues       int      `json:"total_issues"`
	NewIssues         int      `json:"new_issues"`
	FixedIssues       int      `json:"fixed_issues"`
	CriticalIssues    int      `json:"critical_issues"`
	HighIssues        int      `json:"high_issues"`
	MediumIssues      int      `json:"medium_issues"`
	LowIssues         int      `json:"low_issues"`
	OverallScore      float64  `json:"overall_score"`
	QualityScore      float64  `json:"quality_score"`
	SecurityScore     float64  `json:"security_score"`
	Status            string   `json:"status"` // "passed", "failed", "warning"
	Recommendations   []string `json:"recommendations"`
}

// CIIssue represents a detected issue
type CIIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"` // "security", "quality", "duplicate"
	Severity    string                 `json:"severity"`
	Category    string                 `json:"category"`
	Rule        string                 `json:"rule"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	ArtifactID  string                 `json:"artifact_id"`
	Path        string                 `json:"path"`
	Line        int                    `json:"line"`
	Column      int                    `json:"column"`
	EndLine     int                    `json:"end_line"`
	EndColumn   int                    `json:"end_column"`
	Message     string                 `json:"message"`
	Context     string                 `json:"context"`
	Suggestion  string                 `json:"suggestion"`
	Confidence  float64                `json:"confidence"`
	New         bool                   `json:"new"`      // Is this a new issue?
	Baseline    bool                   `json:"baseline"` // Was this in baseline?
	Hash        string                 `json:"hash"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CIDuplicateResult represents a duplicate detection result
type CIDuplicateResult struct {
	ArtifactID1  string    `json:"artifact_id1"`
	ArtifactID2  string    `json:"artifact_id2"`
	Path1        string    `json:"path1"`
	Path2        string    `json:"path2"`
	Similarity   float64   `json:"similarity"`
	Method       string    `json:"method"`
	MatchType    string    `json:"match_type"`
	SharedTokens []string  `json:"shared_tokens"`
	Differences  []string  `json:"differences"`
	Lines1       int       `json:"lines1"`
	Lines2       int       `json:"lines2"`
	Timestamp    time.Time `json:"timestamp"`
}

// CITrends represents trend analysis
type CITrends struct {
	QualityTrend  []float64   `json:"quality_trend"`
	SecurityTrend []float64   `json:"security_trend"`
	CoverageTrend []float64   `json:"coverage_trend"`
	DebtTrend     []float64   `json:"debt_trend"`
	CommitHistory []string    `json:"commit_history"`
	Timestamps    []time.Time `json:"timestamps"`
}

// DefaultCIConfig returns default CI configuration
func DefaultCIConfig() *CIConfig {
	return &CIConfig{
		AnalyzeAllFiles:   false,
		IncrementalMode:   true,
		FailOnNewIssues:   true,
		FailOnSeverity:    "high",
		Parallelism:       runtime.NumCPU(),
		Timeout:           "30m",
		CacheResults:      true,
		IncludePatterns:   []string{"**/*.go", "**/*.js", "**/*.ts", "**/*.py", "**/*.java", "**/*.c", "**/*.cpp", "**/*.h"},
		ExcludePatterns:   []string{"**/vendor/**", "**/node_modules/**", "**/dist/**", "**/build/**", "**/.git/**"},
		EnabledAnalyzers:  []string{"duplicate-detector", "security-scanner", "quality-analyzer"},
		ReportFormats:     []string{"json", "markdown", "junit"},
		OutputDirectory:   "./cass-reports",
		Annotations:       true,
		Gatekeeper:        true,
		EnableSearchIndex: false,
		UpdateBaseline:    false,
		BaselineFile:      ".cass-baseline.json",
	}
}

// NewCIRunner creates a new CI runner
func NewCIRunner(engine *Engine, config *CIConfig, ctx *CIContext) (*CIRunner, error) {
	runner := &CIRunner{
		engine:    engine,
		config:    config,
		context:   ctx,
		reporters: make(map[string]CIReporter),
		startTime: time.Now(),
	}

	// Register default reporters
	runner.registerReporters()

	// Load baseline if exists
	if err := runner.loadBaseline(); err != nil {
		log.Printf("Warning: Could not load baseline: %v", err)
	}

	return runner, nil
}

// Run executes the CI analysis
func (r *CIRunner) Run(ctx context.Context) (*CIResults, error) {
	log.Printf("Starting CASS CI analysis for %s/%s", r.context.Repository, r.context.Branch)

	// Parse timeout
	timeout, err := time.ParseDuration(r.config.Timeout)
	if err != nil {
		timeout = 30 * time.Minute
	}

	// Create timeout context
	analysisCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Collect files to analyze
	artifacts, err := r.collectArtifacts(analysisCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	}

	log.Printf("Found %d artifacts to analyze", len(artifacts))

	// Analyze artifacts
	results, err := r.analyzeArtifacts(analysisCtx, artifacts)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze artifacts: %w", err)
	}

	// Find duplicates
	duplicates, err := r.findDuplicates(analysisCtx, artifacts)
	if err != nil {
		log.Printf("Warning: Duplicate detection failed: %v", err)
	}

	// Generate comprehensive results
	ciResults := r.generateResults(results, duplicates)

	// Compare with baseline
	if r.baseline != nil {
		r.compareWithBaseline(ciResults)
	}

	// Generate reports
	if err := r.generateReports(analysisCtx, ciResults); err != nil {
		log.Printf("Warning: Report generation failed: %v", err)
	}

	// Update baseline if requested
	if r.config.UpdateBaseline {
		if err := r.updateBaseline(ciResults); err != nil {
			log.Printf("Warning: Failed to update baseline: %v", err)
		}
	}

	// Print summary
	r.printSummary(ciResults)

	return ciResults, nil
}

// collectArtifacts collects artifacts to analyze
func (r *CIRunner) collectArtifacts(ctx context.Context) ([]*Artifact, error) {
	var artifacts []*Artifact

	// Determine which files to analyze
	var filesToAnalyze []string
	if r.config.IncrementalMode && len(r.context.ChangedFiles) > 0 {
		filesToAnalyze = r.context.ChangedFiles
		log.Printf("Incremental analysis: %d changed files", len(filesToAnalyze))
	} else {
		// Collect all files
		err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// Check include patterns
			included := false
			for _, pattern := range r.config.IncludePatterns {
				if matched, _ := filepath.Match(pattern, path); matched {
					included = true
					break
				}
			}

			if !included {
				return nil
			}

			// Check exclude patterns
			for _, pattern := range r.config.ExcludePatterns {
				if matched, _ := filepath.Match(pattern, path); matched {
					return nil
				}
			}

			filesToAnalyze = append(filesToAnalyze, path)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk directory: %w", err)
		}
	}

	// Create artifacts
	for _, filePath := range filesToAnalyze {
		artifact, err := r.createArtifact(filePath)
		if err != nil {
			log.Printf("Warning: Failed to create artifact for %s: %v", filePath, err)
			continue
		}
		artifacts = append(artifacts, artifact)
	}

	return artifacts, nil
}

// createArtifact creates an artifact from file path
func (r *CIRunner) createArtifact(filePath string) (*Artifact, error) {
	// Read file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine language
	language := r.detectLanguage(filePath)

	// Create artifact
	artifact := &Artifact{
		ID:        r.generateArtifactID(filePath),
		TenantID:  "default",
		ProjectID: r.context.Repository,
		Type:      ArtifactTypeSource,
		Language:  language,
		Path:      filePath,
		Name:      filepath.Base(filePath),
		Content:   content,
		Size:      int64(len(content)),
		Hash:      r.calculateHash(content),
		Stage:     StageRaw,
		Features:  make(map[FeatureType][]byte),
		Metadata: map[string]interface{}{
			"ci_context":  r.context,
			"file_stats":  r.getFileStats(content),
			"detected_at": time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Version:   1,
	}

	return artifact, nil
}

// analyzeArtifacts analyzes artifacts in parallel
func (r *CIRunner) analyzeArtifacts(ctx context.Context, artifacts []*Artifact) ([]*CIArtifactResult, error) {
	results := make([]*CIArtifactResult, 0, len(artifacts))

	// Create semaphore for parallelism
	semaphore := make(chan struct{}, r.config.Parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Process artifacts in parallel
	for _, artifact := range artifacts {
		wg.Add(1)
		go func(art *Artifact) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				break
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()

			// Analyze artifact
			result := r.analyzeArtifact(ctx, art)

			// Thread-safe append
			mu.Lock()
			results = append(results, result)
			mu.Unlock()

		}(artifact)
	}

	// Wait for completion
	wg.Wait()

	return results, nil
}

// analyzeArtifact analyzes a single artifact
func (r *CIRunner) analyzeArtifact(ctx context.Context, artifact *Artifact) *CIArtifactResult {
	start := time.Now()

	result := &CIArtifactResult{
		ArtifactID: artifact.ID,
		Path:       artifact.Path,
		Type:       fmt.Sprint(rune(artifact.Type)),
		Language:   artifact.Language,
		Size:       artifact.Size,
		Hash:       artifact.Hash,
		Analyzers:  r.config.EnabledAnalyzers,
		Status:     "passed",
		Metadata:   make(map[string]interface{}),
	}

	// Run analysis
	analysisResults, err := r.engine.Analyze(ctx, artifact)
	if err != nil {
		log.Printf("Analysis failed for %s: %v", artifact.Path, err)
		result.Status = "failed"
		result.Results = []*AnalysisResult{}
	} else {
		result.Results = analysisResults
	}

	// Calculate score
	result.Score = r.calculateArtifactScore(analysisResults)
	result.Duration = time.Since(start)

	// Determine status based on thresholds
	if r.shouldFailArtifact(result) {
		result.Status = "failed"
	} else if r.shouldWarnArtifact(result) {
		result.Status = "warning"
	}

	return result
}

// findDuplicates finds duplicates across artifacts
func (r *CIRunner) findDuplicates(ctx context.Context, artifacts []*Artifact) ([]*CIDuplicateResult, error) {
	var duplicates []*CIDuplicateResult

	// Get duplicate detector from engine
	detector := NewDuplicateDetector()

	// Process artifacts in batches to avoid memory issues
	batchSize := 100
	for i := 0; i < len(artifacts); i += batchSize {
		end := i + batchSize
		if end > len(artifacts) {
			end = len(artifacts)
		}

		batch := artifacts[i:end]

		// Compare artifacts within batch
		for j, art1 := range batch {
			for k := j + 1; k < len(batch); k++ {
				art2 := batch[k]

				// Only compare same language artifacts
				if art1.Language != art2.Language {
					continue
				}

				// Compare artifacts
				similarity, err := detector.Compare(ctx, art1, art2)
				if err != nil {
					continue
				}

				// Only include significant similarities
				if similarity.Score >= 0.7 {
					duplicate := &CIDuplicateResult{
						ArtifactID1:  art1.ID,
						ArtifactID2:  art2.ID,
						Path1:        art1.Path,
						Path2:        art2.Path,
						Similarity:   similarity.Score,
						Method:       similarity.Method,
						MatchType:    similarity.MatchType,
						SharedTokens: similarity.SharedTokens,
						Differences:  similarity.Differences,
						Lines1:       strings.Count(string(art1.Content), "\n") + 1,
						Lines2:       strings.Count(string(art2.Content), "\n") + 1,
						Timestamp:    time.Now(),
					}
					duplicates = append(duplicates, duplicate)
				}
			}
		}
	}

	return duplicates, nil
}

// generateResults generates comprehensive CI results
func (r *CIRunner) generateResults(artifactResults []*CIArtifactResult, duplicates []*CIDuplicateResult) *CIResults {
	results := &CIResults{
		Context:     r.context,
		Config:      r.config,
		Artifacts:   artifactResults,
		Duplicates:  duplicates,
		Metrics:     make(map[string]float64),
		Issues:      make(map[string][]*CIIssue),
		Duration:    time.Since(r.startTime),
		GeneratedAt: time.Now(),
	}

	// Generate summary
	results.Summary = r.generateSummary(artifactResults)

	// Extract issues
	r.extractIssues(results)

	// Calculate metrics
	r.calculateMetrics(results)

	// Generate trends
	results.Trends = r.generateTrends(results)

	return results
}

// generateSummary generates high-level summary
func (r *CIRunner) generateSummary(artifactResults []*CIArtifactResult) *CISummary {
	summary := &CISummary{
		TotalArtifacts:  len(artifactResults),
		OverallScore:    0.0,
		QualityScore:    0.0,
		SecurityScore:   0.0,
		Recommendations: []string{},
	}

	var totalScore, totalQuality, totalSecurity float64
	issueCount := make(map[string]int)
	severityCount := map[string]int{"critical": 0, "high": 0, "medium": 0, "low": 0}

	for _, result := range artifactResults {
		// Count statuses
		switch result.Status {
		case "passed":
			summary.PassedArtifacts++
		case "failed":
			summary.FailedArtifacts++
		case "warning":
			summary.WarningArtifacts++
		}

		summary.AnalyzedArtifacts++

		// Accumulate scores
		if len(result.Results) > 0 {
			for _, analysisResult := range result.Results {
				totalScore += analysisResult.Score

				switch analysisResult.Type {
				case "quality":
					totalQuality += analysisResult.Score
				case "security":
					totalSecurity += analysisResult.Score
				}

				// Count issues
				summary.TotalIssues += len(analysisResult.Findings)
				issueCount[analysisResult.Type] += len(analysisResult.Findings)

				// Count severities
				for _, finding := range analysisResult.Findings {
					severityCount[finding.Severity]++
				}
			}
		}
	}

	// Calculate averages
	if summary.AnalyzedArtifacts > 0 {
		summary.OverallScore = totalScore / float64(summary.AnalyzedArtifacts)
		summary.QualityScore = totalQuality / float64(summary.AnalyzedArtifacts)
		summary.SecurityScore = totalSecurity / float64(summary.AnalyzedArtifacts)
	}

	// Set severity counts
	summary.CriticalIssues = severityCount["critical"]
	summary.HighIssues = severityCount["high"]
	summary.MediumIssues = severityCount["medium"]
	summary.LowIssues = severityCount["low"]

	// Determine overall status
	summary.Status = r.determineOverallStatus(summary)

	// Generate recommendations
	summary.Recommendations = r.generateRecommendations(summary)

	return summary
}

// Helper functions for language detection, hashing, etc.

func (r *CIRunner) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h":
		return "c"
	case ".hpp":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	case ".cs":
		return "csharp"
	case ".rs":
		return "rust"
	default:
		return "unknown"
	}
}

func (r *CIRunner) generateArtifactID(filePath string) string {
	return fmt.Sprintf("%s_%d", strings.ReplaceAll(filePath, "/", "_"), time.Now().UnixNano())
}

func (r *CIRunner) calculateHash(content []byte) string {
	// Simple hash - in production, use proper cryptographic hash
	return fmt.Sprintf("%x", len(content))
}

func (r *CIRunner) getFileStats(content []byte) map[string]interface{} {
	lines := strings.Count(string(content), "\n") + 1
	return map[string]interface{}{
		"lines": lines,
		"chars": len(content),
		"size":  len(content),
		"utf8":  true,
	}
}

func (r *CIRunner) calculateArtifactScore(results []*AnalysisResult) float64 {
	if len(results) == 0 {
		return 100.0
	}

	var totalScore float64
	for _, result := range results {
		totalScore += result.Score
	}

	return totalScore / float64(len(results))
}

func (r *CIRunner) shouldFailArtifact(result *CIArtifactResult) bool {
	// Check against failure thresholds
	if r.config.Thresholds.QualityScore > 0 && result.Score < r.config.Thresholds.QualityScore {
		return true
	}

	// Check for critical issues
	for _, analysisResult := range result.Results {
		for _, finding := range analysisResult.Findings {
			if finding.Severity == "critical" {
				return true
			}
		}
	}

	return false
}

func (r *CIRunner) shouldWarnArtifact(result *CIArtifactResult) bool {
	// Check for high severity issues
	for _, analysisResult := range result.Results {
		for _, finding := range analysisResult.Findings {
			if finding.Severity == "high" {
				return true
			}
		}
	}

	return false
}

func (r *CIRunner) determineOverallStatus(summary *CISummary) string {
	if summary.FailedArtifacts > 0 {
		return "failed"
	}
	if summary.WarningArtifacts > 0 {
		return "warning"
	}
	return "passed"
}

func (r *CIRunner) generateRecommendations(summary *CISummary) []string {
	var recommendations []string

	if summary.CriticalIssues > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d critical security vulnerabilities immediately", summary.CriticalIssues))
	}

	if summary.HighIssues > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Fix %d high-priority issues", summary.HighIssues))
	}

	if summary.QualityScore < r.config.Thresholds.QualityScore {
		recommendations = append(recommendations, "Improve overall code quality")
	}

	if summary.SecurityScore < r.config.Thresholds.SecurityScore {
		recommendations = append(recommendations, "Address security concerns")
	}

	return recommendations
}

func (r *CIRunner) extractIssues(results *CIResults) {
	for _, artifactResult := range results.Artifacts {
		for _, analysisResult := range artifactResult.Results {
			if results.Issues[analysisResult.Type] == nil {
				results.Issues[analysisResult.Type] = make([]*CIIssue, 0)
			}

			for _, finding := range analysisResult.Findings {
				issue := &CIIssue{
					ID:          finding.ID,
					Type:        analysisResult.Type,
					Severity:    finding.Severity,
					Category:    finding.Category,
					Rule:        finding.Rule,
					Title:       finding.Rule,
					Description: finding.Message,
					ArtifactID:  artifactResult.ArtifactID,
					Path:        artifactResult.Path,
					Line:        finding.Line,
					Column:      finding.Column,
					EndLine:     finding.EndLine,
					EndColumn:   finding.EndColumn,
					Message:     finding.Message,
					Context:     finding.Context,
					Suggestion:  finding.Suggestion,
					Confidence:  finding.Confidence,
					New:         true, // Will be updated during baseline comparison
					Hash:        r.calculateIssueHash(finding),
					Metadata:    finding.Metadata,
				}
				results.Issues[analysisResult.Type] = append(results.Issues[analysisResult.Type], issue)
			}
		}
	}
}

func (r *CIRunner) calculateIssueHash(finding Finding) string {
	content := fmt.Sprintf("%s:%s:%s:%d", finding.Rule, finding.Severity, finding.Type, finding.Line)
	return fmt.Sprintf("%x", len(content))
}

func (r *CIRunner) calculateMetrics(results *CIResults) {
	// Implementation for calculating various metrics
	results.Metrics["analysis_duration"] = results.Duration.Seconds()
	results.Metrics["artifacts_per_second"] = float64(len(results.Artifacts)) / results.Duration.Seconds()

	if len(results.Artifacts) > 0 {
		var totalSize int64
		for _, artifact := range results.Artifacts {
			totalSize += artifact.Size
		}
		results.Metrics["avg_artifact_size"] = float64(totalSize) / float64(len(results.Artifacts))
	}
}

func (r *CIRunner) generateTrends(results *CIResults) *CITrends {
	// Mock trends - in production, load from historical data
	return &CITrends{
		QualityTrend:  []float64{85.0, 87.0, 89.0, results.Summary.QualityScore},
		SecurityTrend: []float64{90.0, 88.0, 92.0, results.Summary.SecurityScore},
		CoverageTrend: []float64{70.0, 75.0, 80.0, 78.0},
		DebtTrend:     []float64{10.0, 8.0, 6.0, 5.0},
		CommitHistory: []string{"abc123", "def456", "ghi789", r.context.Commit},
		Timestamps: []time.Time{
			time.Now().Add(-3 * 24 * time.Hour),
			time.Now().Add(-2 * 24 * time.Hour),
			time.Now().Add(-1 * 24 * time.Hour),
			time.Now(),
		},
	}
}

func (r *CIRunner) printSummary(results *CIResults) {
	fmt.Printf("\n=== CASS CI Analysis Results ===\n")
	fmt.Printf("Repository: %s\n", results.Context.Repository)
	fmt.Printf("Branch: %s\n", results.Context.Branch)
	fmt.Printf("Commit: %s\n", results.Context.Commit)
	fmt.Printf("Duration: %s\n", results.Duration)
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Total Artifacts: %d\n", results.Summary.TotalArtifacts)
	fmt.Printf("  Analyzed: %d\n", results.Summary.AnalyzedArtifacts)
	fmt.Printf("  Passed: %d\n", results.Summary.PassedArtifacts)
	fmt.Printf("  Failed: %d\n", results.Summary.FailedArtifacts)
	fmt.Printf("  Warnings: %d\n", results.Summary.WarningArtifacts)
	fmt.Printf("  Total Issues: %d\n", results.Summary.TotalIssues)
	fmt.Printf("  Critical: %d\n", results.Summary.CriticalIssues)
	fmt.Printf("  High: %d\n", results.Summary.HighIssues)
	fmt.Printf("  Overall Score: %.1f\n", results.Summary.OverallScore)
	fmt.Printf("  Status: %s\n", results.Summary.Status)

	if len(results.Summary.Recommendations) > 0 {
		fmt.Printf("\nRecommendations:\n")
		for _, rec := range results.Summary.Recommendations {
			fmt.Printf("  • %s\n", rec)
		}
	}
	fmt.Printf("\n")
}

// Additional methods for baseline management, report generation, etc.
// These would be implemented in the full system

func (r *CIRunner) loadBaseline() error {
	// Load baseline from file if exists
	if _, err := os.Stat(r.config.BaselineFile); err == nil {
		data, err := os.ReadFile(r.config.BaselineFile)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &r.baseline)
	}
	return nil
}

func (r *CIRunner) compareWithBaseline(results *CIResults) {
	// Compare current results with baseline and mark new/fixed issues
	if r.baseline == nil {
		return
	}

	for issueType, issues := range results.Issues {
		baselineIssues := r.baseline.Issues[issueType]
		baselineMap := make(map[string]BaselineIssue)
		for _, baselineIssue := range baselineIssues {
			baselineMap[baselineIssue.Hash] = baselineIssue
		}

		for _, issue := range issues {
			if _, exists := baselineMap[issue.Hash]; exists {
				issue.New = false
				issue.Baseline = true
			}
		}
	}
}

func (r *CIRunner) updateBaseline(results *CIResults) error {
	// Create new baseline
	baseline := &CIBaseline{
		Commit:    r.context.Commit,
		Branch:    r.context.Branch,
		Timestamp: time.Now(),
		Metrics:   results.Metrics,
		Issues:    make(map[string][]BaselineIssue),
		Artifacts: len(results.Artifacts),
		Version:   "1.0.0",
	}

	// Convert issues to baseline issues
	for issueType, issues := range results.Issues {
		baselineIssues := make([]BaselineIssue, 0, len(issues))
		for _, issue := range issues {
			baselineIssues = append(baselineIssues, BaselineIssue{
				ID:       issue.ID,
				Type:     issue.Type,
				Severity: issue.Severity,
				Rule:     issue.Rule,
				File:     issue.Path,
				Line:     issue.Line,
				Hash:     issue.Hash,
				Metadata: issue.Metadata,
			})
		}
		baseline.Issues[issueType] = baselineIssues
	}

	// Save to file
	data, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.config.BaselineFile, data, 0644)
}

func (r *CIRunner) generateReports(ctx context.Context, results *CIResults) error {
	// Generate reports in configured formats
	for _, format := range r.config.ReportFormats {
		if reporter, exists := r.reporters[format]; exists {
			if err := reporter.Generate(ctx, results); err != nil {
				log.Printf("Failed to generate %s report: %v", format, err)
			}
		}
	}
	return nil
}

func (r *CIRunner) registerReporters() {
	// Register built-in reporters
	r.reporters["json"] = NewJSONReporter(r.config.OutputDirectory)
	r.reporters["markdown"] = NewMarkdownReporter(r.config.OutputDirectory)
	r.reporters["junit"] = NewJunitReporter(r.config.OutputDirectory)
}
