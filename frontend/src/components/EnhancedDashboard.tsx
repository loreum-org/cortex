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
  Loader2,
  Zap,
  Brain,
  RefreshCw
} from 'lucide-react';
import { enhancedCortexWebSocket, type ConnectionStatus } from '../services/enhanced-websocket';
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
import { Terminal } from './Terminal';

interface DashboardState {
  systemStatus: any;
  systemMetrics: any;
  networkInfo: any;
  agentMetrics: any;
  economicData: any;
  events: EventData[];
  loading: boolean;
  error: string | null;
  lastUpdated: Date | null;
  connectionStatus: ConnectionStatus;
  realtimeData: {
    metrics: any;
    consciousness: any;
    systemEvents: any[];
  };
}

export function EnhancedDashboard() {
  const [searchParams] = useSearchParams();
  const port = parseInt(searchParams.get('port') || '4891');

  const [state, setState] = useState<DashboardState>({
    systemStatus: null,
    systemMetrics: null,
    networkInfo: null,
    agentMetrics: null,
    economicData: null,
    events: [],
    loading: true,
    error: null,
    lastUpdated: null,
    connectionStatus: 'disconnected',
    realtimeData: {
      metrics: null,
      consciousness: null,
      systemEvents: [],
    },
  });

  // Update connection status
  useEffect(() => {
    const unsubscribe = enhancedCortexWebSocket.onConnectionStatusChange((status) => {
      setState(prev => ({ ...prev, connectionStatus: status }));
    });

    // Initial status
    setState(prev => ({ 
      ...prev, 
      connectionStatus: enhancedCortexWebSocket.getConnectionStatus() 
    }));

    return unsubscribe;
  }, []);

  // Set up real-time subscriptions
  useEffect(() => {
    const unsubscribeFunctions: (() => void)[] = [];

    // Subscribe to real-time metrics
    const unsubscribeMetrics = enhancedCortexWebSocket.subscribeToMetrics((metrics) => {
      setState(prev => ({
        ...prev,
        realtimeData: {
          ...prev.realtimeData,
          metrics,
        },
        lastUpdated: new Date(),
      }));
    });
    unsubscribeFunctions.push(unsubscribeMetrics);

    // Subscribe to consciousness updates
    const unsubscribeConsciousness = enhancedCortexWebSocket.subscribeToConsciousness((consciousness) => {
      setState(prev => ({
        ...prev,
        realtimeData: {
          ...prev.realtimeData,
          consciousness,
        },
      }));
    });
    unsubscribeFunctions.push(unsubscribeConsciousness);

    // Subscribe to system events
    const unsubscribeSystemEvents = enhancedCortexWebSocket.subscribeToSystemEvents((event) => {
      setState(prev => ({
        ...prev,
        realtimeData: {
          ...prev.realtimeData,
          systemEvents: [event, ...prev.realtimeData.systemEvents].slice(0, 50), // Keep last 50 events
        },
      }));
    });
    unsubscribeFunctions.push(unsubscribeSystemEvents);

    // Subscribe to agent events
    const unsubscribeAgentEvents = enhancedCortexWebSocket.subscribeToAgentEvents((event) => {
      // Handle agent state changes
      loadAgentMetrics(); // Refresh agent metrics when agents change
    });
    unsubscribeFunctions.push(unsubscribeAgentEvents);

    // Subscribe to economic events
    const unsubscribeEconomicEvents = enhancedCortexWebSocket.subscribeToEconomicEvents((event) => {
      // Handle economic updates
      loadEconomicData(); // Refresh economic data when transactions occur
    });
    unsubscribeFunctions.push(unsubscribeEconomicEvents);

    return () => {
      unsubscribeFunctions.forEach(fn => fn());
    };
  }, []);

  // Load initial data
  const loadInitialData = async () => {
    setState(prev => ({ ...prev, loading: true, error: null }));

    try {
      // Load all dashboard data in parallel using the event-driven API
      const [
        systemStatus,
        systemMetrics,
        networkInfo,
        agentData,
        economicBalance,
      ] = await Promise.all([
        enhancedCortexWebSocket.getSystemStatus(),
        enhancedCortexWebSocket.getSystemMetrics(),
        enhancedCortexWebSocket.getNetworkInfo(),
        enhancedCortexWebSocket.getAgents(),
        enhancedCortexWebSocket.getBalance().catch(() => null), // Economic data might not be available
      ]);

      setState(prev => ({
        ...prev,
        systemStatus,
        systemMetrics,
        networkInfo,
        agentMetrics: agentData,
        economicData: economicBalance,
        loading: false,
        lastUpdated: new Date(),
      }));
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
      setState(prev => ({
        ...prev,
        error: error instanceof Error ? error.message : 'Failed to load data',
        loading: false,
      }));
    }
  };

  // Individual data loaders for real-time updates
  const loadAgentMetrics = async () => {
    try {
      const agentData = await enhancedCortexWebSocket.getAgents();
      setState(prev => ({ ...prev, agentMetrics: agentData }));
    } catch (error) {
      console.error('Failed to load agent metrics:', error);
    }
  };

  const loadEconomicData = async () => {
    try {
      const economicBalance = await enhancedCortexWebSocket.getBalance();
      setState(prev => ({ ...prev, economicData: economicBalance }));
    } catch (error) {
      console.error('Failed to load economic data:', error);
    }
  };

  // Load data on mount and when connection is established
  useEffect(() => {
    if (state.connectionStatus === 'connected') {
      loadInitialData();
    }
  }, [state.connectionStatus]);

  // Manual refresh
  const handleRefresh = () => {
    loadInitialData();
  };

  // Get connection indicator
  const getConnectionIndicator = () => {
    switch (state.connectionStatus) {
      case 'connected':
        return <Wifi className="h-4 w-4 text-green-500" />;
      case 'connecting':
        return <Loader2 className="h-4 w-4 text-yellow-500 animate-spin" />;
      case 'disconnected':
        return <WifiOff className="h-4 w-4 text-red-500" />;
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-500" />;
      default:
        return <WifiOff className="h-4 w-4 text-gray-500" />;
    }
  };

  // Calculate derived metrics
  const derivedMetrics = {
    totalAgents: state.agentMetrics?.count || 0,
    activeAgents: state.agentMetrics?.agents?.filter((a: any) => a.status === 'active').length || 0,
    totalPeers: state.networkInfo?.peer_count || 0,
    totalConnections: state.networkInfo?.connections || 0,
    systemUptime: state.systemStatus?.uptime || 0,
    economicBalance: state.economicData?.balance || '0',
    stakingRewards: state.economicData?.rewards || '0',
  };

  if (state.loading && !state.systemStatus) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-gray-600">Loading dashboard...</p>
          <p className="text-sm text-gray-400 mt-2">
            Connection: {state.connectionStatus}
          </p>
        </div>
      </div>
    );
  }

  if (state.error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6">
        <div className="flex items-center">
          <AlertCircle className="h-5 w-5 text-red-500 mr-2" />
          <h3 className="text-lg font-medium text-red-800">Dashboard Error</h3>
        </div>
        <p className="text-red-700 mt-2">{state.error}</p>
        <button
          onClick={handleRefresh}
          className="mt-4 bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 flex items-center"
        >
          <RefreshCw className="h-4 w-4 mr-2" />
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Enhanced Dashboard</h1>
          <p className="text-gray-600 mt-1">
            Real-time event-driven monitoring
          </p>
        </div>
        
        <div className="flex items-center space-x-4">
          {/* Connection Status */}
          <div className="flex items-center space-x-2">
            {getConnectionIndicator()}
            <span className="text-sm text-gray-600 capitalize">
              {state.connectionStatus}
            </span>
          </div>
          
          {/* Last Updated */}
          {state.lastUpdated && (
            <span className="text-sm text-gray-500">
              Updated: {state.lastUpdated.toLocaleTimeString()}
            </span>
          )}
          
          {/* Refresh Button */}
          <button
            onClick={handleRefresh}
            disabled={state.loading}
            className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50 flex items-center"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${state.loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* System Overview Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="System Status"
          value={state.systemStatus?.status || 'Unknown'}
          icon={<Server className="h-6 w-6" />}
          trend={state.systemStatus?.status === 'healthy' ? 'up' : 'down'}
          color={state.systemStatus?.status === 'healthy' ? 'green' : 'red'}
        />
        
        <MetricCard
          title="Active Agents"
          value={`${derivedMetrics.activeAgents}/${derivedMetrics.totalAgents}`}
          icon={<Users className="h-6 w-6" />}
          trend="neutral"
          color="blue"
        />
        
        <MetricCard
          title="Network Peers"
          value={derivedMetrics.totalPeers.toString()}
          icon={<Network className="h-6 w-6" />}
          trend="neutral"
          color="purple"
        />
        
        <MetricCard
          title="CTX Balance"
          value={parseFloat(derivedMetrics.economicBalance).toFixed(2)}
          icon={<Zap className="h-6 w-6" />}
          trend="up"
          color="green"
        />
      </div>

      {/* Real-time Metrics */}
      {state.realtimeData.metrics && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="CPU Usage"
            value={`${state.realtimeData.metrics.system?.cpu_usage || 0}%`}
            icon={<Activity className="h-6 w-6" />}
            trend="neutral"
            color="orange"
          />
          
          <MetricCard
            title="Memory Usage"
            value={`${state.realtimeData.metrics.system?.memory_usage || 0}%`}
            icon={<Database className="h-6 w-6" />}
            trend="neutral"
            color="indigo"
          />
          
          <MetricCard
            title="Query Processing"
            value={state.realtimeData.metrics.queries_processed || 0}
            icon={<Search className="h-6 w-6" />}
            trend="up"
            color="teal"
          />
          
          <MetricCard
            title="Uptime"
            value={`${Math.floor(derivedMetrics.systemUptime / 3600)}h`}
            icon={<Server className="h-6 w-6" />}
            trend="up"
            color="gray"
          />
        </div>
      )}

      {/* Consciousness Status */}
      {state.realtimeData.consciousness && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center mb-4">
            <Brain className="h-6 w-6 text-purple-600 mr-2" />
            <h2 className="text-xl font-semibold">Consciousness State</h2>
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-purple-50 rounded-lg p-4">
              <h3 className="font-medium text-purple-900">Working Memory</h3>
              <p className="text-2xl font-bold text-purple-600">
                {Object.keys(state.realtimeData.consciousness.working_memory || {}).length}
              </p>
              <p className="text-sm text-purple-700">Active concepts</p>
            </div>
            
            <div className="bg-blue-50 rounded-lg p-4">
              <h3 className="font-medium text-blue-900">Processing State</h3>
              <p className="text-2xl font-bold text-blue-600">
                {state.realtimeData.consciousness.state?.processing_mode || 'Idle'}
              </p>
              <p className="text-sm text-blue-700">Current mode</p>
            </div>
            
            <div className="bg-green-50 rounded-lg p-4">
              <h3 className="font-medium text-green-900">Context Depth</h3>
              <p className="text-2xl font-bold text-green-600">
                {state.realtimeData.consciousness.metrics?.context_depth || 0}
              </p>
              <p className="text-sm text-green-700">Conversation layers</p>
            </div>
          </div>
        </div>
      )}

      {/* Agent Details */}
      {state.agentMetrics && state.agentMetrics.agents && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">Agent Status</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {state.agentMetrics.agents.map((agent: any) => (
              <div key={agent.id} className="border rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <h3 className="font-medium">{agent.name}</h3>
                  <span className={`px-2 py-1 rounded-full text-xs ${
                    agent.status === 'active' 
                      ? 'bg-green-100 text-green-800' 
                      : 'bg-gray-100 text-gray-800'
                  }`}>
                    {agent.status}
                  </span>
                </div>
                <p className="text-sm text-gray-600">{agent.type}</p>
                <div className="mt-2 text-xs text-gray-500">
                  <p>Capabilities: {agent.capabilities?.length || 0}</p>
                  <p>Created: {new Date(agent.created_at * 1000).toLocaleDateString()}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Economic Overview */}
      {state.economicData && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">Economic Overview</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="bg-green-50 rounded-lg p-4">
              <h3 className="font-medium text-green-900">Available Balance</h3>
              <p className="text-2xl font-bold text-green-600">
                {parseFloat(state.economicData.balance).toFixed(2)} CTX
              </p>
            </div>
            
            <div className="bg-blue-50 rounded-lg p-4">
              <h3 className="font-medium text-blue-900">Staked Amount</h3>
              <p className="text-2xl font-bold text-blue-600">
                {parseFloat(state.economicData.staked || '0').toFixed(2)} CTX
              </p>
            </div>
            
            <div className="bg-yellow-50 rounded-lg p-4">
              <h3 className="font-medium text-yellow-900">Rewards Earned</h3>
              <p className="text-2xl font-bold text-yellow-600">
                {parseFloat(state.economicData.rewards || '0').toFixed(2)} CTX
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Network Information */}
      {state.networkInfo && (
        <div className="bg-white rounded-lg shadow p-6">
          <h2 className="text-xl font-semibold mb-4">Network Information</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <h3 className="font-medium mb-2">Node Details</h3>
              <div className="space-y-2 text-sm">
                <p><span className="font-medium">Node ID:</span> {state.networkInfo.node_id?.substring(0, 20)}...</p>
                <p><span className="font-medium">Peer Count:</span> {state.networkInfo.peer_count}</p>
                <p><span className="font-medium">Connections:</span> {state.networkInfo.connections}</p>
                <p><span className="font-medium">Protocol:</span> {state.networkInfo.protocol}</p>
              </div>
            </div>
            
            <div>
              <h3 className="font-medium mb-2">Network Addresses</h3>
              <div className="space-y-1 text-xs text-gray-600 max-h-32 overflow-y-auto">
                {state.networkInfo.addresses?.map((addr: string, index: number) => (
                  <p key={index} className="font-mono">{addr}</p>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Real-time Events */}
      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold mb-4">Real-time System Events</h2>
        {state.realtimeData.systemEvents.length > 0 ? (
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {state.realtimeData.systemEvents.map((event, index) => (
              <div key={index} className="border-l-4 border-blue-400 bg-blue-50 p-3">
                <div className="flex items-center justify-between">
                  <span className="font-medium">{event.type || 'System Event'}</span>
                  <span className="text-xs text-gray-500">
                    {new Date(event.timestamp || Date.now()).toLocaleTimeString()}
                  </span>
                </div>
                <p className="text-sm text-gray-600 mt-1">
                  {event.message || JSON.stringify(event.data || {})}
                </p>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-gray-500 text-center py-8">
            No recent events. System events will appear here in real-time.
          </p>
        )}
      </div>

      {/* Terminal Component */}
      <div className="bg-white rounded-lg shadow">
        <Terminal />
      </div>
    </div>
  );
}