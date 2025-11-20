import { MetaBaseClient } from './client';
import {
  APIKey,
  CreateKeyRequest,
  UpdateKeyRequest,
  KeyFilter,
  KeyUsageStats,
  APIResponse
} from './types';

/**
 * Authentication and API Key Management
 */
export class AuthManager {
  constructor(private client: MetaBaseClient) {}

  /**
   * Create a new API key
   */
  async createKey(request: CreateKeyRequest): Promise<APIResponse<{ key: APIKey; api_key: string }>> {
    return this.client.request({
      method: 'POST',
      url: '/admin/v1/keys',
      data: request,
    });
  }

  /**
   * Get API key by ID
   */
  async getKey(id: string): Promise<APIResponse<APIKey>> {
    return this.client.request({
      method: 'GET',
      url: `/admin/v1/keys/${id}`,
    });
  }

  /**
   * List API keys with filtering
   */
  async listKeys(filter?: KeyFilter): Promise<APIResponse<{ keys: APIKey[]; total: number }>> {
    const params = new URLSearchParams();

    if (filter?.tenant_id) {
      params.append('tenant_id', filter.tenant_id);
    }
    if (filter?.project_id) {
      params.append('project_id', filter.project_id);
    }
    if (filter?.type) {
      params.append('type', filter.type);
    }
    if (filter?.status) {
      params.append('status', filter.status);
    }
    if (filter?.user_id) {
      params.append('user_id', filter.user_id);
    }
    if (filter?.limit) {
      params.append('limit', String(filter.limit));
    }
    if (filter?.offset) {
      params.append('offset', String(filter.offset));
    }

    const url = params.toString()
      ? `/admin/v1/keys?${params.toString()}`
      : '/admin/v1/keys';

    return this.client.request({
      method: 'GET',
      url,
    });
  }

  /**
   * Update an API key
   */
  async updateKey(id: string, request: UpdateKeyRequest): Promise<APIResponse<APIKey>> {
    return this.client.request({
      method: 'PUT',
      url: `/admin/v1/keys/${id}`,
      data: request,
    });
  }

  /**
   * Revoke an API key
   */
  async revokeKey(id: string): Promise<APIResponse<void>> {
    return this.client.request({
      method: 'POST',
      url: `/admin/v1/keys/${id}/revoke`,
    });
  }

  /**
   * Delete an API key
   */
  async deleteKey(id: string): Promise<APIResponse<void>> {
    return this.client.request({
      method: 'DELETE',
      url: `/admin/v1/keys/${id}`,
    });
  }

  /**
   * Get API key usage statistics
   */
  async getKeyStats(id: string): Promise<APIResponse<KeyUsageStats>> {
    return this.client.request({
      method: 'GET',
      url: `/admin/v1/keys/${id}/stats`,
    });
  }

  /**
   * Validate API key format
   */
  static validateKeyFormat(key: string): boolean {
    // New format: metabase_sys_xxxx_base64, metabase_usr_xxxx_base64, etc.
    const newFormatRegex = /^metabase_(sys|usr|svc|key)_[a-z0-9]{4}_[A-Za-z0-9_-]+$/;

    // Legacy format: prefix_base64string
    const legacyFormatRegex = /^(sys|usr|svc|key)_[A-Za-z0-9_-]+$/;

    return newFormatRegex.test(key) || legacyFormatRegex.test(key);
  }

  /**
   * Extract key type from API key
   */
  static extractKeyType(key: string): 'system' | 'user' | 'service' | 'unknown' {
    if (key.startsWith('metabase_sys_') || key.startsWith('sys_')) {
      return 'system';
    }
    if (key.startsWith('metabase_usr_') || key.startsWith('usr_')) {
      return 'user';
    }
    if (key.startsWith('metabase_svc_') || key.startsWith('svc_')) {
      return 'service';
    }
    return 'unknown';
  }

  /**
   * Check if key has required scope
   */
  static keyHasScope(key: APIKey, requiredScope: string): boolean {
    return key.scopes.includes(requiredScope);
  }

  /**
   * Check if key is expired
   */
  static isKeyExpired(key: APIKey): boolean {
    if (!key.expires_at) {
      return false; // No expiration set
    }
    return new Date(key.expires_at) < new Date();
  }

  /**
   * Get default scopes for key type
   */
  static getDefaultScopes(type: 'system' | 'user' | 'service'): string[] {
    switch (type) {
      case 'system':
        return [
          'read',
          'write',
          'delete',
          'table:create',
          'table:read',
          'table:update',
          'table:delete',
          'user:read',
          'user:write',
          'system:read',
          'system:write',
          'file:read',
          'file:write',
          'file:delete',
          'analytics:read',
          'realtime'
        ];
      case 'user':
        return [
          'read',
          'write',
          'table:read',
          'table:update',
          'file:read',
          'file:write'
        ];
      case 'service':
        return [
          'read',
          'write',
          'table:read',
          'table:update',
          'analytics:read'
        ];
      default:
        return ['read'];
    }
  }
}