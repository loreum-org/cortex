import React, { useState, useEffect, useRef } from 'react';
import Chart from 'chart.js/auto';

interface NetworkStats {
  total_supply: string;
  active_nodes: number;
  total_staked: string;
  avg_reputation: number;
  total_accounts: number;
  total_transactions: number;
  transaction_counts?: Record<string, number>;
}

interface NodeStake {
  total_user_stake: string;
  stake_count: number;
  reputation_boost: number;
  performance_data?: {
    success_rate: number;
  };
}

interface UserAccount {
  account: string;
  address: string;
  balance_formatted: string;
  token_symbol: string;
  total_spent: string;
  queries_submitted: number;
  created_at: string;
}

interface UserStake {
  id: string;
  node_id: string;
  amount: string;
  staked_at: string;
  withdrawal_requested: boolean;
}

interface Transaction {
  id: string;
  type: string;
  amount: string;
  from_id: string;
  to_id: string;
  status: string;
  timestamp: string;
  description: string;
}

interface RewardEvent {
  id: string;
  node_id: string;
  amount: string;
  timestamp: string;
  reason: string;
  performance_data?: {
    performance_score: number;
  };
}

interface TransferForm {
  fromAccount: string;
  toAccount: string;
  amount: string;
  description: string;
  processing: boolean;
  error: string | null;
  success: string | null;
}

interface StakeForm {
  account: string;
  nodeId: string;
  amount: string;
  processing: boolean;
  error: string | null;
  success: string | null;
}

interface ChartData {
  labels: string[];
  data: number[];
}

interface LoadingState {
  networkStats: boolean;
  nodeStakes: boolean;
  userAccount: boolean;
  transactions: boolean;
  rewards: boolean;
}

const EconomicDashboard: React.FC = () => {
  // Chart refs
  const transactionTypesChartRef = useRef<HTMLCanvasElement>(null);
  const nodeStakesChartRef = useRef<HTMLCanvasElement>(null);
  const rewardsChartRef = useRef<HTMLCanvasElement>(null);
  
  // Chart instances
  const txTypesChartInstanceRef = useRef<Chart | null>(null);
  const nodeStakesChartInstanceRef = useRef<Chart | null>(null);
  const rewardsChartInstanceRef = useRef<Chart | null>(null);

  // Loading states
  const [loading, setLoading] = useState<LoadingState>({
    networkStats: true,
    nodeStakes: true,
    userAccount: true,
    transactions: true,
    rewards: true
  });

  // Data states
  const [networkStats, setNetworkStats] = useState<NetworkStats>({} as NetworkStats);
  const [nodeStakes, setNodeStakes] = useState<Record<string, NodeStake>>({});
  const [userAccount, setUserAccount] = useState<UserAccount | null>(null);
  const [userStakes, setUserStakes] = useState<UserStake[]>([]);
  const [userTransactions, setUserTransactions] = useState<Transaction[]>([]);
  const [rewardEvents, setRewardEvents] = useState<RewardEvent[]>([]);

  // Chart data
  const [chartData, setChartData] = useState({
    transactionTypes: { labels: [], data: [] } as ChartData,
    nodeStakes: { labels: [], data: [] } as ChartData,
    rewards: { labels: [], data: [] } as ChartData
  });

  // User selection
  const [selectedAccount, setSelectedAccount] = useState('default_user');
  const [customAccountId, setCustomAccountId] = useState('');

  // Forms
  const [transferForm, setTransferForm] = useState<TransferForm>({
    fromAccount: '',
    toAccount: '',
    amount: '',
    description: '',
    processing: false,
    error: null,
    success: null
  });

  const [stakeForm, setStakeForm] = useState<StakeForm>({
    account: '',
    nodeId: '',
    amount: '',
    processing: false,
    error: null,
    success: null
  });

  // Pricing tiers data
  const pricingTiers = [
    {
      id: 'basic',
      name: 'Basic Tier',
      basePrice: '1',
      description: 'Affordable AI inference for simple tasks',
      features: [
        'Simple text completion',
        'Basic chat functionality',
        'Limited context window',
        'Standard response time'
      ],
      queryTypes: {
        completion: '1',
        chat: '2',
        embedding: '0.5'
      }
    },
    {
      id: 'standard',
      name: 'Standard Tier',
      basePrice: '5',
      description: 'Balanced performance for most use cases',
      features: [
        'Enhanced text completion',
        'Advanced chat capabilities',
        'Code generation',
        'Improved response quality',
        'Medium context window'
      ],
      queryTypes: {
        completion: '5',
        chat: '10',
        codegen: '15'
      }
    },
    {
      id: 'premium',
      name: 'Premium Tier',
      basePrice: '20',
      description: 'High-performance AI for demanding applications',
      features: [
        'High-quality text generation',
        'Complex reasoning capabilities',
        'RAG-enhanced responses',
        'Priority processing',
        'Large context window',
        'Specialized domain knowledge'
      ],
      queryTypes: {
        completion: '20',
        chat: '30',
        rag: '40'
      }
    }
  ];

  // Load data on component mount
  useEffect(() => {
    const loadInitialData = async () => {
      await Promise.all([
        fetchNetworkStats(),
        fetchNodeStakes(),
        fetchRewardDistribution()
      ]);
      
      // Set default account in forms
      setTransferForm(prev => ({ ...prev, fromAccount: selectedAccount }));
      setStakeForm(prev => ({ ...prev, account: selectedAccount }));
      
      // Load user data
      loadUserData();
    };

    loadInitialData();
  }, []);

  // Update charts when data changes
  useEffect(() => {
    updateTransactionTypesChart();
  }, [chartData.transactionTypes]);

  useEffect(() => {
    updateNodeStakesChart();
  }, [chartData.nodeStakes]);

  useEffect(() => {
    updateRewardsChart();
  }, [chartData.rewards]);

  // Fetch network statistics
  const fetchNetworkStats = async () => {
    setLoading(prev => ({ ...prev, networkStats: true }));
    try {
      const response = await fetch('/metrics/economy');
      const data = await response.json();
      setNetworkStats(data);
      
      // Prepare transaction types chart data
      if (data.transaction_counts) {
        const labels: string[] = [];
        const values: number[] = [];
        
        for (const [type, count] of Object.entries(data.transaction_counts)) {
          labels.push(formatTransactionType(type));
          values.push(count as number);
        }
        
        setChartData(prev => ({
          ...prev,
          transactionTypes: { labels, data: values }
        }));
      }
    } catch (error) {
      console.error('Failed to fetch network statistics:', error);
    } finally {
      setLoading(prev => ({ ...prev, networkStats: false }));
    }
  };

  // Fetch node staking information
  const fetchNodeStakes = async () => {
    setLoading(prev => ({ ...prev, nodeStakes: true }));
    try {
      const response = await fetch('/staking/nodes');
      const data = await response.json();
      setNodeStakes(data.nodes || {});
      
      // Prepare node stakes chart data
      const labels: string[] = [];
      const values: number[] = [];
      
      for (const [nodeId, stake] of Object.entries(data.nodes || {})) {
        if ((stake as NodeStake).total_user_stake) {
          labels.push(formatNodeId(nodeId));
          // Convert from string to number for chart
          const stakeValue = parseFloat((stake as NodeStake).total_user_stake) / Math.pow(10, 18);
          values.push(stakeValue);
        }
      }
      
      setChartData(prev => ({
        ...prev,
        nodeStakes: { labels, data: values }
      }));
    } catch (error) {
      console.error('Failed to fetch node stakes:', error);
    } finally {
      setLoading(prev => ({ ...prev, nodeStakes: false }));
    }
  };

  // Load user data (account, stakes, transactions)
  const loadUserData = async () => {
    const accountId = selectedAccount === 'custom' ? customAccountId : selectedAccount;
    if (!accountId) return;
    
    setLoading(prev => ({ ...prev, userAccount: true, transactions: true }));
    
    // Fetch account details
    try {
      const accountResponse = await fetch(`/wallet/account/${accountId}`);
      if (accountResponse.ok) {
        setUserAccount(await accountResponse.json());
      } else {
        setUserAccount(null);
        console.error('Failed to fetch user account:', await accountResponse.text());
      }
    } catch (error) {
      console.error('Error fetching user account:', error);
      setUserAccount(null);
    } finally {
      setLoading(prev => ({ ...prev, userAccount: false }));
    }
    
    // Fetch user stakes
    try {
      const stakesResponse = await fetch(`/staking/user/${accountId}`);
      if (stakesResponse.ok) {
        const data = await stakesResponse.json();
        setUserStakes(data.stakes || []);
      } else {
        setUserStakes([]);
      }
    } catch (error) {
      console.error('Error fetching user stakes:', error);
      setUserStakes([]);
    }
    
    // Fetch user transactions
    try {
      const txResponse = await fetch(`/wallet/transactions/${accountId}?limit=20`);
      if (txResponse.ok) {
        setUserTransactions(await txResponse.json());
      } else {
        setUserTransactions([]);
      }
    } catch (error) {
      console.error('Error fetching transactions:', error);
      setUserTransactions([]);
    } finally {
      setLoading(prev => ({ ...prev, transactions: false }));
    }
    
    // Update form values
    setTransferForm(prev => ({ ...prev, fromAccount: accountId }));
  };

  // Fetch reward distribution data
  const fetchRewardDistribution = async () => {
    setLoading(prev => ({ ...prev, rewards: true }));
    try {
      // Simulated reward data
      const mockRewards: RewardEvent[] = [
        {
          id: 'reward_1',
          node_id: 'node-1',
          amount: '500000000000000000', // 0.5 LORE
          timestamp: new Date().toISOString(),
          reason: 'periodic_staking_reward',
          performance_data: { performance_score: 0.92 }
        },
        {
          id: 'reward_2',
          node_id: 'node-2',
          amount: '750000000000000000', // 0.75 LORE
          timestamp: new Date(Date.now() - 86400000).toISOString(), // 1 day ago
          reason: 'periodic_staking_reward',
          performance_data: { performance_score: 0.85 }
        },
        {
          id: 'reward_3',
          node_id: 'node-3',
          amount: '1200000000000000000', // 1.2 LORE
          timestamp: new Date(Date.now() - 172800000).toISOString(), // 2 days ago
          reason: 'periodic_staking_reward',
          performance_data: { performance_score: 0.95 }
        },
        {
          id: 'reward_4',
          node_id: 'node-1',
          amount: '600000000000000000', // 0.6 LORE
          timestamp: new Date(Date.now() - 259200000).toISOString(), // 3 days ago
          reason: 'periodic_staking_reward',
          performance_data: { performance_score: 0.88 }
        }
      ];
      
      setRewardEvents(mockRewards);
      
      // Prepare rewards chart data
      const nodeRewards: Record<string, number> = {};
      
      for (const reward of mockRewards) {
        const nodeId = formatNodeId(reward.node_id);
        if (!nodeRewards[nodeId]) {
          nodeRewards[nodeId] = 0;
        }
        // Convert from string to number
        const amount = parseFloat(reward.amount) / Math.pow(10, 18);
        nodeRewards[nodeId] += amount;
      }
      
      setChartData(prev => ({
        ...prev,
        rewards: {
          labels: Object.keys(nodeRewards),
          data: Object.values(nodeRewards)
        }
      }));
    } catch (error) {
      console.error('Failed to fetch reward distribution:', error);
    } finally {
      setLoading(prev => ({ ...prev, rewards: false }));
    }
  };

  // Update transaction types chart
  const updateTransactionTypesChart = () => {
    if (txTypesChartInstanceRef.current) {
      txTypesChartInstanceRef.current.destroy();
    }
    
    const ctx = transactionTypesChartRef.current?.getContext('2d');
    if (ctx && chartData.transactionTypes.labels.length > 0) {
      txTypesChartInstanceRef.current = new Chart(ctx, {
        type: 'pie',
        data: {
          labels: chartData.transactionTypes.labels,
          datasets: [{
            data: chartData.transactionTypes.data,
            backgroundColor: [
              '#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF', '#FF9F40'
            ]
          }]
        },
        options: {
          responsive: true,
          plugins: {
            legend: {
              position: 'right',
              labels: {
                color: '#e2e8f0'
              }
            },
            title: {
              display: true,
              text: 'Transaction Types',
              color: '#f1f5f9'
            }
          }
        }
      });
    }
  };

  // Update node stakes chart
  const updateNodeStakesChart = () => {
    if (nodeStakesChartInstanceRef.current) {
      nodeStakesChartInstanceRef.current.destroy();
    }
    
    const ctx = nodeStakesChartRef.current?.getContext('2d');
    if (ctx && chartData.nodeStakes.labels.length > 0) {
      nodeStakesChartInstanceRef.current = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: chartData.nodeStakes.labels,
          datasets: [{
            label: 'Staked LORE',
            data: chartData.nodeStakes.data,
            backgroundColor: '#36A2EB'
          }]
        },
        options: {
          responsive: true,
          scales: {
            x: {
              ticks: { color: '#e2e8f0' },
              grid: { color: '#475569' }
            },
            y: {
              beginAtZero: true,
              ticks: { color: '#e2e8f0' },
              grid: { color: '#475569' },
              title: {
                display: true,
                text: 'LORE Tokens',
                color: '#f1f5f9'
              }
            }
          },
          plugins: {
            legend: {
              labels: { color: '#e2e8f0' }
            },
            title: {
              display: true,
              text: 'Node Stake Distribution',
              color: '#f1f5f9'
            }
          }
        }
      });
    }
  };

  // Update rewards chart
  const updateRewardsChart = () => {
    if (rewardsChartInstanceRef.current) {
      rewardsChartInstanceRef.current.destroy();
    }
    
    const ctx = rewardsChartRef.current?.getContext('2d');
    if (ctx && chartData.rewards.labels.length > 0) {
      rewardsChartInstanceRef.current = new Chart(ctx, {
        type: 'bar',
        data: {
          labels: chartData.rewards.labels,
          datasets: [{
            label: 'Rewards (LORE)',
            data: chartData.rewards.data,
            backgroundColor: '#4BC0C0'
          }]
        },
        options: {
          responsive: true,
          scales: {
            x: {
              ticks: { color: '#e2e8f0' },
              grid: { color: '#475569' }
            },
            y: {
              beginAtZero: true,
              ticks: { color: '#e2e8f0' },
              grid: { color: '#475569' },
              title: {
                display: true,
                text: 'LORE Tokens',
                color: '#f1f5f9'
              }
            }
          },
          plugins: {
            legend: {
              labels: { color: '#e2e8f0' }
            },
            title: {
              display: true,
              text: 'Reward Distribution by Node',
              color: '#f1f5f9'
            }
          }
        }
      });
    }
  };

  // Perform token transfer
  const performTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    setTransferForm(prev => ({ ...prev, processing: true, error: null, success: null }));
    
    try {
      // Convert LORE amount to smallest unit
      const amountInSmallestUnit = BigInt(Math.floor(parseFloat(transferForm.amount) * Math.pow(10, 18))).toString();
      
      const response = await fetch('/wallet/transfer', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          from_account: transferForm.fromAccount,
          to_account: transferForm.toAccount,
          amount: amountInSmallestUnit,
          description: transferForm.description || 'Token transfer'
        })
      });
      
      if (response.ok) {
        setTransferForm(prev => ({
          ...prev,
          success: `Successfully transferred ${transferForm.amount} LORE to ${transferForm.toAccount}`,
          amount: '',
          description: ''
        }));
        
        // Refresh data
        await Promise.all([
          fetchNetworkStats(),
          loadUserData()
        ]);
      } else {
        const error = await response.json();
        setTransferForm(prev => ({ ...prev, error: error.error || 'Transfer failed' }));
      }
    } catch (error) {
      console.error('Transfer error:', error);
      setTransferForm(prev => ({ ...prev, error: 'An unexpected error occurred' }));
    } finally {
      setTransferForm(prev => ({ ...prev, processing: false }));
    }
  };

  // Perform token staking
  const performStake = async (e: React.FormEvent) => {
    e.preventDefault();
    setStakeForm(prev => ({ ...prev, processing: true, error: null, success: null }));
    
    try {
      // Convert LORE amount to smallest unit
      const amountInSmallestUnit = BigInt(Math.floor(parseFloat(stakeForm.amount) * Math.pow(10, 18))).toString();
      
      const response = await fetch('/staking/stake', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          account: stakeForm.account,
          node_id: stakeForm.nodeId,
          amount: amountInSmallestUnit
        })
      });
      
      if (response.ok) {
        setStakeForm(prev => ({
          ...prev,
          success: `Successfully staked ${stakeForm.amount} LORE to node ${stakeForm.nodeId}`,
          amount: ''
        }));
        
        // Refresh data
        await Promise.all([
          fetchNetworkStats(),
          fetchNodeStakes(),
          loadUserData()
        ]);
      } else {
        const error = await response.json();
        setStakeForm(prev => ({ ...prev, error: error.error || 'Staking failed' }));
      }
    } catch (error) {
      console.error('Staking error:', error);
      setStakeForm(prev => ({ ...prev, error: 'An unexpected error occurred' }));
    } finally {
      setStakeForm(prev => ({ ...prev, processing: false }));
    }
  };

  // Request withdrawal of staked tokens
  const requestWithdrawal = async (stake: UserStake) => {
    try {
      const response = await fetch('/staking/withdraw/request', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          account: selectedAccount === 'custom' ? customAccountId : selectedAccount,
          node_id: stake.node_id,
          amount: stake.amount
        })
      });
      
      if (response.ok) {
        alert('Withdrawal request submitted successfully');
        // Refresh user stakes
        loadUserData();
      } else {
        const error = await response.json();
        alert(`Withdrawal request failed: ${error.error || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Withdrawal request error:', error);
      alert('An unexpected error occurred');
    }
  };

  // Execute withdrawal of staked tokens
  const executeWithdrawal = async (stake: UserStake) => {
    try {
      const response = await fetch('/staking/withdraw/execute', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          account: selectedAccount === 'custom' ? customAccountId : selectedAccount,
          node_id: stake.node_id
        })
      });
      
      if (response.ok) {
        alert('Withdrawal executed successfully');
        // Refresh data
        await Promise.all([
          fetchNetworkStats(),
          fetchNodeStakes(),
          loadUserData()
        ]);
      } else {
        const error = await response.json();
        alert(`Withdrawal execution failed: ${error.error || 'Unknown error'}`);
      }
    } catch (error) {
      console.error('Withdrawal execution error:', error);
      alert('An unexpected error occurred');
    }
  };

  // Helper functions
  const formatTokenAmount = (amount: string) => {
    if (!amount) return '0 LORE';
    
    // Convert from smallest unit (10^18) to LORE
    const amountBigInt = BigInt(amount);
    const loreAmount = Number(amountBigInt) / Math.pow(10, 18);
    
    return `${loreAmount.toLocaleString(undefined, { maximumFractionDigits: 4 })} LORE`;
  };

  const formatDate = (dateString: string) => {
    if (!dateString) return '';
    return new Date(dateString).toLocaleString();
  };

  const formatNodeId = (nodeId: string) => {
    if (!nodeId) return '';
    if (nodeId.length <= 12) return nodeId;
    return `${nodeId.substring(0, 6)}...${nodeId.substring(nodeId.length - 6)}`;
  };

  const formatAccountId = (accountId: string) => {
    if (!accountId) return '';
    if (accountId === 'network') return 'Network';
    if (accountId === 'staking_pool') return 'Staking Pool';
    return accountId;
  };

  const formatTransactionType = (type: string) => {
    if (!type) return '';
    
    // Remove "TransactionType" prefix if present
    type = type.replace('TransactionType', '');
    
    // Convert snake_case to Title Case
    return type.split('_')
      .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
      .join(' ');
  };

  const formatQueryType = (type: string) => {
    if (!type) return '';
    
    // Convert camelCase to Title Case
    return type.replace(/([A-Z])/g, ' $1')
      .replace(/^./, str => str.toUpperCase());
  };

  return (
    <div className="max-w-7xl mx-auto p-6 bg-slate-900 text-white min-h-screen">
      <h1 className="text-3xl font-bold mb-6 text-white">Economic Monitoring Dashboard</h1>
      
      {/* Network Statistics Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Network Statistics</h2>
        {loading.networkStats ? (
          <div className="flex justify-center items-center h-24 text-slate-400 italic">Loading network statistics...</div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Total Supply</div>
              <div className="text-xl font-semibold text-white">{formatTokenAmount(networkStats.total_supply)}</div>
            </div>
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Active Nodes</div>
              <div className="text-xl font-semibold text-white">{networkStats.active_nodes}</div>
            </div>
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Total Staked</div>
              <div className="text-xl font-semibold text-white">{formatTokenAmount(networkStats.total_staked)}</div>
            </div>
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Avg. Reputation</div>
              <div className="text-xl font-semibold text-white">{networkStats.avg_reputation ? networkStats.avg_reputation.toFixed(2) : '0.00'}</div>
            </div>
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Total Accounts</div>
              <div className="text-xl font-semibold text-white">{networkStats.total_accounts}</div>
            </div>
            <div className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
              <div className="text-sm text-slate-300 mb-2">Total Transactions</div>
              <div className="text-xl font-semibold text-white">{networkStats.total_transactions}</div>
            </div>
          </div>
        )}

        {/* Transaction Types Chart */}
        {!loading.networkStats && chartData.transactionTypes.labels.length > 0 && (
          <div className="h-80 mt-6 bg-slate-700 rounded-md p-4 border border-slate-600">
            <h3 className="text-lg font-semibold text-white mb-4">Transaction Distribution</h3>
            <canvas ref={transactionTypesChartRef}></canvas>
          </div>
        )}
      </div>

      {/* Node Stakes Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Node Stakes</h2>
        {loading.nodeStakes ? (
          <div className="flex justify-center items-center h-24 text-slate-400 italic">Loading node stakes...</div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 mb-6">
              {Object.entries(nodeStakes).map(([nodeId, stake]) => (
                <div key={nodeId} className="bg-slate-700 rounded-md p-4 text-center border border-slate-600">
                  <div className="text-sm text-slate-300 mb-2">{formatNodeId(nodeId)}</div>
                  <div className="text-xl font-semibold text-white">{formatTokenAmount(stake.total_user_stake)}</div>
                  <div className="text-xs text-slate-400 mt-1">
                    {stake.stake_count} stakers
                  </div>
                </div>
              ))}
            </div>
            
            {/* Node Stakes Chart */}
            {chartData.nodeStakes.labels.length > 0 && (
              <div className="h-80 bg-slate-700 rounded-md p-4 border border-slate-600">
                <canvas ref={nodeStakesChartRef}></canvas>
              </div>
            )}
          </>
        )}
      </div>

      {/* User Account Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">User Account Management</h2>
        
        <div className="flex flex-wrap gap-4 items-center mb-6">
          <label className="text-white">
            Account:
            <select 
              value={selectedAccount} 
              onChange={(e) => setSelectedAccount(e.target.value)}
              className="ml-2 px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="default_user">Default User</option>
              <option value="test_user">Test User</option>
              <option value="custom">Custom Account ID</option>
            </select>
          </label>
          
          {selectedAccount === 'custom' && (
            <input
              type="text"
              placeholder="Enter account ID"
              value={customAccountId}
              onChange={(e) => setCustomAccountId(e.target.value)}
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          )}
          
          <button 
            onClick={loadUserData} 
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Load Account
          </button>
        </div>

        {loading.userAccount ? (
          <div className="flex justify-center items-center h-24 text-slate-400 italic">Loading account...</div>
        ) : userAccount ? (
          <div className="mt-4">
            <h3 className="text-lg font-semibold text-white mb-3">Account Details</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 text-sm">
              <div className="text-slate-300"><strong className="text-white">Address:</strong> {userAccount.address}</div>
              <div className="text-slate-300"><strong className="text-white">Balance:</strong> {userAccount.balance_formatted} {userAccount.token_symbol}</div>
              <div className="text-slate-300"><strong className="text-white">Total Spent:</strong> {formatTokenAmount(userAccount.total_spent)}</div>
              <div className="text-slate-300"><strong className="text-white">Queries:</strong> {userAccount.queries_submitted}</div>
              <div className="text-slate-300"><strong className="text-white">Created:</strong> {formatDate(userAccount.created_at)}</div>
            </div>
          </div>
        ) : (
          <div className="text-center text-slate-400 italic p-8">No account data found</div>
        )}
      </div>

      {/* Token Transfer Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Token Transfer</h2>
        <form onSubmit={performTransfer} className="mt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <input
              type="text"
              placeholder="From Account"
              value={transferForm.fromAccount}
              onChange={(e) => setTransferForm(prev => ({...prev, fromAccount: e.target.value}))}
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="text"
              placeholder="To Account"
              value={transferForm.toAccount}
              onChange={(e) => setTransferForm(prev => ({...prev, toAccount: e.target.value}))}
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="number"
              placeholder="Amount (LORE)"
              value={transferForm.amount}
              onChange={(e) => setTransferForm(prev => ({...prev, amount: e.target.value}))}
              step="0.0001"
              min="0"
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="text"
              placeholder="Description (optional)"
              value={transferForm.description}
              onChange={(e) => setTransferForm(prev => ({...prev, description: e.target.value}))}
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button 
            type="submit" 
            disabled={transferForm.processing}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {transferForm.processing ? 'Processing...' : 'Transfer Tokens'}
          </button>
          {transferForm.error && <div className="mt-2 p-3 bg-red-900 border border-red-700 text-red-300 rounded-md">{transferForm.error}</div>}
          {transferForm.success && <div className="mt-2 p-3 bg-green-900 border border-green-700 text-green-300 rounded-md">{transferForm.success}</div>}
        </form>
      </div>

      {/* Staking Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Token Staking</h2>
        <form onSubmit={performStake} className="mt-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
            <input
              type="text"
              placeholder="Account"
              value={stakeForm.account}
              onChange={(e) => setStakeForm(prev => ({...prev, account: e.target.value}))}
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="text"
              placeholder="Node ID"
              value={stakeForm.nodeId}
              onChange={(e) => setStakeForm(prev => ({...prev, nodeId: e.target.value}))}
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <input
              type="number"
              placeholder="Amount (LORE)"
              value={stakeForm.amount}
              onChange={(e) => setStakeForm(prev => ({...prev, amount: e.target.value}))}
              step="0.0001"
              min="0"
              required
              className="px-3 py-2 bg-slate-700 border border-slate-600 rounded-md text-white placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <button 
            type="submit" 
            disabled={stakeForm.processing}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-slate-600 text-white rounded-md font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {stakeForm.processing ? 'Processing...' : 'Stake Tokens'}
          </button>
          {stakeForm.error && <div className="mt-2 p-3 bg-red-900 border border-red-700 text-red-300 rounded-md">{stakeForm.error}</div>}
          {stakeForm.success && <div className="mt-2 p-3 bg-green-900 border border-green-700 text-green-300 rounded-md">{stakeForm.success}</div>}
        </form>

        {/* User Stakes */}
        {userStakes.length > 0 && (
          <div className="mt-8">
            <h3 className="text-lg font-semibold text-white mb-4">Your Stakes</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {userStakes.map((stake) => (
                <div key={stake.id} className="bg-slate-700 border border-slate-600 rounded-md p-4">
                  <div className="text-slate-300 mb-1"><strong className="text-white">Node:</strong> {formatNodeId(stake.node_id)}</div>
                  <div className="text-slate-300 mb-1"><strong className="text-white">Amount:</strong> {formatTokenAmount(stake.amount)}</div>
                  <div className="text-slate-300 mb-3"><strong className="text-white">Staked:</strong> {formatDate(stake.staked_at)}</div>
                  <div>
                    {!stake.withdrawal_requested ? (
                      <button 
                        onClick={() => requestWithdrawal(stake)}
                        className="px-3 py-1 bg-slate-600 hover:bg-slate-500 text-white text-sm rounded transition-colors"
                      >
                        Request Withdrawal
                      </button>
                    ) : (
                      <button 
                        onClick={() => executeWithdrawal(stake)}
                        className="px-3 py-1 bg-slate-600 hover:bg-slate-500 text-white text-sm rounded transition-colors"
                      >
                        Execute Withdrawal
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Rewards Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Reward Distribution</h2>
        {loading.rewards ? (
          <div className="flex justify-center items-center h-24 text-slate-400 italic">Loading rewards...</div>
        ) : (
          <>
            {/* Rewards Chart */}
            {chartData.rewards.labels.length > 0 && (
              <div className="h-80 bg-slate-700 rounded-md p-4 border border-slate-600 mb-6">
                <canvas ref={rewardsChartRef}></canvas>
              </div>
            )}
            
            {/* Recent Rewards */}
            {rewardEvents.length > 0 && (
              <div className="mt-6">
                <h3 className="text-lg font-semibold text-white mb-4">Recent Rewards</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {rewardEvents.slice(0, 10).map((reward) => (
                    <div key={reward.id} className="bg-blue-900 bg-opacity-30 border border-blue-700 rounded-md p-4">
                      <div className="text-slate-300 mb-1"><strong className="text-white">Node:</strong> {formatNodeId(reward.node_id)}</div>
                      <div className="text-slate-300 mb-1"><strong className="text-white">Amount:</strong> {formatTokenAmount(reward.amount)}</div>
                      <div className="text-slate-300 mb-1"><strong className="text-white">Reason:</strong> {reward.reason.replace(/_/g, ' ')}</div>
                      <div className="text-slate-300 mb-1"><strong className="text-white">Time:</strong> {formatDate(reward.timestamp)}</div>
                      {reward.performance_data && (
                        <div className="text-slate-300"><strong className="text-white">Performance:</strong> {(reward.performance_data.performance_score * 100).toFixed(1)}%</div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>

      {/* Transactions Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">Recent Transactions</h2>
        {loading.transactions ? (
          <div className="flex justify-center items-center h-24 text-slate-400 italic">Loading transactions...</div>
        ) : userTransactions.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {userTransactions.map((tx) => (
              <div key={tx.id} className="bg-slate-700 border border-slate-600 rounded-md p-4">
                <div className="text-slate-300 mb-1"><strong className="text-white">Type:</strong> {formatTransactionType(tx.type)}</div>
                <div className="text-slate-300 mb-1"><strong className="text-white">Amount:</strong> {formatTokenAmount(tx.amount)}</div>
                <div className="text-slate-300 mb-1"><strong className="text-white">From:</strong> {formatAccountId(tx.from_id)}</div>
                <div className="text-slate-300 mb-1"><strong className="text-white">To:</strong> {formatAccountId(tx.to_id)}</div>
                <div className="text-slate-300 mb-1"><strong className="text-white">Status:</strong> {tx.status}</div>
                <div className="text-slate-300 mb-1"><strong className="text-white">Time:</strong> {formatDate(tx.timestamp)}</div>
                {tx.description && <div className="text-slate-300"><strong className="text-white">Description:</strong> {tx.description}</div>}
              </div>
            ))}
          </div>
        ) : (
          <div className="text-center text-slate-400 italic p-8">No transactions found</div>
        )}
      </div>

      {/* Pricing Tiers Section */}
      <div className="bg-slate-800 rounded-lg shadow-lg p-6 mb-6 border border-slate-700">
        <h2 className="text-2xl font-semibold mb-4 text-white">AI Query Pricing Tiers</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {pricingTiers.map((tier) => (
            <div key={tier.id} className="bg-slate-700 border-2 border-slate-600 rounded-lg p-6 shadow-lg">
              <h3 className="text-xl font-semibold text-white mb-2">{tier.name}</h3>
              <div className="text-lg font-semibold text-blue-400 mb-4">Base Price: {tier.basePrice} LORE</div>
              <p className="text-slate-300 mb-4">{tier.description}</p>
              
              <div className="mb-4">
                <h4 className="text-white font-medium mb-2">Features:</h4>
                <ul className="list-disc list-inside text-sm text-slate-300 space-y-1">
                  {tier.features.map((feature, index) => (
                    <li key={index}>{feature}</li>
                  ))}
                </ul>
              </div>
              
              <div className="border-t border-slate-600 pt-4">
                <h4 className="text-white font-medium mb-2">Query Pricing:</h4>
                {Object.entries(tier.queryTypes).map(([type, price]) => (
                  <div key={type} className="flex justify-between py-1 text-sm text-slate-300">
                    <span>{formatQueryType(type)}:</span>
                    <span className="text-white font-medium">{price} LORE</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};


export default EconomicDashboard;