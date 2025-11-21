package rls

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

// PolicyParser parses policy expressions
type PolicyParser struct {
	fset *token.FileSet
}

// NewPolicyParser creates a new policy parser
func NewPolicyParser() *PolicyParser {
	return &PolicyParser{
		fset: token.NewFileSet(),
	}
}

// ParseExpression parses an expression string into an AST
func (p *PolicyParser) ParseExpression(expr string) (ast.Expr, error) {
	// Wrap expression in a valid Go syntax for parsing
	wrapped := fmt.Sprintf("package main\nfunc test() {\n  _ = %s\n}", expr)

	// Parse the expression
	node, err := parser.ParseFile(p.fset, "", wrapped, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse expression: %w", err)
	}

	// Extract the expression from the AST
	decl := node.Decls[0].(*ast.GenDecl)
	if decl.Tok != token.VAR {
		return nil, fmt.Errorf("expected variable declaration")
	}

	valueSpec := decl.Specs[0].(*ast.ValueSpec)
	exprStmt := valueSpec.Values[0].(*ast.ExprStmt)

	return exprStmt.X, nil
}

// ParsePolicyDefinition parses a complete policy definition
func (p *PolicyParser) ParsePolicyDefinition(definition string) (*PolicyDefinition, error) {
	// This is a simplified parser for policy definitions
	// In a real implementation, you might want to use a more sophisticated parser

	def := &PolicyDefinition{
		Rules: make([]PolicyRule, 0),
	}

	lines := strings.Split(definition, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		rule, err := p.parseRule(line)
		if err != nil {
			return nil, err
		}

		if rule != nil {
			def.Rules = append(def.Rules, *rule)
		}
	}

	return def, nil
}

// parseRule parses a single policy rule
func (p *PolicyParser) parseRule(line string) (*PolicyRule, error) {
	// Simple rule parsing
	// Examples:
	// current_user_id = row.user_id
	// row.tenant_id = current_tenant_id
	// ${user.roles} CONTAINS 'admin'
	// row.created_at > '2024-01-01'

	rule := &PolicyRule{}

	// Handle common patterns
	if strings.Contains(line, "=") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			rule.Field = strings.TrimSpace(parts[0])
			rule.Operator = "=="
			rule.Value = strings.TrimSpace(parts[1])
			return rule, nil
		}
	}

	if strings.Contains(line, ">") {
		parts := strings.SplitN(line, ">", 2)
		if len(parts) == 2 {
			rule.Field = strings.TrimSpace(parts[0])
			rule.Operator = ">"
			rule.Value = strings.TrimSpace(parts[1])
			return rule, nil
		}
	}

	if strings.Contains(line, "<") {
		parts := strings.SplitN(line, "<", 2)
		if len(parts) == 2 {
			rule.Field = strings.TrimSpace(parts[0])
			rule.Operator = "<"
			rule.Value = strings.TrimSpace(parts[1])
			return rule, nil
		}
	}

	if strings.Contains(line, "CONTAINS") {
		parts := strings.SplitN(line, "CONTAINS", 2)
		if len(parts) == 2 {
			rule.Field = strings.TrimSpace(parts[0])
			rule.Operator = "CONTAINS"
			rule.Value = strings.TrimSpace(parts[1])
			return rule, nil
		}
	}

	if strings.Contains(line, "IN") {
		parts := strings.SplitN(line, "IN", 2)
		if len(parts) == 2 {
			rule.Field = strings.TrimSpace(parts[0])
			rule.Operator = "IN"
			rule.Value = strings.TrimSpace(parts[1])
			return rule, nil
		}
	}

	return nil, nil // Skip unrecognized lines
}

// PolicyDefinition represents a parsed policy definition
type PolicyDefinition struct {
	Rules []PolicyRule `json:"rules"`
}

// PolicyRule represents a single rule in a policy definition
type PolicyRule struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
	Negated  bool   `json:"negated"`
}

// PolicyEvaluator evaluates parsed policies
type PolicyEvaluator struct {
}

// NewPolicyEvaluator creates a new policy evaluator
func NewPolicyEvaluator() *PolicyEvaluator {
	return &PolicyEvaluator{}
}

// Evaluate evaluates a rule against data
func (e *PolicyEvaluator) Evaluate(rule PolicyRule, data map[string]interface{}, userContext map[string]interface{}) (bool, error) {
	// Get field value from data
	fieldValue, exists := e.getFieldValue(rule.Field, data, userContext)
	if !exists {
		return false, fmt.Errorf("field not found: %s", rule.Field)
	}

	// Parse rule value
	ruleValue := e.parseValue(rule.Value, userContext)

	// Evaluate based on operator
	switch rule.Operator {
	case "==":
		return e.compareEqual(fieldValue, ruleValue), nil
	case "!=":
		return !e.compareEqual(fieldValue, ruleValue), nil
	case ">":
		return e.compareGreater(fieldValue, ruleValue), nil
	case ">=":
		return e.compareGreaterEqual(fieldValue, ruleValue), nil
	case "<":
		return e.compareLess(fieldValue, ruleValue), nil
	case "<=":
		return e.compareLessEqual(fieldValue, ruleValue), nil
	case "CONTAINS":
		return e.contains(fieldValue, ruleValue), nil
	case "IN":
		return e.inArray(fieldValue, ruleValue), nil
	case "LIKE":
		return e.like(fieldValue, ruleValue), nil
	default:
		return false, fmt.Errorf("unsupported operator: %s", rule.Operator)
	}
}

// getFieldValue gets field value from data or user context
func (e *PolicyEvaluator) getFieldValue(field string, data map[string]interface{}, userContext map[string]interface{}) (interface{}, bool) {
	// Check if it's a row field
	if strings.HasPrefix(field, "row.") {
		rowField := strings.TrimPrefix(field, "row.")
		value, exists := data[rowField]
		return value, exists
	}

	// Check if it's a user field
	if strings.HasPrefix(field, "user.") {
		userField := strings.TrimPrefix(field, "user.")
		value, exists := userContext[userField]
		return value, exists
	}

	// Check if it's a current_user field
	if strings.HasPrefix(field, "current_user_") {
		userField := strings.TrimPrefix(field, "current_user_")
		value, exists := userContext[userField]
		return value, exists
	}

	// Direct field lookup in data
	value, exists := data[field]
	if exists {
		return value, true
	}

	// Direct field lookup in user context
	value, exists = userContext[field]
	return value, exists
}

// parseValue parses a rule value, handling variables
func (e *PolicyEvaluator) parseValue(value string, userContext map[string]interface{}) interface{} {
	// Handle quoted strings
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return strings.Trim(value, "'")
	}

	// Handle numeric values
	if e.isNumeric(value) {
		return value // Return as string for comparison
	}

	// Handle boolean values
	if strings.ToLower(value) == "true" {
		return true
	}
	if strings.ToLower(value) == "false" {
		return false
	}

	// Handle array values
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		return e.parseArray(value)
	}

	// Handle variables
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		varName := strings.Trim(value, "${}")
		if val, exists := userContext[varName]; exists {
			return val
		}
	}

	return value
}

// isNumeric checks if a string represents a number
func (e *PolicyEvaluator) isNumeric(s string) bool {
	// Simple numeric check
	for _, char := range s {
		if char != '.' && (char < '0' || char > '9') && char != '-' {
			return false
		}
	}
	return len(s) > 0
}

// parseArray parses an array value like ['value1', 'value2']
func (e *PolicyEvaluator) parseArray(value string) []interface{} {
	// Remove brackets and split by comma
	content := strings.Trim(value, "[]")
	if content == "" {
		return []interface{}{}
	}

	parts := strings.Split(content, ",")
	result := make([]interface{}, 0)

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "'") && strings.HasSuffix(part, "'") {
			result = append(result, strings.Trim(part, "'"))
		} else if e.isNumeric(part) {
			result = append(result, part)
		}
	}

	return result
}

// Comparison functions
func (e *PolicyEvaluator) compareEqual(a, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func (e *PolicyEvaluator) compareGreater(a, b interface{}) bool {
	// Simple numeric comparison
	aStr, bStr := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	if e.isNumeric(aStr) && e.isNumeric(bStr) {
		// In a real implementation, convert to float64 and compare
		return aStr > bStr
	}
	return aStr > bStr
}

func (e *PolicyEvaluator) compareGreaterEqual(a, b interface{}) bool {
	return e.compareEqual(a, b) || e.compareGreater(a, b)
}

func (e *PolicyEvaluator) compareLess(a, b interface{}) bool {
	return !e.compareGreaterEqual(a, b)
}

func (e *PolicyEvaluator) compareLessEqual(a, b interface{}) bool {
	return !e.compareGreater(a, b)
}

func (e *PolicyEvaluator) contains(a, b interface{}) bool {
	aStr, bStr := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	return strings.Contains(aStr, bStr)
}

func (e *PolicyEvaluator) inArray(a, b interface{}) bool {
	// a should be the value, b should be the array
	array, ok := b.([]interface{})
	if !ok {
		return false
	}

	for _, item := range array {
		if e.compareEqual(a, item) {
			return true
		}
	}

	return false
}

func (e *PolicyEvaluator) like(a, b interface{}) bool {
	aStr, bStr := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	// Convert SQL LIKE pattern to regex pattern
	pattern := strings.ReplaceAll(bStr, "%", ".*")
	pattern = strings.ReplaceAll(pattern, "_", ".")
	matched, _ := regexp.MatchString("^"+pattern+"$", aStr)
	return matched
}
