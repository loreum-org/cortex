import { useState, useEffect } from 'react';
import { Plus, Trash2, Eye, Cpu, Activity, Users, Bot, Zap } from 'lucide-react';
import { cortexWebSocket } from '../services/websocket';

interface AgentCapability {
  name: string;
  description: string;
  parameters?: Record<string, any>;
  version: string;
}

interface AgentMetrics {
  response_time: number;
  success_rate: number;
  error_rate: number;
  requests_per_min: number;
  total_queries: number;
  uptime: number;
}

interface Agent {
  id: string;
  name: string;
  description: string;
  type: 'solver' | 'sensor' | 'analyzer' | 'generator' | 'coordinator';
  status: 'active' | 'inactive' | 'error' | 'initializing';
  capabilities: AgentCapability[];
  metrics: AgentMetrics;
  model_id?: string;
  created_at: string;
  updated_at: string;
  last_query_at?: string;
}

interface CreateAgentRequest {
  name: string;
  description: string;
  type: Agent['type'];
  capabilities: AgentCapability[];
  model_id?: string;
}

export function AgentManagement() {
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadAgents();
    // Set up periodic refresh for metrics
    const interval = setInterval(loadAgents, 30000); // Every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const loadAgents = async () => {
    try {
      console.log('🚀 Starting to load agents...');
      setLoading(true);
      
      // Fetch agents from WebSocket API
      console.log('📤 Requesting agents from WebSocket API...');
      const agentsData = await cortexWebSocket.getAgents();
      console.log('📊 Raw agents response:', agentsData);
      
      // Check if we have agents data
      const agentsResponse = agentsData.agents || [];
      if (!Array.isArray(agentsResponse)) {
        console.warn('⚠️ No agents data in response:', agentsResponse);
        setAgents([]);
        setError(null);
        return;
      }
      
      console.log('📊 Agents array length:', agentsResponse.length);
      
      // Transform the backend response to match our frontend interface
      const transformedAgents: Agent[] = agentsResponse.map((agentData: any, index: number) => {
        console.log(`📊 Processing agent ${index}:`, agentData);
        
        // Handle the nested structure returned by the backend
        const info = agentData.info || agentData;
        const metrics = agentData.metrics || {};
        const status = agentData.status || info.status || 'inactive';
        
        return {
          id: info.id || `agent-${index}`,
          name: info.name || 'Unknown Agent',
          description: info.description || 'No description available',
          type: info.type as Agent['type'] || 'solver',
          status: status as Agent['status'],
          capabilities: info.capabilities || [],
          metrics: {
            response_time: metrics.response_time || 0,
            success_rate: (metrics.success_rate || 0) * 100,
            error_rate: metrics.error_rate || 0,
            requests_per_min: metrics.requests_per_min || 0,
            total_queries: metrics.total_queries || 0,
            uptime: (metrics.uptime || 0) * 100
          },
          model_id: info.model_id || '',
          created_at: info.created_at || new Date().toISOString(),
          updated_at: info.updated_at || new Date().toISOString(),
          last_query_at: metrics.last_query_at || info.updated_at || ''
        };
      });
      
      console.log('📊 Transformed agents:', transformedAgents);
      setAgents(transformedAgents);
      setError(null);
    } catch (err) {
      console.error('Error loading agents:', err);
      setError(err instanceof Error ? err.message : 'Failed to load agents');
      
      // Fallback to empty array on error
      setAgents([]);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteAgent = async (agentId: string) => {
    if (!confirm('Are you sure you want to delete this agent? This action cannot be undone.')) return;
    
    try {
      // TODO: Implement WebSocket agent deletion
      console.log('Delete agent:', agentId);
      setError('Agent deletion not yet implemented via WebSocket');
      
      // For now, just reload agents to refresh the list
      await loadAgents();
    } catch (err) {
      console.error('Error deleting agent:', err);
      setError(err instanceof Error ? err.message : 'Failed to delete agent');
    }
  };

  const getStatusColor = (status: Agent['status']) => {
    switch (status) {
      case 'active': return 'text-green-400 bg-green-400/10 border-green-400/20';
      case 'inactive': return 'text-gray-400 bg-gray-400/10 border-gray-400/20';
      case 'initializing': return 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20';
      case 'error': return 'text-red-400 bg-red-400/10 border-red-400/20';
      default: return 'text-gray-400 bg-gray-400/10 border-gray-400/20';
    }
  };

  const getTypeColor = (type: Agent['type']) => {
    switch (type) {
      case 'solver': return 'text-blue-400 bg-blue-400/10 border-blue-400/20';
      case 'sensor': return 'text-green-400 bg-green-400/10 border-green-400/20';
      case 'analyzer': return 'text-purple-400 bg-purple-400/10 border-purple-400/20';
      case 'generator': return 'text-orange-400 bg-orange-400/10 border-orange-400/20';
      case 'coordinator': return 'text-cyan-400 bg-cyan-400/10 border-cyan-400/20';
      default: return 'text-gray-400 bg-gray-400/10 border-gray-400/20';
    }
  };

  const getTypeIcon = (type: Agent['type']) => {
    switch (type) {
      case 'solver': return Bot;
      case 'sensor': return Activity;
      case 'analyzer': return Cpu;
      case 'generator': return Zap;
      case 'coordinator': return Users;
      default: return Bot;
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-tesla-black text-tesla-white p-6">
        <div className="flex items-center justify-center h-64">
          <div className="flex items-center gap-3">
            <Bot className="w-8 h-8 animate-pulse text-tesla-white" />
            <span className="text-xl uppercase tracking-wider">Loading Agents...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-tesla-black text-tesla-white p-6">
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <div className="flex items-center gap-3">
            <Bot className="h-8 w-8 text-tesla-white" />
            <h1 className="text-3xl font-bold uppercase tracking-wider">Agent Management</h1>
          </div>
          <button
            onClick={() => setShowCreateForm(true)}
            className="tesla-button flex items-center gap-2 text-xs uppercase tracking-wider"
          >
            <Plus className="h-4 w-4" />
            Create Agent
          </button>
        </div>

        {error && (
          <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 mb-6">
            {error}
          </div>
        )}

        {/* System Overview */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-tesla-dark-gray border border-tesla-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-tesla-text-gray uppercase tracking-wider text-xs">Total Agents</h3>
              <Bot className="h-5 w-5 text-tesla-white" />
            </div>
            <div className="text-3xl font-bold text-tesla-white mb-2">{agents.length}</div>
            <p className="text-xs text-tesla-text-gray uppercase tracking-wider">
              {agents.filter(a => a.status === 'active').length} Active
            </p>
          </div>

          <div className="bg-tesla-dark-gray border border-tesla-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-tesla-text-gray uppercase tracking-wider text-xs">Avg Response Time</h3>
              <Activity className="h-5 w-5 text-tesla-white" />
            </div>
            <div className="text-3xl font-bold text-tesla-white mb-2">
              {agents.length > 0 ? (agents.reduce((sum, agent) => sum + agent.metrics.response_time, 0) / agents.length).toFixed(1) : '0.0'}s
            </div>
            <p className="text-xs text-tesla-text-gray uppercase tracking-wider">System Average</p>
          </div>

          <div className="bg-tesla-dark-gray border border-tesla-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-tesla-text-gray uppercase tracking-wider text-xs">Success Rate</h3>
              <Zap className="h-5 w-5 text-tesla-white" />
            </div>
            <div className="text-3xl font-bold text-tesla-white mb-2">
              {agents.length > 0 ? (agents.reduce((sum, agent) => sum + agent.metrics.success_rate, 0) / agents.length).toFixed(1) : '0.0'}%
            </div>
            <p className="text-xs text-tesla-text-gray uppercase tracking-wider">Overall Performance</p>
          </div>

          <div className="bg-tesla-dark-gray border border-tesla-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="font-semibold text-tesla-text-gray uppercase tracking-wider text-xs">Total Queries</h3>
              <Cpu className="h-5 w-5 text-tesla-white" />
            </div>
            <div className="text-3xl font-bold text-tesla-white mb-2">
              {agents.reduce((sum, agent) => sum + agent.metrics.total_queries, 0).toLocaleString()}
            </div>
            <p className="text-xs text-tesla-text-gray uppercase tracking-wider">Lifetime Processed</p>
          </div>
        </div>

        {/* Agents Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {agents.map((agent) => {
            const TypeIcon = getTypeIcon(agent.type);
            return (
              <div
                key={agent.id}
                className="bg-tesla-dark-gray border border-tesla-border p-6 hover:border-tesla-light-gray transition-all duration-150"
              >
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-start gap-3">
                    <div className="bg-tesla-medium-gray border border-tesla-border p-2">
                      <TypeIcon className="h-5 w-5 text-tesla-white" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold mb-1 uppercase tracking-wider">{agent.name}</h3>
                      <div className="flex items-center gap-2 mb-2">
                        <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider border ${getTypeColor(agent.type)}`}>
                          {agent.type}
                        </span>
                        <span className={`px-2 py-1 text-xs font-medium uppercase tracking-wider border ${getStatusColor(agent.status)}`}>
                          {agent.status}
                        </span>
                      </div>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => setSelectedAgent(agent)}
                      className="p-2 text-tesla-text-gray hover:text-tesla-white hover:bg-tesla-medium-gray transition-all duration-150"
                      title="View Details"
                    >
                      <Eye className="h-4 w-4" />
                    </button>
                    <button
                      onClick={() => handleDeleteAgent(agent.id)}
                      className="p-2 text-tesla-text-gray hover:text-red-400 hover:bg-red-400/10 transition-all duration-150"
                      title="Delete Agent"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                <p className="text-tesla-text-gray text-sm mb-4 line-clamp-2">
                  {agent.description}
                </p>

                {/* Metrics */}
                <div className="space-y-3">
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Success Rate</span>
                      <div className="text-green-400 font-medium">{agent.metrics.success_rate.toFixed(1)}%</div>
                    </div>
                    <div>
                      <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Response Time</span>
                      <div className="text-blue-400 font-medium">{agent.metrics.response_time.toFixed(1)}s</div>
                    </div>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4 text-sm">
                    <div>
                      <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Capabilities</span>
                      <div className="text-tesla-white font-medium">{agent.capabilities.length}</div>
                    </div>
                    <div>
                      <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Total Queries</span>
                      <div className="text-purple-400 font-medium">{agent.metrics.total_queries.toLocaleString()}</div>
                    </div>
                  </div>

                  {agent.last_query_at && (
                    <div className="text-xs">
                      <span className="text-tesla-text-gray uppercase tracking-wider">Last Query: </span>
                      <span className="text-tesla-white">
                        {new Date(agent.last_query_at).toLocaleString()}
                      </span>
                    </div>
                  )}
                  
                  {/* Agent Control Buttons */}
                  <div className="flex gap-2 mt-4 pt-4 border-t border-tesla-border">
                    <button
                      onClick={() => startAgent(agent.id)}
                      disabled={agent.status === 'active' || isOperationInProgress(`starting_${agent.id}`)}
                      className="tesla-button-sm flex items-center gap-1 text-xs uppercase tracking-wider"
                    >
                      {isOperationInProgress(`starting_${agent.id}`) ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <Play className="h-3 w-3" />
                      )}
                      START
                    </button>
                    <button
                      onClick={() => stopAgent(agent.id)}
                      disabled={agent.status === 'inactive' || isOperationInProgress(`stopping_${agent.id}`)}
                      className="tesla-button-sm flex items-center gap-1 text-xs uppercase tracking-wider"
                    >
                      {isOperationInProgress(`stopping_${agent.id}`) ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <Square className="h-3 w-3" />
                      )}
                      STOP
                    </button>
                    <button
                      onClick={() => restartAgent(agent.id)}
                      disabled={isOperationInProgress(`restarting_${agent.id}`)}
                      className="tesla-button-sm flex items-center gap-1 text-xs uppercase tracking-wider"
                    >
                      {isOperationInProgress(`restarting_${agent.id}`) ? (
                        <Loader2 className="h-3 w-3 animate-spin" />
                      ) : (
                        <RotateCcw className="h-3 w-3" />
                      )}
                      RESTART
                    </button>
                    <button
                      className="tesla-button-sm flex items-center gap-1 text-xs uppercase tracking-wider"
                      title="Configuration management coming soon"
                      disabled
                    >
                      <Settings className="h-3 w-3" />
                      CONFIG
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        {agents.length === 0 && !loading && (
          <div className="text-center py-16">
            <Bot className="h-16 w-16 text-tesla-text-gray mx-auto mb-4" />
            <h3 className="text-xl font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">No Agents Available</h3>
            <p className="text-tesla-text-gray mb-6 uppercase tracking-wider text-sm">Create your first agent to start processing queries.</p>
            <button
              onClick={() => setShowCreateForm(true)}
              className="tesla-button text-xs uppercase tracking-wider"
            >
              Create Agent
            </button>
          </div>
        )}
      </div>

      {/* Agent Details Modal */}
      {selectedAgent && (
        <AgentDetailsModal 
          agent={selectedAgent} 
          onClose={() => setSelectedAgent(null)} 
        />
      )}

      {/* Create Agent Modal */}
      {showCreateForm && (
        <CreateAgentModal 
          onClose={() => setShowCreateForm(false)}
          onSuccess={() => {
            setShowCreateForm(false);
            loadAgents();
          }}
        />
      )}
    </div>
  );
}

interface AgentDetailsModalProps {
  agent: Agent;
  onClose: () => void;
}

function AgentDetailsModal({ agent, onClose }: AgentDetailsModalProps) {
  const [codeQuality, setCodeQuality] = useState<any>(null);
  const [loadingCodeQuality, setLoadingCodeQuality] = useState(false);
  const [analysisError, setAnalysisError] = useState<string | null>(null);

  const TypeIcon = agent.type === 'solver' ? Bot : 
                   agent.type === 'sensor' ? Activity : 
                   agent.type === 'analyzer' ? Cpu : 
                   agent.type === 'generator' ? Zap : Users;

  // Check if this is the Code Reflection Agent
  const isCodeReflectionAgent = agent.id === 'code-reflection-agent' || 
                               agent.name.toLowerCase().includes('code reflection');

  useEffect(() => {
    if (isCodeReflectionAgent) {
      loadCodeQualityData();
    }
  }, [isCodeReflectionAgent]);

  const loadCodeQualityData = async () => {
    try {
      setLoadingCodeQuality(true);
      setAnalysisError(null);
      
      // Send code quality request via WebSocket
      const requestId = `code_quality_${Date.now()}`;
      const response = await new Promise((resolve, reject) => {
        const unsubscribe = cortexWebSocket.subscribe('response', (message: any) => {
          if (message.id === requestId) {
            unsubscribe();
            if (message.error) {
              reject(new Error(message.error));
            } else {
              resolve(message.data);
            }
          }
        });

        cortexWebSocket.sendMessage({
          type: 'request',
          method: 'getCodeQuality',
          id: requestId,
          data: { agent_id: agent.id }
        });

        setTimeout(() => {
          unsubscribe();
          reject(new Error('Request timeout'));
        }, 10000);
      });
      
      setCodeQuality(response as any);
    } catch (err) {
      console.error('Error loading code quality data:', err);
      setAnalysisError(err instanceof Error ? err.message : 'Failed to load code quality data');
      
      // Fallback for now
      setCodeQuality({ 
        issues: { agent_response: 'Code quality analysis data via WebSocket - feature in development' },
        metrics: { agent_response: 'Code quality metrics via WebSocket - feature in development' }
      });
    } finally {
      setLoadingCodeQuality(false);
    }
  };

  const triggerAnalysis = async () => {
    try {
      setLoadingCodeQuality(true);
      setAnalysisError(null);
      
      // Trigger code analysis via WebSocket
      const requestId = `trigger_analysis_${Date.now()}`;
      await new Promise((resolve, reject) => {
        const unsubscribe = cortexWebSocket.subscribe('response', (message: any) => {
          if (message.id === requestId) {
            unsubscribe();
            if (message.error) {
              reject(new Error(message.error));
            } else {
              resolve(message.data);
            }
          }
        });

        cortexWebSocket.sendMessage({
          type: 'request',
          method: 'triggerCodeAnalysis',
          id: requestId,
          data: { 
            agent_id: agent.id,
            analysis_type: 'full',
            target_path: '.'
          }
        });

        setTimeout(() => {
          unsubscribe();
          reject(new Error('Request timeout'));
        }, 10000);
      });
      
      // Wait a moment then reload data
      setTimeout(() => {
        loadCodeQualityData();
      }, 2000);
      
    } catch (err) {
      console.error('Error triggering analysis:', err);
      setAnalysisError(err instanceof Error ? err.message : 'Failed to trigger analysis');
      setLoadingCodeQuality(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50">
      <div className="bg-tesla-dark-gray border border-tesla-border max-w-4xl w-full max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <div className="flex items-center gap-3">
              <div className="bg-tesla-medium-gray border border-tesla-border p-3">
                <TypeIcon className="h-6 w-6 text-tesla-white" />
              </div>
              <h2 className="text-2xl font-bold uppercase tracking-wider">{agent.name}</h2>
            </div>
            <button
              onClick={onClose}
              className="text-tesla-text-gray hover:text-tesla-white transition-all duration-150 text-xl"
            >
              ✕
            </button>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* Main Info */}
            <div className="lg:col-span-2 space-y-6">
              <div>
                <h3 className="text-lg font-semibold mb-3 uppercase tracking-wider">Description</h3>
                <p className="text-tesla-text-gray">{agent.description}</p>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <h4 className="font-medium mb-1 uppercase tracking-wider text-sm">Type</h4>
                  <span className={`px-3 py-1 text-xs font-medium uppercase tracking-wider border ${
                    agent.type === 'solver' ? 'text-blue-400 bg-blue-400/10 border-blue-400/20' :
                    agent.type === 'sensor' ? 'text-green-400 bg-green-400/10 border-green-400/20' :
                    agent.type === 'analyzer' ? 'text-purple-400 bg-purple-400/10 border-purple-400/20' :
                    agent.type === 'generator' ? 'text-orange-400 bg-orange-400/10 border-orange-400/20' :
                    'text-cyan-400 bg-cyan-400/10 border-cyan-400/20'
                  }`}>
                    {agent.type}
                  </span>
                </div>
                <div>
                  <h4 className="font-medium mb-1 uppercase tracking-wider text-sm">Status</h4>
                  <span className={`px-3 py-1 text-xs font-medium uppercase tracking-wider border ${
                    agent.status === 'active' ? 'text-green-400 bg-green-400/10 border-green-400/20' :
                    agent.status === 'inactive' ? 'text-gray-400 bg-gray-400/10 border-gray-400/20' :
                    agent.status === 'initializing' ? 'text-yellow-400 bg-yellow-400/10 border-yellow-400/20' :
                    'text-red-400 bg-red-400/10 border-red-400/20'
                  }`}>
                    {agent.status}
                  </span>
                </div>
              </div>

              {agent.model_id && (
                <div>
                  <h4 className="font-medium mb-1 uppercase tracking-wider text-sm">Model ID</h4>
                  <code className="bg-tesla-medium-gray px-2 py-1 text-sm">{agent.model_id}</code>
                </div>
              )}

              <div>
                <h3 className="text-lg font-semibold mb-3 uppercase tracking-wider">Capabilities</h3>
                <div className="space-y-3">
                  {agent.capabilities.map((capability, index) => (
                    <div key={index} className="bg-tesla-medium-gray border border-tesla-border p-4">
                      <div className="flex items-center justify-between mb-2">
                        <h4 className="font-medium uppercase tracking-wider">{capability.name}</h4>
                        <span className="text-xs text-tesla-text-gray">v{capability.version}</span>
                      </div>
                      <p className="text-sm text-tesla-text-gray mb-3">{capability.description}</p>
                      {capability.parameters && Object.keys(capability.parameters).length > 0 && (
                        <div className="bg-tesla-black p-3">
                          <h5 className="text-xs uppercase tracking-wider text-tesla-text-gray mb-2">Parameters</h5>
                          <div className="space-y-1">
                            {Object.entries(capability.parameters || {}).map(([key, value]) => (
                              <div key={key} className="flex justify-between text-sm">
                                <span className="text-tesla-text-gray">{key}:</span>
                                <span className="text-tesla-white">{JSON.stringify(value)}</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>

              {/* Code Quality Results - Only show for Code Reflection Agent */}
              {isCodeReflectionAgent && (
                <div>
                  <div className="flex items-center justify-between mb-3">
                    <h3 className="text-lg font-semibold uppercase tracking-wider">Code Analysis Results</h3>
                    <button
                      onClick={triggerAnalysis}
                      disabled={loadingCodeQuality}
                      className="tesla-button text-xs uppercase tracking-wider disabled:opacity-50"
                    >
                      {loadingCodeQuality ? 'Analyzing...' : 'Run Analysis'}
                    </button>
                  </div>

                  {analysisError && (
                    <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-3 mb-4 text-sm">
                      {analysisError}
                    </div>
                  )}

                  {loadingCodeQuality && (
                    <div className="bg-tesla-medium-gray border border-tesla-border p-4 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <div className="animate-spin h-4 w-4 border-2 border-tesla-white border-t-transparent rounded-full"></div>
                        <span className="text-sm text-tesla-text-gray uppercase tracking-wider">Loading Analysis...</span>
                      </div>
                    </div>
                  )}

                  {codeQuality && !loadingCodeQuality && (
                    <div className="space-y-4">
                      {/* Code Issues */}
                      <div className="bg-tesla-medium-gray border border-tesla-border p-4">
                        <h4 className="font-medium mb-3 uppercase tracking-wider text-sm">Code Issues</h4>
                        <div className="bg-tesla-black p-3">
                          <pre className="text-sm text-tesla-white whitespace-pre-wrap font-mono">
                            {codeQuality.issues?.agent_response || 'No issues found'}
                          </pre>
                        </div>
                        {codeQuality.issues?.retrieved_at && (
                          <div className="mt-2 text-xs text-tesla-text-gray">
                            Last updated: {new Date(codeQuality.issues.retrieved_at).toLocaleString()}
                          </div>
                        )}
                      </div>

                      {/* Code Metrics */}
                      <div className="bg-tesla-medium-gray border border-tesla-border p-4">
                        <h4 className="font-medium mb-3 uppercase tracking-wider text-sm">Quality Metrics</h4>
                        <div className="bg-tesla-black p-3">
                          <pre className="text-sm text-tesla-white whitespace-pre-wrap font-mono">
                            {codeQuality.metrics?.agent_response || 'No metrics available'}
                          </pre>
                        </div>
                        {codeQuality.metrics?.retrieved_at && (
                          <div className="mt-2 text-xs text-tesla-text-gray">
                            Last updated: {new Date(codeQuality.metrics.retrieved_at).toLocaleString()}
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>

            {/* Metrics Sidebar */}
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-semibold mb-3 uppercase tracking-wider">Performance Metrics</h3>
                <div className="bg-tesla-medium-gray border border-tesla-border p-4 space-y-4">
                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-tesla-text-gray text-sm uppercase tracking-wider">Success Rate</span>
                      <span className="text-green-400 font-medium">{agent.metrics.success_rate.toFixed(1)}%</span>
                    </div>
                    <div className="w-full bg-tesla-black h-2">
                      <div 
                        className="bg-green-400 h-2"
                        style={{ width: `${agent.metrics.success_rate}%` }}
                      />
                    </div>
                  </div>

                  <div>
                    <div className="flex justify-between items-center mb-1">
                      <span className="text-tesla-text-gray text-sm uppercase tracking-wider">Uptime</span>
                      <span className="text-blue-400 font-medium">{agent.metrics.uptime.toFixed(1)}%</span>
                    </div>
                    <div className="w-full bg-tesla-black h-2">
                      <div 
                        className="bg-blue-400 h-2"
                        style={{ width: `${agent.metrics.uptime}%` }}
                      />
                    </div>
                  </div>

                  <div className="grid grid-cols-1 gap-3 pt-2">
                    <div className="text-center">
                      <div className="text-2xl font-bold text-tesla-white">{agent.metrics.response_time.toFixed(1)}s</div>
                      <div className="text-xs text-tesla-text-gray uppercase tracking-wider">Avg Response</div>
                    </div>
                    <div className="text-center">
                      <div className="text-2xl font-bold text-tesla-white">{agent.metrics.requests_per_min}</div>
                      <div className="text-xs text-tesla-text-gray uppercase tracking-wider">Req/Min</div>
                    </div>
                    <div className="text-center">
                      <div className="text-2xl font-bold text-tesla-white">{agent.metrics.total_queries.toLocaleString()}</div>
                      <div className="text-xs text-tesla-text-gray uppercase tracking-wider">Total Queries</div>
                    </div>
                  </div>
                </div>
              </div>

              <div>
                <h3 className="text-lg font-semibold mb-3 uppercase tracking-wider">Timestamps</h3>
                <div className="space-y-3 text-sm">
                  <div>
                    <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Created:</span>
                    <div className="text-tesla-white">{new Date(agent.created_at).toLocaleString()}</div>
                  </div>
                  <div>
                    <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Last Updated:</span>
                    <div className="text-tesla-white">{new Date(agent.updated_at).toLocaleString()}</div>
                  </div>
                  {agent.last_query_at && (
                    <div>
                      <span className="text-tesla-text-gray uppercase tracking-wider text-xs">Last Query:</span>
                      <div className="text-tesla-white">{new Date(agent.last_query_at).toLocaleString()}</div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface CreateAgentModalProps {
  onClose: () => void;
  onSuccess: () => void;
}

function CreateAgentModal({ onClose, onSuccess }: CreateAgentModalProps) {
  const [formData, setFormData] = useState<CreateAgentRequest>({
    name: '',
    description: '',
    type: 'solver',
    capabilities: [],
    model_id: ''
  });
  const [newCapability, setNewCapability] = useState<AgentCapability>({
    name: '',
    description: '',
    parameters: {},
    version: '1.0.0'
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name || !formData.description) {
      setError('Name and description are required');
      return;
    }

    try {
      setLoading(true);
      setError(null);
      // TODO: Implement actual API call
      await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate API call
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create agent');
    } finally {
      setLoading(false);
    }
  };

  const addCapability = () => {
    if (!newCapability.name || !newCapability.description) return;
    setFormData(prev => ({
      ...prev,
      capabilities: [...prev.capabilities, newCapability]
    }));
    setNewCapability({
      name: '',
      description: '',
      parameters: {},
      version: '1.0.0'
    });
  };

  const removeCapability = (index: number) => {
    setFormData(prev => ({
      ...prev,
      capabilities: prev.capabilities.filter((_, i) => i !== index)
    }));
  };

  return (
    <div className="fixed inset-0 bg-black/80 flex items-center justify-center p-4 z-50">
      <div className="bg-tesla-dark-gray border border-tesla-border max-w-2xl w-full max-h-[80vh] overflow-y-auto">
        <div className="p-6">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-2xl font-bold uppercase tracking-wider">Create New Agent</h2>
            <button
              onClick={onClose}
              className="text-tesla-text-gray hover:text-tesla-white transition-all duration-150 text-xl"
            >
              ✕
            </button>
          </div>

          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-4 mb-6">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2 uppercase tracking-wider">Agent Type</label>
                <select
                  value={formData.type}
                  onChange={(e) => setFormData(prev => ({ ...prev, type: e.target.value as Agent['type'] }))}
                  className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                >
                  <option value="solver">Solver</option>
                  <option value="sensor">Sensor</option>
                  <option value="analyzer">Analyzer</option>
                  <option value="generator">Generator</option>
                  <option value="coordinator">Coordinator</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium mb-2 uppercase tracking-wider">Model ID (Optional)</label>
                <input
                  type="text"
                  value={formData.model_id}
                  onChange={(e) => setFormData(prev => ({ ...prev, model_id: e.target.value }))}
                  className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                  placeholder="ollama-cogito"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2 uppercase tracking-wider">Agent Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                placeholder="Enter agent name"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-2 uppercase tracking-wider">Description</label>
              <textarea
                value={formData.description}
                onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
                className="w-full bg-tesla-medium-gray border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                rows={3}
                placeholder="Describe the agent's purpose and functionality"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-3 uppercase tracking-wider">Capabilities</label>
              
              {/* Add Capability Form */}
              <div className="bg-tesla-medium-gray border border-tesla-border p-4 mb-4 space-y-3">
                <div className="grid grid-cols-2 gap-3">
                  <input
                    type="text"
                    value={newCapability.name}
                    onChange={(e) => setNewCapability(prev => ({ ...prev, name: e.target.value }))}
                    className="bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                    placeholder="Capability name"
                  />
                  <input
                    type="text"
                    value={newCapability.version}
                    onChange={(e) => setNewCapability(prev => ({ ...prev, version: e.target.value }))}
                    className="bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                    placeholder="Version"
                  />
                </div>
                <textarea
                  value={newCapability.description}
                  onChange={(e) => setNewCapability(prev => ({ ...prev, description: e.target.value }))}
                  className="w-full bg-tesla-black border border-tesla-border text-tesla-white px-3 py-2 focus:outline-none focus:border-tesla-white transition-all duration-150"
                  rows={2}
                  placeholder="Capability description"
                />
                <button
                  type="button"
                  onClick={addCapability}
                  className="tesla-button text-xs uppercase tracking-wider"
                >
                  Add Capability
                </button>
              </div>

              {/* Capabilities List */}
              <div className="space-y-2">
                {formData.capabilities.map((capability, index) => (
                  <div key={index} className="bg-tesla-black border border-tesla-border p-3 flex items-center justify-between">
                    <div>
                      <div className="font-medium uppercase tracking-wider">{capability.name} v{capability.version}</div>
                      <div className="text-sm text-tesla-text-gray">{capability.description}</div>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeCapability(index)}
                      className="text-red-400 hover:text-red-300 transition-all duration-150"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex gap-4 pt-4">
              <button
                type="button"
                onClick={onClose}
                className="flex-1 bg-tesla-medium-gray text-tesla-white px-4 py-3 font-medium hover:bg-tesla-black transition-all duration-150 border border-tesla-border text-xs uppercase tracking-wider"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex-1 tesla-button disabled:opacity-50 text-xs uppercase tracking-wider"
              >
                {loading ? 'Creating...' : 'Create Agent'}
              </button>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}