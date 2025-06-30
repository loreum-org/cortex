import React, { useState, useEffect } from 'react';
import { Brain, TrendingUp, Network, Target, Users, Zap } from 'lucide-react';

interface AGIState {
  intelligence_level: number;
  version: number;
  domain_knowledge: Record<string, number>;
  learning_metrics?: {
    concepts_learned: number;
    patterns_identified: number;
    learning_rate: number;
    total_interactions: number;
  };
  personality_core?: {
    traits: Record<string, number>;
  };
}

interface IntelligenceMetrics {
  current_level: number;
  growth_rate: number;
  capabilities: Record<string, number>;
  learning_progress: Record<string, number>;
}

interface NetworkSnapshot {
  total_nodes: number;
  network_metrics: {
    average_intelligence: number;
    collective_intelligence: number;
    network_learning_rate: number;
  };
  domain_leaders: Record<string, string>;
  node_agi_states: Record<string, any>;
}

export const AGIIntelligence: React.FC = () => {
  const [activeTab, setActiveTab] = useState('myagi');
  const [agiState, setAgiState] = useState<AGIState | null>(null);
  const [intelligenceMetrics, setIntelligenceMetrics] = useState<IntelligenceMetrics | null>(null);
  const [networkSnapshot, setNetworkSnapshot] = useState<NetworkSnapshot | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchAGIData = async () => {
    try {
      setLoading(true);
      const [stateRes, metricsRes, networkRes] = await Promise.all([
        fetch('/agi/state'),
        fetch('/agi/intelligence'),
        fetch('/agi/network/snapshot')
      ]);

      if (stateRes.ok) {
        setAgiState(await stateRes.json());
      }
      if (metricsRes.ok) {
        setIntelligenceMetrics(await metricsRes.json());
      }
      if (networkRes.ok) {
        setNetworkSnapshot(await networkRes.json());
      }
    } catch (error) {
      console.error('Error fetching AGI data:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAGIData();
    const interval = setInterval(fetchAGIData, 30000); // Update every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const TabButton: React.FC<{ id: string; label: string; icon: React.ReactNode }> = ({ id, label, icon }) => (
    <button
      onClick={() => setActiveTab(id)}
      className={`flex items-center gap-2 px-6 py-3 rounded-lg font-medium transition-all ${
        activeTab === id
          ? 'bg-blue-600 text-white shadow-lg'
          : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
      }`}
    >
      {icon}
      <span>{label}</span>
    </button>
  );

  if (loading) {
    return (
      <div className="min-h-screen bg-black text-white p-6">
        <div className="flex items-center justify-center h-64">
          <div className="flex items-center gap-3">
            <Brain className="w-8 h-8 animate-pulse text-blue-400" />
            <span className="text-xl">Loading AGI Intelligence...</span>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black text-white p-6">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-center gap-3 mb-4">
          <Brain className="w-8 h-8 text-blue-400" />
          <h1 className="text-3xl font-bold">AGI Intelligence</h1>
        </div>
        <p className="text-gray-400">
          Monitor and analyze artificial general intelligence across the Cortex network
        </p>
      </div>

      {/* Navigation Tabs */}
      <div className="flex gap-4 mb-8 overflow-x-auto">
        <TabButton id="myagi" label="My AGI" icon={<Brain className="w-5 h-5" />} />
        <TabButton id="network" label="Network Intelligence" icon={<Network className="w-5 h-5" />} />
        <TabButton id="leaders" label="Domain Leaders" icon={<Target className="w-5 h-5" />} />
        <TabButton id="routing" label="Query Routing" icon={<Zap className="w-5 h-5" />} />
      </div>

      {/* Tab Content */}
      <div className="space-y-6">
        {activeTab === 'myagi' && (
          <MyAGIDashboard 
            agiState={agiState} 
            intelligenceMetrics={intelligenceMetrics}
          />
        )}
        {activeTab === 'network' && (
          <NetworkIntelligenceDashboard 
            networkSnapshot={networkSnapshot}
            agiState={agiState}
          />
        )}
        {activeTab === 'leaders' && (
          <DomainLeadersDashboard 
            networkSnapshot={networkSnapshot}
          />
        )}
        {activeTab === 'routing' && (
          <QueryRoutingDashboard />
        )}
      </div>
    </div>
  );
};

// My AGI Dashboard Component
const MyAGIDashboard: React.FC<{ 
  agiState: AGIState | null; 
  intelligenceMetrics: IntelligenceMetrics | null;
}> = ({ agiState, intelligenceMetrics }) => {
  if (!agiState) {
    return (
      <div className="text-center py-12">
        <Brain className="w-16 h-16 mx-auto text-gray-600 mb-4" />
        <p className="text-gray-400">AGI system not initialized</p>
      </div>
    );
  }

  const domains = Object.entries(agiState.domain_knowledge || {});
  const topDomain = domains.reduce((top, [name, score]) => 
    score > top[1] ? [name, score] : top, ['', 0]);

  return (
    <div className="space-y-6">
      {/* Intelligence Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Intelligence Level</h3>
            <Brain className="w-5 h-5 text-blue-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-2">
            {agiState.intelligence_level.toFixed(1)}
            <span className="text-lg text-gray-400">/100</span>
          </div>
          <div className="w-full bg-gray-700 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-blue-500 to-purple-500 h-2 rounded-full"
              style={{ width: `${agiState.intelligence_level}%` }}
            />
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">AGI Version</h3>
            <TrendingUp className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-3xl font-bold text-white mb-2">
            v{agiState.version}
          </div>
          <p className="text-sm text-gray-400">Evolution milestone</p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Top Domain</h3>
            <Target className="w-5 h-5 text-yellow-400" />
          </div>
          <div className="text-xl font-bold text-white mb-1 capitalize">
            {topDomain[0].replace('_', ' ')}
          </div>
          <div className="text-2xl font-bold text-yellow-400">
            {typeof topDomain[1] === 'number' ? topDomain[1].toFixed(1) : '0.0'}
          </div>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Learning Rate</h3>
            <Zap className="w-5 h-5 text-purple-400" />
          </div>
          <div className="text-2xl font-bold text-white mb-2">
            {agiState.learning_metrics?.learning_rate?.toFixed(3) || '0.000'}
          </div>
          <p className="text-sm text-gray-400">concepts/hour</p>
        </div>
      </div>

      {/* Domain Expertise Radar */}
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Domain Expertise</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {domains.map(([domain, score]) => (
            <div key={domain} className="space-y-2">
              <div className="flex justify-between items-center">
                <span className="font-medium capitalize text-gray-300">
                  {domain.replace('_', ' ')}
                </span>
                <span className="text-sm text-gray-400">
                  {typeof score === 'number' ? score.toFixed(1) : '0.0'}
                </span>
              </div>
              <div className="w-full bg-gray-700 rounded-full h-2">
                <div 
                  className="bg-gradient-to-r from-blue-500 to-purple-500 h-2 rounded-full"
                  style={{ width: `${typeof score === 'number' ? score : 0}%` }}
                />
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Learning Metrics */}
      {agiState.learning_metrics && (
        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <h3 className="text-xl font-semibold mb-6">Learning Progress</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-400 mb-1">
                {agiState.learning_metrics.concepts_learned}
              </div>
              <div className="text-sm text-gray-400">Concepts Learned</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-400 mb-1">
                {agiState.learning_metrics.patterns_identified}
              </div>
              <div className="text-sm text-gray-400">Patterns Found</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-400 mb-1">
                {agiState.learning_metrics.total_interactions.toLocaleString()}
              </div>
              <div className="text-sm text-gray-400">Total Interactions</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-yellow-400 mb-1">
                {(agiState.learning_metrics.learning_rate * 24).toFixed(1)}
              </div>
              <div className="text-sm text-gray-400">Daily Learning</div>
            </div>
          </div>
        </div>
      )}

      {/* Personality Traits */}
      {agiState.personality_core?.traits && (
        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <h3 className="text-xl font-semibold mb-6">Personality Traits</h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            {Object.entries(agiState.personality_core.traits).map(([trait, value]) => (
              <div key={trait} className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="font-medium capitalize text-gray-300">
                    {trait}
                  </span>
                  <span className="text-sm text-gray-400">
                    {(typeof value === 'number' ? value * 100 : 0).toFixed(0)}%
                  </span>
                </div>
                <div className="w-full bg-gray-700 rounded-full h-2">
                  <div 
                    className="bg-gradient-to-r from-pink-500 to-red-500 h-2 rounded-full"
                    style={{ width: `${typeof value === 'number' ? value * 100 : 0}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

// Network Intelligence Dashboard Component
const NetworkIntelligenceDashboard: React.FC<{ 
  networkSnapshot: NetworkSnapshot | null;
  agiState: AGIState | null;
}> = ({ networkSnapshot, agiState }) => {
  if (!networkSnapshot) {
    return (
      <div className="text-center py-12">
        <Network className="w-16 h-16 mx-auto text-gray-600 mb-4" />
        <p className="text-gray-400">Network data not available</p>
      </div>
    );
  }

  const myIntelligence = agiState?.intelligence_level || 0;
  const networkAvg = networkSnapshot.network_metrics.average_intelligence;
  const performanceVsNetwork = ((myIntelligence / networkAvg - 1) * 100);

  return (
    <div className="space-y-6">
      {/* Network Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Active Nodes</h3>
            <Users className="w-5 h-5 text-blue-400" />
          </div>
          <div className="text-3xl font-bold text-white">
            {networkSnapshot.total_nodes}
          </div>
          <p className="text-sm text-gray-400">Contributing to network</p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Avg Intelligence</h3>
            <Brain className="w-5 h-5 text-green-400" />
          </div>
          <div className="text-3xl font-bold text-white">
            {networkAvg.toFixed(1)}
          </div>
          <p className="text-sm text-gray-400">Network average</p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">Collective Intelligence</h3>
            <Network className="w-5 h-5 text-purple-400" />
          </div>
          <div className="text-3xl font-bold text-white">
            {networkSnapshot.network_metrics.collective_intelligence.toFixed(1)}
          </div>
          <p className="text-sm text-gray-400">Network effect bonus</p>
        </div>

        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold text-gray-300">My vs Network</h3>
            <TrendingUp className={`w-5 h-5 ${performanceVsNetwork >= 0 ? 'text-green-400' : 'text-red-400'}`} />
          </div>
          <div className={`text-3xl font-bold ${performanceVsNetwork >= 0 ? 'text-green-400' : 'text-red-400'}`}>
            {performanceVsNetwork >= 0 ? '+' : ''}{performanceVsNetwork.toFixed(1)}%
          </div>
          <p className="text-sm text-gray-400">
            {performanceVsNetwork >= 0 ? 'Above' : 'Below'} network average
          </p>
        </div>
      </div>

      {/* Network Learning Rate */}
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Network Learning Dynamics</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div className="text-center">
            <div className="text-3xl font-bold text-blue-400 mb-2">
              {networkSnapshot.network_metrics.network_learning_rate.toFixed(3)}
            </div>
            <div className="text-gray-400">Network Learning Rate</div>
            <div className="text-sm text-gray-500">concepts/hour/node</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-green-400 mb-2">
              {Object.keys(networkSnapshot.domain_leaders).length}
            </div>
            <div className="text-gray-400">Active Domains</div>
            <div className="text-sm text-gray-500">knowledge areas</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-purple-400 mb-2">
              {(networkSnapshot.network_metrics.collective_intelligence / networkAvg).toFixed(2)}x
            </div>
            <div className="text-gray-400">Network Effect</div>
            <div className="text-sm text-gray-500">intelligence multiplier</div>
          </div>
        </div>
      </div>

      {/* Intelligence Distribution */}
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Intelligence Distribution</h3>
        <div className="space-y-4">
          <div className="flex items-center justify-between text-sm">
            <span className="text-gray-400">Intelligence Levels Across Network</span>
            <span className="text-gray-400">{networkSnapshot.total_nodes} nodes</span>
          </div>
          
          {/* Mock distribution bars - in real implementation, this would show actual distribution */}
          <div className="space-y-3">
            <div className="flex items-center gap-4">
              <span className="text-gray-400 w-20">80-100</span>
              <div className="flex-1 bg-gray-700 rounded-full h-3">
                <div className="bg-green-500 h-3 rounded-full" style={{ width: '5%' }} />
              </div>
              <span className="text-gray-400 w-12 text-right">2 nodes</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-gray-400 w-20">60-79</span>
              <div className="flex-1 bg-gray-700 rounded-full h-3">
                <div className="bg-blue-500 h-3 rounded-full" style={{ width: '15%' }} />
              </div>
              <span className="text-gray-400 w-12 text-right">8 nodes</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-gray-400 w-20">40-59</span>
              <div className="flex-1 bg-gray-700 rounded-full h-3">
                <div className="bg-yellow-500 h-3 rounded-full" style={{ width: '45%' }} />
              </div>
              <span className="text-gray-400 w-12 text-right">22 nodes</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-gray-400 w-20">20-39</span>
              <div className="flex-1 bg-gray-700 rounded-full h-3">
                <div className="bg-orange-500 h-3 rounded-full" style={{ width: '30%' }} />
              </div>
              <span className="text-gray-400 w-12 text-right">15 nodes</span>
            </div>
            <div className="flex items-center gap-4">
              <span className="text-gray-400 w-20">0-19</span>
              <div className="flex-1 bg-gray-700 rounded-full h-3">
                <div className="bg-red-500 h-3 rounded-full" style={{ width: '5%' }} />
              </div>
              <span className="text-gray-400 w-12 text-right">3 nodes</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

// Domain Leaders Dashboard Component
const DomainLeadersDashboard: React.FC<{ 
  networkSnapshot: NetworkSnapshot | null;
}> = ({ networkSnapshot }) => {
  const [leaderDetails, setLeaderDetails] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchLeaderDetails = async () => {
      try {
        const response = await fetch('/agi/network/leaders');
        if (response.ok) {
          const data = await response.json();
          setLeaderDetails(data);
        }
      } catch (error) {
        console.error('Error fetching leader details:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchLeaderDetails();
  }, []);

  if (loading) {
    return (
      <div className="text-center py-12">
        <Target className="w-16 h-16 mx-auto text-gray-600 mb-4 animate-pulse" />
        <p className="text-gray-400">Loading domain leaders...</p>
      </div>
    );
  }

  if (!leaderDetails?.leader_details) {
    return (
      <div className="text-center py-12">
        <Target className="w-16 h-16 mx-auto text-gray-600 mb-4" />
        <p className="text-gray-400">No domain leaders available</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Domain Expertise Leaders</h3>
        <div className="space-y-6">
          {Object.entries(leaderDetails.leader_details).map(([domain, details]: [string, any]) => (
            <div key={domain} className="border border-gray-700 rounded-lg p-4">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h4 className="text-lg font-semibold capitalize text-white">
                    {domain.replace('_', ' ')}
                  </h4>
                  <p className="text-sm text-gray-400">Node: {details.node_id}</p>
                </div>
                <div className="text-right">
                  <div className="text-2xl font-bold text-yellow-400">
                    {details.expertise?.toFixed(1) || '0.0'}
                  </div>
                  <div className="text-sm text-gray-400">expertise</div>
                </div>
              </div>
              
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-4">
                <div className="text-center">
                  <div className="text-lg font-semibold text-blue-400">
                    {details.intelligence_level?.toFixed(1) || '0.0'}
                  </div>
                  <div className="text-xs text-gray-400">Intelligence</div>
                </div>
                <div className="text-center">
                  <div className="text-lg font-semibold text-green-400">
                    {details.reputation?.toFixed(2) || '0.00'}
                  </div>
                  <div className="text-xs text-gray-400">Reputation</div>
                </div>
                <div className="text-center">
                  <div className="text-lg font-semibold text-purple-400">
                    {details.last_update ? new Date(details.last_update * 1000).toLocaleDateString() : 'Unknown'}
                  </div>
                  <div className="text-xs text-gray-400">Last Update</div>
                </div>
              </div>
              
              <div className="mt-4">
                <div className="w-full bg-gray-700 rounded-full h-2">
                  <div 
                    className="bg-gradient-to-r from-yellow-500 to-orange-500 h-2 rounded-full"
                    style={{ width: `${details.expertise || 0}%` }}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

// Query Routing Dashboard Component
const QueryRoutingDashboard: React.FC = () => {
  const [query, setQuery] = useState('');
  const [recommendations, setRecommendations] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);

  const analyzeQuery = async () => {
    if (!query.trim()) return;
    
    setLoading(true);
    try {
      // This would integrate with the actual query routing system
      // For now, we'll simulate recommendations
      setRecommendations([
        {
          node_id: 'QmTech123...',
          match_percentage: 94,
          domains: { technology: 87.2, reasoning: 76.3 },
          success_rate: 98.2,
          avg_response_time: 0.8,
          cost: 12
        },
        {
          node_id: 'QmReason456...',
          match_percentage: 89,
          domains: { reasoning: 91.5, technology: 72.1 },
          success_rate: 96.8,
          avg_response_time: 1.2,
          cost: 15
        }
      ]);
    } catch (error) {
      console.error('Error analyzing query:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Query Input */}
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Find the Best Node for Your Query</h3>
        <div className="flex gap-4">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Enter your query to find the best expert nodes..."
            className="flex-1 bg-gray-800 border border-gray-600 rounded-lg px-4 py-3 text-white placeholder-gray-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          <button
            onClick={analyzeQuery}
            disabled={loading || !query.trim()}
            className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-6 py-3 rounded-lg font-medium transition-colors"
          >
            {loading ? 'Analyzing...' : 'Analyze'}
          </button>
        </div>
      </div>

      {/* Recommendations */}
      {recommendations.length > 0 && (
        <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
          <h3 className="text-xl font-semibold mb-6">Recommended Expert Nodes</h3>
          <div className="space-y-4">
            {recommendations.map((rec, index) => (
              <div key={rec.node_id} className="border border-gray-700 rounded-lg p-4">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="text-2xl">
                      {index === 0 ? '🥇' : index === 1 ? '🥈' : '🥉'}
                    </div>
                    <div>
                      <h4 className="font-semibold text-white">{rec.node_id}</h4>
                      <p className="text-sm text-gray-400">Match: {rec.match_percentage}%</p>
                    </div>
                  </div>
                  <button className="bg-green-600 hover:bg-green-700 px-4 py-2 rounded-lg font-medium transition-colors">
                    Query This Node
                  </button>
                </div>
                
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-gray-400">Success Rate</div>
                    <div className="font-semibold text-green-400">{rec.success_rate}%</div>
                  </div>
                  <div>
                    <div className="text-gray-400">Response Time</div>
                    <div className="font-semibold text-blue-400">{rec.avg_response_time}s</div>
                  </div>
                  <div>
                    <div className="text-gray-400">Cost</div>
                    <div className="font-semibold text-yellow-400">{rec.cost} tokens</div>
                  </div>
                  <div>
                    <div className="text-gray-400">Top Domain</div>
                    <div className="font-semibold text-purple-400">
                      {Object.keys(rec.domains)[0]}: {Object.values(rec.domains)[0]}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Routing Options */}
      <div className="bg-gray-900 rounded-xl p-6 border border-gray-700">
        <h3 className="text-xl font-semibold mb-6">Auto-Route Options</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="border border-blue-500 bg-blue-500/10 rounded-lg p-4 cursor-pointer hover:bg-blue-500/20 transition-colors">
            <div className="flex items-center gap-3 mb-2">
              <Zap className="w-5 h-5 text-blue-400" />
              <span className="font-medium">Fastest Response</span>
            </div>
            <p className="text-sm text-gray-400">Avg: 0.6s</p>
          </div>
          
          <div className="border border-gray-600 rounded-lg p-4 cursor-pointer hover:bg-gray-800 transition-colors">
            <div className="flex items-center gap-3 mb-2">
              <Target className="w-5 h-5 text-green-400" />
              <span className="font-medium">Best Expertise</span>
            </div>
            <p className="text-sm text-gray-400">Match: 95%+</p>
          </div>
          
          <div className="border border-gray-600 rounded-lg p-4 cursor-pointer hover:bg-gray-800 transition-colors">
            <div className="flex items-center gap-3 mb-2">
              <span className="text-yellow-400">💰</span>
              <span className="font-medium">Most Cost-Effective</span>
            </div>
            <p className="text-sm text-gray-400">8-12 tokens</p>
          </div>
          
          <div className="border border-gray-600 rounded-lg p-4 cursor-pointer hover:bg-gray-800 transition-colors">
            <div className="flex items-center gap-3 mb-2">
              <span className="text-purple-400">⭐</span>
              <span className="font-medium">Highest Reputation</span>
            </div>
            <p className="text-sm text-gray-400">8.5+ rating</p>
          </div>
        </div>
      </div>
    </div>
  );
};

export default AGIIntelligence;