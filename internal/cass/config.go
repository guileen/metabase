package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads CASS configuration from file
func LoadConfig(configPath string) (*CIConfig, error) {
	config := DefaultCIConfig()

	// Load from file if exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			// File doesn't exist, use defaults
			return config, nil
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Load environment overrides
	loadEnvironmentOverrides(config)

	return config, nil
}

// loadEnvironmentOverrides loads configuration from environment variables
func loadEnvironmentOverrides(config *CIConfig) {
	// Analysis settings
	if val := os.Getenv("CASS_ANALYZE_ALL_FILES"); val == "true" {
		config.AnalyzeAllFiles = true
	}
	if val := os.Getenv("CASS_INCREMENTAL_MODE"); val == "false" {
		config.IncrementalMode = false
	}
	if val := os.Getenv("CASS_FAIL_ON_NEW_ISSUES"); val == "false" {
		config.FailOnNewIssues = false
	}
	if val := os.Getenv("CASS_FAIL_ON_SEVERITY"); val != "" {
		config.FailOnSeverity = val
	}

	// Thresholds
	if val := os.Getenv("CASS_QUALITY_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &config.Thresholds.QualityScore)
	}
	if val := os.Getenv("CASS_SECURITY_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &config.Thresholds.SecurityScore)
	}
	if val := os.Getenv("CASS_DUPLICATION_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &config.Thresholds.DuplicationRatio)
	}
	if val := os.Getenv("CASS_COVERAGE_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &config.Thresholds.TestCoverage)
	}

	// Reporting
	if val := os.Getenv("CASS_REPORT_FORMATS"); val != "" {
		config.ReportFormats = strings.Split(val, ",")
		for i, format := range config.ReportFormats {
			config.ReportFormats[i] = strings.TrimSpace(format)
		}
	}
	if val := os.Getenv("CASS_OUTPUT_DIR"); val != "" {
		config.OutputDirectory = val
	}
	if val := os.Getenv("CASS_BASELINE_FILE"); val != "" {
		config.BaselineFile = val
	}

	// Analyzers
	if val := os.Getenv("CASS_ENABLED_ANALYZERS"); val != "" {
		config.EnabledAnalyzers = strings.Split(val, ",")
		for i, analyzer := range config.EnabledAnalyzers {
			config.EnabledAnalyzers[i] = strings.TrimSpace(analyzer)
		}
	}

	// Performance
	if val := os.Getenv("CASS_PARALLELISM"); val != "" {
		var parallelism int
		fmt.Sscanf(val, "%d", &parallelism)
		if parallelism > 0 {
			config.Parallelism = parallelism
		}
	}
	if val := os.Getenv("CASS_TIMEOUT"); val != "" {
		config.Timeout = val
	}
}

// DetectCIContext detects CI/CD context from environment variables
func DetectCIContext() *CIContext {
	ctx := &CIContext{
		Environment: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}

	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		ctx.Provider = "github"
		ctx.Repository = os.Getenv("GITHUB_REPOSITORY")
		ctx.BuildNumber = os.Getenv("GITHUB_RUN_ID")
		ctx.Branch = os.Getenv("GITHUB_HEAD_REF")
		if ctx.Branch == "" {
			ctx.Branch = os.Getenv("GITHUB_REF_NAME")
		}
		ctx.Commit = os.Getenv("GITHUB_SHA")
		ctx.Actor = os.Getenv("GITHUB_ACTOR")
		ctx.Workflow = os.Getenv("GITHUB_WORKFLOW")
		ctx.PullRequest = os.Getenv("GITHUB_REF_NAME")
		if strings.HasPrefix(ctx.PullRequest, "PR-") {
			ctx.BaseBranch = os.Getenv("GITHUB_BASE_REF")
		}

		// Get changed files for pull requests
		if event := os.Getenv("GITHUB_EVENT_NAME"); event == "pull_request" {
			// In real implementation, use GitHub API to get changed files
			ctx.ChangedFiles = getGitHubChangedFiles()
		}

		ctx.Environment["CI"] = "true"
		ctx.Environment["GITHUB_ACTIONS"] = "true"
	}

	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		ctx.Provider = "gitlab"
		ctx.Repository = os.Getenv("CI_PROJECT_PATH")
		ctx.BuildNumber = os.Getenv("CI_JOB_ID")
		ctx.Branch = os.Getenv("CI_COMMIT_REF_NAME")
		ctx.Commit = os.Getenv("CI_COMMIT_SHA")
		ctx.Actor = os.Getenv("GITLAB_USER_LOGIN")
		ctx.Workflow = os.Getenv("CI_PIPELINE_NAME")
		ctx.Tag = os.Getenv("CI_COMMIT_TAG")

		// Get changed files
		if changed := os.Getenv("CI_COMMIT_CHANGED_FILES"); changed != "" {
			ctx.ChangedFiles = strings.Split(changed, " ")
		}

		ctx.Environment["CI"] = "true"
		ctx.Environment["GITLAB_CI"] = "true"
	}

	// Jenkins
	if os.Getenv("JENKINS_URL") != "" {
		ctx.Provider = "jenkins"
		ctx.Repository = os.Getenv("GIT_URL")
		ctx.BuildNumber = os.Getenv("BUILD_NUMBER")
		ctx.Branch = os.Getenv("GIT_BRANCH")
		ctx.Commit = os.Getenv("GIT_COMMIT")
		ctx.Actor = os.Getenv("BUILD_USER_ID")
		ctx.Workflow = os.Getenv("JOB_NAME")

		ctx.Environment["CI"] = "true"
		ctx.Environment["JENKINS_URL"] = os.Getenv("JENKINS_URL")
	}

	// Azure DevOps
	if os.Getenv("TF_BUILD") == "true" {
		ctx.Provider = "azure"
		ctx.Repository = os.Getenv("BUILD_REPOSITORY_NAME")
		ctx.BuildNumber = os.Getenv("BUILD_BUILDID")
		ctx.Branch = os.Getenv("BUILD_SOURCEBRANCHNAME")
		ctx.Commit = os.Getenv("BUILD_SOURCEVERSION")
		ctx.Actor = os.Getenv("BUILD_REQUESTEDFOREMAIL")
		ctx.Workflow = os.Getenv("BUILD_DEFINITIONNAME")
		ctx.PullRequest = os.Getenv("SYSTEM_PULLREQUEST_PULLREQUESTID")

		ctx.Environment["CI"] = "true"
		ctx.Environment["TF_BUILD"] = "true"
	}

	// CircleCI
	if os.Getenv("CIRCLECI") == "true" {
		ctx.Provider = "circleci"
		ctx.Repository = fmt.Sprintf("%s/%s", os.Getenv("CIRCLE_PROJECT_USERNAME"), os.Getenv("CIRCLE_PROJECT_REPONAME"))
		ctx.BuildNumber = os.Getenv("CIRCLE_BUILD_NUM")
		ctx.Branch = os.Getenv("CIRCLE_BRANCH")
		ctx.Commit = os.Getenv("CIRCLE_SHA1")
		ctx.Actor = os.Getenv("CIRCLE_USERNAME")
		ctx.Workflow = os.Getenv("CIRCLE_WORKFLOW_ID")

		ctx.Environment["CI"] = "true"
		ctx.Environment["CIRCLECI"] = "true"
	}

	// Generic CI detection
	if ctx.Provider == "" && os.Getenv("CI") == "true" {
		ctx.Provider = "generic"
		ctx.BuildNumber = os.Getenv("CI_BUILD_NUMBER")
		ctx.Branch = os.Getenv("CI_BRANCH")
		ctx.Commit = os.Getenv("CI_COMMIT_SHA")
		ctx.Environment["CI"] = "true"
	}

	return ctx
}

// getGitHubChangedFiles gets changed files from GitHub Actions
// This is a mock implementation - in production, use GitHub API
func getGitHubChangedFiles() []string {
	files := []string{
		// Mock changed files for demonstration
		"internal/analysis/engine.go",
		"internal/analysis/analyzers.go",
		"cmd/metabase/server.go",
	}
	return files
}

// SaveConfig saves configuration to file
func SaveConfig(config *CIConfig, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates configuration
func ValidateConfig(config *CIConfig) error {
	// Validate thresholds
	if config.Thresholds.QualityScore < 0 || config.Thresholds.QualityScore > 100 {
		return fmt.Errorf("quality threshold must be between 0 and 100")
	}
	if config.Thresholds.SecurityScore < 0 || config.Thresholds.SecurityScore > 100 {
		return fmt.Errorf("security threshold must be between 0 and 100")
	}
	if config.Thresholds.DuplicationRatio < 0 || config.Thresholds.DuplicationRatio > 100 {
		return fmt.Errorf("duplication threshold must be between 0 and 100")
	}
	if config.Thresholds.TestCoverage < 0 || config.Thresholds.TestCoverage > 100 {
		return fmt.Errorf("coverage threshold must be between 0 and 100")
	}

	// Validate severity
	validSeverities := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
	if !validSeverities[config.FailOnSeverity] {
		return fmt.Errorf("invalid fail_on_severity: %s (must be low, medium, high, or critical)", config.FailOnSeverity)
	}

	// Validate analyzers
	validAnalyzers := map[string]bool{
		"duplicate-detector": true,
		"security-scanner":   true,
		"quality-analyzer":   true,
	}
	for _, analyzer := range config.EnabledAnalyzers {
		if !validAnalyzers[analyzer] {
			return fmt.Errorf("invalid analyzer: %s", analyzer)
		}
	}

	// Validate report formats
	validFormats := map[string]bool{
		"json":               true,
		"markdown":           true,
		"junit":              true,
		"github-annotations": true,
		"sarif":              true,
	}
	for _, format := range config.ReportFormats {
		if !validFormats[format] {
			return fmt.Errorf("invalid report format: %s", format)
		}
	}

	// Validate timeout
	if _, err := parseDuration(config.Timeout); err != nil {
		return fmt.Errorf("invalid timeout format: %w", err)
	}

	return nil
}

// parseDuration parses duration string with common formats
func parseDuration(s string) (time.Duration, error) {
	// Support common duration formats
	if !strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "m") &&
		!strings.HasSuffix(s, "h") && !strings.HasSuffix(s, "d") {
		// Assume seconds if no unit
		s += "s"
	}
	return time.ParseDuration(s)
}
