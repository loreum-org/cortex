export interface NetworkMetrics {
  peer_count: number;
  bytes_received: number;
  bytes_sent: number;
  connections_opened: number;
  connections_closed: number;
}

export interface SystemMetrics {
  uptime_seconds: number;
  goroutines: number;
  memory_allocated: number;
  memory_total: number;
  memory_sys: number;
  gc_cycles: number;
  cpu_cores: number;
}

export interface QueryMetrics {
  queries_processed: number;
  query_successes: number;
  query_failures: number;
  avg_latency_ms: number;
  success_rate: number;
}

export interface RAGMetrics {
  document_count: number;
  documents_added: number;
  rag_queries_total: number;
  rag_query_failures: number;
  avg_latency_ms: number;
}

export interface EventData {
  id: string;
  type: string;
  timestamp: string;
  data: Record<string, any>;
}

export interface Peer {
  id: string;
  protocols: string[];
}

export interface ChatMessage {
  id: string;
  content: string;
  role: 'user' | 'assistant';
  timestamp: string;
}

export interface QueryRequest {
  query: string;
  use_rag?: boolean;
}

export interface QueryResponse {
  query_id: string;
  text: string;
  data?: any;
  metadata?: Record<string, string>;
  status: string;
  timestamp: number;
}

export interface NodeInfo {
  nodeId: string;
  port: number;
  url: string;
  name: string;
  status: 'online' | 'offline' | 'unknown';
}

export interface NetworkOverview {
  totalNodes: number;
  onlineNodes: number;
  totalPeers: number;
  totalTransactions: number;
  networkHealth: number;
  nodes: NodeInfo[];
}

export interface TransactionData {
  id: string;
  type: string;
  timestamp: string;
  node_id: string;
  status: 'pending' | 'confirmed' | 'failed';
  data: Record<string, any>;
}

export interface WalletBalance {
  user_id: string;
  balance: string;
  balance_formatted: string;
  token_symbol: string;
}

export interface WalletAccount {
  user_id: string;
  address: string;
  balance: string;
  balance_formatted: string;
  total_spent: string;
  queries_submitted: number;
  created_at: string;
  updated_at: string;
  token_symbol: string;
}

export interface WalletTransferRequest {
  from_user_id: string;
  to_user_id: string;
  amount: string;
  description?: string;
}

export interface WalletTransferResponse {
  transaction_id: string;
  from_user_id: string;
  to_user_id: string;
  amount: string;
  description: string;
  status: string;
  created_at: string;
}

export interface WalletTransaction {
  id: string;
  type: string;
  from_id: string;
  to_id: string;
  amount: string;
  description: string;
  status: string;
  created_at: string;
}

export interface ReputationScore {
  node_id: string;
  score: number;
  last_update: string;
}

export interface ReputationScoresResponse {
  reputation_scores: ReputationScore[];
  total_nodes: number;
  timestamp: string;
}

export interface ConsensusStatus {
  total_transactions: number;
  finalized_transactions: number;
  pending_transactions: number;
  conflicts_detected: number;
  conflicts_resolved: number;
  dag_nodes: number;
  topological_order?: string[];
  reputation_stats: {
    average?: number;
    maximum?: number;
    minimum?: number;
    nodes_count?: number;
  };
  timestamp: string;
}

export interface ConflictData {
  id: string;
  type: string;
  transaction_a: string;
  transaction_b: string;
  description: string;
  severity: number;
  detected_at: string;
  resolved_at?: string;
  resolution_method?: string;
}

export interface ConflictsResponse {
  conflicts: ConflictData[];
  total_conflicts: number;
  unresolved_count: number;
  resolved_count: number;
  timestamp: string;
}

class CortexAPI {
  private baseURL: string;

  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
  }

  // Create a new API instance for a specific port
  static forPort(port: number): CortexAPI {
    return new CortexAPI(`http://localhost:${port}`);
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const url = `${this.baseURL}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    });

    if (!response.ok) {
      throw new Error(`API request failed: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }

  // Network metrics
  async getNetworkMetrics(): Promise<NetworkMetrics> {
    return this.request<NetworkMetrics>('/metrics/network');
  }

  async getPeers(): Promise<Peer[]> {
    return this.request<Peer[]>('/node/peers');
  }

  // System metrics
  async getSystemMetrics(): Promise<SystemMetrics> {
    return this.request<SystemMetrics>('/metrics/system');
  }

  // Query metrics
  async getQueryMetrics(): Promise<QueryMetrics> {
    return this.request<QueryMetrics>('/metrics/queries');
  }

  // RAG metrics
  async getRAGMetrics(): Promise<RAGMetrics> {
    return this.request<RAGMetrics>('/metrics/rag');
  }

  // Events
  async getEvents(limit = 20): Promise<EventData[]> {
    return this.request<EventData[]>(`/events?limit=${limit}`);
  }

  // Chat/Query functionality
  async submitQuery(query: QueryRequest): Promise<QueryResponse> {
    return this.request<QueryResponse>('/queries', {
      method: 'POST',
      body: JSON.stringify(query),
    });
  }

  // Health check
  async healthCheck(): Promise<{ status: string; timestamp: string }> {
    return this.request<{ status: string; timestamp: string }>('/health');
  }

  // Wallet methods
  async getWalletBalance(userId: string): Promise<WalletBalance> {
    return this.request<WalletBalance>(`/wallet/balance/${userId}`);
  }

  async getWalletAccount(userId: string): Promise<WalletAccount> {
    return this.request<WalletAccount>(`/wallet/account/${userId}`);
  }

  async transferTokens(transfer: WalletTransferRequest): Promise<WalletTransferResponse> {
    return this.request<WalletTransferResponse>('/wallet/transfer', {
      method: 'POST',
      body: JSON.stringify(transfer),
    });
  }

  async getWalletTransactions(userId: string, limit = 50, offset = 0): Promise<WalletTransaction[]> {
    return this.request<WalletTransaction[]>(`/wallet/transactions/${userId}?limit=${limit}&offset=${offset}`);
  }

  // Network overview methods
  async getNetworkOverview(ports: number[] = [8080]): Promise<NetworkOverview> {
    const nodes: NodeInfo[] = [];
    let totalPeers = 0;
    let totalTransactions = 0;
    let onlineNodes = 0;

    for (const port of ports) {
      const nodeAPI = CortexAPI.forPort(port);
      try {
        const health = await nodeAPI.healthCheck();
        const networkMetrics = await nodeAPI.getNetworkMetrics();
        const events = await nodeAPI.getEvents(100);

        const node: NodeInfo = {
          nodeId: `node-${port}`,
          port,
          url: `http://localhost:${port}`,
          name: `Cortex Node ${port}`,
          status: health.status === 'ok' ? 'online' : 'offline'
        };

        nodes.push(node);
        
        if (node.status === 'online') {
          onlineNodes++;
          totalPeers += networkMetrics.peer_count;
          totalTransactions += events.length;
        }
      } catch (error) {
        nodes.push({
          nodeId: `node-${port}`,
          port,
          url: `http://localhost:${port}`,
          name: `Cortex Node ${port}`,
          status: 'offline'
        });
      }
    }

    const networkHealth = onlineNodes > 0 ? (onlineNodes / ports.length) * 100 : 0;

    return {
      totalNodes: ports.length,
      onlineNodes,
      totalPeers,
      totalTransactions,
      networkHealth,
      nodes
    };
  }

  // Get aggregated transactions from all nodes  
  async getNetworkTransactions(ports: number[] = [8080]): Promise<TransactionData[]> {
    const allTransactions: TransactionData[] = [];

    for (const port of ports) {
      const nodeAPI = CortexAPI.forPort(port);
      try {
        const events = await nodeAPI.getEvents(50);
        const transactions = events
          .filter(event => 
            event.type.includes('query') || 
            event.type.includes('payment') || 
            event.type.includes('reward') ||
            event.type.includes('economic') ||
            event.type.includes('document') ||
            event.type.includes('rag')
          )
          .map(event => ({
            id: event.id,
            type: event.type,
            timestamp: event.timestamp,
            node_id: `node-${port}`,
            status: event.type.includes('_failed') ? 'failed' as const : 
                   event.type.includes('_processed') || event.type.includes('_distributed') ? 'confirmed' as const : 'pending' as const,
            data: event.data
          }));
        
        allTransactions.push(...transactions);
      } catch (error) {
        // Skip offline nodes
        continue;
      }
    }

    return allTransactions.sort((a, b) => 
      new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );
  }

  // Reputation API methods
  async getReputationScores(): Promise<ReputationScoresResponse> {
    return this.request<ReputationScoresResponse>('/reputation/scores');
  }

  async getNodeReputation(nodeId: string): Promise<ReputationScore> {
    return this.request<ReputationScore>(`/reputation/score/${nodeId}`);
  }

  async getConsensusStatus(): Promise<ConsensusStatus> {
    return this.request<ConsensusStatus>('/consensus/status');
  }

  async getConsensusConflicts(): Promise<ConflictsResponse> {
    return this.request<ConflictsResponse>('/consensus/conflicts');
  }

  // Staking API methods
  async stakeToNode(userID: string, nodeID: string, amount: string): Promise<any> {
    return this.request<any>('/staking/stake', {
      method: 'POST',
      body: JSON.stringify({
        user_id: userID,
        node_id: nodeID,
        amount: amount
      })
    });
  }

  async requestWithdrawal(userID: string, nodeID: string, amount: string): Promise<any> {
    return this.request<any>('/staking/withdraw/request', {
      method: 'POST',
      body: JSON.stringify({
        user_id: userID,
        node_id: nodeID,
        amount: amount
      })
    });
  }

  async executeWithdrawal(userID: string, nodeID: string): Promise<any> {
    return this.request<any>('/staking/withdraw/execute', {
      method: 'POST',
      body: JSON.stringify({
        user_id: userID,
        node_id: nodeID
      })
    });
  }

  async getUserStakes(userID: string): Promise<any> {
    return this.request<any>(`/staking/user/${userID}`);
  }

  async getNodeStakes(nodeID: string): Promise<any> {
    return this.request<any>(`/staking/node/${nodeID}`);
  }

  async getAllNodeStakes(): Promise<any> {
    return this.request<any>('/staking/nodes');
  }

  // Service Registry API methods
  async getNetworkServices(): Promise<import('../types/services').NetworkServiceStats> {
    return this.request<import('../types/services').NetworkServiceStats>('/services/network');
  }

  async getNodeServices(nodeId?: string): Promise<import('../types/services').ServiceOffering[]> {
    const endpoint = nodeId ? `/services/node/${nodeId}` : '/services/node';
    return this.request<import('../types/services').ServiceOffering[]>(endpoint);
  }

  async registerService(service: import('../types/services').RegisterServiceRequest): Promise<import('../types/services').ServiceOffering> {
    return this.request<import('../types/services').ServiceOffering>('/services/register', {
      method: 'POST',
      body: JSON.stringify(service),
    });
  }

  async deregisterService(serviceId: string): Promise<{ status: string; message: string }> {
    return this.request<{ status: string; message: string }>(`/services/deregister/${serviceId}`, {
      method: 'DELETE',
    });
  }

  async getServiceAttestations(serviceId?: string): Promise<import('../types/services').ServiceAttestation[]> {
    const endpoint = serviceId ? `/services/attestations/${serviceId}` : '/services/attestations';
    return this.request<import('../types/services').ServiceAttestation[]>(endpoint);
  }

  async findServices(requirements: import('../types/services').ServiceRequirement): Promise<import('../types/services').ServiceOffering[]> {
    return this.request<import('../types/services').ServiceOffering[]>('/services/find', {
      method: 'POST',
      body: JSON.stringify(requirements),
    });
  }
}

export const cortexAPI = new CortexAPI();
export { CortexAPI };
export default CortexAPI;