package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JSONReporter generates JSON format reports
type JSONReporter struct {
	outputDir string
}

func NewJSONReporter(outputDir string) *JSONReporter {
	return &JSONReporter{outputDir: outputDir}
}

func (r *JSONReporter) GetFormat() string    { return "json" }
func (r *JSONReporter) GetExtension() string { return ".json" }

func (r *JSONReporter) Generate(ctx context.Context, results *CIResults) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate comprehensive report
	reportFile := filepath.Join(r.outputDir, "cass-report.json")
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON report: %w", err)
	}

	if err := os.WriteFile(reportFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write JSON report: %w", err)
	}

	// Generate summary report
	summaryFile := filepath.Join(r.outputDir, "cass-summary.json")
	summary := map[string]interface{}{
		"timestamp":       results.GeneratedAt,
		"context":         results.Context,
		"summary":         results.Summary,
		"metrics":         results.Metrics,
		"duration":        results.Duration,
		"status":          results.Summary.Status,
		"total_artifacts": results.Summary.TotalArtifacts,
		"total_issues":    results.Summary.TotalIssues,
		"critical_issues": results.Summary.CriticalIssues,
		"high_issues":     results.Summary.HighIssues,
		"overall_score":   results.Summary.OverallScore,
		"recommendations": results.Summary.Recommendations,
	}

	summaryData, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal summary JSON: %w", err)
	}

	if err := os.WriteFile(summaryFile, summaryData, 0644); err != nil {
		return fmt.Errorf("failed to write summary JSON: %w", err)
	}

	return nil
}

// MarkdownReporter generates Markdown format reports
type MarkdownReporter struct {
	outputDir string
}

func NewMarkdownReporter(outputDir string) *MarkdownReporter {
	return &MarkdownReporter{outputDir: outputDir}
}

func (r *MarkdownReporter) GetFormat() string    { return "markdown" }
func (r *MarkdownReporter) GetExtension() string { return ".md" }

func (r *MarkdownReporter) Generate(ctx context.Context, results *CIResults) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportFile := filepath.Join(r.outputDir, "cass-report.md")

	var md strings.Builder

	// Header
	md.WriteString("# CASS CI Analysis Report\n\n")
	md.WriteString(fmt.Sprintf("**Repository:** %s\n", results.Context.Repository))
	md.WriteString(fmt.Sprintf("**Branch:** %s\n", results.Context.Branch))
	md.WriteString(fmt.Sprintf("**Commit:** %s\n", results.Context.Commit))
	md.WriteString(fmt.Sprintf("**Build Number:** %s\n", results.Context.BuildNumber))
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", results.GeneratedAt.Format(time.RFC3339)))

	// Summary
	md.WriteString("## Summary\n\n")
	summary := results.Summary
	md.WriteString("| Metric | Value |\n")
	md.WriteString("|--------|-------|\n")
	md.WriteString(fmt.Sprintf("| **Status** | %s |\n", r.getStatusBadge(summary.Status)))
	md.WriteString(fmt.Sprintf("| Total Artifacts | %d |\n", summary.TotalArtifacts))
	md.WriteString(fmt.Sprintf("| Analyzed | %d |\n", summary.AnalyzedArtifacts))
	md.WriteString(fmt.Sprintf("| Passed | %d |\n", summary.PassedArtifacts))
	md.WriteString(fmt.Sprintf("| Failed | %d |\n", summary.FailedArtifacts))
	md.WriteString(fmt.Sprintf("| Warnings | %d |\n", summary.WarningArtifacts))
	md.WriteString(fmt.Sprintf("| Total Issues | %d |\n", summary.TotalIssues))
	md.WriteString(fmt.Sprintf("| Critical | %d |\n", summary.CriticalIssues))
	md.WriteString(fmt.Sprintf("| High | %d |\n", summary.HighIssues))
	md.WriteString(fmt.Sprintf("| Medium | %d |\n", summary.MediumIssues))
	md.WriteString(fmt.Sprintf("| Low | %d |\n", summary.LowIssues))
	md.WriteString(fmt.Sprintf("| Overall Score | %.1f/100 |\n", summary.OverallScore))
	md.WriteString(fmt.Sprintf("| Quality Score | %.1f/100 |\n", summary.QualityScore))
	md.WriteString(fmt.Sprintf("| Security Score | %.1f/100 |\n", summary.SecurityScore))
	md.WriteString(fmt.Sprintf("| Analysis Duration | %s |\n", results.Duration.Round(time.Second)))

	// Issues by Type
	md.WriteString("\n## Issues by Type\n\n")
	for issueType, issues := range results.Issues {
		if len(issues) > 0 {
			md.WriteString(fmt.Sprintf("### %s (%d issues)\n\n", strings.Title(issueType), len(issues)))

			// Severity breakdown
			severityCount := make(map[string]int)
			for _, issue := range issues {
				severityCount[issue.Severity]++
			}

			md.WriteString("| Severity | Count |\n")
			md.WriteString("|----------|-------|\n")
			for _, severity := range []string{"critical", "high", "medium", "low"} {
				if count := severityCount[severity]; count > 0 {
					md.WriteString(fmt.Sprintf("| %s | %d |\n", strings.Title(severity), count))
				}
			}
			md.WriteString("\n")

			// Top issues
			md.WriteString("#### Top Issues\n\n")
			count := min(10, len(issues))
			for i := 0; i < count; i++ {
				issue := issues[i]
				md.WriteString(fmt.Sprintf("- **[%s]** `%s:%d` - %s\n",
					strings.ToUpper(issue.Severity), issue.Path, issue.Line, issue.Message))
			}
			md.WriteString("\n")
		}
	}

	// Failed Artifacts
	if len(results.Summary.FailedArtifacts) > 0 {
		md.WriteString("## Failed Artifacts\n\n")
		for _, artifact := range results.Artifacts {
			if artifact.Status == "failed" {
				md.WriteString(fmt.Sprintf("### `%s`\n\n", artifact.Path))
				md.WriteString(fmt.Sprintf("**Score:** %.1f/100  \n", artifact.Score))
				md.WriteString(fmt.Sprintf("**Language:** %s  \n", artifact.Language))
				md.WriteString(fmt.Sprintf("**Duration:** %s\n\n", artifact.Duration))

				if len(artifact.Results) > 0 {
					md.WriteString("**Issues:**\n")
					for _, result := range artifact.Results {
						for _, finding := range result.Findings {
							md.WriteString(fmt.Sprintf("- **[%s]** %s (line %d)\n",
								strings.ToUpper(finding.Severity), finding.Message, finding.Line))
						}
					}
				}
				md.WriteString("\n")
			}
		}
	}

	// Duplicates
	if len(results.Duplicates) > 0 {
		md.WriteString(fmt.Sprintf("## Code Duplicates (%d found)\n\n", len(results.Duplicates)))
		md.WriteString("| File 1 | File 2 | Similarity | Lines |\n")
		md.WriteString("|--------|--------|------------|-------|\n")

		count := min(20, len(results.Duplicates))
		for i := 0; i < count; i++ {
			dup := results.Duplicates[i]
			md.WriteString(fmt.Sprintf("| `%s` | `%s` | %.1f%% | %d/%d |\n",
				dup.Path1, dup.Path2, dup.Similarity*100, dup.Lines1, dup.Lines2))
		}
		md.WriteString("\n")
	}

	// Recommendations
	if len(summary.Recommendations) > 0 {
		md.WriteString("## Recommendations\n\n")
		for _, rec := range summary.Recommendations {
			md.WriteString(fmt.Sprintf("- %s\n", rec))
		}
		md.WriteString("\n")
	}

	// Metrics
	md.WriteString("## Detailed Metrics\n\n")
	md.WriteString("```json\n")
	metricsJSON, _ := json.MarshalIndent(results.Metrics, "", "  ")
	md.Write(metricsJSON)
	md.WriteString("\n```\n\n")

	// Footer
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("*Report generated by CASS (Code Analysis & Search System) on %s*\n",
		results.GeneratedAt.Format(time.RFC3339)))

	// Write file
	if err := os.WriteFile(reportFile, []byte(md.String()), 0644); err != nil {
		return fmt.Errorf("failed to write markdown report: %w", err)
	}

	return nil
}

func (r *MarkdownReporter) getStatusBadge(status string) string {
	switch status {
	case "passed":
		return "✅ Passed"
	case "failed":
		return "❌ Failed"
	case "warning":
		return "⚠️ Warning"
	default:
		return status
	}
}

// JunitReporter generates JUnit XML format reports for CI systems
type JunitReporter struct {
	outputDir string
}

func NewJunitReporter(outputDir string) *JunitReporter {
	return &JunitReporter{outputDir: outputDir}
}

func (r *JunitReporter) GetFormat() string    { return "junit" }
func (r *JunitReporter) GetExtension() string { return ".xml" }

func (r *JunitReporter) Generate(ctx context.Context, results *CIResults) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportFile := filepath.Join(r.outputDir, "cass-junit.xml")

	var xml strings.Builder

	// XML header
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	// Test suite
	summary := results.Summary
	failures := summary.FailedArtifacts
	errors := summary.CriticalIssues
	xml.WriteString(fmt.Sprintf(`<testsuite name="CASS Analysis" tests="%d" failures="%d" errors="%d" time="%.3f">`+"\n",
		summary.AnalyzedArtifacts, failures, errors, results.Duration.Seconds()))

	// Test cases for each artifact
	for _, artifact := range results.Artifacts {
		xml.WriteString(fmt.Sprintf(`  <testcase classname="cass.analysis" name="%s" time="%.3f">`+"\n",
			artifact.Path, artifact.Duration.Seconds()))

		if artifact.Status == "failed" {
			// Find the primary failure reason
			var failureMessage string
			if len(artifact.Results) > 0 {
				for _, result := range artifact.Results {
					for _, finding := range result.Findings {
						if finding.Severity == "critical" || finding.Severity == "high" {
							failureMessage = fmt.Sprintf("%s: %s", finding.Rule, finding.Message)
							break
						}
					}
					if failureMessage != "" {
						break
					}
				}
			}

			if failureMessage == "" {
				failureMessage = fmt.Sprintf("Analysis failed with score %.1f", artifact.Score)
			}

			xml.WriteString(fmt.Sprintf(`    <failure message="%s">`+"\n", escapeXML(failureMessage)))
			xml.WriteString("      <![CDATA[\n")
			if len(artifact.Results) > 0 {
				for _, result := range artifact.Results {
					xml.WriteString(fmt.Sprintf("Analyzer: %s\n", result.Type))
					for _, finding := range result.Findings {
						xml.WriteString(fmt.Sprintf("- [%s] %s (line %d)\n",
							strings.ToUpper(finding.Severity), finding.Message, finding.Line))
					}
				}
			}
			xml.WriteString("      ]]>\n")
			xml.WriteString("    </failure>\n")
		}

		if artifact.Status == "warning" {
			var warningMessage string
			if len(artifact.Results) > 0 {
				for _, result := range artifact.Results {
					for _, finding := range result.Findings {
						if finding.Severity == "medium" {
							warningMessage = fmt.Sprintf("%s: %s", finding.Rule, finding.Message)
							break
						}
					}
					if warningMessage != "" {
						break
					}
				}
			}

			if warningMessage == "" {
				warningMessage = fmt.Sprintf("Analysis warning with score %.1f", artifact.Score)
			}

			xml.WriteString(fmt.Sprintf(`    <error message="%s">`+"\n", escapeXML(warningMessage)))
			xml.WriteString("      <![CDATA[\n")
			if len(artifact.Results) > 0 {
				for _, result := range artifact.Results {
					xml.WriteString(fmt.Sprintf("Analyzer: %s\n", result.Type))
					for _, finding := range result.Findings {
						xml.WriteString(fmt.Sprintf("- [%s] %s (line %d)\n",
							strings.ToUpper(finding.Severity), finding.Message, finding.Line))
					}
				}
			}
			xml.WriteString("      ]]>\n")
			xml.WriteString("    </error>\n")
		}

		xml.WriteString("  </testcase>\n")
	}

	xml.WriteString("</testsuite>\n")

	// Write file
	if err := os.WriteFile(reportFile, []byte(xml.String()), 0644); err != nil {
		return fmt.Errorf("failed to write JUnit report: %w", err)
	}

	return nil
}

// Helper function to escape XML special characters
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// GitHubAnnotationsReporter generates GitHub Actions annotations
type GitHubAnnotationsReporter struct {
	outputDir string
}

func NewGitHubAnnotationsReporter(outputDir string) *GitHubAnnotationsReporter {
	return &GitHubAnnotationsReporter{outputDir: outputDir}
}

func (r *GitHubAnnotationsReporter) GetFormat() string    { return "github-annotations" }
func (r *GitHubAnnotationsReporter) GetExtension() string { return ".txt" }

func (r *GitHubAnnotationsReporter) Generate(ctx context.Context, results *CIResults) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportFile := filepath.Join(r.outputDir, "github-annotations.txt")

	var annotations strings.Builder

	// Generate annotations for issues
	for _, issues := range results.Issues {
		for _, issue := range issues {
			// Skip low severity issues for annotations to avoid noise
			if issue.Severity == "low" {
				continue
			}

			annotationLevel := "notice"
			switch issue.Severity {
			case "critical", "high":
				annotationLevel = "error"
			case "medium":
				annotationLevel = "warning"
			}

			title := fmt.Sprintf("%s: %s", strings.Title(issue.Type), issue.Title)
			message := fmt.Sprintf("%s\n\n**Suggestion:** %s", issue.Message, issue.Suggestion)

			annotations.WriteString(fmt.Sprintf("::%s file=%s,line=%d::%s - %s\n",
				annotationLevel, issue.Path, issue.Line, title, message))
		}
	}

	// Generate summary annotation
	summary := results.Summary
	summaryMessage := fmt.Sprintf("CASS Analysis Complete: %d artifacts analyzed, %d issues found (Score: %.1f/100)",
		summary.AnalyzedArtifacts, summary.TotalIssues, summary.OverallScore)

	var summaryLevel string
	switch summary.Status {
	case "failed":
		summaryLevel = "error"
	case "warning":
		summaryLevel = "warning"
	default:
		summaryLevel = "notice"
	}

	annotations.WriteString(fmt.Sprintf("::%s title=CASS Analysis::%s\n", summaryLevel, summaryMessage))

	// Write file
	if err := os.WriteFile(reportFile, []byte(annotations.String()), 0644); err != nil {
		return fmt.Errorf("failed to write GitHub annotations: %w", err)
	}

	return nil
}

// SARIFReporter generates SARIF (Static Analysis Results Interchange Format) reports
type SARIFReporter struct {
	outputDir string
}

func NewSARIFReporter(outputDir string) *SARIFReporter {
	return &SARIFReporter{outputDir: outputDir}
}

func (r *SARIFReporter) GetFormat() string    { return "sarif" }
func (r *SARIFReporter) GetExtension() string { return ".sarif" }

func (r *SARIFReporter) Generate(ctx context.Context, results *CIResults) error {
	// Ensure output directory exists
	if err := os.MkdirAll(r.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	reportFile := filepath.Join(r.outputDir, "cass-report.sarif")

	// Create SARIF structure
	sarif := map[string]interface{}{
		"$schema": "https://json.schemastore.org/sarif-2.1.0",
		"version": "2.1.0",
		"runs": []map[string]interface{}{
			{
				"tool": map[string]interface{}{
					"driver": map[string]interface{}{
						"name":           "CASS",
						"version":        "1.0.0",
						"informationUri": "https://github.com/metabase/cass",
						"rules":          r.generateSARIFRules(results),
					},
				},
				"results":    r.generateSARIFResults(results),
				"columnKind": "utf16CodeUnits",
			},
		},
	}

	data, err := json.MarshalIndent(sarif, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal SARIF report: %w", err)
	}

	if err := os.WriteFile(reportFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write SARIF report: %w", err)
	}

	return nil
}

func (r *SARIFReporter) generateSARIFRules(results *CIResults) []map[string]interface{} {
	rulesMap := make(map[string]map[string]interface{})

	for _, issues := range results.Issues {
		for _, issue := range issues {
			if _, exists := rulesMap[issue.Rule]; !exists {
				rule := map[string]interface{}{
					"id":   issue.Rule,
					"name": issue.Title,
					"shortDescription": map[string]interface{}{
						"text": issue.Description,
					},
					"fullDescription": map[string]interface{}{
						"text": fmt.Sprintf("%s: %s", issue.Type, issue.Message),
					},
					"defaultConfiguration": map[string]interface{}{
						"level": r.getSARIFLevel(issue.Severity),
					},
					"help": map[string]interface{}{
						"text": fmt.Sprintf("Suggestion: %s", issue.Suggestion),
					},
					"properties": map[string]interface{}{
						"category":          issue.Category,
						"precision":         "high",
						"security-severity": r.getSecuritySeverity(issue.Severity),
					},
				}
				rulesMap[issue.Rule] = rule
			}
		}
	}

	rules := make([]map[string]interface{}, 0, len(rulesMap))
	for _, rule := range rulesMap {
		rules = append(rules, rule)
	}

	return rules
}

func (r *SARIFReporter) generateSARIFResults(results *CIResults) []map[string]interface{} {
	var sarifResults []map[string]interface{}

	for _, issues := range results.Issues {
		for _, issue := range issues {
			result := map[string]interface{}{
				"ruleId": issue.Rule,
				"level":  r.getSARIFLevel(issue.Severity),
				"message": map[string]interface{}{
					"text": issue.Message,
				},
				"locations": []map[string]interface{}{
					{
						"physicalLocation": map[string]interface{}{
							"artifactLocation": map[string]interface{}{
								"uri": issue.Path,
							},
							"region": map[string]interface{}{
								"startLine":   issue.Line,
								"startColumn": issue.Column,
								"endLine":     issue.EndLine,
								"endColumn":   issue.EndColumn,
							},
						},
						"logicalLocations": []map[string]interface{}{
							{
								"fullyQualifiedName": fmt.Sprintf("%s::%s", issue.Type, issue.Category),
								"kind":               "function",
							},
						},
					},
				},
			}

			if issue.Context != "" {
				result["locations"].([]map[string]interface{})[0]["physicalLocation"].(map[string]interface{})["contextRegion"] = map[string]interface{}{
					"snippet": map[string]interface{}{
						"text": issue.Context,
					},
				}
			}

			sarifResults = append(sarifResults, result)
		}
	}

	return sarifResults
}

func (r *SARIFReporter) getSARIFLevel(severity string) string {
	switch severity {
	case "critical":
		return "error"
	case "high":
		return "error"
	case "medium":
		return "warning"
	case "low":
		return "note"
	default:
		return "note"
	}
}

func (r *SARIFReporter) getSecuritySeverity(severity string) float64 {
	switch severity {
	case "critical":
		return 9.0
	case "high":
		return 7.0
	case "medium":
		return 5.0
	case "low":
		return 3.0
	default:
		return 1.0
	}
}
