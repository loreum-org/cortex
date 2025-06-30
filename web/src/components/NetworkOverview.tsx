import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { 
  Network, 
  Server, 
  Activity, 
  Users, 
  Zap,
  AlertCircle,
  CheckCircle,
  XCircle,
  ExternalLink,
  RefreshCw,
  Settings
} from 'lucide-react';
import { cortexAPI } from '../services/api';
import type { 
  NetworkOverview as NetworkOverviewType, 
  TransactionData,
  NodeInfo
} from '../services/api';
import type { NetworkServiceStats } from '../types/services';
import { MetricCard } from './MetricCard';

interface NetworkState {
  overview: NetworkOverviewType | null;
  transactions: TransactionData[];
  services: NetworkServiceStats | null;
  loading: boolean;
  error: string | null;
  lastUpdated: Date | null;
}

export function NetworkOverview() {
  const [state, setState] = useState<NetworkState>({
    overview: null,
    transactions: [],
    services: null,
    loading: true,
    error: null,
    lastUpdated: null,
  });

  const [autoRefresh, setAutoRefresh] = useState(true);

  const fetchNetworkData = async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }));

      const [overview, transactions, services] = await Promise.all([
        cortexAPI.getNetworkOverview(),
        cortexAPI.getNetworkTransactions(),
        cortexAPI.getNetworkServices().catch(() => null), // Services might not be available
      ]);

      setState(prev => ({
        ...prev,
        overview,
        transactions,
        services,
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
    fetchNetworkData();
  }, []);

  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(fetchNetworkData, 3000);
    return () => clearInterval(interval);
  }, [autoRefresh]);

  const getStatusIcon = (status: NodeInfo['status']) => {
    switch (status) {
      case 'online':
        return <CheckCircle className="h-4 w-4 text-tesla-accent-green" />;
      case 'offline':
        return <XCircle className="h-4 w-4 text-tesla-accent-red" />;
      default:
        return <AlertCircle className="h-4 w-4 text-tesla-text-gray" />;
    }
  };

  const getHealthColor = (health: number) => {
    if (health >= 80) return 'text-tesla-accent-green';
    if (health >= 60) return 'text-tesla-text-gray';
    return 'text-tesla-accent-red';
  };

  if (state.loading && !state.overview) {
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
        <h1 className="text-2xl font-medium text-tesla-white uppercase tracking-wide">NETWORK OVERVIEW</h1>
        <div className="flex items-center gap-6">
          {state.lastUpdated && (
            <span className="text-xs text-tesla-text-gray uppercase tracking-wider">
              LAST UPDATED: {state.lastUpdated.toLocaleTimeString()}
            </span>
          )}
          <label className="flex items-center gap-3">
            <input
              type="checkbox"
              checked={autoRefresh}
              onChange={(e) => setAutoRefresh(e.target.checked)}
              className="w-4 h-4 bg-tesla-dark-gray border border-tesla-border focus:ring-0 focus:ring-offset-0"
            />
            <span className="text-xs text-tesla-text-gray uppercase tracking-wider">AUTO-REFRESH</span>
          </label>
          <button
            onClick={fetchNetworkData}
            className="tesla-button text-xs flex items-center gap-2"
            disabled={state.loading}
          >
            <RefreshCw className={`h-4 w-4 ${state.loading ? 'animate-spin' : ''}`} />
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

      {/* Network Stats */}
      {state.overview && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="Network Health"
            icon={<Network className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className={`text-3xl font-light ${getHealthColor(state.overview.networkHealth)}`}>
                {state.overview.networkHealth.toFixed(1)}%
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                {state.overview.onlineNodes} OF {state.overview.totalNodes} NODES ONLINE
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Total Peers"
            icon={<Users className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-tesla-white">{state.overview.totalPeers}</div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                ACROSS ALL NODES
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Transactions"
            icon={<Activity className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-tesla-white">{state.transactions.length}</div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                RECENT NETWORK ACTIVITY
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Active Nodes"
            icon={<Server className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-tesla-accent-green">
                {state.overview.onlineNodes}
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                RUNNING NODES
              </div>
            </div>
          </MetricCard>
        </div>
      )}

      {/* Services Overview */}
      {state.services && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
          <MetricCard
            title="Total Services"
            icon={<Settings className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-tesla-white">{state.services.total}</div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                REGISTERED SERVICES
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Agents"
            icon={<Users className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-blue-400">
                {state.services.by_type.agent?.length || 0}
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                AGENT SERVICES
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Models"
            icon={<Activity className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-purple-400">
                {state.services.by_type.model?.length || 0}
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                MODEL SERVICES
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Tools & Sensors"
            icon={<Zap className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-orange-400">
                {(state.services.by_type.tool?.length || 0) + (state.services.by_type.sensor?.length || 0)}
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                UTILITY SERVICES
              </div>
            </div>
          </MetricCard>
        </div>
      )}

      {/* Node List */}
      {state.overview && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <MetricCard title="Network Nodes" icon={<Server className="h-6 w-6" />}>
            <div className="space-y-2">
              {state.overview.nodes.map((node) => (
                <div
                  key={node.nodeId}
                  className="flex items-center justify-between p-4 border border-tesla-border hover:border-tesla-light-gray transition-all duration-150 bg-tesla-medium-gray"
                >
                  <div className="flex items-center gap-3">
                    {getStatusIcon(node.status)}
                    <div>
                      <div className="font-medium text-tesla-white text-sm">{node.name}</div>
                      <div className="text-xs text-tesla-text-gray uppercase tracking-wider">PORT: {node.port}</div>
                    </div>
                  </div>
                  <div className="flex items-center gap-3">
                    <div className={`px-3 py-1 text-xs font-medium uppercase tracking-wider ${
                      node.status === 'online' 
                        ? 'bg-tesla-accent-green text-tesla-black' 
                        : 'bg-tesla-accent-red text-tesla-white'
                    }`}>
                      {node.status}
                    </div>
                    <Link
                      to={`/dashboard?port=${node.port}`}
                      className="p-2 hover:bg-tesla-light-gray transition-all duration-150 text-tesla-white"
                      title="View node details"
                    >
                      <ExternalLink className="h-4 w-4" />
                    </Link>
                  </div>
                </div>
              ))}
            </div>
          </MetricCard>

          {/* Recent Transactions */}
          <MetricCard title="Recent Transactions" icon={<Zap className="h-6 w-6" />}>
            <div className="space-y-2 max-h-96 overflow-y-auto">
              {state.transactions.length === 0 ? (
                <div className="text-center text-tesla-text-gray py-8">
                  <div className="text-xs uppercase tracking-wider">NO TRANSACTIONS FOUND</div>
                </div>
              ) : (
                state.transactions.slice(0, 10).map((tx) => (
                  <div
                    key={tx.id}
                    className="flex items-center justify-between p-3 border border-tesla-border bg-tesla-medium-gray hover:border-tesla-light-gray transition-all duration-150"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <div className={`w-2 h-2 ${
                          tx.status === 'confirmed' ? 'bg-tesla-accent-green' : 
                          tx.status === 'pending' ? 'bg-tesla-text-gray' : 'bg-tesla-accent-red'
                        }`} />
                        <span className="text-sm font-medium truncate text-tesla-white uppercase tracking-wider">{tx.type}</span>
                      </div>
                      <div className="text-xs text-tesla-text-gray">
                        {tx.node_id} • {new Date(tx.timestamp).toLocaleTimeString()}
                      </div>
                    </div>
                    <div className={`px-3 py-1 text-xs font-medium uppercase tracking-wider ${
                      tx.status === 'confirmed' 
                        ? 'bg-tesla-accent-green text-tesla-black' 
                        : tx.status === 'pending'
                        ? 'bg-tesla-text-gray text-tesla-white'
                        : 'bg-tesla-accent-red text-tesla-white'
                    }`}>
                      {tx.status}
                    </div>
                  </div>
                ))
              )}
            </div>
          </MetricCard>
        </div>
      )}
    </div>
  );
}