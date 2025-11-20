import { writable, derived, readable } from 'svelte/store';
import { apiClient, SystemMetrics, SearchStatus, StorageStats } from '$lib/api/client';

// Types
export interface SystemAlert {
  id: string;
  type: 'error' | 'warning' | 'info';
  title: string;
  message: string;
  timestamp: number;
  acknowledged: boolean;
  source: string;
  metadata?: Record<string, any>;
}

export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message: string;
  timestamp: number;
  autoHide?: boolean;
  duration?: number;
  actions?: Array<{
    label: string;
    action: () => void;
  }>;
}

export interface ConnectionState {
  status: 'disconnected' | 'connecting' | 'connected' | 'reconnecting' | 'error';
  lastConnected?: number;
  lastDisconnected?: number;
  reconnectAttempts: number;
  error?: string;
}

// System state store
function createSystemStore() {
  const { subscribe, set, update } = writable({
    alerts: [] as SystemAlert[],
    notifications: [] as Notification[],
    connectionStatus: {
      status: 'disconnected' as const,
      reconnectAttempts: 0,
    } as ConnectionState,
    sidebarCollapsed: false,
    theme: 'light' as 'light' | 'dark' | 'auto',
  });

  return {
    subscribe,

    // Alerts management
    addAlert: (alert: Omit<SystemAlert, 'id' | 'timestamp'>) => {
      const id = `alert_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const newAlert: SystemAlert = {
        ...alert,
        id,
        timestamp: Date.now(),
        acknowledged: false,
      };

      update(state => ({
        ...state,
        alerts: [...state.alerts, newAlert],
      }));
    },

    acknowledgeAlert: (alertId: string) => {
      update(state => ({
        ...state,
        alerts: state.alerts.map(alert =>
          alert.id === alertId ? { ...alert, acknowledged: true } : alert
        ),
      }));
    },

    removeAlert: (alertId: string) => {
      update(state => ({
        ...state,
        alerts: state.alerts.filter(alert => alert.id !== alertId),
      }));
    },

    clearAlerts: () => {
      update(state => ({ ...state, alerts: [] }));
    },

    // Notifications management
    addNotification: (notification: Omit<Notification, 'id' | 'timestamp'>) => {
      const id = `notif_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      const newNotification: Notification = {
        ...notification,
        id,
        timestamp: Date.now(),
        autoHide: notification.autoHide ?? true,
        duration: notification.duration ?? 5000,
      };

      update(state => ({
        ...state,
        notifications: [...state.notifications, newNotification],
      }));

      // Auto-hide notification if specified
      if (newNotification.autoHide && newNotification.duration) {
        setTimeout(() => {
          update(state => ({
            ...state,
            notifications: state.notifications.filter(n => n.id !== id),
          }));
        }, newNotification.duration);
      }

      return id;
    },

    removeNotification: (notificationId: string) => {
      update(state => ({
        ...state,
        notifications: state.notifications.filter(n => n.id !== notificationId),
      }));
    },

    clearNotifications: () => {
      update(state => ({ ...state, notifications: [] }));
    },

    // Connection status management
    setConnectionStatus: (status: ConnectionState['status'], error?: string) => {
      update(state => {
        const now = Date.now();
        const currentStatus = state.connectionStatus.status;
        let lastConnected = state.connectionStatus.lastConnected;
        let lastDisconnected = state.connectionStatus.lastDisconnected;
        let reconnectAttempts = state.connectionStatus.reconnectAttempts;

        if (status === 'connected') {
          lastConnected = now;
          reconnectAttempts = 0;
        } else if (currentStatus === 'connected' && status !== 'connected') {
          lastDisconnected = now;
        }

        if (status === 'connecting' || status === 'reconnecting') {
          reconnectAttempts++;
        }

        return {
          ...state,
          connectionStatus: {
            status,
            lastConnected,
            lastDisconnected,
            reconnectAttempts,
            error,
          },
        };
      });
    },

    // UI preferences
    toggleSidebar: () => {
      update(state => ({ ...state, sidebarCollapsed: !state.sidebarCollapsed }));
    },

    setTheme: (theme: 'light' | 'dark' | 'auto') => {
      update(state => ({ ...state, theme }));
    },
  };
}

export const systemStore = createSystemStore();

// Reactive stores
export const alerts = derived(systemStore, $system => $system.alerts);
export const notifications = derived(systemStore, $system => $system.notifications);
export const connectionStatus = derived(systemStore, $system => $system.connectionStatus);
export const sidebarCollapsed = derived(systemStore, $system => $system.sidebarCollapsed);
export const theme = derived(systemStore, $system => $system.theme);

// Unacknowledged alerts count
export const unacknowledgedAlertsCount = derived(
  alerts,
  $alerts => $alerts.filter(alert => !alert.acknowledged).length
);

// Critical alerts
export const criticalAlerts = derived(
  alerts,
  $alerts => $alerts.filter(alert => alert.type === 'error' && !alert.acknowledged)
);

// Connection state helpers
export const isConnected = derived(
  connectionStatus,
  $connection => $connection.status === 'connected'
);

export const isConnecting = derived(
  connectionStatus,
  $connection => ['connecting', 'reconnecting'].includes($connection.status)
);

export const hasConnectionError = derived(
  connectionStatus,
  $connection => $connection.status === 'error'
);

// Real-time data stores
export const systemMetrics = writable<SystemMetrics | null>(null);
export const searchStatus = writable<SearchStatus | null>(null);
export const storageStats = writable<StorageStats | null>(null);

// Loading states
export const metricsLoading = writable(false);
export const searchLoading = writable(false);
export const storageLoading = writable(false);

// Error states
export const metricsError = writable<string | null>(null);
export const searchError = writable<string | null>(null);
export const storageError = writable<string | null>(null);

// Derived health status
export const overallHealth = derived(
  [systemMetrics, searchStatus, storageStats],
  ([$metrics, $search, $storage]) => {
    if (!$metrics || !$search || !$storage) {
      return 'unknown';
    }

    const checks = [
      $metrics.cpu_usage.percentage < 90,
      $metrics.memory_usage.percentage < 90,
      $metrics.disk_usage.percentage < 90,
      !$search.indexing,
      $search.search_performance.avg_query_time < 500,
      $storage.query_performance.avg_query_time < 1000,
    ];

    const passedChecks = checks.filter(Boolean).length;
    const totalChecks = checks.length;

    if (passedChecks === totalChecks) {
      return 'healthy';
    } else if (passedChecks >= totalChecks * 0.7) {
      return 'degraded';
    } else {
      return 'unhealthy';
    }
  }
);

// Performance metrics
export const performanceMetrics = derived(
  [systemMetrics, searchStatus, storageStats],
  ([$metrics, $search, $storage]) => {
    return {
      cpu: $metrics?.cpu_usage.percentage ?? 0,
      memory: $metrics?.memory_usage.percentage ?? 0,
      disk: $metrics?.disk_usage.percentage ?? 0,
      queriesPerSecond: $metrics?.requests_per_second ?? 0,
      avgSearchTime: $search?.search_performance.avg_query_time ?? 0,
      avgQueryTime: $storage?.query_performance.avg_query_time ?? 0,
      cacheHitRate: $search?.search_performance.cache_hit_rate ?? 0,
    };
  }
);

// Utility functions
export async function refreshMetrics() {
  metricsLoading.set(true);
  metricsError.set(null);

  try {
    const response = await apiClient.getMetrics();
    if (response.data) {
      systemMetrics.set(response.data);
    }
  } catch (error: any) {
    metricsError.set(error.message);
    systemStore.addNotification({
      type: 'error',
      title: 'Failed to load metrics',
      message: error.message,
    });
  } finally {
    metricsLoading.set(false);
  }
}

export async function refreshSearchStatus() {
  searchLoading.set(true);
  searchError.set(null);

  try {
    const response = await apiClient.getSearchStatus();
    if (response.data) {
      searchStatus.set(response.data);
    }
  } catch (error: any) {
    searchError.set(error.message);
    systemStore.addNotification({
      type: 'error',
      title: 'Failed to load search status',
      message: error.message,
    });
  } finally {
    searchLoading.set(false);
  }
}

export async function refreshStorageStats() {
  storageLoading.set(true);
  storageError.set(null);

  try {
    const response = await apiClient.getStorageStats();
    if (response.data) {
      storageStats.set(response.data);
    }
  } catch (error: any) {
    storageError.set(error.message);
    systemStore.addNotification({
      type: 'error',
      title: 'Failed to load storage stats',
      message: error.message,
    });
  } finally {
    storageLoading.set(false);
  }
}

// Refresh all data
export async function refreshAllData() {
  await Promise.all([
    refreshMetrics(),
    refreshSearchStatus(),
    refreshStorageStats(),
  ]);
}

// Auto-refresh timer (every 30 seconds)
export const autoRefresh = readable(
  { enabled: false, interval: 30000 },
  (set, update) => {
    let intervalId: NodeJS.Timeout;

    function start() {
      stop();
      intervalId = setInterval(() => {
        refreshAllData();
      }, 30000);
      update({ enabled: true, interval: 30000 });
    }

    function stop() {
      if (intervalId) {
        clearInterval(intervalId);
        intervalId = null as any;
      }
      update({ enabled: false, interval: 30000 });
    }

    return {
      start,
      stop,
    };
  }
);

// Initialize WebSocket connection and real-time updates
export function initializeRealTimeUpdates() {
  systemStore.setConnectionStatus('connecting');

  apiClient.connectWebSocket()
    .then(() => {
      systemStore.setConnectionStatus('connected');
      systemStore.addNotification({
        type: 'success',
        title: 'Connected',
        message: 'Real-time updates enabled',
      });
    })
    .catch(error => {
      systemStore.setConnectionStatus('error', error.message);
      systemStore.addNotification({
        type: 'error',
        title: 'Connection Failed',
        message: 'Could not establish real-time connection',
      });
    });

  // Subscribe to real-time channels
  apiClient.subscribe('system.metrics', (message) => {
    if (message.data) {
      systemMetrics.set(message.data);
    }
  });

  apiClient.subscribe('search.updates', (message) => {
    if (message.data) {
      searchStatus.set(message.data);
      systemStore.addNotification({
        type: 'info',
        title: 'Search Update',
        message: 'Search index updated',
        autoHide: true,
        duration: 3000,
      });
    }
  });

  apiClient.subscribe('storage.updates', (message) => {
    if (message.data) {
      storageStats.set(message.data);
    }
  });

  apiClient.subscribe('system.alerts', (message) => {
    if (message.data) {
      systemStore.addAlert(message.data);
    }
  });

  // Handle connection events
  const reconnectSubscription = apiClient.subscribe('connection.reconnecting', () => {
    systemStore.setConnectionStatus('reconnecting');
  });

  const disconnectSubscription = apiClient.subscribe('connection.disconnected', () => {
    systemStore.setConnectionStatus('disconnected');
  });

  return () => {
    // Cleanup subscriptions
    reconnectSubscription?.();
    disconnectSubscription?.();
    apiClient.disconnectWebSocket();
  };
}