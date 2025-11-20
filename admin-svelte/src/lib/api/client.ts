// API Client for MetaBase Admin Interface
import { writable, readable, derived } from 'svelte/store';

// Types
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
    details?: string;
  };
  meta?: {
    timestamp: number;
    request_id: string;
    version: string;
  };
}

export interface WebSocketMessage {
  type: string;
  channel?: string;
  data?: any;
  timestamp: number;
  id?: string;
}

export interface ApiConfig {
  baseUrl: string;
  wsUrl?: string;
  apiKey?: string;
  timeout?: number;
  retryAttempts?: number;
  retryDelay?: number;
}

export interface SystemMetrics {
  timestamp: number;
  uptime: number;
  requests_per_second: number;
  active_connections: number;
  memory_usage: {
    total: number;
    used: number;
    percentage: number;
  };
  cpu_usage: {
    percentage: number;
    cores: number;
  };
  disk_usage: {
    total: number;
    used: number;
    percentage: number;
  };
}

export interface SearchStatus {
  indexing: boolean;
  total_documents: number;
  indexed_documents: number;
  pending_documents: number;
  index_size: number;
  search_performance: {
    avg_query_time: number;
    queries_per_second: number;
    cache_hit_rate: number;
  };
}

export interface StorageStats {
  databases: number;
  tables: number;
  total_records: number;
  storage_size: number;
  cache_size: number;
  query_performance: {
    avg_query_time: number;
    slow_queries: number;
    cache_hit_rate: number;
  };
}

// Error handling
export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
    public status?: number,
    public details?: any
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

// Main API Client
export class MetaBaseClient {
  private config: Required<ApiConfig>;
  private wsConnection: WebSocket | null = null;
  private wsReconnectAttempts = 0;
  private wsMaxReconnectAttempts = 5;
  private wsReconnectDelay = 1000;
  private wsSubscriptions = new Map<string, Set<(message: WebSocketMessage) => void>>();
  private wsMessageQueue: WebSocketMessage[] = [];

  constructor(config: ApiConfig) {
    this.config = {
      baseUrl: config.baseUrl,
      wsUrl: config.wsUrl || config.baseUrl.replace('http', 'ws'),
      apiKey: config.apiKey || '',
      timeout: config.timeout || 30000,
      retryAttempts: config.retryAttempts || 3,
      retryDelay: config.retryDelay || 1000,
    };
  }

  // HTTP Methods
  async request<T = any>(
    method: string,
    endpoint: string,
    data?: any,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    const url = `${this.config.baseUrl}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...options.headers,
    };

    if (this.config.apiKey) {
      headers['Authorization'] = `Bearer ${this.config.apiKey}`;
    }

    const requestOptions: RequestInit = {
      method,
      headers,
      ...options,
    };

    if (data && method !== 'GET') {
      requestOptions.body = JSON.stringify(data);
    }

    let lastError: Error | null = null;

    for (let attempt = 0; attempt <= this.config.retryAttempts; attempt++) {
      if (attempt > 0) {
        await this.delay(this.config.retryDelay * attempt);
      }

      try {
        const response = await fetch(url, requestOptions);

        if (!response.ok) {
          throw new ApiError(
            `HTTP_${response.status}`,
            `HTTP ${response.status}: ${response.statusText}`,
            response.status
          );
        }

        const result: ApiResponse<T> = await response.json();

        if (!result.success && result.error) {
          throw new ApiError(result.error.code, result.error.message, undefined, result.error.details);
        }

        return result;
      } catch (error) {
        lastError = error as Error;

        if (error instanceof ApiError && error.status && error.status < 500) {
          // Don't retry client errors
          break;
        }

        console.warn(`API request attempt ${attempt + 1} failed:`, error);
      }
    }

    throw lastError || new Error('Request failed');
  }

  async get<T = any>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>('GET', endpoint);
  }

  async post<T = any>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>('POST', endpoint, data);
  }

  async put<T = any>(endpoint: string, data?: any): Promise<ApiResponse<T>> {
    return this.request<T>('PUT', endpoint, data);
  }

  async delete<T = any>(endpoint: string): Promise<ApiResponse<T>> {
    return this.request<T>('DELETE', endpoint);
  }

  // WebSocket Methods
  connectWebSocket(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.wsConnection?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      try {
        this.wsConnection = new WebSocket(this.config.wsUrl);

        this.wsConnection.onopen = () => {
          console.log('WebSocket connected');
          this.wsReconnectAttempts = 0;

          // Send queued messages
          this.wsMessageQueue.forEach(message => {
            this.sendWebSocketMessage(message);
          });
          this.wsMessageQueue = [];

          resolve();
        };

        this.wsConnection.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
          } catch (error) {
            console.error('Failed to parse WebSocket message:', error);
          }
        };

        this.wsConnection.onclose = () => {
          console.log('WebSocket disconnected');
          this.handleWebSocketDisconnect();
        };

        this.wsConnection.onerror = (error) => {
          console.error('WebSocket error:', error);
          reject(error);
        };
      } catch (error) {
        reject(error);
      }
    });
  }

  disconnectWebSocket() {
    if (this.wsConnection) {
      this.wsConnection.close();
      this.wsConnection = null;
    }
    this.wsSubscriptions.clear();
  }

  subscribe(channel: string, callback: (message: WebSocketMessage) => void) {
    if (!this.wsSubscriptions.has(channel)) {
      this.wsSubscriptions.set(channel, new Set());
    }
    this.wsSubscriptions.get(channel)!.add(callback);

    // Send subscription message
    this.sendWebSocketMessage({
      type: 'subscribe',
      channel,
      timestamp: Date.now(),
    });

    return () => {
      const callbacks = this.wsSubscriptions.get(channel);
      if (callbacks) {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          this.wsSubscriptions.delete(channel);
          // Send unsubscribe message
          this.sendWebSocketMessage({
            type: 'unsubscribe',
            channel,
            timestamp: Date.now(),
          });
        }
      }
    };
  }

  publish(channel: string, data: any) {
    this.sendWebSocketMessage({
      type: 'publish',
      channel,
      data,
      timestamp: Date.now(),
    });
  }

  private sendWebSocketMessage(message: WebSocketMessage) {
    if (this.wsConnection?.readyState === WebSocket.OPEN) {
      this.wsConnection.send(JSON.stringify(message));
    } else {
      // Queue message for when connection is established
      this.wsMessageQueue.push(message);
    }
  }

  private handleWebSocketMessage(message: WebSocketMessage) {
    const callbacks = this.wsSubscriptions.get(message.channel || '');
    if (callbacks) {
      callbacks.forEach(callback => {
        try {
          callback(message);
        } catch (error) {
          console.error('Error in WebSocket callback:', error);
        }
      });
    }
  }

  private handleWebSocketDisconnect() {
    this.wsConnection = null;

    // Attempt to reconnect
    if (this.wsReconnectAttempts < this.wsMaxReconnectAttempts) {
      this.wsReconnectAttempts++;
      const delay = this.wsReconnectDelay * Math.pow(2, this.wsReconnectAttempts - 1);

      setTimeout(() => {
        console.log(`Attempting WebSocket reconnection (${this.wsReconnectAttempts}/${this.wsMaxReconnectAttempts})`);
        this.connectWebSocket().catch(error => {
          console.error('WebSocket reconnection failed:', error);
        });
      }, delay);
    } else {
      console.error('Max WebSocket reconnection attempts reached');
    }
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Specific API Methods
  async getHealth() {
    return this.get('/api/health');
  }

  async getMetrics(): Promise<ApiResponse<SystemMetrics>> {
    return this.get('/api/metrics');
  }

  async getSearchStatus(): Promise<ApiResponse<SearchStatus>> {
    return this.get('/api/search/status');
  }

  async getStorageStats(): Promise<ApiResponse<StorageStats>> {
    return this.get('/api/storage/stats');
  }

  async getIndexStats() {
    return this.get('/api/search/index/stats');
  }

  async rebuildIndex() {
    return this.post('/api/search/index/rebuild');
  }

  async optimizeIndex() {
    return this.post('/api/search/index/optimize');
  }

  async getDatabaseInfo() {
    return this.get('/api/admin/database/info');
  }

  async getTenantInfo() {
    return this.get('/api/admin/tenants');
  }

  async createTenant(data: any) {
    return this.post('/api/admin/tenants', data);
  }

  async getUserInfo(userId: string) {
    return this.get(`/api/admin/users/${userId}`);
  }

  async getAdminStatus() {
    return this.get('/api/admin/status');
  }

  async getAdminMetrics() {
    return this.get('/api/admin/metrics');
  }
}

// Create singleton client
export const apiClient = new MetaBaseClient({
  baseUrl: 'http://localhost:7609',
  wsUrl: 'ws://localhost:7609/ws/realtime',
});

// Svelte stores for reactive state management
export const connectionStatus = writable<'disconnected' | 'connecting' | 'connected'>('disconnected');
export const connectionError = writable<string | null>(null);

// WebSocket connection management
export function initWebSocket() {
  connectionStatus.set('connecting');

  apiClient.connectWebSocket()
    .then(() => {
      connectionStatus.set('connected');
      connectionError.set(null);
    })
    .catch(error => {
      connectionStatus.set('disconnected');
      connectionError.set(error.message);
    });
}

// Reactive stores for data
export const systemMetrics = writable<SystemMetrics | null>(null);
export const searchStatus = writable<SearchStatus | null>(null);
export const storageStats = writable<StorageStats | null>(null);

// Subscribe to real-time updates
export const metricsSubscription = apiClient.subscribe('system.metrics', (message) => {
  if (message.data) {
    systemMetrics.set(message.data);
  }
});

export const searchSubscription = apiClient.subscribe('search.updates', (message) => {
  if (message.data) {
    searchStatus.set(message.data);
  }
});

export const storageSubscription = apiClient.subscribe('storage.updates', (message) => {
  if (message.data) {
    storageStats.set(message.data);
  }
});

// Derived stores
export const isConnected = readable(false, (set) => {
  const unsubscribe = connectionStatus.subscribe(status => {
    set(status === 'connected');
  });

  return unsubscribe;
});

export const healthStatus = derived(
  [systemMetrics, searchStatus, storageStats],
  ([$metrics, $search, $storage]) => {
    const healthy = [
      $metrics?.cpu_usage.percentage ?? 0 < 90,
      $metrics?.memory_usage.percentage ?? 0 < 90,
      $search?.indexing !== true,
      $storage?.query_performance?.avg_query_time ?? 0 < 1000,
    ].every(Boolean);

    return healthy ? 'healthy' : 'degraded';
  }
);

// Utility functions
export async function fetchInitialData() {
  try {
    const [metrics, search, storage] = await Promise.all([
      apiClient.getMetrics(),
      apiClient.getSearchStatus(),
      apiClient.getStorageStats(),
    ]);

    if (metrics.data) systemMetrics.set(metrics.data);
    if (search.data) searchStatus.set(search.data);
    if (storage.data) storageStats.set(storage.data);
  } catch (error) {
    console.error('Failed to fetch initial data:', error);
    throw error;
  }
}

export function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${size.toFixed(1)} ${units[unitIndex]}`;
}

export function formatDuration(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  } else if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  } else {
    return `${secs}s`;
  }
}