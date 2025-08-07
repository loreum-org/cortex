import { useState, useEffect } from 'react';
import { 
  Play, 
  Square, 
  RotateCcw, 
  Settings, 
  Activity, 
  AlertCircle, 
  CheckCircle, 
  Clock, 
  Loader2,
  BarChart3,
  RefreshCw,
  Plus
} from 'lucide-react';
import { enhancedCortexWebSocket } from '../services/enhanced-websocket';

interface Agent {
  id: string;
  name: string;
  type: string;
  status: string;
  capabilities: string[];
  created_at: number;
  metrics?: {
    queries_processed: number;
    average_latency: number;
    success_rate: number;
    last_updated: number;
  };
}

interface AgentMetrics {
  agent_id: string;
  queries_processed: number;
  success_rate: number;
  average_latency: number;
  last_activity: string;
  uptime: number;
}

export function EnhancedAgentManagement() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [agentMetrics, setAgentMetrics] = useState<Record<string, AgentMetrics>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<Record<string, boolean>>({});
  
  // Real-time state
  const [connectionStatus, setConnectionStatus] = useState(enhancedCortexWebSocket.getConnectionStatus());
  const [realtimeEvents, setRealtimeEvents] = useState<any[]>([]);

  // Connection status tracking
  useEffect(() => {
    const unsubscribe = enhancedCortexWebSocket.onConnectionStatusChange(setConnectionStatus);
    return unsubscribe;
  }, []);

  // Real-time agent event subscriptions
  useEffect(() => {
    const unsubscribeFunctions: (() => void)[] = [];

    // Subscribe to agent events
    const unsubscribeAgentEvents = enhancedCortexWebSocket.subscribeToAgentEvents((event, message) => {
      console.log('Agent event received:', event);
      
      // Add to realtime events
      setRealtimeEvents(prev => [
        { ...event, timestamp: Date.now(), message_type: message?.type },
        ...prev.slice(0, 49) // Keep last 50 events
      ]);

      // Refresh agent list when agents change
      if (['agent.registered', 'agent.deregistered', 'agent.started', 'agent.stopped'].includes(message?.type || '')) {
        loadAgents();
      }
    });
    unsubscribeFunctions.push(unsubscribeAgentEvents);

    // Subscribe to metrics updates
    const unsubscribeMetrics = enhancedCortexWebSocket.subscribeToMetrics((metrics) => {
      if (metrics.agents) {
        setAgentMetrics(metrics.agents);
      }
    });
    unsubscribeFunctions.push(unsubscribeMetrics);

    return () => {
      unsubscribeFunctions.forEach(fn => fn());
    };
  }, []);

  // Load agents when connected
  useEffect(() => {
    if (connectionStatus === 'connected') {
      loadAgents();
    }
  }, [connectionStatus]);

  const loadAgents = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await enhancedCortexWebSocket.getAgents();
      setAgents(response.agents || []);
      
      // Load metrics for each agent
      await loadAllAgentMetrics(response.agents || []);
    } catch (err) {
      console.error('Failed to load agents:', err);
      setError(err instanceof Error ? err.message : 'Failed to load agents');
    } finally {
      setLoading(false);
    }
  };

  const loadAllAgentMetrics = async (agentList: Agent[]) => {
    const metricsPromises = agentList.map(async (agent) => {
      try {
        const metrics = await enhancedCortexWebSocket.getAgentMetrics(agent.id);
        return { agentId: agent.id, metrics };
      } catch (error) {
        console.warn(`Failed to load metrics for agent ${agent.id}:`, error);
        return null;
      }
    });

    const results = await Promise.allSettled(metricsPromises);
    const newMetrics: Record<string, AgentMetrics> = {};

    results.forEach((result) => {
      if (result.status === 'fulfilled' && result.value) {
        const { agentId, metrics } = result.value;
        newMetrics[agentId] = metrics;
      }
    });

    setAgentMetrics(prev => ({ ...prev, ...newMetrics }));
  };

  const handleAgentAction = async (agentId: string, action: 'start' | 'stop' | 'restart') => {
    setActionLoading(prev => ({ ...prev, [`${agentId}-${action}`]: true }));

    try {
      let response;
      switch (action) {
        case 'start':
          response = await enhancedCortexWebSocket.startAgent(agentId);
          break;
        case 'stop':
          response = await enhancedCortexWebSocket.stopAgent(agentId);
          break;
        case 'restart':
          response = await enhancedCortexWebSocket.restartAgent(agentId);
          break;
      }

      console.log(`Agent ${action} response:`, response);
      
      // Refresh agents list after action
      setTimeout(() => loadAgents(), 1000);
      
    } catch (err) {
      console.error(`Failed to ${action} agent:`, err);
      setError(`Failed to ${action} agent: ${err instanceof Error ? err.message : 'Unknown error'}`);
    } finally {
      setActionLoading(prev => ({ ...prev, [`${agentId}-${action}`]: false }));
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'running':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'stopped':
      case 'inactive':
        return <Square className="h-5 w-5 text-gray-500" />;
      case 'starting':
      case 'stopping':
        return <Loader2 className="h-5 w-5 text-yellow-500 animate-spin" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      default:
        return <Clock className="h-5 w-5 text-gray-400" />;
    }
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
      case 'running':
        return 'bg-green-100 text-green-800';
      case 'stopped':
      case 'inactive':
        return 'bg-gray-100 text-gray-800';
      case 'starting':
      case 'stopping':
        return 'bg-yellow-100 text-yellow-800';
      case 'error':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-600';
    }
  };

  const formatUptime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    }
    return `${minutes}m`;
  };

  if (connectionStatus !== 'connected') {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-gray-600">Connecting to Cortex...</p>
          <p className="text-sm text-gray-400">Status: {connectionStatus}</p>
        </div>
      </div>
    );
  }

  if (loading && agents.length === 0) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin mx-auto mb-4" />
          <p className="text-gray-600">Loading agents...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Agent Management</h1>
          <p className="text-gray-600 mt-1">
            Manage and monitor AI agents in real-time
          </p>
        </div>
        
        <div className="flex items-center space-x-4">
          <span className="text-sm text-gray-500">
            {agents.length} agent{agents.length !== 1 ? 's' : ''}
          </span>
          
          <button
            onClick={loadAgents}
            disabled={loading}
            className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50 flex items-center"
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>
      </div>

      {/* Error Display */}
      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4">
          <div className="flex items-center">
            <AlertCircle className="h-5 w-5 text-red-500 mr-2" />
            <span className="font-medium text-red-800">Error</span>
          </div>
          <p className="text-red-700 mt-1">{error}</p>
          <button
            onClick={() => setError(null)}
            className="mt-2 text-red-600 hover:text-red-800 text-sm underline"
          >
            Dismiss
          </button>
        </div>
      )}

      {/* Agent Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <Users className="h-8 w-8 text-blue-600" />
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-600">Total Agents</p>
              <p className="text-2xl font-bold text-gray-900">{agents.length}</p>
            </div>
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <CheckCircle className="h-8 w-8 text-green-600" />
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-600">Active</p>
              <p className="text-2xl font-bold text-gray-900">
                {agents.filter(a => a.status.toLowerCase() === 'active').length}
              </p>
            </div>
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <Activity className="h-8 w-8 text-purple-600" />
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-600">Processing</p>
              <p className="text-2xl font-bold text-gray-900">
                {Object.values(agentMetrics).reduce((sum, m) => sum + m.queries_processed, 0)}
              </p>
            </div>
          </div>
        </div>
        
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center">
            <BarChart3 className="h-8 w-8 text-orange-600" />
            <div className="ml-3">
              <p className="text-sm font-medium text-gray-600">Avg Success Rate</p>
              <p className="text-2xl font-bold text-gray-900">
                {Object.values(agentMetrics).length > 0 
                  ? Math.round(Object.values(agentMetrics).reduce((sum, m) => sum + m.success_rate, 0) / Object.values(agentMetrics).length)
                  : 0}%
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Agents List */}
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b border-gray-200">
          <h2 className="text-lg font-medium text-gray-900">Agents</h2>
        </div>
        
        {agents.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            <Users className="h-12 w-12 mx-auto mb-4 text-gray-300" />
            <p>No agents found</p>
            <p className="text-sm mt-1">Agents will appear here when they are registered</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200">
            {agents.map((agent) => {
              const metrics = agentMetrics[agent.id];
              const isSelected = selectedAgent === agent.id;
              
              return (
                <div
                  key={agent.id}
                  className={`p-6 cursor-pointer transition-colors ${
                    isSelected ? 'bg-blue-50' : 'hover:bg-gray-50'
                  }`}
                  onClick={() => setSelectedAgent(isSelected ? null : agent.id)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center space-x-4">
                      {getStatusIcon(agent.status)}
                      
                      <div>
                        <h3 className="font-medium text-gray-900">{agent.name}</h3>
                        <p className="text-sm text-gray-600">{agent.type}</p>
                        <p className="text-xs text-gray-500">ID: {agent.id.substring(0, 16)}...</p>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-4">
                      {/* Status Badge */}
                      <span className={`px-3 py-1 rounded-full text-xs font-medium ${getStatusColor(agent.status)}`}>
                        {agent.status}
                      </span>
                      
                      {/* Metrics Preview */}
                      {metrics && (
                        <div className="text-right">
                          <p className="text-sm font-medium text-gray-900">
                            {metrics.queries_processed} queries
                          </p>
                          <p className="text-xs text-gray-500">
                            {(metrics.success_rate * 100).toFixed(1)}% success
                          </p>
                        </div>
                      )}
                      
                      {/* Action Buttons */}
                      <div className="flex items-center space-x-2">
                        {agent.status.toLowerCase() === 'active' ? (
                          <>
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleAgentAction(agent.id, 'restart');
                              }}
                              disabled={actionLoading[`${agent.id}-restart`]}
                              className="p-2 text-yellow-600 hover:bg-yellow-100 rounded-lg transition-colors disabled:opacity-50"
                              title="Restart Agent"
                            >
                              {actionLoading[`${agent.id}-restart`] ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                              ) : (
                                <RotateCcw className="h-4 w-4" />
                              )}
                            </button>
                            
                            <button
                              onClick={(e) => {
                                e.stopPropagation();
                                handleAgentAction(agent.id, 'stop');
                              }}
                              disabled={actionLoading[`${agent.id}-stop`]}
                              className="p-2 text-red-600 hover:bg-red-100 rounded-lg transition-colors disabled:opacity-50"
                              title="Stop Agent"
                            >
                              {actionLoading[`${agent.id}-stop`] ? (
                                <Loader2 className="h-4 w-4 animate-spin" />
                              ) : (
                                <Square className="h-4 w-4" />
                              )}
                            </button>
                          </>
                        ) : (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              handleAgentAction(agent.id, 'start');
                            }}
                            disabled={actionLoading[`${agent.id}-start`]}
                            className="p-2 text-green-600 hover:bg-green-100 rounded-lg transition-colors disabled:opacity-50"
                            title="Start Agent"
                          >
                            {actionLoading[`${agent.id}-start`] ? (
                              <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                              <Play className="h-4 w-4" />
                            )}
                          </button>
                        )}
                        
                        <button
                          onClick={(e) => e.stopPropagation()}
                          className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
                          title="Configure Agent"
                        >
                          <Settings className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                  </div>
                  
                  {/* Expanded Details */}
                  {isSelected && (
                    <div className="mt-6 pt-6 border-t border-gray-200">
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        {/* Agent Info */}
                        <div>
                          <h4 className="font-medium text-gray-900 mb-3">Agent Information</h4>
                          <div className="space-y-2 text-sm">
                            <div className="flex justify-between">
                              <span className="text-gray-600">Type:</span>
                              <span className="font-medium">{agent.type}</span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-600">Created:</span>
                              <span className="font-medium">
                                {new Date(agent.created_at * 1000).toLocaleDateString()}
                              </span>
                            </div>
                            <div className="flex justify-between">
                              <span className="text-gray-600">Capabilities:</span>
                              <span className="font-medium">{agent.capabilities?.length || 0}</span>
                            </div>
                          </div>
                          
                          {agent.capabilities && agent.capabilities.length > 0 && (
                            <div className="mt-4">
                              <h5 className="text-sm font-medium text-gray-700 mb-2">Capabilities</h5>
                              <div className="flex flex-wrap gap-2">
                                {agent.capabilities.map((capability, index) => (
                                  <span
                                    key={index}
                                    className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded"
                                  >
                                    {capability}
                                  </span>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                        
                        {/* Metrics */}
                        {metrics && (
                          <div>
                            <h4 className="font-medium text-gray-900 mb-3">Performance Metrics</h4>
                            <div className="space-y-3">
                              <div className="bg-gray-50 rounded-lg p-3">
                                <div className="flex justify-between items-center">
                                  <span className="text-sm text-gray-600">Queries Processed</span>
                                  <span className="font-bold text-gray-900">{metrics.queries_processed}</span>
                                </div>
                              </div>
                              
                              <div className="bg-gray-50 rounded-lg p-3">
                                <div className="flex justify-between items-center">
                                  <span className="text-sm text-gray-600">Success Rate</span>
                                  <span className="font-bold text-gray-900">
                                    {(metrics.success_rate * 100).toFixed(1)}%
                                  </span>
                                </div>
                              </div>
                              
                              <div className="bg-gray-50 rounded-lg p-3">
                                <div className="flex justify-between items-center">
                                  <span className="text-sm text-gray-600">Avg Latency</span>
                                  <span className="font-bold text-gray-900">
                                    {metrics.average_latency.toFixed(2)}ms
                                  </span>
                                </div>
                              </div>
                              
                              <div className="bg-gray-50 rounded-lg p-3">
                                <div className="flex justify-between items-center">
                                  <span className="text-sm text-gray-600">Uptime</span>
                                  <span className="font-bold text-gray-900">
                                    {formatUptime(metrics.uptime)}
                                  </span>
                                </div>
                              </div>
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Real-time Events */}
      {realtimeEvents.length > 0 && (
        <div className="bg-white rounded-lg shadow">
          <div className="px-6 py-4 border-b border-gray-200">
            <h2 className="text-lg font-medium text-gray-900">Recent Agent Events</h2>
          </div>
          
          <div className="p-6">
            <div className="space-y-2 max-h-64 overflow-y-auto">
              {realtimeEvents.slice(0, 10).map((event, index) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div>
                    <span className="font-medium text-gray-900">{event.message_type || event.type}</span>
                    <p className="text-sm text-gray-600">
                      Agent: {event.agent_id || event.data?.agent_id || 'Unknown'}
                    </p>
                  </div>
                  <span className="text-xs text-gray-500">
                    {new Date(event.timestamp).toLocaleTimeString()}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}