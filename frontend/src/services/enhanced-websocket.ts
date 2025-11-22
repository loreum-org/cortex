// Enhanced WebSocket service for event-driven communication with Cortex API
export interface WebSocketMessage {
  type: string;
  method?: string;
  id?: string;
  data?: any;
  timestamp?: string;
  error?: string;
  correlation_id?: string;
  event_type?: string;
  metadata?: Record<string, any>;
}

export interface WebSocketSubscription {
  type: string;
  callback: (data: any, message?: WebSocketMessage) => void;
  filter?: SubscriptionFilter;
}

export interface SubscriptionFilter {
  event_types?: string[];
  categories?: string[];
  service_types?: string[];
  source_patterns?: string[];
  custom_filters?: Record<string, any>;
  include_metadata?: boolean;
}

export interface EventSubscription {
  event_type: string;
  filter?: SubscriptionFilter;
  callback: (data: any, message?: WebSocketMessage) => void;
}

export interface RequestOptions {
  timeout?: number;
  retry?: boolean;
  priority?: 'low' | 'normal' | 'high' | 'critical';
}

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

export class EnhancedCortexWebSocketService {
  private ws: WebSocket | null = null;
  private subscriptions: Map<string, WebSocketSubscription[]> = new Map();
  private eventSubscriptions: Map<string, EventSubscription[]> = new Map();
  private messageQueue: WebSocketMessage[] = [];
  private pendingRequests: Map<string, {
    resolve: (data: any) => void;
    reject: (error: Error) => void;
    timeout: number;
  }> = new Map();
  
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 5000;
  private reconnectTimer: number | null = null;
  private connectionStatus: ConnectionStatus = 'disconnected';
  private statusCallbacks: ((status: ConnectionStatus) => void)[] = [];
  private baseURL: string;
  private userId: string = 'default_user';
  
  // Enhanced features
  private correlationIdCounter = 0;
  private enableEventLogging = false;
  private messageHistory: WebSocketMessage[] = [];
  private maxHistorySize = 1000;
  
  constructor(baseURL = 'ws://localhost:4891', userId = 'default_user') {
    this.baseURL = baseURL;
    this.userId = userId;
  }

  // Configuration methods
  setUserId(userId: string): void {
    this.userId = userId;
  }

  enableLogging(enabled = true): void {
    this.enableEventLogging = enabled;
  }

  getMessageHistory(): WebSocketMessage[] {
    return [...this.messageHistory];
  }

  clearMessageHistory(): void {
    this.messageHistory = [];
  }

  // Connection management
  async connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      if (this.ws?.readyState === WebSocket.OPEN) {
        resolve();
        return;
      }

      this.setConnectionStatus('connecting');
      
      const wsUrl = `${this.baseURL}/ws`;
      this.log('🔌 Attempting to connect to:', wsUrl);
      this.ws = new WebSocket(wsUrl);

      // Set headers if supported
      if (this.userId !== 'default_user') {
        // Note: WebSocket headers are limited, this would be handled in authentication
      }

      this.ws.onopen = () => {
        this.log('🚀 Connected to Cortex WebSocket');
        this.setConnectionStatus('connected');
        this.reconnectAttempts = 0;
        
        // Send queued messages
        this.flushMessageQueue();
        
        // Set up default subscriptions
        this.setupDefaultSubscriptions();
        
        resolve();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          this.addToHistory(message);
          this.log('📨 Received:', message.type, message.id ? `(${message.id})` : '');
          this.handleMessage(message);
        } catch (error) {
          console.error('❌ Error parsing WebSocket message:', error);
        }
      };

      this.ws.onclose = () => {
        this.log('🔌 WebSocket connection closed');
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
    
    // Reject all pending requests
    this.pendingRequests.forEach(({ reject }) => {
      reject(new Error('Connection closed'));
    });
    this.pendingRequests.clear();
    
    this.setConnectionStatus('disconnected');
  }

  // Enhanced event-driven methods

  // Send a request and wait for response using correlation
  async sendRequest(method: string, data: any = {}, options: RequestOptions = {}): Promise<any> {
    const requestId = this.generateRequestId();
    const correlationId = this.generateCorrelationId();
    const timeout = options.timeout || 30000;

    const message: WebSocketMessage = {
      type: 'request',
      method,
      id: requestId,
      data,
      correlation_id: correlationId,
      timestamp: new Date().toISOString(),
      metadata: {
        user_id: this.userId,
        priority: options.priority || 'normal',
      },
    };

    return new Promise((resolve, reject) => {
      // Set up timeout
      const timeoutId = window.setTimeout(() => {
        this.pendingRequests.delete(requestId);
        reject(new Error(`Request timeout: ${method}`));
      }, timeout);

      // Store request for correlation
      this.pendingRequests.set(requestId, {
        resolve: (data) => {
          window.clearTimeout(timeoutId);
          resolve(data);
        },
        reject: (error) => {
          window.clearTimeout(timeoutId);
          reject(error);
        },
        timeout: timeoutId,
      });

      // Send message
      this.sendMessage(message);
    });
  }

  // Send a command without expecting a response
  sendCommand(command: string, data: any = {}): void {
    const message: WebSocketMessage = {
      type: 'command',
      method: command,
      id: this.generateRequestId(),
      data,
      timestamp: new Date().toISOString(),
      metadata: {
        user_id: this.userId,
      },
    };

    this.sendMessage(message);
  }

  // Enhanced subscription with filters
  subscribeToEvents(subscriptions: EventSubscription[]): () => void {
    const unsubscribeFunctions: (() => void)[] = [];

    subscriptions.forEach(({ event_type, filter, callback }) => {
      if (!this.eventSubscriptions.has(event_type)) {
        this.eventSubscriptions.set(event_type, []);
      }

      const subscription: EventSubscription = { event_type, filter, callback };
      this.eventSubscriptions.get(event_type)!.push(subscription);

      unsubscribeFunctions.push(() => {
        const subs = this.eventSubscriptions.get(event_type) || [];
        const index = subs.indexOf(subscription);
        if (index > -1) {
          subs.splice(index, 1);
        }
      });
    });

    // Send subscription request to backend
    const subscriptionData = {
      event_types: subscriptions.map(s => s.event_type),
      filters: subscriptions.reduce((acc, s) => {
        if (s.filter) {
          acc[s.event_type] = s.filter;
        }
        return acc;
      }, {} as Record<string, SubscriptionFilter>),
    };

    this.sendMessage({
      type: 'subscribe',
      data: subscriptionData,
      timestamp: new Date().toISOString(),
    });

    // Return combined unsubscribe function
    return () => {
      unsubscribeFunctions.forEach(fn => fn());
      
      // Send unsubscribe request
      this.sendMessage({
        type: 'unsubscribe',
        data: {
          event_types: subscriptions.map(s => s.event_type),
        },
        timestamp: new Date().toISOString(),
      });
    };
  }

  // Simplified single event subscription
  subscribe(eventType: string, callback: (data: any, message?: WebSocketMessage) => void, filter?: SubscriptionFilter): () => void {
    return this.subscribeToEvents([{ event_type: eventType, callback, filter }]);
  }

  // Bulk subscription for multiple event types
  subscribeToMultiple(eventTypes: string[], callback: (data: any, message?: WebSocketMessage) => void): () => void {
    const subscriptions = eventTypes.map(event_type => ({ event_type, callback }));
    return this.subscribeToEvents(subscriptions);
  }

  // High-level API methods using the new event system

  // Agent Management
  async getAgents(): Promise<{ agents: any[]; count: number }> {
    return this.sendRequest('getAgents');
  }

  async startAgent(agentId: string): Promise<any> {
    return this.sendRequest('agent.start', { agent_id: agentId });
  }

  async stopAgent(agentId: string): Promise<any> {
    return this.sendRequest('agent.stop', { agent_id: agentId });
  }

  async restartAgent(agentId: string): Promise<any> {
    return this.sendRequest('agent.restart', { agent_id: agentId });
  }

  async getAgentMetrics(agentId?: string): Promise<any> {
    if (agentId) {
      return this.sendRequest('agent.get_metrics', { agent_id: agentId });
    } else {
      // Get all agent metrics
      return this.sendRequest('agent.get_metrics');
    }
  }

  async startAllAgents(): Promise<any> {
    return this.sendRequest('agent.start_all');
  }

  async stopAllAgents(): Promise<any> {
    return this.sendRequest('agent.stop_all');
  }

  async restartAllAgents(): Promise<any> {
    return this.sendRequest('agent.restart_all');
  }

  async updateAgentConfig(agentId: string, config: Record<string, any>): Promise<any> {
    return this.sendRequest('agent.update_config', { agent_id: agentId, config });
  }

  // Model Management
  async getModels(): Promise<{ installed: any[]; available: any[] }> {
    return this.sendRequest('getModels');
  }

  async downloadModel(modelName: string): Promise<any> {
    return this.sendRequest('downloadModel', { model_name: modelName });
  }

  async deleteModel(modelName: string): Promise<any> {
    return this.sendRequest('deleteModel', { model_name: modelName });
  }

  // Economic System
  async getBalance(userId?: string): Promise<any> {
    return this.sendRequest('economy.get_balance', { user_id: userId || this.userId });
  }

  async createAccount(userId?: string): Promise<any> {
    return this.sendRequest('economy.create_account', { user_id: userId || this.userId });
  }

  async transfer(fromAccount: string, toAccount: string, amount: string): Promise<any> {
    return this.sendRequest('economy.transfer', { from_account: fromAccount, to_account: toAccount, amount });
  }

  async stake(amount: string, userId?: string): Promise<any> {
    return this.sendRequest('economy.stake', { user_id: userId || this.userId, amount });
  }

  async unstake(stakingId: string): Promise<any> {
    return this.sendRequest('economy.unstake', { staking_id: stakingId });
  }

  async getStakingInfo(userId?: string): Promise<any> {
    return this.sendRequest('economy.get_staking_info', { user_id: userId || this.userId });
  }

  // RAG System
  async addDocument(content: string, metadata?: Record<string, any>): Promise<any> {
    return this.sendRequest('rag.add_document', { content, metadata });
  }

  async searchRAG(query: string): Promise<any> {
    return this.sendRequest('rag.search', { query });
  }

  async getRAGContext(userId?: string): Promise<any> {
    return this.sendRequest('rag.get_context', { user_id: userId || this.userId });
  }

  async clearRAGMemory(userId?: string): Promise<any> {
    return this.sendRequest('rag.clear_memory', { user_id: userId || this.userId });
  }

  // Service Registry
  async discoverServices(serviceType?: string): Promise<any> {
    return this.sendRequest('service.discover', { type: serviceType });
  }

  async registerService(name: string, type: string, address: string, metadata?: Record<string, any>): Promise<any> {
    return this.sendRequest('service.register', { name, type, address, metadata });
  }

  async deregisterService(serviceId: string): Promise<any> {
    return this.sendRequest('service.deregister', { service_id: serviceId });
  }

  // P2P Network
  async getNetworkInfo(): Promise<any> {
    return this.sendRequest('p2p.get_network_info');
  }

  async connectPeer(address: string): Promise<any> {
    return this.sendRequest('p2p.connect_peer', { address });
  }

  async disconnectPeer(peerId: string): Promise<any> {
    return this.sendRequest('p2p.disconnect_peer', { peer_id: peerId });
  }

  async broadcastMessage(message: string, type = 'general'): Promise<any> {
    return this.sendRequest('p2p.broadcast', { message, type });
  }

  // System Operations
  async getSystemStatus(): Promise<any> {
    return this.sendRequest('getStatus');
  }

  async getSystemMetrics(): Promise<any> {
    return this.sendRequest('getMetrics');
  }

  async getSystemHealth(): Promise<any> {
    return this.sendRequest('system.health_check');
  }

  async getSystemLogs(level?: string, limit = 100): Promise<any> {
    return this.sendRequest('system.get_logs', { level, limit });
  }

  // Query Processing (with streaming support)
  async submitQuery(
    query: string, 
    useRAG = true, 
    onChunk?: (chunk: string, metadata?: any) => void
  ): Promise<string> {
    const queryId = this.generateRequestId();
    let fullResponse = '';
    
    // Subscribe to query chunks
    const unsubscribeChunk = this.subscribe('query_chunk', (data: any, message?: WebSocketMessage) => {
      if (message?.id === queryId) {
        const chunk = data?.chunk || '';
        const metadata = data?.metadata || null;
        fullResponse += chunk;
        if (onChunk) {
          onChunk(chunk, metadata);
        }
      }
    });

    try {
      // Send query request
      const response = await this.sendRequest('submitQuery', {
        query,
        use_rag: useRAG,
        stream: !!onChunk, // Enable streaming if chunk handler provided
      });

      unsubscribeChunk();
      return fullResponse || response?.response || response?.text || 'No response';
    } catch (error) {
      unsubscribeChunk();
      throw error;
    }
  }

  // Real-time subscriptions with enhanced filtering
  subscribeToMetrics(callback: (metrics: any) => void): () => void {
    return this.subscribe('metrics_updated', callback);
  }

  subscribeToConsciousness(callback: (state: any) => void): () => void {
    return this.subscribe('consciousness_updated', callback);
  }

  subscribeToSystemEvents(callback: (event: any) => void): () => void {
    return this.subscribe('system_events', callback);
  }

  subscribeToAgentEvents(callback: (event: any) => void): () => void {
    return this.subscribeToMultiple([
      'agent.registered',
      'agent.deregistered',
      'agent.started',
      'agent.stopped',
      'agent.restarted',
      'agent.start_success',
      'agent.stop_success',
      'agent.restart_success',
      'agent.start_error',
      'agent.stop_error',
      'agent.restart_error',
      'agent.config_updated',
      'agent.start_all_success',
      'agent.stop_all_success',
      'agent.restart_all_success',
      'agent.start_all_error',
      'agent.stop_all_error',
      'agent.restart_all_error',
    ], callback);
  }

  subscribeToEconomicEvents(callback: (event: any) => void): () => void {
    return this.subscribeToMultiple([
      'economy.transfer',
      'economy.stake',
      'economy.unstake',
      'economy.reward',
    ], callback);
  }

  // Connection status management
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

  // Private methods

  private sendMessage(message: WebSocketMessage): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const messageWithTimestamp = {
        ...message,
        timestamp: message.timestamp || new Date().toISOString(),
      };
      
      this.addToHistory(messageWithTimestamp);
      this.log('📤 Sending:', message.type, message.id ? `(${message.id})` : '');
      this.ws.send(JSON.stringify(messageWithTimestamp));
    } else {
      this.log('⏰ Queueing message (connection not ready):', message.type);
      this.messageQueue.push(message);
    }
  }

  private handleMessage(message: WebSocketMessage): void {
    const { type, id } = message;
    
    // Handle response messages for pending requests
    if (type === 'response' && id && this.pendingRequests.has(id)) {
      const request = this.pendingRequests.get(id)!;
      this.pendingRequests.delete(id);
      
      if (message.error) {
        request.reject(new Error(message.error));
      } else {
        request.resolve(message.data);
      }
      return;
    }

    // Handle event subscriptions
    this.notifyEventSubscribers(message);
    
    // Handle legacy subscriptions
    this.notifySubscribers(type, message.data, message);
  }

  private notifyEventSubscribers(message: WebSocketMessage): void {
    const { type: messageType, event_type } = message;
    const eventType = event_type || messageType;
    
    const subscriptions = this.eventSubscriptions.get(eventType) || [];
    subscriptions.forEach(({ filter, callback }) => {
      // Apply filters if specified
      if (filter && !this.matchesFilter(message, filter)) {
        return;
      }
      
      try {
        callback(message.data, message);
      } catch (error) {
        console.error(`❌ Error in event subscription callback for ${eventType}:`, error);
      }
    });
  }

  private notifySubscribers(type: string, data: any, message?: WebSocketMessage): void {
    const subscriptions = this.subscriptions.get(type) || [];
    subscriptions.forEach(subscription => {
      try {
        subscription.callback(data, message);
      } catch (error) {
        console.error(`❌ Error in subscription callback for ${type}:`, error);
      }
    });
  }

  private matchesFilter(message: WebSocketMessage, filter: SubscriptionFilter): boolean {
    // Implement filter matching logic
    if (filter.event_types && filter.event_types.length > 0) {
      const eventType = message.event_type || message.type;
      if (!filter.event_types.includes(eventType)) {
        return false;
      }
    }

    if (filter.custom_filters) {
      for (const [key, value] of Object.entries(filter.custom_filters)) {
        if (message.metadata?.[key] !== value) {
          return false;
        }
      }
    }

    return true;
  }

  private setupDefaultSubscriptions(): void {
    // Auto-subscribe to important system events
    const defaultSubscriptions = [
      'metrics_updated',
      'consciousness_updated',
      'system_events',
      'connection_status',
      'error',
    ];

    this.sendMessage({
      type: 'subscribe',
      data: {
        event_types: defaultSubscriptions,
      },
      timestamp: new Date().toISOString(),
    });
  }

  private setConnectionStatus(status: ConnectionStatus): void {
    if (this.connectionStatus !== status) {
      this.connectionStatus = status;
      this.statusCallbacks.forEach(callback => callback(status));
    }
  }

  private attemptReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.log('❌ Max reconnection attempts reached');
      return;
    }

    this.reconnectAttempts++;
    this.log(`🔄 Attempting to reconnect (${this.reconnectAttempts}/${this.maxReconnectAttempts})...`);
    
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

  private generateRequestId(): string {
    return `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
  }

  private generateCorrelationId(): string {
    return `corr_${Date.now()}_${++this.correlationIdCounter}`;
  }

  private addToHistory(message: WebSocketMessage): void {
    this.messageHistory.push(message);
    if (this.messageHistory.length > this.maxHistorySize) {
      this.messageHistory.shift();
    }
  }

  private log(...args: any[]): void {
    if (this.enableEventLogging) {
      console.log('[EnhancedWebSocket]', ...args);
    }
  }
}

// Create singleton instance
export const enhancedCortexWebSocket = new EnhancedCortexWebSocketService();

// Auto-connect when module loads
if (typeof window !== 'undefined') {
  enhancedCortexWebSocket.enableLogging(process.env.NODE_ENV === 'development');
  enhancedCortexWebSocket.connect().catch(error => {
    console.error('❌ Failed to auto-connect enhanced WebSocket:', error);
  });
}

export default enhancedCortexWebSocket;