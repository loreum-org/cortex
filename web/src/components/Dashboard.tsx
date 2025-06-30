import { useState, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';
import { 
  Network, 
  Server, 
  Search, 
  Database, 
  Users, 
  Activity,
  AlertCircle,
  Wifi,
  WifiOff,
  Loader2
} from 'lucide-react';
import { CortexAPI } from '../services/api';
import { cortexWebSocket, type ConnectionStatus } from '../services/websocket';
import type { 
  NetworkMetrics, 
  SystemMetrics, 
  QueryMetrics, 
  RAGMetrics, 
  EventData, 
  Peer 
} from '../services/api';
import { MetricCard } from './MetricCard';
import { EventList } from './EventList';
import { PeerList } from './PeerList';

interface DashboardState {
  networkMetrics: NetworkMetrics | null;
  systemMetrics: SystemMetrics | null;
  queryMetrics: QueryMetrics | null;
  ragMetrics: RAGMetrics | null;
  events: EventData[];
  peers: Peer[];
  loading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  connectionStatus: ConnectionStatus;
  realTimeMetrics: any;
}

export function Dashboard() {
  const [searchParams] = useSearchParams();
  const port = parseInt(searchParams.get('port') || '8080');
  const nodeAPI = CortexAPI.forPort(port);

  const [state, setState] = useState<DashboardState>({
    networkMetrics: null,
    systemMetrics: null,
    queryMetrics: null,
    ragMetrics: null,
    events: [],
    peers: [],
    loading: true,
    error: null,
    lastUpdated: null,
    connectionStatus: 'disconnected',
    realTimeMetrics: null,
  });

  const [autoRefresh, setAutoRefresh] = useState(true);
  const [useRealTime, setUseRealTime] = useState(true);

  const fetchAllData = async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }));

      const [networkMetrics, systemMetrics, queryMetrics, ragMetrics, events, peers] = 
        await Promise.all([
          nodeAPI.getNetworkMetrics(),
          nodeAPI.getSystemMetrics(),
          nodeAPI.getQueryMetrics(),
          nodeAPI.getRAGMetrics(),
          nodeAPI.getEvents(20),
          nodeAPI.getPeers(),
        ]);

      setState(prev => ({
        ...prev,
        networkMetrics,
        systemMetrics,
        queryMetrics,
        ragMetrics,
        events,
        peers,
        loading: false,
        lastUpdated: new Date(),
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : 'Unknown error occurred',
      }));
    }
  };

  useEffect(() => {
    // Initial data fetch
    fetchAllData();
  }, []);

  useEffect(() => {
    // Set up WebSocket connection and real-time subscriptions
    setState(prev => ({ ...prev, connectionStatus: cortexWebSocket.getConnectionStatus() }));
    
    const unsubscribeStatus = cortexWebSocket.onConnectionStatusChange((status) => {
      setState(prev => ({ ...prev, connectionStatus: status }));
    });

    const unsubscribeMetrics = cortexWebSocket.subscribeToMetrics((metrics) => {
      if (useRealTime && metrics.system_metrics) {
        setState(prev => ({
          ...prev,
          realTimeMetrics: metrics.system_metrics,
          lastUpdated: new Date(),
        }));
      }
    });

    const unsubscribeEvents = cortexWebSocket.subscribeToSystemEvents((event) => {
      if (useRealTime) {
        setState(prev => ({
          ...prev,
          events: [
            {
              id: `event_${Date.now()}`,
              type: event.event_type || 'system_event',
              timestamp: new Date().toISOString(),
              data: event.data || {}
            },
            ...prev.events.slice(0, 19) // Keep only last 20 events
          ],
          lastUpdated: new Date(),
        }));
      }
    });

    // Connect if not already connected
    if (cortexWebSocket.getConnectionStatus() === 'disconnected') {
      cortexWebSocket.connect().catch(console.error);
    }

    return () => {
      unsubscribeStatus();
      unsubscribeMetrics();
      unsubscribeEvents();
    };
  }, [useRealTime]);

  useEffect(() => {
    if (!autoRefresh || useRealTime) return;

    const interval = setInterval(fetchAllData, 5000);
    return () => clearInterval(interval);
  }, [autoRefresh, useRealTime]);

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const formatUptime = (seconds: number): string => {
    if (seconds < 60) return `${Math.floor(seconds)}s`;
    if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60);
      const remainingSeconds = Math.floor(seconds % 60);
      return `${minutes}m ${remainingSeconds}s`;
    }
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };


  if (state.loading && !state.networkMetrics) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-tesla-black">
        <div className="animate-spin h-8 w-8 border-2 border-tesla-white border-t-transparent"></div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
      {/* Header */}
      <div className="flex justify-between items-center border-b border-tesla-border pb-6">
        <div className="flex items-center gap-6">
          <div>
            <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE DASHBOARD</h1>
            <p className="text-tesla-text-gray text-sm mt-1">PORT: {port}</p>
          </div>
          <div className={`flex items-center gap-2 text-xs px-3 py-1 uppercase tracking-wider border ${
            state.connectionStatus === 'connected' 
              ? 'text-green-400 bg-green-900/20 border-green-500/30' 
              : state.connectionStatus === 'connecting'
              ? 'text-yellow-400 bg-yellow-900/20 border-yellow-500/30'
              : 'text-red-400 bg-red-900/20 border-red-500/30'
          }`}>
            {state.connectionStatus === 'connected' ? (
              <>
                <Wifi className="h-3 w-3" />
                WEBSOCKET LIVE
              </>
            ) : state.connectionStatus === 'connecting' ? (
              <>
                <Loader2 className="h-3 w-3 animate-spin" />
                CONNECTING...
              </>
            ) : (
              <>
                <WifiOff className="h-3 w-3" />
                OFFLINE
              </>
            )}
          </div>
        </div>
        <div className="flex items-center gap-6">
          {state.lastUpdated && (
            <span className="text-xs text-tesla-text-gray uppercase tracking-wider">
              LAST UPDATED: {state.lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={useRealTime}
              onChange={(e) => setUseRealTime(e.target.checked)}
              className="w-4 h-4 bg-tesla-dark-gray border border-tesla-border focus:ring-0 focus:ring-offset-0"
            />
            <span className="text-xs text-tesla-text-gray uppercase tracking-wider">REAL-TIME</span>
          </label>
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4 bg-tesla-dark-gray border border-tesla-border focus:ring-0 focus:ring-offset-0"
              disabled={useRealTime}
            />
            <span className="text-xs text-tesla-text-gray uppercase tracking-wider">AUTO-REFRESH</span>
          </label>
          <button
            onClick={fetchAllData}
            className="tesla-button text-xs"
            disabled={useRealTime}
          >
            REFRESH
          </button>
        </div>
      </div>

      {state.error && (
        <div className="bg-tesla-dark-gray border border-tesla-accent-red p-4 flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-tesla-accent-red" />
          <span className="text-tesla-white text-sm">ERROR: {state.error}</span>
        </div>
      )}

      {/* Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Network Status */}
        <MetricCard
          title={`Network Status ${useRealTime && state.connectionStatus === 'connected' ? '(LIVE)' : ''}`}
          icon={<Network className="h-6 w-6" />}
          className="col-span-1 md:col-span-2"
        >
          {state.networkMetrics && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <div className={`w-2 h-2 ${
                    state.networkMetrics.peer_count > 0 ? 'bg-tesla-accent-green' : 'bg-tesla-text-gray'
                  } ${useRealTime && state.connectionStatus === 'connected' ? 'animate-pulse' : ''}`} />
                  <span className="text-xs text-tesla-text-gray uppercase tracking-wider">PEERS</span>
                </div>
                <div className="text-3xl font-light text-tesla-white">{state.networkMetrics.peer_count}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">DATA RECEIVED</div>
                <div className="text-3xl font-light text-tesla-white">{formatBytes(state.networkMetrics.bytes_received)}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">DATA SENT</div>
                <div className="text-3xl font-light text-tesla-white">{formatBytes(state.networkMetrics.bytes_sent)}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">WS CONNECTIONS</div>
                <div className="text-3xl font-light text-tesla-white">
                  {state.realTimeMetrics?.websocket_connections || (state.networkMetrics.connections_opened - state.networkMetrics.connections_closed)}
                </div>
              </div>
            </div>
          )}
        </MetricCard>

        {/* System Health */}
        <MetricCard
          title="System Health"
          icon={<Server className="h-6 w-6" />}
          className="col-span-1 md:col-span-2"
        >
          {state.systemMetrics && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">UPTIME</div>
                <div className="text-3xl font-light text-tesla-white">{formatUptime(state.systemMetrics.uptime_seconds)}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">CPU CORES</div>
                <div className="text-3xl font-light text-tesla-white">{state.systemMetrics.cpu_cores}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">MEMORY USAGE</div>
                <div className="text-3xl font-light text-tesla-white">
                  {state.systemMetrics.memory_sys > 0 
                    ? Math.round((state.systemMetrics.memory_allocated / state.systemMetrics.memory_sys) * 100)
                    : 0}%
                </div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">GOROUTINES</div>
                <div className="text-3xl font-light text-tesla-white">{state.systemMetrics.goroutines}</div>
              </div>
            </div>
          )}
        </MetricCard>

        {/* Query Processing */}
        <MetricCard
          title={`Query Processing ${useRealTime && state.connectionStatus === 'connected' ? '(LIVE)' : ''}`}
          icon={<Search className="h-6 w-6" />}
          className="col-span-1 md:col-span-2"
        >
          {state.queryMetrics && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">QUERIES PROCESSED</div>
                  <div className="text-3xl font-light text-tesla-white">
                    {state.realTimeMetrics?.queries_processed || state.queryMetrics.queries_processed}
                  </div>
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <div className={`w-2 h-2 ${
                      state.queryMetrics.success_rate >= 90 ? 'bg-tesla-accent-green' : 
                      state.queryMetrics.success_rate >= 70 ? 'bg-tesla-text-gray' : 'bg-tesla-accent-red'
                    } ${useRealTime && state.connectionStatus === 'connected' ? 'animate-pulse' : ''}`} />
                    <span className="text-xs text-tesla-text-gray uppercase tracking-wider">SUCCESS RATE</span>
                  </div>
                  <div className="text-3xl font-light text-tesla-white">{state.queryMetrics.success_rate.toFixed(1)}%</div>
                </div>
                <div>
                  <div className="flex items-center gap-2 mb-2">
                    <div className={`w-2 h-2 ${
                      state.queryMetrics.avg_latency_ms <= 500 ? 'bg-tesla-accent-green' : 
                      state.queryMetrics.avg_latency_ms <= 1000 ? 'bg-tesla-text-gray' : 'bg-tesla-accent-red'
                    }`} />
                    <span className="text-xs text-tesla-text-gray uppercase tracking-wider">AVG. LATENCY</span>
                  </div>
                  <div className="text-3xl font-light text-tesla-white">{Math.round(state.queryMetrics.avg_latency_ms)}ms</div>
                </div>
                <div>
                  <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">FAILED QUERIES</div>
                  <div className="text-3xl font-light text-tesla-white">
                    {state.realTimeMetrics?.query_failures || state.queryMetrics.query_failures}
                  </div>
                </div>
              </div>
            </div>
          )}
        </MetricCard>

        {/* RAG System */}
        <MetricCard
          title="RAG System"
          icon={<Database className="h-6 w-6" />}
          className="col-span-1 md:col-span-2"
        >
          {state.ragMetrics && (
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">DOCUMENTS</div>
                <div className="text-3xl font-light text-tesla-white">{state.ragMetrics.document_count}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">DOCUMENTS ADDED</div>
                <div className="text-3xl font-light text-tesla-white">{state.ragMetrics.documents_added}</div>
              </div>
              <div>
                <div className="text-xs text-tesla-text-gray uppercase tracking-wider mb-2">RAG QUERIES</div>
                <div className="text-3xl font-light text-tesla-white">{state.ragMetrics.rag_queries_total}</div>
              </div>
              <div>
                <div className="flex items-center gap-2 mb-2">
                  <div className={`w-2 h-2 ${
                    state.ragMetrics.avg_latency_ms <= 500 ? 'bg-tesla-accent-green' : 
                    state.ragMetrics.avg_latency_ms <= 1000 ? 'bg-tesla-text-gray' : 'bg-tesla-accent-red'
                  }`} />
                  <span className="text-xs text-tesla-text-gray uppercase tracking-wider">AVG. LATENCY</span>
                </div>
                <div className="text-3xl font-light text-tesla-white">{Math.round(state.ragMetrics.avg_latency_ms)}ms</div>
              </div>
            </div>
          )}
        </MetricCard>
      </div>

      {/* Bottom Section */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Connected Peers */}
        <MetricCard title="Connected Peers" icon={<Users className="h-6 w-6" />}>
          <PeerList peers={state.peers} />
        </MetricCard>

        {/* Recent Events */}
        <MetricCard title="Recent Events" icon={<Activity className="h-6 w-6" />}>
          <EventList events={state.events} />
        </MetricCard>
      </div>
    </div>
  );
}