import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import {
  MetaBaseConfig,
  APIResponse,
  APIError,
  HealthResponse,
  QueryOptions,
  InsertOptions,
  UpdateOptions
} from './types';

/**
 * MetaBase Client - Main client class for interacting with MetaBase API
 */
export class MetaBaseClient {
  private axios: AxiosInstance;
  private config: MetaBaseConfig;

  constructor(config: MetaBaseConfig) {
    this.config = {
      timeout: 30000,
      headers: {},
      debug: false,
      ...config,
    };

    // Validate required config
    if (!this.config.url) {
      throw new Error('URL is required in MetaBaseConfig');
    }
    if (!this.config.apiKey) {
      throw new Error('API key is required in MetaBaseConfig');
    }

    // Create axios instance
    this.axios = axios.create({
      baseURL: this.config.url,
      timeout: this.config.timeout,
      headers: {
        'Authorization': `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json',
        'User-Agent': '@metabase/client/1.0.0',
        ...this.config.headers,
      },
    });

    // Request interceptor for debugging
    if (this.config.debug) {
      this.axios.interceptors.request.use((request) => {
        console.log('MetaBase Request:', {
          method: request.method,
          url: request.url,
          headers: request.headers,
          data: request.data,
        });
        return request;
      });

      this.axios.interceptors.response.use((response) => {
        console.log('MetaBase Response:', {
          status: response.status,
          data: response.data,
          headers: response.headers,
        });
        return response;
      });
    }
  }

  /**
   * Update configuration
   */
  updateConfig(newConfig: Partial<MetaBaseConfig>): void {
    this.config = { ...this.config, ...newConfig };

    if (newConfig.url) {
      this.axios.defaults.baseURL = newConfig.url;
    }
    if (newConfig.apiKey) {
      this.axios.defaults.headers['Authorization'] = `Bearer ${newConfig.apiKey}`;
    }
    if (newConfig.timeout) {
      this.axios.defaults.timeout = newConfig.timeout;
    }
    if (newConfig.headers) {
      this.axios.defaults.headers = {
        ...this.axios.defaults.headers,
        ...newConfig.headers,
      };
    }
  }

  /**
   * Get current configuration
   */
  getConfig(): MetaBaseConfig {
    return { ...this.config };
  }

  /**
   * Handle API errors
   */
  private handleError(error: any): APIError {
    if (error.response) {
      // Server responded with error status
      const response = error.response;
      return {
        code: response.data?.error?.code || 'http_error',
        message: response.data?.error?.message || response.statusText || 'HTTP Error',
        details: response.data?.error?.details,
        timestamp: response.data?.error?.timestamp || new Date().toISOString(),
      };
    } else if (error.request) {
      // Request was made but no response received
      return {
        code: 'network_error',
        message: 'Network error - no response received',
        details: error.message,
        timestamp: new Date().toISOString(),
      };
    } else {
      // Other error
      return {
        code: 'client_error',
        message: error.message || 'Unknown client error',
        timestamp: new Date().toISOString(),
      };
    }
  }

  /**
   * Make HTTP request with error handling
   */
  private async request<T = any>(config: AxiosRequestConfig): Promise<APIResponse<T>> {
    try {
      const response: AxiosResponse<APIResponse<T>> = await this.axios.request(config);
      return response.data;
    } catch (error: any) {
      const apiError = this.handleError(error);
      return {
        error: apiError,
      };
    }
  }

  /**
   * Health check
   */
  async health(): Promise<APIResponse<HealthResponse>> {
    return this.request<HealthResponse>({
      method: 'GET',
      url: '/rest/health',
    });
  }

  /**
   * Simple ping
   */
  async ping(): Promise<APIResponse<string>> {
    return this.request<string>({
      method: 'GET',
      url: '/ping',
    });
  }

  /**
   * Query data from a table
   */
  async query<T = any>(
    table: string,
    options?: QueryOptions
  ): Promise<APIResponse<T[]>> {
    const params = new URLSearchParams();

    if (options?.select) {
      params.append('select', options.select);
    }

    if (options?.where) {
      Object.entries(options.where).forEach(([key, value]) => {
        if (typeof value === 'object') {
          params.append(key, JSON.stringify(value));
        } else {
          params.append(key, String(value));
        }
      });
    }

    if (options?.order) {
      params.append('order', options.order);
    }

    if (options?.limit) {
      params.append('limit', String(options.limit));
    }

    if (options?.offset) {
      params.append('offset', String(options.offset));
    }

    const url = params.toString()
      ? `/rest/v1/${table}?${params.toString()}`
      : `/rest/v1/${table}`;

    return this.request<T[]>({
      method: 'GET',
      url,
    });
  }

  /**
   * Get a single record by ID
   */
  async get<T = any>(
    table: string,
    id: string | number,
    options?: { select?: string }
  ): Promise<APIResponse<T>> {
    const params = new URLSearchParams();

    if (options?.select) {
      params.append('select', options.select);
    }

    const url = params.toString()
      ? `/rest/v1/${table}/${id}?${params.toString()}`
      : `/rest/v1/${table}/${id}`;

    return this.request<T>({
      method: 'GET',
      url,
    });
  }

  /**
   * Insert data into a table
   */
  async insert<T = any>(
    table: string,
    data: any | any[],
    options?: InsertOptions
  ): Promise<APIResponse<T | T[]>> {
    const params = new URLSearchParams();

    if (options?.returning) {
      params.append('returning', options.returning.join(','));
    }

    const url = params.toString()
      ? `/rest/v1/${table}?${params.toString()}`
      : `/rest/v1/${table}`;

    return this.request<T | T[]>({
      method: 'POST',
      url,
      data,
    });
  }

  /**
   * Update data in a table (batch update)
   */
  async update<T = any>(
    table: string,
    data: any,
    options?: UpdateOptions & QueryOptions
  ): Promise<APIResponse<T[]>> {
    const params = new URLSearchParams();

    if (options?.where) {
      Object.entries(options.where).forEach(([key, value]) => {
        if (typeof value === 'object') {
          params.append(key, JSON.stringify(value));
        } else {
          params.append(key, String(value));
        }
      });
    }

    if (options?.returning) {
      params.append('returning', options.returning.join(','));
    }

    const url = params.toString()
      ? `/rest/v1/${table}?${params.toString()}`
      : `/rest/v1/${table}`;

    return this.request<T[]>({
      method: 'PATCH',
      url,
      data,
    });
  }

  /**
   * Update a single record by ID
   */
  async updateOne<T = any>(
    table: string,
    id: string | number,
    data: any,
    options?: UpdateOptions
  ): Promise<APIResponse<T>> {
    const params = new URLSearchParams();

    if (options?.returning) {
      params.append('returning', options.returning.join(','));
    }

    const url = params.toString()
      ? `/rest/v1/${table}/${id}?${params.toString()}`
      : `/rest/v1/${table}/${id}`;

    return this.request<T>({
      method: 'PATCH',
      url,
      data,
    });
  }

  /**
   * Delete data from a table (batch delete)
   */
  async delete<T = any>(
    table: string,
    options?: QueryOptions & { returning?: string[] }
  ): Promise<APIResponse<T[]>> {
    const params = new URLSearchParams();

    if (options?.where) {
      Object.entries(options.where).forEach(([key, value]) => {
        if (typeof value === 'object') {
          params.append(key, JSON.stringify(value));
        } else {
          params.append(key, String(value));
        }
      });
    }

    if (options?.returning) {
      params.append('returning', options.returning.join(','));
    }

    const url = params.toString()
      ? `/rest/v1/${table}?${params.toString()}`
      : `/rest/v1/${table}`;

    return this.request<T[]>({
      method: 'DELETE',
      url,
    });
  }

  /**
   * Delete a single record by ID
   */
  async deleteOne<T = any>(
    table: string,
    id: string | number,
    options?: { returning?: string[] }
  ): Promise<APIResponse<T>> {
    const params = new URLSearchParams();

    if (options?.returning) {
      params.append('returning', options.returning.join(','));
    }

    const url = params.toString()
      ? `/rest/v1/${table}/${id}?${params.toString()}`
      : `/rest/v1/${table}/${id}`;

    return this.request<T>({
      method: 'DELETE',
      url,
    });
  }

  /**
   * List tables
   */
  async listTables(): Promise<APIResponse<string[]>> {
    return this.request<string[]>({
      method: 'GET',
      url: '/rest/v1',
    });
  }

  /**
   * Get table schema
   */
  async getTableSchema(table: string): Promise<APIResponse<any>> {
    return this.request({
      method: 'GET',
      url: `/rest/v1/${table}/schema`,
    });
  }
}