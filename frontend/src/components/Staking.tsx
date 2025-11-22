import { useState, useEffect } from 'react';
import { cortexAPI, type ReputationScore } from '../services/api';
import { Shield, Users, Lock, Unlock, AlertTriangle, Clock, Coins } from 'lucide-react';

interface NodeStakeInfo {
  node_id: string;
  total_user_stake: string;
  user_stakes: UserStake[];
  stake_count: number;
  reputation_boost: number;
  performance_data?: NodePerformance;
}

interface UserStake {
  id: string;
  user_id: string;
  node_id: string;
  amount: string;
  staked_at: string;
  last_update: string;
  withdrawal_requested: boolean;
  withdrawal_at?: string;
  withdrawal_amount?: string;
  accumulated_rewards: string;
  last_reward_time: string;
}

interface NodePerformance {
  node_id: string;
  queries_completed: number;
  queries_failed: number;
  success_rate: number;
  avg_response_time_ms: number;
  last_active: string;
  slashing_events: SlashingEvent[];
  total_slashed: string;
}

interface SlashingEvent {
  id: string;
  node_id: string;
  reason: string;
  severity: number;
  slashed_amount: string;
  affected_users: string[];
  timestamp: string;
  query_id?: string;
}

export default function Staking() {
  const [nodeStakes, setNodeStakes] = useState<Record<string, NodeStakeInfo>>({});
  const [userStakes, setUserStakes] = useState<UserStake[]>([]);
  const [reputationScores, setReputationScores] = useState<ReputationScore[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  
  // Form states
  const [showStakeForm, setShowStakeForm] = useState(false);
  const [stakeForm, setStakeForm] = useState({
    user_id: 'user-demo',
    node_id: '',
    amount: ''
  });
  // Removed unused withdrawForm state
  const [submitting, setSubmitting] = useState(false);

  const fetchStakingData = async () => {
    try {
      setLoading(true);
      const [stakesResponse, userStakesResponse, reputationResponse] = await Promise.all([
        cortexAPI.getAllNodeStakes(),
        cortexAPI.getUserStakes('user-demo'),
        cortexAPI.getReputationScores()
      ]);
      
      setNodeStakes(stakesResponse.nodes || {});
      setUserStakes(userStakesResponse.stakes || []);
      setReputationScores(reputationResponse.reputation_scores || []);
      setError('');
    } catch (err) {
      console.error('Failed to fetch staking data:', err);
      setError('Failed to load staking data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStakingData();
    const interval = setInterval(fetchStakingData, 15000); // Refresh every 15 seconds
    return () => clearInterval(interval);
  }, []);

  const formatTokenAmount = (amount: string): string => {
    try {
      const num = parseFloat(amount) / Math.pow(10, 18); // Convert from smallest unit
      return num.toFixed(4).replace(/\.?0+$/, '') + ' λore';
    } catch {
      return '0 λore';
    }
  };

  const formatDate = (dateString: string): string => {
    try {
      return new Date(dateString).toLocaleString();
    } catch {
      return 'Unknown';
    }
  };

  const getPerformanceColor = (successRate: number): string => {
    if (successRate >= 0.95) return 'text-tesla-accent-green';
    if (successRate >= 0.85) return 'text-tesla-accent-blue';
    if (successRate >= 0.75) return 'text-tesla-text-gray';
    if (successRate >= 0.60) return 'text-tesla-text-gray';
    return 'text-tesla-accent-red';
  };

  const getPerformanceBadge = (successRate: number): { label: string; color: string } => {
    if (successRate >= 0.95) return { label: 'EXCELLENT', color: 'bg-tesla-accent-green text-tesla-black border-tesla-accent-green' };
    if (successRate >= 0.85) return { label: 'GOOD', color: 'bg-tesla-accent-blue text-tesla-white border-tesla-accent-blue' };
    if (successRate >= 0.75) return { label: 'FAIR', color: 'bg-tesla-text-gray text-tesla-white border-tesla-text-gray' };
    if (successRate >= 0.60) return { label: 'POOR', color: 'bg-tesla-text-gray text-tesla-white border-tesla-text-gray' };
    return { label: 'CRITICAL', color: 'bg-tesla-accent-red text-tesla-white border-tesla-accent-red' };
  };

  const handleStake = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!stakeForm.node_id || !stakeForm.amount) return;

    setSubmitting(true);
    try {
      // Convert λoreto smallest unit
      const amountInSmallestUnit = (parseFloat(stakeForm.amount) * Math.pow(10, 18)).toString();
      
      await cortexAPI.stakeToNode(stakeForm.user_id, stakeForm.node_id, amountInSmallestUnit);

      setShowStakeForm(false);
      setStakeForm({ user_id: 'user-demo', node_id: '', amount: '' });
      await fetchStakingData();
    } catch (err) {
      setError(`Failed to stake: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  const handleRequestWithdrawal = async (userStake: UserStake) => {
    setSubmitting(true);
    try {
      await cortexAPI.requestWithdrawal(userStake.user_id, userStake.node_id, userStake.amount);

      await fetchStakingData();
    } catch (err) {
      setError(`Failed to request withdrawal: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  const handleExecuteWithdrawal = async (userStake: UserStake) => {
    setSubmitting(true);
    try {
      await cortexAPI.executeWithdrawal(userStake.user_id, userStake.node_id);

      await fetchStakingData();
    } catch (err) {
      setError(`Failed to execute withdrawal: ${err}`);
    } finally {
      setSubmitting(false);
    }
  };

  const isWithdrawalReady = (userStake: UserStake): boolean => {
    if (!userStake.withdrawal_requested || !userStake.withdrawal_at) return false;
    return new Date(userStake.withdrawal_at) <= new Date();
  };

  if (loading) {
    return (
      <div className="max-w-7xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
        <div className="flex items-center space-x-3 mb-6 border-b border-tesla-border pb-6">
          <Lock className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE STAKING</h1>
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
          <Lock className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE STAKING</h1>
        </div>
        <div className="text-center py-8">
          <AlertTriangle className="h-12 w-12 text-tesla-accent-red mx-auto mb-4" />
          <p className="text-tesla-accent-red text-sm uppercase tracking-wider">{error}</p>
          <button
            onClick={fetchStakingData}
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
      {/* Header */}
      <div className="flex items-center justify-between border-b border-tesla-border pb-6">
        <div className="flex items-center space-x-3">
          <Lock className="h-6 w-6 text-tesla-white" />
          <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NODE STAKING</h1>
        </div>
        <div className="flex space-x-4">
          <button
            onClick={fetchStakingData}
            className="tesla-button text-xs"
          >
            REFRESH
          </button>
          <button
            onClick={() => setShowStakeForm(true)}
            className="tesla-button text-xs"
          >
            STAKE TO NODE
          </button>
        </div>
      </div>

      {/* Stake Form Modal */}
      {showStakeForm && (
        <div className="fixed inset-0 bg-tesla-black bg-opacity-90 flex items-center justify-center z-50">
          <div className="bg-tesla-dark-gray border border-tesla-border p-6 w-full max-w-md">
            <h3 className="text-lg font-medium text-tesla-white mb-6 uppercase tracking-wider">STAKE λoreTO NODE</h3>
            <form onSubmit={handleStake}>
              <div className="space-y-6">
                <div>
                  <label className="block text-xs font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">
                    NODE ID
                  </label>
                  <select
                    value={stakeForm.node_id}
                    onChange={(e) => setStakeForm({ ...stakeForm, node_id: e.target.value })}
                    className="w-full px-3 py-3 bg-tesla-medium-gray border border-tesla-border text-tesla-white text-sm focus:border-tesla-white transition-all duration-150"
                    required
                  >
                    <option value="">SELECT A NODE...</option>
                    {reputationScores.map((score) => (
                      <option key={score.node_id} value={score.node_id}>
                        {score.node_id.substring(0, 16)}... (REP: {score.score.toFixed(2)})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">
                    AMOUNT (λore)
                  </label>
                  <input
                    type="number"
                    step="0.0001"
                    min="10"
                    value={stakeForm.amount}
                    onChange={(e) => setStakeForm({ ...stakeForm, amount: e.target.value })}
                    className="w-full px-3 py-3 bg-tesla-medium-gray border border-tesla-border text-tesla-white text-sm focus:border-tesla-white transition-all duration-150"
                    placeholder="MINIMUM 10 λore"
                    required
                  />
                </div>
              </div>
              <div className="flex space-x-3 mt-8">
                <button
                  type="button"
                  onClick={() => setShowStakeForm(false)}
                  className="flex-1 px-4 py-3 bg-tesla-medium-gray hover:bg-tesla-light-gray text-tesla-white border border-tesla-border text-xs uppercase tracking-wider transition-all duration-150"
                >
                  CANCEL
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="flex-1 tesla-button text-xs disabled:opacity-50"
                >
                  {submitting ? 'STAKING...' : 'STAKE'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* My Stakes */}
      {userStakes.length > 0 && (
        <div className="bg-tesla-dark-gray border border-tesla-border p-6">
          <h2 className="text-lg font-medium text-tesla-white mb-6 flex items-center uppercase tracking-wider">
            <Coins className="h-5 w-5 mr-3 text-tesla-white" />
            MY STAKES ({userStakes.length})
          </h2>
          <div className="space-y-3">
            {userStakes.map((stake) => (
              <div key={stake.id} className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="flex items-center space-x-3">
                      <span className="font-medium text-tesla-white text-sm">
                        {stake.node_id.substring(0, 16)}...
                      </span>
                      <span className="text-lg font-light text-tesla-white">
                        {formatTokenAmount(stake.amount)}
                      </span>
                    </div>
                    <p className="text-xs text-tesla-text-gray uppercase tracking-wider mt-1">
                      STAKED: {formatDate(stake.staked_at)}
                    </p>
                    {stake.withdrawal_requested && (
                      <div className="flex items-center space-x-2 mt-2">
                        <Clock className="h-4 w-4 text-tesla-text-gray" />
                        <span className="text-xs text-tesla-text-gray uppercase tracking-wider">
                          {isWithdrawalReady(stake) 
                            ? 'WITHDRAWAL READY' 
                            : `WITHDRAWAL AVAILABLE: ${formatDate(stake.withdrawal_at!)}`
                          }
                        </span>
                      </div>
                    )}
                  </div>
                  <div className="flex space-x-3">
                    {stake.withdrawal_requested ? (
                      isWithdrawalReady(stake) ? (
                        <button
                          onClick={() => handleExecuteWithdrawal(stake)}
                          disabled={submitting}
                          className="px-3 py-2 bg-tesla-accent-green hover:bg-tesla-accent-green text-tesla-black text-xs uppercase tracking-wider disabled:opacity-50 transition-all duration-150"
                        >
                          <Unlock className="h-4 w-4" />
                        </button>
                      ) : (
                        <span className="px-3 py-2 bg-tesla-text-gray text-tesla-white text-xs uppercase tracking-wider">
                          PENDING
                        </span>
                      )
                    ) : (
                      <button
                        onClick={() => handleRequestWithdrawal(stake)}
                        disabled={submitting}
                        className="px-3 py-2 bg-tesla-text-gray hover:bg-tesla-light-gray text-tesla-white text-xs uppercase tracking-wider disabled:opacity-50 transition-all duration-150"
                      >
                        WITHDRAW
                      </button>
                    )}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Available Nodes */}
      <div className="bg-tesla-dark-gray border border-tesla-border p-6">
        <h2 className="text-lg font-medium text-tesla-white mb-6 flex items-center uppercase tracking-wider">
          <Users className="h-5 w-5 mr-3 text-tesla-white" />
          AVAILABLE NODES ({Object.keys(nodeStakes).length})
        </h2>

        {Object.keys(nodeStakes).length === 0 ? (
          <div className="text-center py-8 text-tesla-text-gray">
            <Shield className="h-12 w-12 mx-auto mb-4 text-tesla-text-gray" />
            <p className="text-xs uppercase tracking-wider">NO NODES AVAILABLE FOR STAKING</p>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {Object.entries(nodeStakes).map(([nodeId, stakeInfo]) => {
              const reputation = reputationScores.find(r => r.node_id === nodeId);
              const performance = stakeInfo.performance_data;
              const badge = performance ? getPerformanceBadge(performance.success_rate) : null;
              
              return (
                <div key={nodeId} className="bg-tesla-medium-gray border border-tesla-border p-4 hover:border-tesla-light-gray transition-all duration-150">
                  <div className="flex items-start justify-between mb-4">
                    <div>
                      <h3 className="font-medium text-tesla-white text-sm">
                        {nodeId.substring(0, 16)}...
                      </h3>
                      <div className="flex items-center space-x-3 mt-2">
                        <span className="text-xs text-tesla-text-gray uppercase tracking-wider">
                          REP: {reputation ? reputation.score.toFixed(2) : 'N/A'}
                        </span>
                        {stakeInfo.reputation_boost > 0 && (
                          <span className="text-xs text-tesla-accent-green uppercase tracking-wider">
                            (+{stakeInfo.reputation_boost.toFixed(3)} BOOST)
                          </span>
                        )}
                      </div>
                    </div>
                    {badge && (
                      <span className={`inline-flex items-center px-2 py-1 text-xs font-medium uppercase tracking-wider border ${badge.color}`}>
                        {badge.label}
                      </span>
                    )}
                  </div>

                  <div className="space-y-3 text-xs">
                    <div className="flex justify-between">
                      <span className="text-tesla-text-gray uppercase tracking-wider">TOTAL STAKED:</span>
                      <span className="text-tesla-white font-medium">
                        {formatTokenAmount(stakeInfo.total_user_stake)}
                      </span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-tesla-text-gray uppercase tracking-wider">STAKERS:</span>
                      <span className="text-tesla-white">{stakeInfo.stake_count}</span>
                    </div>
                    
                    {performance && (
                      <>
                        <div className="flex justify-between">
                          <span className="text-tesla-text-gray uppercase tracking-wider">SUCCESS RATE:</span>
                          <span className={getPerformanceColor(performance.success_rate)}>
                            {(performance.success_rate * 100).toFixed(1)}%
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span className="text-tesla-text-gray uppercase tracking-wider">QUERIES:</span>
                          <span className="text-tesla-white">
                            {performance.queries_completed + performance.queries_failed}
                          </span>
                        </div>
                        {performance.slashing_events.length > 0 && (
                          <div className="flex justify-between">
                            <span className="text-tesla-text-gray uppercase tracking-wider">SLASHED:</span>
                            <span className="text-tesla-accent-red">
                              {formatTokenAmount(performance.total_slashed)}
                            </span>
                          </div>
                        )}
                      </>
                    )}
                  </div>

                  <button
                    onClick={() => {
                      setStakeForm({ ...stakeForm, node_id: nodeId });
                      setShowStakeForm(true);
                    }}
                    className="w-full mt-4 tesla-button text-xs"
                  >
                    STAKE TO THIS NODE
                  </button>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Staking Info */}
      <div className="bg-tesla-dark-gray border border-tesla-border p-6">
        <h2 className="text-lg font-medium text-tesla-white mb-6 uppercase tracking-wider">STAKING INFORMATION</h2>
        <div className="grid md:grid-cols-2 gap-8 text-xs">
          <div>
            <h3 className="font-medium text-tesla-white mb-4 uppercase tracking-wider">HOW IT WORKS:</h3>
            <ul className="space-y-2 text-tesla-text-gray">
              <li>• STAKE λoreTOKENS TO NODES TO BOOST THEIR REPUTATION</li>
              <li>• HIGHER REPUTATION NODES GET MORE QUERIES AND REWARDS</li>
              <li>• EARN A SHARE OF NODE REWARDS BASED ON YOUR STAKE</li>
              <li>• SLASHING OCCURS IF NODES FAIL TO FULFILL QUERIES</li>
            </ul>
          </div>
          <div>
            <h3 className="font-medium text-tesla-white mb-4 uppercase tracking-wider">RISK & REWARDS:</h3>
            <ul className="space-y-2 text-tesla-text-gray">
              <li>• MINIMUM STAKE: 10 λorePER NODE</li>
              <li>• MAXIMUM STAKE: 10,000 λorePER NODE</li>
              <li>• 7-DAY WITHDRAWAL DELAY FOR SECURITY</li>
              <li>• STAKES ARE SLASHED IF NODES PERFORM POORLY</li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  );
}