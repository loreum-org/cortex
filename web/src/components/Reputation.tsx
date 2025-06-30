import { useState, useEffect } from 'react';
import { cortexAPI, type ReputationScore, type ConsensusStatus } from '../services/api';
import { Shield, TrendingUp, Users, Activity, AlertTriangle, CheckCircle } from 'lucide-react';

interface ReputationProps {
  className?: string;
}

export default function Reputation({ className = '' }: ReputationProps) {
  const [reputationScores, setReputationScores] = useState<ReputationScore[]>([]);
  const [consensusStatus, setConsensusStatus] = useState<ConsensusStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');

  const fetchReputationData = async () => {
    try {
      setLoading(true);
      const [scoresResponse, statusResponse] = await Promise.all([
        cortexAPI.getReputationScores(),
        cortexAPI.getConsensusStatus()
      ]);
      
      setReputationScores(scoresResponse.reputation_scores || []);
      setConsensusStatus(statusResponse);
      setError('');
    } catch (err) {
      console.error('Failed to fetch reputation data:', err);
      setError('Failed to load reputation data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchReputationData();
    const interval = setInterval(fetchReputationData, 10000); // Refresh every 10 seconds
    return () => clearInterval(interval);
  }, []);

  const getReputationColor = (score: number) => {
    if (score >= 8) return 'text-tesla-accent-green';
    if (score >= 6) return 'text-tesla-accent-blue';
    if (score >= 4) return 'text-tesla-text-gray';
    if (score >= 2) return 'text-tesla-text-gray';
    return 'text-tesla-accent-red';
  };

  const getReputationBadge = (score: number) => {
    if (score >= 8) return { label: 'EXCELLENT', color: 'bg-tesla-accent-green text-tesla-black' };
    if (score >= 6) return { label: 'GOOD', color: 'bg-tesla-accent-blue text-tesla-white' };
    if (score >= 4) return { label: 'FAIR', color: 'bg-tesla-text-gray text-tesla-white' };
    if (score >= 2) return { label: 'POOR', color: 'bg-tesla-text-gray text-tesla-white' };
    return { label: 'CRITICAL', color: 'bg-tesla-accent-red text-tesla-white' };
  };

  const formatNodeId = (nodeId: string) => {
    if (nodeId.length > 12) {
      return `${nodeId.substring(0, 6)}...${nodeId.substring(nodeId.length - 6)}`;
    }
    return nodeId;
  };

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
        <div className="flex items-center space-x-3 mb-6 border-b border-tesla-border pb-6">
          <Shield className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE REPUTATION</h1>
        </div>
        <div className="flex justify-center py-8">
          <div className="animate-spin h-8 w-8 border-2 border-tesla-white border-t-transparent"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
        <div className="flex items-center space-x-3 mb-6 border-b border-tesla-border pb-6">
          <Shield className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE REPUTATION</h1>
        </div>
        <div className="text-center py-8">
          <AlertTriangle className="h-12 w-12 text-tesla-accent-red mx-auto mb-4" />
          <p className="text-tesla-accent-red text-sm uppercase tracking-wider">{error}</p>
          <button
            onClick={fetchReputationData}
            className="mt-4 tesla-button text-xs"
          >
            RETRY
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
      <div className="flex items-center justify-between mb-6 border-b border-tesla-border pb-6">
        <div className="flex items-center space-x-3">
          <Shield className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE REPUTATION</h1>
        </div>
        <button
          onClick={fetchReputationData}
          className="tesla-button text-xs"
        >
          REFRESH
        </button>
      </div>

    <div className={`bg-tesla-dark-gray border border-tesla-border p-6 ${className}`}>

      {/* Consensus Status Summary */}
      {consensusStatus && (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
          <div className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
            <div className="flex items-center space-x-2 mb-2">
              <Activity className="h-4 w-4 text-tesla-accent-green" />
              <span className="text-xs text-tesla-text-gray uppercase tracking-wider">ACTIVE NODES</span>
            </div>
            <p className="text-2xl font-light text-tesla-white">
              {consensusStatus.reputation_stats.nodes_count || 0}
            </p>
          </div>
          
          <div className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
            <div className="flex items-center space-x-2 mb-2">
              <TrendingUp className="h-4 w-4 text-tesla-accent-blue" />
              <span className="text-xs text-tesla-text-gray uppercase tracking-wider">AVG SCORE</span>
            </div>
            <p className="text-2xl font-light text-tesla-white">
              {consensusStatus.reputation_stats.average?.toFixed(2) || 'N/A'}
            </p>
          </div>
          
          <div className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
            <div className="flex items-center space-x-2 mb-2">
              <CheckCircle className="h-4 w-4 text-tesla-accent-green" />
              <span className="text-xs text-tesla-text-gray uppercase tracking-wider">FINALIZED</span>
            </div>
            <p className="text-2xl font-light text-tesla-white">
              {consensusStatus.finalized_transactions}
            </p>
          </div>
          
          <div className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
            <div className="flex items-center space-x-2 mb-2">
              <AlertTriangle className="h-4 w-4 text-tesla-accent-red" />
              <span className="text-xs text-tesla-text-gray uppercase tracking-wider">CONFLICTS</span>
            </div>
            <p className="text-2xl font-light text-tesla-white">
              {consensusStatus.conflicts_detected}
            </p>
          </div>
        </div>
      )}

      {/* Reputation Scores List */}
      <div className="space-y-4">
        <div className="flex items-center space-x-2 mb-6">
          <Users className="h-4 w-4 text-tesla-white" />
          <h4 className="text-sm font-medium text-tesla-white uppercase tracking-wider">
            NETWORK REPUTATION SCORES ({reputationScores.length} NODES)
          </h4>
        </div>

        {reputationScores.length === 0 ? (
          <div className="text-center py-8 text-tesla-text-gray">
            <Users className="h-12 w-12 mx-auto mb-4 text-tesla-text-gray" />
            <p className="text-xs uppercase tracking-wider">NO REPUTATION DATA AVAILABLE</p>
            <p className="text-xs text-tesla-text-gray mt-2">NODES WILL APPEAR HERE AS THEY PARTICIPATE IN CONSENSUS</p>
          </div>
        ) : (
          <div className="space-y-2">
            {reputationScores
              .sort((a, b) => b.score - a.score) // Sort by highest score first
              .map((node) => {
                const badge = getReputationBadge(node.score);
                return (
                  <div
                    key={node.node_id}
                    className="flex items-center justify-between p-4 bg-tesla-medium-gray border border-tesla-border hover:border-tesla-light-gray transition-all duration-150"
                  >
                    <div className="flex items-center space-x-4">
                      <div className="flex-shrink-0">
                        <div className="w-10 h-10 bg-tesla-white flex items-center justify-center">
                          <span className="text-xs font-medium text-tesla-black uppercase">
                            {node.node_id.substring(0, 2)}
                          </span>
                        </div>
                      </div>
                      <div>
                        <p className="font-medium text-tesla-white text-sm">
                          {formatNodeId(node.node_id)}
                        </p>
                        <p className="text-xs text-tesla-text-gray uppercase tracking-wider">
                          UPDATED: {new Date(node.last_update).toLocaleTimeString()}
                        </p>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-4">
                      <span className={`inline-flex items-center px-3 py-1 text-xs font-medium uppercase tracking-wider ${badge.color}`}>
                        {badge.label}
                      </span>
                      <span className={`text-xl font-light ${getReputationColor(node.score)}`}>
                        {node.score.toFixed(2)}
                      </span>
                    </div>
                  </div>
                );
              })}
          </div>
        )}
      </div>

      {/* Reputation Scale Reference */}
      <div className="mt-8 pt-6 border-t border-tesla-border">
        <h5 className="text-xs font-medium text-tesla-text-gray mb-4 uppercase tracking-wider">REPUTATION SCALE:</h5>
        <div className="grid grid-cols-5 gap-3 text-xs">
          <div className="text-center">
            <div className="w-full h-1 bg-tesla-accent-green mb-2"></div>
            <span className="text-tesla-accent-green uppercase tracking-wider">8-10 EXCELLENT</span>
          </div>
          <div className="text-center">
            <div className="w-full h-1 bg-tesla-accent-blue mb-2"></div>
            <span className="text-tesla-accent-blue uppercase tracking-wider">6-8 GOOD</span>
          </div>
          <div className="text-center">
            <div className="w-full h-1 bg-tesla-text-gray mb-2"></div>
            <span className="text-tesla-text-gray uppercase tracking-wider">4-6 FAIR</span>
          </div>
          <div className="text-center">
            <div className="w-full h-1 bg-tesla-text-gray mb-2"></div>
            <span className="text-tesla-text-gray uppercase tracking-wider">2-4 POOR</span>
          </div>
          <div className="text-center">
            <div className="w-full h-1 bg-tesla-accent-red mb-2"></div>
            <span className="text-tesla-accent-red uppercase tracking-wider">0-2 CRITICAL</span>
          </div>
        </div>
      </div>
    </div>
    </div>
  );
}