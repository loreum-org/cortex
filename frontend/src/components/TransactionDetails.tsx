import { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import { ArrowLeft, Hash, Clock, User, Coins, GitBranch, CheckCircle, AlertCircle } from 'lucide-react';

interface TransactionData {
  id: string;
  type: string;
  data: string;
  parentIds: string[];
  signature: string;
  metadata: Record<string, string>;
  finalized: boolean;
  timestamp: number;
  economicData?: {
    fromAccount: string;
    toAccount: string;
    amount: string;
    fee: string;
    economicType: string;
    queryId?: string;
    modelTier?: string;
    qualityScore?: number;
    responseTime?: number;
  };
  agiData?: {
    nodeId: string;
    agiVersion: number;
    intelligenceLevel: number;
    domainKnowledge: Record<string, number>;
    updateType: string;
  };
}

export function TransactionDetails() {
  const { txId } = useParams<{ txId: string }>();
  const [transaction, setTransaction] = useState<TransactionData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (txId) {
      fetchTransactionDetails(txId);
    }
  }, [txId]);

  const fetchTransactionDetails = async (id: string) => {
    try {
      const response = await fetch(`/api/transactions/${id}`);
      if (response.ok) {
        const tx = await response.json();
        
        // Transform backend transaction format to frontend format
        const transformedTx = {
          id: tx.ID || tx.id || id,
          type: tx.Type || tx.type || 'Unknown',
          data: tx.Data || tx.data || '',
          parentIds: tx.ParentIDs || tx.parentIds || [],
          signature: tx.Signature || tx.signature || '',
          metadata: tx.Metadata || tx.metadata || {},
          finalized: tx.Finalized !== undefined ? tx.Finalized : tx.finalized !== undefined ? tx.finalized : true,
          timestamp: tx.Timestamp || tx.timestamp || Date.now() / 1000,
          economicData: tx.FromID && tx.ToID ? {
            fromAccount: tx.FromID || tx.from,
            toAccount: tx.ToID || tx.to,
            amount: tx.Amount ? tx.Amount.toString() : tx.amount || '0',
            fee: tx.Fee ? tx.Fee.toString() : tx.fee || '0',
            economicType: tx.TransactionType || tx.economicType || 'transfer',
            queryId: tx.QueryID || tx.queryId,
            modelTier: tx.ModelTier || tx.modelTier,
            qualityScore: tx.QualityScore || tx.qualityScore,
            responseTime: tx.ResponseTime || tx.responseTime
          } : undefined,
          agiData: tx.AGIData || tx.agiData
        };
        
        setTransaction(transformedTx);
      } else {
        setError('Transaction not found');
      }
    } catch (err) {
      setError('Failed to fetch transaction details');
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

  const getTransactionTypeColor = (type: string) => {
    switch (type) {
      case 'Economic': return 'text-green-400 bg-green-400/10';
      case 'Query': return 'text-blue-400 bg-blue-400/10';
      case 'Reward': return 'text-yellow-400 bg-yellow-400/10';
      case 'AGIUpdate': return 'text-purple-400 bg-purple-400/10';
      case 'AGIEvolution': return 'text-pink-400 bg-pink-400/10';
      default: return 'text-gray-400 bg-gray-400/10';
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

  if (error || !transaction) {
    return (
      <div className="p-6">
        <div className="text-center">
          <AlertCircle className="mx-auto h-12 w-12 text-red-400 mb-4" />
          <h2 className="text-xl font-semibold mb-2">Transaction Not Found</h2>
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
          <h1 className="text-2xl font-bold">Transaction Details</h1>
        </div>
        <div className="flex items-center space-x-2">
          {transaction.finalized ? (
            <CheckCircle className="h-5 w-5 text-green-400" />
          ) : (
            <AlertCircle className="h-5 w-5 text-yellow-400" />
          )}
          <span className={`px-3 py-1 rounded-full text-sm font-semibold ${
            transaction.finalized 
              ? 'bg-green-400/10 text-green-400' 
              : 'bg-yellow-400/10 text-yellow-400'
          }`}>
            {transaction.finalized ? 'Finalized' : 'Pending'}
          </span>
        </div>
      </div>

      {/* Basic Info */}
      <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
        <h2 className="text-lg font-semibold mb-4">Transaction Overview</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Transaction ID
              </label>
              <div className="flex items-center space-x-2">
                <Hash className="h-4 w-4 text-tesla-text-gray" />
                <span className="font-mono text-sm break-all">{transaction.id}</span>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Type
              </label>
              <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-semibold ${getTransactionTypeColor(transaction.type)}`}>
                {transaction.type}
              </span>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Timestamp
              </label>
              <div className="flex items-center space-x-2">
                <Clock className="h-4 w-4 text-tesla-text-gray" />
                <span className="text-sm">{formatTimestamp(transaction.timestamp)}</span>
              </div>
            </div>
          </div>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Parent Transactions
              </label>
              <div className="space-y-1">
                {transaction.parentIds.length > 0 ? (
                  transaction.parentIds.map((parentId, index) => (
                    <Link
                      key={index}
                      to={`/explorer/transaction/${parentId}`}
                      className="block text-sm font-mono text-blue-400 hover:text-blue-300 break-all"
                    >
                      {parentId}
                    </Link>
                  ))
                ) : (
                  <span className="text-sm text-tesla-text-gray">Genesis Transaction</span>
                )}
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                DAG Position
              </label>
              <div className="flex items-center space-x-2">
                <GitBranch className="h-4 w-4 text-tesla-text-gray" />
                <span className="text-sm">Level {transaction.parentIds.length}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Economic Data */}
      {transaction.economicData && (
        <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
          <h2 className="text-lg font-semibold mb-4">Economic Details</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                From Account
              </label>
              <Link
                to={`/explorer/account/${transaction.economicData.fromAccount}`}
                className="flex items-center space-x-2 text-blue-400 hover:text-blue-300"
              >
                <User className="h-4 w-4" />
                <span className="font-mono text-sm break-all">
                  {transaction.economicData.fromAccount}
                </span>
              </Link>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                To Account
              </label>
              <Link
                to={`/explorer/account/${transaction.economicData.toAccount}`}
                className="flex items-center space-x-2 text-blue-400 hover:text-blue-300"
              >
                <User className="h-4 w-4" />
                <span className="font-mono text-sm break-all">
                  {transaction.economicData.toAccount}
                </span>
              </Link>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Amount
              </label>
              <div className="flex items-center space-x-2">
                <Coins className="h-4 w-4 text-green-400" />
                <span className="text-sm font-semibold text-green-400">
                  {formatAmount(transaction.economicData.amount)}
                </span>
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Transaction Fee
              </label>
              <span className="text-sm">{formatAmount(transaction.economicData.fee)}</span>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Economic Type
              </label>
              <span className="text-sm capitalize">{transaction.economicData.economicType}</span>
            </div>
            {transaction.economicData.modelTier && (
              <div>
                <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                  Model Tier
                </label>
                <span className="text-sm capitalize">{transaction.economicData.modelTier}</span>
              </div>
            )}
          </div>
          {transaction.economicData.qualityScore && (
            <div className="mt-4 pt-4 border-t border-tesla-border">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                    Quality Score
                  </label>
                  <span className="text-sm">{transaction.economicData.qualityScore.toFixed(2)}/10</span>
                </div>
                {transaction.economicData.responseTime && (
                  <div>
                    <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                      Response Time
                    </label>
                    <span className="text-sm">{transaction.economicData.responseTime}ms</span>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* AGI Data */}
      {transaction.agiData && (
        <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
          <h2 className="text-lg font-semibold mb-4">AGI Details</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Node ID
              </label>
              <Link
                to={`/explorer/node/${transaction.agiData.nodeId}`}
                className="text-blue-400 hover:text-blue-300 font-mono text-sm break-all"
              >
                {transaction.agiData.nodeId}
              </Link>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                AGI Version
              </label>
              <span className="text-sm">v{transaction.agiData.agiVersion}</span>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Intelligence Level
              </label>
              <span className="text-sm font-semibold text-purple-400">
                {transaction.agiData.intelligenceLevel.toFixed(2)}
              </span>
            </div>
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Update Type
              </label>
              <span className="text-sm capitalize">{transaction.agiData.updateType}</span>
            </div>
          </div>
          {Object.keys(transaction.agiData.domainKnowledge).length > 0 && (
            <div className="mt-4 pt-4 border-t border-tesla-border">
              <label className="block text-sm font-medium text-tesla-text-gray mb-2">
                Domain Knowledge
              </label>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                {Object.entries(transaction.agiData.domainKnowledge).map(([domain, level]) => (
                  <div key={domain} className="flex justify-between items-center py-1">
                    <span className="text-sm capitalize">{domain}</span>
                    <span className="text-sm font-mono">{level.toFixed(2)}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Raw Data */}
      <div className="bg-tesla-dark-gray rounded-lg border border-tesla-border p-6">
        <h2 className="text-lg font-semibold mb-4">Raw Transaction Data</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Data Payload
            </label>
            <div className="bg-tesla-black rounded border border-tesla-border p-3">
              <pre className="text-sm text-tesla-text-gray font-mono whitespace-pre-wrap break-all">
                {transaction.data}
              </pre>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-tesla-text-gray mb-1">
              Signature
            </label>
            <div className="bg-tesla-black rounded border border-tesla-border p-3">
              <span className="text-sm text-tesla-text-gray font-mono break-all">
                {transaction.signature}
              </span>
            </div>
          </div>
          {Object.keys(transaction.metadata).length > 0 && (
            <div>
              <label className="block text-sm font-medium text-tesla-text-gray mb-1">
                Metadata
              </label>
              <div className="bg-tesla-black rounded border border-tesla-border p-3">
                <pre className="text-sm text-tesla-text-gray font-mono">
                  {JSON.stringify(transaction.metadata, null, 2)}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}