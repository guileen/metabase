import { MetaBaseClient } from './client';
import { APIResponse } from './types';

/**
 * Real-time subscription manager
 */
export interface RealtimeSubscription {
  /** Subscription ID */
  id: string;
  /** Table name */
  table: string;
  /** Event filter */
  filter?: Record<string, any>;
  /** Event callback */
  callback: (event: RealtimeEvent) => void;
  /** WebSocket connection */
  ws?: WebSocket;
  /** Active status */
  active: boolean;
}

export interface RealtimeEvent {
  /** Event type */
  type: 'INSERT' | 'UPDATE' | 'DELETE';
  /** Table name */
  table: string;
  /** Record ID */
  record?: any;
  /** Old record (for UPDATE/DELETE) */
  old?: any;
  /** New record (for INSERT/UPDATE) */
  new?: any;
  /** Event timestamp */
  timestamp: string;
  /** Additional metadata */
  metadata?: Record<string, any>;
}

/**
 * Real-time manager for WebSocket subscriptions
 */
export class RealtimeManager {
  private subscriptions: Map<string, RealtimeSubscription> = new Map();
  private reconnectAttempts: Map<string, number> = new Map();
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second

  constructor(private client: MetaBaseClient) {}

  /**
   * Subscribe to real-time events for a table
   */
  async subscribe(
    table: string,
    callback: (event: RealtimeEvent) => void,
    filter?: Record<string, any>
  ): Promise<RealtimeSubscription> {
    const subscriptionId = this.generateSubscriptionId();

    const subscription: RealtimeSubscription = {
      id: subscriptionId,
      table,
      callback,
      filter,
      active: false,
    };

    this.subscriptions.set(subscriptionId, subscription);

    // Start WebSocket connection
    await this.connectWebSocket(subscription);

    return subscription;
  }

  /**
   * Unsubscribe from real-time events
   */
  unsubscribe(subscriptionId: string): void {
    const subscription = this.subscriptions.get(subscriptionId);
    if (subscription) {
      subscription.active = false;
      if (subscription.ws) {
        subscription.ws.close();
      }
      this.subscriptions.delete(subscriptionId);
      this.reconnectAttempts.delete(subscriptionId);
    }
  }

  /**
   * Unsubscribe from all real-time events
   */
  unsubscribeAll(): void {
    for (const [id, subscription] of this.subscriptions) {
      subscription.active = false;
      if (subscription.ws) {
        subscription.ws.close();
      }
    }
    this.subscriptions.clear();
    this.reconnectAttempts.clear();
  }

  /**
   * Get active subscriptions
   */
  getActiveSubscriptions(): RealtimeSubscription[] {
    return Array.from(this.subscriptions.values()).filter(sub => sub.active);
  }

  /**
   * Create WebSocket connection
   */
  private async connectWebSocket(subscription: RealtimeSubscription): Promise<void> {
    const config = this.client.getConfig();
    const wsUrl = config.url.replace('http://', 'ws://').replace('https://', 'wss://');

    const params = new URLSearchParams();
    params.append('table', subscription.table);
    params.append('apiKey', config.apiKey);

    if (subscription.filter) {
      params.append('filter', JSON.stringify(subscription.filter));
    }

    const wsUrlWithParams = `${wsUrl}/ws/v1/realtime?${params.toString()}`;

    try {
      subscription.ws = new WebSocket(wsUrlWithParams);

      subscription.ws.onopen = () => {
        console.log(`Realtime subscription connected: ${subscription.id} (${subscription.table})`);
        subscription.active = true;
        this.reconnectAttempts.set(subscription.id, 0);
      };

      subscription.ws.onmessage = (event) => {
        try {
          const realtimeEvent: RealtimeEvent = JSON.parse(event.data);
          subscription.callback(realtimeEvent);
        } catch (error) {
          console.error(`Error parsing realtime event:`, error);
        }
      };

      subscription.ws.onclose = () => {
        console.log(`Realtime subscription closed: ${subscription.id} (${subscription.table})`);
        subscription.active = false;

        // Attempt to reconnect
        this.attemptReconnect(subscription);
      };

      subscription.ws.onerror = (error) => {
        console.error(`WebSocket error for subscription ${subscription.id}:`, error);
      };

    } catch (error) {
      console.error(`Failed to create WebSocket connection:`, error);
      this.attemptReconnect(subscription);
    }
  }

  /**
   * Attempt to reconnect a subscription
   */
  private attemptReconnect(subscription: RealtimeSubscription): void {
    const attempts = this.reconnectAttempts.get(subscription.id) || 0;

    if (attempts >= this.maxReconnectAttempts) {
      console.error(`Max reconnect attempts reached for subscription ${subscription.id}`);
      this.subscriptions.delete(subscription.id);
      this.reconnectAttempts.delete(subscription.id);
      return;
    }

    const delay = this.reconnectDelay * Math.pow(2, attempts); // Exponential backoff
    this.reconnectAttempts.set(subscription.id, attempts + 1);

    console.log(`Attempting to reconnect subscription ${subscription.id} in ${delay}ms (attempt ${attempts + 1}/${this.maxReconnectAttempts})`);

    setTimeout(() => {
      if (this.subscriptions.has(subscription.id)) {
        this.connectWebSocket(subscription);
      }
    }, delay);
  }

  /**
   * Generate unique subscription ID
   */
  private generateSubscriptionId(): string {
    return `sub_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Send message through WebSocket (for bi-directional communication)
   */
  sendMessage(subscriptionId: string, message: any): boolean {
    const subscription = this.subscriptions.get(subscriptionId);
    if (subscription && subscription.ws && subscription.active) {
      subscription.ws.send(JSON.stringify(message));
      return true;
    }
    return false;
  }

  /**
   * Get connection status for a subscription
   */
  getConnectionStatus(subscriptionId: string): 'connected' | 'disconnected' | 'reconnecting' | 'not_found' {
    const subscription = this.subscriptions.get(subscriptionId);
    if (!subscription) {
      return 'not_found';
    }

    if (subscription.active) {
      return 'connected';
    }

    const attempts = this.reconnectAttempts.get(subscription.id) || 0;
    if (attempts > 0 && attempts < this.maxReconnectAttempts) {
      return 'reconnecting';
    }

    return 'disconnected';
  }
}