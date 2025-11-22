import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, User, Coins, TrendingUp, Clock, Hash, Activity, Shield, Award } from 'lucide-react';
import { MetricCard } from './MetricCard';

interface AccountData {
  id: string;
  balance: string;
  stake: string;
  reputation?: number;
  performance?: {
    successRate: number;
    avgResponseTime: number;
    totalQueries: number;
    uptime: number;
  };
  createdAt: number;
  lastActivity: number;
  transactionCount: number;
  isNode: boolean;
  accumulatedRewards?: string;
  spendingPatterns?: {
    totalSpent: string;
    avgTransactionSize: string;
    preferredModelTier: string;
  };
}

interface Transaction {
  id: string;
  type: string;
  from?: string;
  to?: string;
  amount?: string;
  timestamp: number;
  finalized: boolean;
}

export function AccountDetails() {
  const { accountId } = useParams<{ accountId: string }>();
  const [account, setAccount] = useState<AccountData | null>(null);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [transactionPage] = useState(0);

  useEffect(() => {
    if (accountId) {
      fetchAccountDetails(accountId);
    }
  }, [accountId]);

  const fetchAccountDetails = async (id: string) => {
    try {
      const [accountRes, txRes] = await Promise.all([
        fetch(`/api/accounts/${id}`),
        fetch(`/api/accounts/${id}/transactions?page=${transactionPage}&limit=20`)
      ]);

      if (accountRes.ok) {
        const accountData = await accountRes.json();
        setAccount(accountData);
      } else {
        setError('Account not found');
      }

      if (txRes.ok) {
        const txData = await txRes.json();
        // Transform backend transaction format to frontend format
        const formattedTxs = txData.map((tx: any) => ({
          id: tx.ID || tx.id,
          type: tx.Type || tx.type || 'Unknown',
          from: tx.FromID || tx.from,
          to: tx.ToID || tx.to,
          amount: tx.Amount ? tx.Amount.toString() : tx.amount,
          timestamp: tx.Timestamp || tx.timestamp || Date.now() / 1000,
          finalized: tx.Finalized !== undefined ? tx.Finalized : true
        }));
        setTransactions(formattedTxs);
      }
    } catch (err) {
      setError('Failed to fetch account details');
    } finally {
      setLoading(false);
    }
  };

  const formatTimestamp = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleString();
  };

  const formatAmount = (amount: string) => {
    return `${parseFloat(amount).toFixed(6)} λore`;
  };

  const getReputationColor = (reputation: number) => {
    if (reputation >= 80) return 'text-green-400';
    if (reputation >= 60) return 'text-yellow-400';
    if (reputation >= 40) return 'text-orange-400';
    return 'text-red-400';
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

  if (error || !account) {
    return (
      <div className="p-6">
        <div className="text-center">
          <User className="mx-auto h-12 w-12 text-red-400 mb-4" />
          <h2 className="text-xl font-semibold mb-2">Account Not Found</h2>
          <p className="text-tesla-text-gray mb-4">{error}</p>
          <Link
            to="/explorer"
            className="inline-flex items-center px-4 py-2 bg-tesla-white text-tesla-black rounded-lg hover:bg-gray-200 transition-colors"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Explorer
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/explorer"
            className="p-2 text-tesla-text-gray hover:text-tesla-white hover:bg-tesla-medium-gray rounded-lg transition-colors"
          >
            <ArrowLeft className="h-5 w-5" />
          </Link>
          <div>
            <h1 className="text-2xl font-bold">Account Details</h1>
            <p className="text-tesla-text-gray font-mono text-sm">{account.id}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {account.isNode && (
            <span className="px-3 py-1 bg-blue-400/10 text-blue-400 rounded-full text-sm font-semibold">
              Node Account
            </span>
          )}
          <span className="px-3 py-1 bg-green-400/10 text-green-400 rounded-full text-sm font-semibold">
            Active
          </span>
        </div>
      </div>

      {/* Account Overview */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <MetricCard
          title="Balance"
          icon={<Coins className="h-5 w-5" />}
        >
          <div className="text-2xl font-bold">{formatAmount(account.balance)}</div>
          <div className="text-sm text-green-400 mt-1">+5.2%</div>
        </MetricCard>
        <MetricCard
          title="Staked Amount"
          icon={<Shield className="h-5 w-5" />}
        >
          <div className="text-2xl font-bold">{formatAmount(account.stake)}</div>
          <div className="text-sm text-green-400 mt-1">+12.8%</div>
        </MetricCard>
        {account.reputation !== undefined && (
          <MetricCard
            title="Reputation"
            icon={<TrendingUp className="h-5 w-5" />}
          >
            <div className={`text-2xl font-bold ${getReputationColor(account.reputation)}`}>{account.reputation.toFixed(2)}</div>
            <div className="text-sm text-green-400 mt-1">+3.1%</div>
          </MetricCard>
        )}
        <MetricCard
          title="Transactions"
          icon={<Hash className="h-5 w-5" />}
        >
          <div className="text-2xl font-bold">{account.transactionCount.toLocaleString()}</div>
          <div className="text-sm text-green-400 mt-1">+18.5%</div>
        </MetricCard>
      </div>

      {/* Node Performance (if applicable) */}
      {account.isNode && account.performance && (
        <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center">
            <Activity className="h-5 w-5 mr-2" />
            Node Performance
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div className="text-center">
              <div className="text-2xl font-bold text-green-400">
                {(account.performance.successRate * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-tesla-text-gray">Success Rate</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-400">
                {account.performance.avgResponseTime}ms
              </div>
              <div className="text-sm text-tesla-text-gray">Avg Response Time</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-400">
                {account.performance.totalQueries.toLocaleString()}
              </div>
              <div className="text-sm text-tesla-text-gray">Total Queries</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-yellow-400">
                {(account.performance.uptime * 100).toFixed(1)}%
              </div>
              <div className="text-sm text-tesla-text-gray">Uptime</div>
            </div>
          </div>
        </div>
      )}

      {/* Rewards & Staking */}
      {account.accumulatedRewards && (
        <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
          <h2 className="text-lg font-semibold mb-4 flex items-center">
            <Award className="h-5 w-5 mr-2" />
            Rewards & Staking
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Accumulated Rewards
              </label>
              <div className="text-xl font-semibold text-yellow-400">
                {formatAmount(account.accumulatedRewards)}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Total Staked
              </label>
              <div className="text-xl font-semibold text-blue-400">
                {formatAmount(account.stake)}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Staking Yield
              </label>
              <div className="text-xl font-semibold text-green-400">8.5% APY</div>
            </div>
          </div>
        </div>
      )}

      {/* Spending Patterns (for user accounts) */}
      {!account.isNode && account.spendingPatterns && (
        <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
          <h2 className="text-lg font-semibold mb-4">Usage Patterns</h2>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Total Spent
              </label>
              <div className="text-xl font-semibold">
                {formatAmount(account.spendingPatterns.totalSpent)}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Avg Transaction Size
              </label>
              <div className="text-xl font-semibold">
                {formatAmount(account.spendingPatterns.avgTransactionSize)}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Preferred Model Tier
              </label>
              <div className="text-xl font-semibold capitalize">
                {account.spendingPatterns.preferredModelTier}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Account Information */}
      <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
        <h2 className="text-lg font-semibold mb-4">Account Information</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Account ID
            </label>
            <div className="font-mono text-sm break-all bg-tesla-black rounded border border-tesla-border p-2">
              {account.id}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Account Type
            </label>
            <div className="text-sm">
              {account.isNode ? 'Node Account' : 'User Account'}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Created
            </label>
            <div className="flex items-center space-x-2">
              <Clock className="h-4 w-4 text-tesla-text-gray" />
              <span className="text-sm">{formatTimestamp(account.createdAt)}</span>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Last Activity
            </label>
            <div className="flex items-center space-x-2">
              <Activity className="h-4 w-4 text-tesla-text-gray" />
              <span className="text-sm">{formatTimestamp(account.lastActivity)}</span>
            </div>
          </div>
        </div>
      </div>

      {/* Recent Transactions */}
      <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border">
        <div className="p-6 border-b border-tesla-border">
          <h2 className="text-lg font-semibold">Recent Transactions</h2>
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
                  Direction
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
              {transactions.map((tx) => (
                <tr key={tx.id} className="hover:bg-tesla-medium-gray">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <Link
                      to={`/explorer/transaction/${tx.id}`}
                      className="text-sm font-mono text-blue-400 hover:text-blue-300"
                    >
                      {tx.id.substring(0, 12)}...
                    </Link>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`text-sm font-medium ${getTransactionTypeColor(tx.type)}`}>
                      {tx.type}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    {tx.from === account.id ? (
                      <span className="text-red-400">Outgoing</span>
                    ) : tx.to === account.id ? (
                      <span className="text-green-400">Incoming</span>
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