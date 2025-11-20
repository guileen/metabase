/**
 * Core client configuration and types
 */

export interface MetaBaseConfig {
  /** API base URL */
  url: string;
  /** API key for authentication */
  apiKey: string;
  /** Request timeout in milliseconds (default: 30000) */
  timeout?: number;
  /** Custom headers to send with every request */
  headers?: Record<string, string>;
  /** Enable debug logging */
  debug?: boolean;
}

export interface APIResponse<T = any> {
  /** Response data */
  data?: T;
  /** Total count (for paginated responses) */
  count?: number;
  /** Current limit (for paginated responses) */
  limit?: number;
  /** Current offset (for paginated responses) */
  offset?: number;
  /** Whether there are more results */
  hasNext?: boolean;
  /** Error information */
  error?: APIError;
}

export interface APIError {
  /** Error code */
  code: string;
  /** Human-readable error message */
  message: string;
  /** Detailed error information */
  details?: string;
  /** Error timestamp */
  timestamp?: string;
}

export interface QueryOptions {
  /** Fields to select (comma-separated) */
  select?: string;
  /** WHERE conditions */
  where?: Record<string, any>;
  /** ORDER BY clause */
  order?: string;
  /** LIMIT clause */
  limit?: number;
  /** OFFSET clause */
  offset?: number;
  /** JOIN clauses */
  joins?: JoinClause[];
  /** GROUP BY fields */
  groupBy?: string[];
  /** HAVING conditions */
  having?: Record<string, any>;
}

export interface JoinClause {
  /** Join type */
  type: 'inner' | 'left' | 'right' | 'outer';
  /** Table to join */
  table: string;
  /** Table alias */
  alias?: string;
  /** Join condition */
  condition: string;
}

export interface InsertOptions {
  /** Fields to return after insert */
  returning?: string[];
}

export interface UpdateOptions {
  /** Fields to return after update */
  returning?: string[];
}

export interface HealthResponse {
  /** Service status */
  status: string;
  /** Service version */
  version: string;
  /** Uptime */
  uptime: string;
  /** Database status */
  database: DatabaseStatus;
  /** Cache status */
  cache: CacheStatus;
  /** Timestamp */
  timestamp: string;
}

export interface DatabaseStatus {
  /** Whether database is connected */
  connected: boolean;
  /** Database version */
  version: string;
}

export interface CacheStatus {
  /** Whether cache is connected */
  connected: boolean;
  /** Cache type */
  type: string;
}

/**
 * API Key types
 */
export type KeyType = 'system' | 'user' | 'service';
export type KeyStatus = 'active' | 'inactive' | 'revoked' | 'expired';

export interface APIKey {
  /** Key ID */
  id: string;
  /** Key name */
  name: string;
  /** Key type */
  type: KeyType;
  /** Key status */
  status: KeyStatus;
  /** Permission scopes */
  scopes: string[];
  /** Tenant ID (null for system-level) */
  tenant_id?: string;
  /** Project ID (null for tenant-level or system-level) */
  project_id?: string;
  /** Creator */
  created_by: string;
  /** Associated user ID */
  user_id?: string;
  /** Expiration time */
  expires_at?: string;
  /** Creation time */
  created_at: string;
  /** Update time */
  updated_at: string;
  /** Last used time */
  last_used_at?: string;
  /** Usage count */
  usage_count: number;
  /** Additional metadata */
  metadata?: Record<string, any>;
}

export interface CreateKeyRequest {
  /** Key name */
  name: string;
  /** Key type */
  type: KeyType;
  /** Permission scopes */
  scopes?: string[];
  /** Tenant ID */
  tenant_id?: string;
  /** Project ID */
  project_id?: string;
  /** User ID */
  user_id?: string;
  /** Expiration time */
  expires_at?: string;
  /** Additional metadata */
  metadata?: Record<string, any>;
}

export interface UpdateKeyRequest {
  /** Key name */
  name?: string;
  /** Key status */
  status?: KeyStatus;
  /** Permission scopes */
  scopes?: string[];
  /** Expiration time */
  expires_at?: string;
  /** Additional metadata */
  metadata?: Record<string, any>;
}

export interface KeyFilter {
  /** Tenant ID */
  tenant_id?: string;
  /** Project ID */
  project_id?: string;
  /** Key type */
  type?: KeyType;
  /** Key status */
  status?: KeyStatus;
  /** User ID */
  user_id?: string;
  /** Pagination limit */
  limit?: number;
  /** Pagination offset */
  offset?: number;
}

export interface KeyUsageStats {
  /** Key ID */
  key_id: string;
  /** Usage count */
  usage_count: number;
  /** Last used time */
  last_used_at?: string;
  /** Top endpoints */
  top_endpoints: EndpointUsage[];
}

export interface EndpointUsage {
  /** Endpoint path */
  endpoint: string;
  /** Call count */
  count: number;
}