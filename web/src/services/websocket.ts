// WebSocket service for real-time communication with Cortex API
export interface WebSocketMessage {
  type: string;
  data?: any;
  id?: string;
  timestamp?: string;
  error?: string;
}

export interface WebSocketSubscription {
  type: string;
  callback: (data: any) => void;
}

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export class CortexWebSocketService {
  private ws: WebSocket | null = null;
  private subscriptions: Map<string, WebSocketSubscription[]> = new Map();
  private messageQueue: WebSocketMessage[] = [];
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 5000;
  private reconnectTimer: number | null = null;
  private connectionStatus: ConnectionStatus = 'disconnected';
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private baseURL: string;
  
  constructor(baseURL = 'ws://localhost:8086') {
    this.baseURL = baseURL;
  }

  // Connection management
  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.setConnectionStatus('connecting');
      
      const wsUrl = `${this.baseURL}/ws`;
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('🚀 Connected to Cortex WebSocket');
        this.setConnectionStatus('connected');
        this.reconnectAttempts = 0;
        
        // Send queued messages
        this.flushMessageQueue();
        
        // Auto-subscribe to common real-time updates
        this.subscribeToRealTimeUpdates();
        
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          this.handleMessage(message);
        } catch (error) {
          console.error('❌ Error parsing WebSocket message:', error);
        }
      };

      this.ws.onclose = () => {
        console.log('🔌 WebSocket connection closed');
        this.setConnectionStatus('disconnected');
        this.attemptReconnect();
      };

      this.ws.onerror = (error) => {
        console.error('❌ WebSocket error:', error);
        this.setConnectionStatus('error');
        reject(error);
      };
    });
  }

  disconnect(): void {
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    
    this.setConnectionStatus('disconnected');
  }

  private setConnectionStatus(status: ConnectionStatus): void {
    if (this.connectionStatus !== status) {
      this.connectionStatus = status;
      this.statusCallbacks.forEach(callback => callback(status));
    }
  }

  getConnectionStatus(): ConnectionStatus {
    return this.connectionStatus;
  }

  onConnectionStatusChange(callback: (status: ConnectionStatus) => void): () => void {
    this.statusCallbacks.push(callback);
    return () => {
      const index = this.statusCallbacks.indexOf(callback);
      if (index > -1) {
        this.statusCallbacks.splice(index, 1);
      }
    };
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.log('❌ Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    console.log(`🔄 Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
    
    this.reconnectTimer = window.setTimeout(() => {
      this.connect().catch(error => {
        console.error('❌ Reconnection failed:', error);
      });
    }, this.reconnectInterval);
  }

  private flushMessageQueue(): void {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      if (message) {
        this.sendMessage(message);
      }
    }
  }

  private subscribeToRealTimeUpdates(): void {
    // Subscribe to common real-time updates
    const subscriptions = [
      'metrics',
      'consciousness', 
      'ollama_status',
      'system_events',
      'query_results'
    ];

    subscriptions.forEach(type => {
      this.sendMessage({
        type: 'subscribe',
        data: { type }
      });
    });
  }

  // Message handling
  private handleMessage(message: WebSocketMessage): void {
    const { type, data } = message;
    
    // Handle different message types
    switch (type) {
      case 'status':
        console.log('📊 Connection status:', data);
        break;
      case 'response':
      case 'query_complete':
        this.notifySubscribers('query_response', message);
        break;
      case 'query_start':
        this.notifySubscribers('query_start', message);
        break;
      case 'query_chunk':
        this.notifySubscribers('query_chunk', message);
        break;
      case 'query_progress':
        this.notifySubscribers('query_progress', message);
        break;
      case 'metrics':
        this.notifySubscribers('metrics', data);
        break;
      case 'consciousness':
        this.notifySubscribers('consciousness', data);
        break;
      case 'ollama_status':
        this.notifySubscribers('ollama_status', data);
        break;
      case 'notification':
        this.notifySubscribers('system_events', data);
        break;
      case 'error':
        console.error('❌ WebSocket error:', message.error);
        this.notifySubscribers('error', message);
        break;
      default:
        console.log(`📨 Received ${type}:`, data);
        this.notifySubscribers(type, data);
    }
  }

  private notifySubscribers(type: string, data: any): void {
    const subscriptions = this.subscriptions.get(type) || [];
    subscriptions.forEach(subscription => {
      try {
        subscription.callback(data);
      } catch (error) {
        console.error(`❌ Error in subscription callback for ${type}:`, error);
      }
    });
  }

  // Public API methods
  sendMessage(message: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const messageWithTimestamp = {
        ...message,
        timestamp: new Date().toISOString()
      };
      this.ws.send(JSON.stringify(messageWithTimestamp));
    } else {
      // Queue message for when connection is restored
      this.messageQueue.push(message);
    }
  }

  subscribe(type: string, callback: (data: any) => void): () => void {
    if (!this.subscriptions.has(type)) {
      this.subscriptions.set(type, []);
    }
    
    const subscription: WebSocketSubscription = { type, callback };
    this.subscriptions.get(type)!.push(subscription);
    
    // Return unsubscribe function
    return () => {
      const subscriptions = this.subscriptions.get(type) || [];
      const index = subscriptions.indexOf(subscription);
      if (index > -1) {
        subscriptions.splice(index, 1);
      }
    };
  }

  // Query methods with streaming support
  async submitQuery(
    query: string, 
    useRAG = true, 
    onChunk?: (chunk: string) => void
  ): Promise<string> {
    return new Promise((resolve, reject) => {
      const queryId = `query_${Date.now()}`;
      let fullResponse = '';
      
      // Subscribe to streaming chunks
      const unsubscribeChunk = this.subscribe('query_chunk', (message: WebSocketMessage) => {
        if (message.id === queryId) {
          const chunk = message.data?.chunk || '';
          fullResponse += chunk;
          if (onChunk) {
            onChunk(chunk);
          }
        }
      });

      // Subscribe to query completion
      const unsubscribeComplete = this.subscribe('query_response', (message: WebSocketMessage) => {
        if (message.id === queryId) {
          unsubscribeChunk();
          unsubscribeComplete();
          if (message.type === 'error') {
            reject(new Error(message.error || 'Query failed'));
          } else {
            // Return the accumulated response or single response
            resolve(fullResponse || message.data?.text || message.data?.response || 'No response');
          }
        }
      });

      // Subscribe to errors
      const unsubscribeError = this.subscribe('error', (message: WebSocketMessage) => {
        if (message.id === queryId) {
          unsubscribeChunk();
          unsubscribeComplete();
          unsubscribeError();
          reject(new Error(message.error || 'Query failed'));
        }
      });

      // Send the query
      this.sendMessage({
        type: 'query',
        id: queryId,
        data: {
          query,
          use_rag: useRAG
        }
      });

      // Set timeout for query
      setTimeout(() => {
        unsubscribeChunk();
        unsubscribeComplete();
        unsubscribeError();
        reject(new Error('Query timeout'));
      }, 60000); // 60 second timeout for streaming
    });
  }

  requestStatus(): void {
    this.sendMessage({
      type: 'get_status',
      data: {}
    });
  }

  restartOllama(): void {
    this.sendMessage({
      type: 'restart_ollama',
      data: {}
    });
  }

  // Real-time data subscriptions
  subscribeToMetrics(callback: (metrics: any) => void): () => void {
    return this.subscribe('metrics', callback);
  }

  subscribeToConsciousness(callback: (state: any) => void): () => void {
    return this.subscribe('consciousness', callback);
  }

  subscribeToOllamaStatus(callback: (status: any) => void): () => void {
    return this.subscribe('ollama_status', callback);
  }

  subscribeToSystemEvents(callback: (event: any) => void): () => void {
    return this.subscribe('system_events', callback);
  }

  subscribeToQueryResults(callback: (result: any) => void): () => void {
    return this.subscribe('query_response', callback);
  }
}

// Create singleton instance
export const cortexWebSocket = new CortexWebSocketService();

// Auto-connect when module loads (optional)
if (typeof window !== 'undefined') {
  cortexWebSocket.connect().catch(error => {
    console.error('❌ Failed to auto-connect WebSocket:', error);
  });
}

export default cortexWebSocket;