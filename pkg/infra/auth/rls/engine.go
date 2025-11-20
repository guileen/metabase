package rls

import ("context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/guileen/metabase/pkg/infra/auth")

// PolicyType represents the type of RLS policy
type PolicyType string

const (
	PolicyTypeSelect   PolicyType = "SELECT"
	PolicyTypeInsert   PolicyType = "INSERT"
	PolicyTypeUpdate   PolicyType = "UPDATE"
	PolicyTypeDelete   PolicyType = "DELETE"
	PolicyTypeAll      PolicyType = "ALL"
)

// PolicyEffect represents the effect of RLS policy
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "ALLOW"
	PolicyEffectDeny  PolicyEffect = "DENY"
)

// Policy represents a Row Level Security policy
type Policy struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Table       string                 `json:"table"`
	TenantID    string                 `json:"tenant_id"`
	Type        PolicyType             `json:"type"`
	Effect      PolicyEffect           `json:"effect"`
	Definition  string                 `json:"definition"`
	Using       string                 `json:"using,omitempty"`       // For SELECT, UPDATE, DELETE
	WithCheck   string                 `json:"with_check,omitempty"`   // For INSERT, UPDATE
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	CheckGroups []string               `json:"check_groups,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CreatedBy   string                 `json:"created_by"`
	UpdatedBy   string                 `json:"updated_by"`
}

// ExecutionContext represents the context for policy evaluation
type ExecutionContext struct {
	UserID    string                 `json:"user_id"`
	TenantID  string                 `json:"tenant_id"`
	ProjectID string                 `json:"project_id"`
	Roles     []string               `json:"roles"`
	Claims    map[string]interface{} `json:"claims"`
	RequestID string                 `json:"request_id"`
	IP        string                 `json:"ip"`
	UserAgent string                 `json:"user_agent"`
	Time      time.Time              `json:"time"`
}

// PolicyEvaluationResult represents the result of policy evaluation
type PolicyEvaluationResult struct {
	Allowed    bool                   `json:"allowed"`
	Policy     *Policy                `json:"policy,omitempty"`
	Filter     string                 `json:"filter,omitempty"`
	Check      string                 `json:"check,omitempty"`
	Reason     string                 `json:"reason,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	EvaluatedAt time.Time             `json:"evaluated_at"`
}

// RowFilter represents a row filter for SELECT operations
type RowFilter struct {
	SQL    string                 `json:"sql"`
	Args   []interface{}          `json:"args"`
	Meta   map[string]interface{} `json:"meta"`
}

// RLSEngine manages Row Level Security policies
type RLSEngine struct {
	policies     map[string][]*Policy // table -> policies
	compiled     map[string]*CompiledPolicy
	parser       *PolicyParser
	evaluator    *PolicyEvaluator
	rbac         *auth.RBACManager
	cache        *PolicyCache
	mu           sync.RWMutex
	config       *RLSConfig
}

// RLSConfig represents RLS configuration
type RLSConfig struct {
	Enabled           bool          `json:"enabled"`
	CacheEnabled      bool          `json:"cache_enabled"`
	CacheSize         int           `json:"cache_size"`
	CacheTTL          time.Duration `json:"cache_ttl"`
	DebugMode         bool          `json:"debug_mode"`
	MaxPolicyPerTable int           `json:"max_policy_per_table"`
	Timeout           time.Duration `json:"timeout"`
}

// CompiledPolicy represents a compiled policy for faster execution
type CompiledPolicy struct {
	Policy   *Policy
	Filter   ast.Expr
	Check    ast.Expr
	Compiled bool
}

// PolicyCache provides caching for policy evaluations
type PolicyCache struct {
	entries map[string]*CacheEntry
	maxSize int
	mu      sync.RWMutex
	ttl     time.Duration
}

// CacheEntry represents a cached policy evaluation
type CacheEntry struct {
	Key       string
	Result    *PolicyEvaluationResult
	ExpiresAt time.Time
	HitCount  int64
}

// NewRLSEngine creates a new RLS engine
func NewRLSEngine(rbac *auth.RBACManager, config *RLSConfig) *RLSEngine {
	if config == nil {
		config = &RLSConfig{
			Enabled:           true,
			CacheEnabled:      true,
			CacheSize:         1000,
			CacheTTL:          5 * time.Minute,
			DebugMode:         false,
			MaxPolicyPerTable: 50,
			Timeout:           30 * time.Second,
		}
	}

	return &RLSEngine{
		policies: make(map[string][]*Policy),
		compiled: make(map[string]*CompiledPolicy),
		parser:   NewPolicyParser(),
		evaluator: NewPolicyEvaluator(),
		rbac:      rbac,
		cache:     NewPolicyCache(config.CacheSize, config.CacheTTL),
		config:    config,
	}
}

// AddPolicy adds a new RLS policy
func (r *RLSEngine) AddPolicy(policy *Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate policy
	if err := r.validatePolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	// Initialize policies slice if not exists
	if r.policies[policy.Table] == nil {
		r.policies[policy.Table] = make([]*Policy, 0)
	}

	// Check policy limit
	if len(r.policies[policy.Table]) >= r.config.MaxPolicyPerTable {
		return fmt.Errorf("maximum number of policies per table exceeded: %d", r.config.MaxPolicyPerTable)
	}

	// Add policy
	r.policies[policy.Table] = append(r.policies[policy.Table], policy)

	// Invalidate cache
	r.cache.Invalidate(fmt.Sprintf("table:%s", policy.Table))

	// Precompile policy
	if err := r.precompilePolicy(policy); err != nil {
		return fmt.Errorf("failed to precompile policy: %w", err)
	}

	return nil
}

// RemovePolicy removes an RLS policy
func (r *RLSEngine) RemovePolicy(table, policyID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	policies, exists := r.policies[table]
	if !exists {
		return fmt.Errorf("no policies found for table: %s", table)
	}

	var updatedPolicies []*Policy
	found := false

	for _, policy := range policies {
		if policy.ID == policyID {
			found = true
			continue
		}
		updatedPolicies = append(updatedPolicies, policy)
	}

	if !found {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	if len(updatedPolicies) == 0 {
		delete(r.policies, table)
	} else {
		r.policies[table] = updatedPolicies
	}

	// Remove compiled policy
	delete(r.compiled, policyID)

	// Invalidate cache
	r.cache.Invalidate(fmt.Sprintf("table:%s", table))

	return nil
}

// CheckPermission checks if a user has permission for an operation
func (r *RLSEngine) CheckPermission(ctx context.Context, table, operation string, execCtx *ExecutionContext) (*PolicyEvaluationResult, error) {
	if !r.config.Enabled {
		return &PolicyEvaluationResult{
			Allowed:      true,
			Reason:       "RLS disabled",
			EvaluatedAt:  time.Now(),
		}, nil
	}

	// Check cache first if enabled
	if r.config.CacheEnabled {
		cacheKey := r.getCacheKey(table, operation, execCtx)
		if cached, exists := r.cache.Get(cacheKey); exists {
			cached.HitCount++
			return cached.Result, nil
		}
	}

	// Get policies for table and operation
	policies, err := r.getPoliciesForOperation(table, PolicyType(operation))
	if err != nil {
		return nil, err
	}

	// Evaluate policies in priority order
	result := r.evaluatePolicies(policies, execCtx)

	// Cache result if enabled
	if r.config.CacheEnabled {
		r.cache.Set(cacheKey, &CacheEntry{
			Key:       cacheKey,
			Result:    result,
			ExpiresAt: time.Now().Add(r.config.CacheTTL),
		})
	}

	return result, nil
}

// GetRowFilter generates a row filter for SELECT operations
func (r *RLSEngine) GetRowFilter(ctx context.Context, table string, execCtx *ExecutionContext) (*RowFilter, error) {
	result, err := r.CheckPermission(ctx, table, "SELECT", execCtx)
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return nil, fmt.Errorf("access denied: %s", result.Reason)
	}

	if result.Filter == "" {
		// No filter, allow all rows
		return &RowFilter{SQL: "1=1"}, nil
	}

	return &RowFilter{
		SQL:  result.Filter,
		Args: []interface{}{},
	}, nil
}

// ValidateInsert validates data for INSERT operations
func (r *RLSEngine) ValidateInsert(ctx context.Context, table string, data map[string]interface{}, execCtx *ExecutionContext) (*PolicyEvaluationResult, error) {
	result, err := r.CheckPermission(ctx, table, "INSERT", execCtx)
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return result, nil
	}

	// Apply WITH CHECK constraints
	if result.Check != "" {
		if !r.evaluateConstraint(result.Check, data, execCtx) {
			return &PolicyEvaluationResult{
				Allowed:      false,
				Reason:       "Row check constraint failed",
				Policy:       result.Policy,
				EvaluatedAt:  time.Now(),
			}, nil
		}
	}

	return result, nil
}

// ValidateUpdate validates data for UPDATE operations
func (r *RLSEngine) ValidateUpdate(ctx context.Context, table string, oldData, newData map[string]interface{}, execCtx *ExecutionContext) (*PolicyEvaluationResult, error) {
	result, err := r.CheckPermission(ctx, table, "UPDATE", execCtx)
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return result, nil
	}

	// Apply USING clause (row filter)
	if result.Filter != "" {
		if !r.evaluateConstraint(result.Filter, oldData, execCtx) {
			return &PolicyEvaluationResult{
				Allowed:      false,
				Reason:       "Row access denied by USING clause",
				Policy:       result.Policy,
				EvaluatedAt:  time.Now(),
			}, nil
		}
	}

	// Apply WITH CHECK clause
	if result.Check != "" {
		if !r.evaluateConstraint(result.Check, newData, execCtx) {
			return &PolicyEvaluationResult{
				Allowed:      false,
				Reason:       "Row update denied by CHECK constraint",
				Policy:       result.Policy,
				EvaluatedAt:  time.Now(),
			}, nil
		}
	}

	return result, nil
}

// ValidateDelete validates DELETE operations
func (r *RLSEngine) ValidateDelete(ctx context.Context, table string, data map[string]interface{}, execCtx *ExecutionContext) (*PolicyEvaluationResult, error) {
	result, err := r.CheckPermission(ctx, table, "DELETE", execCtx)
	if err != nil {
		return nil, err
	}

	if !result.Allowed {
		return result, nil
	}

	// Apply USING clause
	if result.Filter != "" {
		if !r.evaluateConstraint(result.Filter, data, execCtx) {
			return &PolicyEvaluationResult{
				Allowed:      false,
				Reason:       "Row delete denied by USING clause",
				Policy:       result.Policy,
				EvaluatedAt:  time.Now(),
			}, nil
		}
	}

	return result, nil
}

// getPoliciesForOperation gets policies for a specific table and operation
func (r *RLSEngine) getPoliciesForOperation(table string, operation PolicyType) ([]*Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	policies, exists := r.policies[table]
	if !exists {
		return []*Policy{}, nil
	}

	var filtered []*Policy
	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		// Check if policy applies to operation
		if policy.Type == operation || policy.Type == PolicyTypeAll {
			filtered = append(filtered, policy)
		}
	}

	// Sort by priority (higher priority first)
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[i].Priority < filtered[j].Priority {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	return filtered, nil
}

// evaluatePolicies evaluates a list of policies
func (r *RLSEngine) evaluatePolicies(policies []*Policy, execCtx *ExecutionContext) *PolicyEvaluationResult {
	// DENY policies take precedence over ALLOW
	for _, policy := range policies {
		if policy.Effect == PolicyEffectDeny {
			if r.evaluatePolicyCondition(policy, execCtx) {
				return &PolicyEvaluationResult{
					Allowed:      false,
					Policy:       policy,
					Reason:       "Denied by policy",
					EvaluatedAt:  time.Now(),
				}
			}
		}
	}

	// Check ALLOW policies
	for _, policy := range policies {
		if policy.Effect == PolicyEffectAllow {
			if r.evaluatePolicyCondition(policy, execCtx) {
				return &PolicyEvaluationResult{
					Allowed:      true,
					Policy:       policy,
					Filter:       policy.Using,
					Check:        policy.WithCheck,
					Reason:       "Allowed by policy",
					EvaluatedAt:  time.Now(),
				}
			}
		}
	}

	// Default deny
	return &PolicyEvaluationResult{
		Allowed:      false,
		Reason:       "No matching policy found (default deny)",
		EvaluatedAt:  time.Now(),
	}
}

// evaluatePolicyCondition evaluates if policy condition matches
func (r *RLSEngine) evaluatePolicyCondition(policy *Policy, execCtx *ExecutionContext) bool {
	if policy.Definition == "" {
		return true
	}

	// Simple evaluation - in a real implementation, this would be more sophisticated
	return r.evaluateSimpleCondition(policy.Definition, execCtx)
}

// evaluateSimpleCondition evaluates a simple policy condition
func (r *RLSEngine) evaluateSimpleCondition(condition string, execCtx *ExecutionContext) bool {
	// Replace common variables
	replacer := strings.NewReplacer(
		"${user.id}", execCtx.UserID,
		"${user.tenant_id}", execCtx.TenantID,
		"${user.project_id}", execCtx.ProjectID,
		"${current_user_id}", execCtx.UserID,
		"${current_tenant_id}", execCtx.TenantID,
	)

	condition = replacer.Replace(condition)

	// Simple evaluations
	if strings.Contains(condition, "${user.roles}") {
		for _, role := range execCtx.Roles {
			if strings.Contains(condition, fmt.Sprintf(`"%s"`, role)) {
				return true
			}
		}
		return false
	}

	// Default to true for simple conditions
	return true
}

// evaluateConstraint evaluates a constraint expression
func (r *RLSEngine) evaluateConstraint(constraint string, data map[string]interface{}, execCtx *ExecutionContext) bool {
	// This is a simplified implementation
	// In a real implementation, you would use a proper expression evaluator

	// Replace data variables
	for key, value := range data {
		placeholder := fmt.Sprintf("${row.%s}", key)
		constraint = strings.ReplaceAll(constraint, placeholder, fmt.Sprintf("%v", value))
	}

	// Replace user variables
	replacer := strings.NewReplacer(
		"${user.id}", execCtx.UserID,
		"${user.tenant_id}", execCtx.TenantID,
		"${user.project_id}", execCtx.ProjectID,
	)

	constraint = replacer.Replace(constraint)

	// Simple evaluation for common patterns
	if strings.Contains(constraint, "==") {
		parts := strings.Split(constraint, "==")
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
		}
	}

	if strings.Contains(constraint, "!=") {
		parts := strings.Split(constraint, "!=")
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
		}
	}

	// Default to true for complex constraints
	return true
}

// validatePolicy validates a policy
func (r *RLSEngine) validatePolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}

	if policy.Table == "" {
		return fmt.Errorf("policy table cannot be empty")
	}

	if policy.TenantID == "" {
		return fmt.Errorf("policy tenant ID cannot be empty")
	}

	if policy.Effect != PolicyEffectAllow && policy.Effect != PolicyEffectDeny {
		return fmt.Errorf("invalid policy effect: %s", policy.Effect)
	}

	// Validate policy type
	validTypes := []PolicyType{
		PolicyTypeSelect, PolicyTypeInsert, PolicyTypeUpdate, PolicyTypeDelete, PolicyTypeAll,
	}
	validType := false
	for _, t := range validTypes {
		if policy.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		return fmt.Errorf("invalid policy type: %s", policy.Type)
	}

	return nil
}

// precompilePolicy precompiles a policy for faster execution
func (r *RLSEngine) precompilePolicy(policy *Policy) error {
	compiled := &CompiledPolicy{
		Policy:   policy,
		Compiled: false,
	}

	// Parse USING clause if present
	if policy.Using != "" {
		filter, err := r.parser.ParseExpression(policy.Using)
		if err != nil {
			return fmt.Errorf("failed to parse USING clause: %w", err)
		}
		compiled.Filter = filter
	}

	// Parse WITH CHECK clause if present
	if policy.WithCheck != "" {
		check, err := r.parser.ParseExpression(policy.WithCheck)
		if err != nil {
			return fmt.Errorf("failed to parse WITH CHECK clause: %w", err)
		}
		compiled.Check = check
	}

	compiled.Compiled = true
	r.compiled[policy.ID] = compiled

	return nil
}

// getCacheKey generates cache key for policy evaluation
func (r *RLSEngine) getCacheKey(table, operation string, execCtx *ExecutionContext) string {
	return fmt.Sprintf("policy:%s:%s:%s:%s:%v", table, operation, execCtx.UserID, execCtx.TenantID, execCtx.Roles)
}

// NewPolicyCache creates a new policy cache
func NewPolicyCache(size int, ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		entries: make(map[string]*CacheEntry),
		maxSize: size,
		ttl:     ttl,
	}
}

// Get gets cached policy evaluation
func (pc *PolicyCache) Get(key string) (*CacheEntry, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()

	entry, exists := pc.entries[key]
	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, false
	}

	return entry, true
}

// Set sets cached policy evaluation
func (pc *PolicyCache) Set(key string, entry *CacheEntry) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	// Remove oldest entries if cache is full
	if len(pc.entries) >= pc.maxSize {
		pc.evictOldest()
	}

	pc.entries[key] = entry
}

// Invalidate invalidates cache entries
func (pc *PolicyCache) Invalidate(pattern string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pattern == "*" {
		pc.entries = make(map[string]*CacheEntry)
		return
	}

	for key := range pc.entries {
		if strings.Contains(key, pattern) {
			delete(pc.entries, key)
		}
	}
}

// evictOldest removes oldest cache entry
func (pc *PolicyCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, entry := range pc.entries {
		if entry.ExpiresAt.Before(oldestTime) {
			oldestTime = entry.ExpiresAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(pc.entries, oldestKey)
	}
}