// Main client
export { MetaBaseClient } from './client';

// Specialized managers
export { AuthManager } from './auth';
export { RealtimeManager } from './realtime';
export { FileManager } from './files';

// Types and interfaces
export type {
  // Core types
  MetaBaseConfig,
  APIResponse,
  APIError,
  QueryOptions,
  JoinClause,
  InsertOptions,
  UpdateOptions,
  HealthResponse,
  DatabaseStatus,
  CacheStatus,

  // Authentication types
  APIKey,
  CreateKeyRequest,
  UpdateKeyRequest,
  KeyFilter,
  KeyUsageStats,
  EndpointUsage,
  KeyType,
  KeyStatus,

  // Realtime types
  RealtimeSubscription,
  RealtimeEvent,

  // File types
  FileUploadOptions,
  FileInfo,
  FileListOptions,
} from './types';

// Re-export managers with types
export type { AuthManager as IAuthManager } from './auth';
export type { RealtimeManager as IRealtimeManager } from './realtime';
export type { FileManager as IFileManager } from './files';

// Factory function for creating a complete client with managers
export function createMetaBaseClient(config: MetaBaseConfig) {
  const client = new MetaBaseClient(config);

  return {
    client,
    auth: new AuthManager(client),
    realtime: new RealtimeManager(client),
    files: new FileManager(client),
  };
}

// Default export
export default {
  MetaBaseClient,
  AuthManager,
  RealtimeManager,
  FileManager,
  createMetaBaseClient,
};