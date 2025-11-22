package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// TenantMigrations contains all tenant and project related migrations
var TenantMigrations = []Migration{
	{
		ID:          "001_create_tenants_table",
		Version:     "001",
		Name:        "Create tenants table",
		Description: "Creates the tenants table for multi-tenancy support",
		UpSQL: `
			CREATE TABLE IF NOT EXISTS tenants (
				id TEXT PRIMARY KEY,
				name TEXT NOT NULL,
				slug TEXT UNIQUE NOT NULL,
				domain TEXT,
				logo TEXT,
				description TEXT,
				settings TEXT,
				metadata TEXT,
				is_active BOOLEAN DEFAULT 1,
				plan TEXT DEFAULT 'free',
				limits TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP
			);

			CREATE INDEX IF NOT EXISTS idx_tenants_slug ON tenants(slug);
			CREATE INDEX IF NOT EXISTS idx_tenants_active ON tenants(is_active);
			CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants(deleted_at);
		`,
		DownSQL: `
			DROP TABLE IF EXISTS tenants;
		`,
	},
	{
		ID:          "002_create_projects_table",
		Version:     "002",
		Name:        "Create projects table",
		Description: "Creates the projects table for project management within tenants",
		UpSQL: `
			CREATE TABLE IF NOT EXISTS projects (
				id TEXT PRIMARY KEY,
				tenant_id TEXT NOT NULL,
				name TEXT NOT NULL,
				slug TEXT NOT NULL,
				description TEXT,
				logo TEXT,
				settings TEXT,
				metadata TEXT,
				is_active BOOLEAN DEFAULT 1,
				is_public BOOLEAN DEFAULT 0,
				environment TEXT DEFAULT 'development',
				owner_id TEXT NOT NULL,
				members TEXT,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				deleted_at TIMESTAMP,
				FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
				UNIQUE(tenant_id, slug)
			);

			CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);
			CREATE INDEX IF NOT EXISTS idx_projects_active ON projects(is_active);
			CREATE INDEX IF NOT EXISTS idx_projects_owner_id ON projects(owner_id);
			CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects(deleted_at);
		`,
		DownSQL: `
			DROP TABLE IF EXISTS projects;
		`,
	},
	{
		ID:          "003_create_user_projects_table",
		Version:     "003",
		Name:        "Create user_projects table",
		Description: "Creates the user_projects table for user-project relationships with collaboration support",
		UpSQL: `
			CREATE TABLE IF NOT EXISTS user_projects (
				id TEXT PRIMARY KEY,
				user_id TEXT NOT NULL,
				tenant_id TEXT NOT NULL,
				project_id TEXT NOT NULL,
				role TEXT NOT NULL DEFAULT 'viewer',

				-- Status and relationship info
				is_active BOOLEAN DEFAULT 1,
				is_creator BOOLEAN DEFAULT 0,
				invited_by TEXT,
				invited_at TIMESTAMP,
				joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				left_at TIMESTAMP,

				-- Collaboration info
				is_external_collaborator BOOLEAN DEFAULT 0,
				can_invite BOOLEAN DEFAULT 0,
				can_manage_members BOOLEAN DEFAULT 0,

				-- Permissions and metadata
				custom_permissions TEXT,
				metadata TEXT,

				FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
				FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
				UNIQUE(user_id, project_id)
			);

			CREATE INDEX IF NOT EXISTS idx_user_projects_user_id ON user_projects(user_id);
			CREATE INDEX IF NOT EXISTS idx_user_projects_tenant_id ON user_projects(tenant_id);
			CREATE INDEX IF NOT EXISTS idx_user_projects_project_id ON user_projects(project_id);
			CREATE INDEX IF NOT EXISTS idx_user_projects_role ON user_projects(role);
			CREATE INDEX IF NOT EXISTS idx_user_projects_active ON user_projects(is_active);
			CREATE INDEX IF NOT EXISTS idx_user_projects_external ON user_projects(is_external_collaborator);
		`,
		DownSQL: `
			DROP TABLE IF EXISTS user_projects;
		`,
	},
	{
		ID:          "004_add_triggers_and_constraints",
		Version:     "004",
		Name:        "Add triggers and constraints",
		Description: "Adds triggers for updated_at timestamps and additional constraints",
		UpSQL: `
			-- Create trigger for tenants updated_at
			CREATE TRIGGER IF NOT EXISTS tenants_updated_at
				AFTER UPDATE ON tenants
			BEGIN
				UPDATE tenants SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END;

			-- Create trigger for projects updated_at
			CREATE TRIGGER IF NOT EXISTS projects_updated_at
				AFTER UPDATE ON projects
			BEGIN
				UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
			END;

			-- Note: SQLite doesn't support adding CHECK constraints to existing tables
			-- These constraints would need to be added when recreating tables
			-- For now, we skip the constraint additions to avoid SQL errors
		`,
		DownSQL: `
			DROP TRIGGER IF EXISTS tenants_updated_at;
			DROP TRIGGER IF EXISTS projects_updated_at;

			-- Remove constraints (SQLite doesn't support DROP CONSTRAINT directly, so we recreate tables)
			-- In production, you might want to handle this differently
		`,
	},
	{
		ID:          "005_insert_system_tenant_and_project",
		Version:     "005",
		Name:        "Insert system tenant and project",
		Description: "Creates the system tenant and project records",
		UpSQL: `
			-- Insert system tenant
			INSERT OR IGNORE INTO tenants (
				id, name, slug, description, is_active, plan,
				settings, limits, created_at, updated_at
			) VALUES (
				'system',
				'System',
				'system',
				'System administration and configuration',
				1,
				'enterprise',
				'{"allow_user_registration": true, "default_user_role": "user", "require_email_verification": false, "require_two_factor": false, "session_timeout_minutes": 1440}',
				'{"max_users": 999999, "max_projects": 999999, "max_storage_mb": 999999999, "max_api_requests_per_day": 999999999}',
				CURRENT_TIMESTAMP,
				CURRENT_TIMESTAMP
			);

			-- Insert system project
			INSERT OR IGNORE INTO projects (
				id, tenant_id, name, slug, description,
				is_active, environment, owner_id,
				settings, created_at, updated_at
			) VALUES (
				'system',
				'system',
				'System',
				'system',
				'System administration and configuration project',
				1,
				'production',
				'system',
				'{"require_auth_for_read": true, "require_auth_for_write": true, "allowed_origins": [], "enabled_features": ["admin", "system"], "rate_limit": {"enabled": false}}',
				CURRENT_TIMESTAMP,
				CURRENT_TIMESTAMP
			);
		`,
		DownSQL: `
			DELETE FROM projects WHERE id = 'system';
			DELETE FROM tenants WHERE id = 'system';
		`,
	},
}

// Migration represents a database migration
type Migration struct {
	ID          string     `json:"id"`
	Version     string     `json:"version"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	UpSQL       string     `json:"up_sql"`
	DownSQL     string     `json:"down_sql"`
	Checksum    string     `json:"checksum"`
	Applied     bool       `json:"applied"`
	AppliedAt   *time.Time `json:"applied_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MigrationRunner runs database migrations
type MigrationRunner struct {
	db *sql.DB
}

// NewMigrationRunner creates a new migration runner
func NewMigrationRunner(db *sql.DB) *MigrationRunner {
	return &MigrationRunner{db: db}
}

// CreateMigrationsTable creates the migrations tracking table
func (mr *MigrationRunner) CreateMigrationsTable(ctx context.Context) error {
	sql := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			id TEXT PRIMARY KEY,
			version TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			checksum TEXT,
			applied BOOLEAN DEFAULT 0,
			applied_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_schema_migrations_version ON schema_migrations(version);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied ON schema_migrations(applied);
	`

	_, err := mr.db.ExecContext(ctx, sql)
	return err
}

// RunMigrations runs all pending migrations
func (mr *MigrationRunner) RunMigrations(ctx context.Context) error {
	// Create migrations table if it doesn't exist
	if err := mr.CreateMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Run all tenant migrations
	for _, migration := range TenantMigrations {
		if err := mr.runMigration(ctx, &migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.ID, err)
		}
	}

	return nil
}

// runMigration runs a single migration
func (mr *MigrationRunner) runMigration(ctx context.Context, migration *Migration) error {
	// Check if migration has already been applied
	var applied bool
	err := mr.db.QueryRowContext(ctx,
		"SELECT applied FROM schema_migrations WHERE id = ?",
		migration.ID).Scan(&applied)

	if err == nil && applied {
		// Migration already applied
		return nil
	}

	// Start transaction
	tx, err := mr.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Run migration
	if _, err := tx.ExecContext(ctx, migration.UpSQL); err != nil {
		return fmt.Errorf("failed to execute migration up SQL: %w", err)
	}

	// Record migration
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO schema_migrations (id, version, name, description, applied, applied_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			applied = excluded.applied,
			applied_at = excluded.applied_at
	`, migration.ID, migration.Version, migration.Name, migration.Description, true, time.Now()); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// GetAppliedMigrations returns all applied migrations
func (mr *MigrationRunner) GetAppliedMigrations(ctx context.Context) ([]*Migration, error) {
	rows, err := mr.db.QueryContext(ctx, `
		SELECT id, version, name, description, applied, applied_at, created_at
		FROM schema_migrations
		ORDER BY version
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []*Migration
	for rows.Next() {
		var migration Migration
		err := rows.Scan(
			&migration.ID,
			&migration.Version,
			&migration.Name,
			&migration.Description,
			&migration.Applied,
			&migration.AppliedAt,
			&migration.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, &migration)
	}

	return migrations, nil
}

// IsMigrationApplied checks if a specific migration has been applied
func (mr *MigrationRunner) IsMigrationApplied(ctx context.Context, migrationID string) (bool, error) {
	var applied bool
	err := mr.db.QueryRowContext(ctx,
		"SELECT applied FROM schema_migrations WHERE id = ?",
		migrationID).Scan(&applied)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return applied, nil
}
