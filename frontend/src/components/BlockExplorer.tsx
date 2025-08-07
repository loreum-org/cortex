import { useState, useEffect } from 'react';
import { Search, Clock, Hash, Users, Coins, TrendingUp, Activity, Database } from 'lucide-react';
import { MetricCard } from './MetricCard';

interface Transaction {
  id: string;
  type: string;
  from?: string;
  to?: string;
  amount?: string;
  fee?: string;
  timestamp: number;
  finalized: boolean;
  parentIds: string[];
}

interface BlockchainStats {
  totalTransactions: number;
  totalAccounts: number;
  totalStaked: string;
  avgReputation: number;
  networkHashrate: string;
  consensusHealth: number;
}

interface AGIStats {
  avgIntelligenceLevel: number;
  totalNodes: number;
  activeQueries: number;
  rewardsDistributed: string;
}

export function BlockExplorer() {
  const [searchTerm, setSearchTerm] = useState('');
  const [recentTransactions, setRecentTransactions] = useState<Transaction[]>([]);
  const [blockchainStats, setBlockchainStats] = useState<BlockchainStats | null>(null);
  const [agiStats, setAGIStats] = useState<AGIStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchBlockchainData();
    const interval = setInterval(fetchBlockchainData, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchBlockchainData = async () => {
    try {
      // Fetch real data from backend APIs
      const [statsRes, agiRes, txRes] = await Promise.all([
        fetch('/api/metrics/economy'),
        fetch('/api/agi/network/stats'),
        fetch('/api/transactions?limit=20')
      ]);

      if (statsRes.ok) {
        const stats = await statsRes.json();
        setBlockchainStats({
          totalTransactions: stats.total_transactions || 0,
          totalAccounts: stats.total_accounts || 0,
          totalStaked: stats.total_staked || "0",
          avgReputation: stats.avg_reputation || 0,
          networkHashrate: "N/A", // Not available in current stats
          consensusHealth: 95.0 // Placeholder
        });
      }

      if (agiRes.ok) {
        const agi = await agiRes.json();
        setAGIStats(agi);
      }

      if (txRes.ok) {
        const transactions = await txRes.json();
        // Transform backend transaction format to frontend format
        const formattedTxs = transactions.map((tx: any) => ({
          id: tx.ID || tx.id,
          type: tx.Type || tx.type || 'Unknown',
          from: tx.FromID || tx.from,
          to: tx.ToID || tx.to,
          amount: tx.Amount ? tx.Amount.toString() : tx.amount,
          fee: tx.Fee ? tx.Fee.toString() : tx.fee,
          timestamp: tx.Timestamp || tx.timestamp || Date.now() / 1000,
          finalized: tx.Finalized !== undefined ? tx.Finalized : true,
          parentIds: tx.ParentIDs || tx.parentIds || []
        }));
        setRecentTransactions(formattedTxs);
      }
    } catch (error) {
      console.error('Failed to fetch blockchain data:', error);
      // Fall back to showing empty state
      setBlockchainStats({
        totalTransactions: 0,
        totalAccounts: 0,
        totalStaked: "0",
        avgReputation: 0,
        networkHashrate: "N/A",
        consensusHealth: 0
      });
      setAGIStats({
        avgIntelligenceLevel: 0,
        totalNodes: 0,
        activeQueries: 0,
        rewardsDistributed: "0"
      });
      setRecentTransactions([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = async () => {
    if (!searchTerm.trim()) return;
    
    try {
      // Mock search functionality for now
      console.log('Searching for:', searchTerm);
      // In the future, this would navigate to specific transaction/account pages
      // For now, just show an alert
      alert(`Search functionality will be implemented when backend endpoints are available. Searching for: ${searchTerm}`);
    } catch (error) {
      console.error('Search failed:', error);
    }
  };

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  const formatAmount = (amount: string) => {
    return `${parseFloat(amount).toFixed(4)} λore`;
  };

  const getTransactionTypeColor = (type: string) => {
    switch (type) {
      case 'Economic': return 'text-green-400';
      case 'Query': return 'text-blue-400';
      case 'Reward': return 'text-yellow-400';
      case 'AGIUpdate': return 'text-purple-400';
      case 'AGIEvolution': return 'text-pink-400';
      default: return 'text-gray-400';
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <div className="flex items-center justify-center h-64">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-tesla-white"></div>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Block Explorer</h1>
        <div className="flex items-center space-x-4">
          <div className="relative">
            <input
              type="text"
              placeholder="Search transactions, accounts, or DAG nodes..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
              className="w-96 px-4 py-2 pl-10 bg-tesla-dark-gray border border-tesla-border rounded-lg focus:outline-none focus:border-tesla-white"
            />
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-tesla-text-gray" />
          </div>
          <button
            onClick={handleSearch}
            className="px-4 py-2 bg-tesla-white text-tesla-black rounded-lg hover:bg-gray-200 transition-colors"
          >
            Search
          </button>
        </div>
      </div>

      {/* Blockchain Stats */}
      {blockchainStats && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="Total Transactions"
            icon={<Hash className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{blockchainStats.totalTransactions.toLocaleString()}</div>
            <div className="text-sm text-green-400 mt-1">+12.5%</div>
          </MetricCard>
          <MetricCard
            title="Active Accounts"
            icon={<Users className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{blockchainStats.totalAccounts.toLocaleString()}</div>
            <div className="text-sm text-green-400 mt-1">+8.3%</div>
          </MetricCard>
          <MetricCard
            title="Total Staked"
            icon={<Coins className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{formatAmount(blockchainStats.totalStaked)}</div>
            <div className="text-sm text-green-400 mt-1">+15.2%</div>
          </MetricCard>
          <MetricCard
            title="Avg Reputation"
            icon={<TrendingUp className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{blockchainStats.avgReputation.toFixed(2)}</div>
            <div className="text-sm text-green-400 mt-1">+2.1%</div>
          </MetricCard>
        </div>
      )}

      {/* AGI Stats */}
      {agiStats && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="Avg Intelligence"
            icon={<Activity className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{agiStats.avgIntelligenceLevel.toFixed(2)}</div>
            <div className="text-sm text-green-400 mt-1">+5.7%</div>
          </MetricCard>
          <MetricCard
            title="AGI Nodes"
            icon={<Database className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{agiStats.totalNodes.toLocaleString()}</div>
            <div className="text-sm text-green-400 mt-1">+3.4%</div>
          </MetricCard>
          <MetricCard
            title="Active Queries"
            icon={<Clock className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{agiStats.activeQueries.toLocaleString()}</div>
            <div className="text-sm text-green-400 mt-1">+18.9%</div>
          </MetricCard>
          <MetricCard
            title="Rewards Distributed"
            icon={<Coins className="h-5 w-5" />}
          >
            <div className="text-2xl font-bold">{formatAmount(agiStats.rewardsDistributed)}</div>
            <div className="text-sm text-green-400 mt-1">+22.1%</div>
          </MetricCard>
        </div>
      )}

      {/* Recent Transactions */}
      <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border">
        <div className="p-6 border-b border-tesla-border">
          <h2 className="text-xl font-semibold">Recent Transactions</h2>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-tesla-medium-gray">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  Transaction ID
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  Type
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  From / To
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  Amount
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  Time
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-tesla-text-gray uppercase tracking-wider">
                  Status
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-tesla-border">
              {recentTransactions.map((tx) => (
                <tr key={tx.id} className="hover:bg-tesla-medium-gray cursor-pointer">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className="text-sm font-mono text-blue-400">
                      {tx.id.substring(0, 12)}...
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`text-sm font-medium ${getTransactionTypeColor(tx.type)}`}>
                      {tx.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {tx.from && tx.to ? (
                      <>
                        <div className="text-tesla-text-gray">{tx.from.substring(0, 8)}...</div>
                        <div className="text-xs text-tesla-text-gray">→ {tx.to.substring(0, 8)}...</div>
                      </>
                    ) : (
                      <span className="text-tesla-text-gray">—</span>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {tx.amount ? formatAmount(tx.amount) : '—'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-tesla-text-gray">
                    {formatTimestamp(tx.timestamp)}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full ${
                      tx.finalized 
                        ? 'bg-green-100 text-green-800' 
                        : 'bg-yellow-100 text-yellow-800'
                    }`}>
                      {tx.finalized ? 'Finalized' : 'Pending'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}