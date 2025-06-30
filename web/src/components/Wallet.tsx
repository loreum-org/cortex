import { useState, useEffect } from 'react';
import { 
  Wallet as WalletIcon, 
  Send, 
  Download, 
  History, 
  Copy,
  CheckCircle,
  AlertCircle,
  Loader2,
  ArrowUpRight,
  ArrowDownLeft,
  Clock
} from 'lucide-react';
import { cortexAPI } from '../services/api';
import type { 
  WalletAccount, 
  WalletTransferRequest, 
  WalletTransaction 
} from '../services/api';
import { MetricCard } from './MetricCard';

interface WalletState {
  account: WalletAccount | null;
  transactions: WalletTransaction[];
  loading: boolean;
  error: string | null;
  transferLoading: boolean;
  transferSuccess: string | null;
  transferError: string | null;
}

const DEFAULT_USER_ID = 'default_user';

export function Wallet() {
  const [state, setState] = useState<WalletState>({
    account: null,
    transactions: [],
    loading: true,
    error: null,
    transferLoading: false,
    transferSuccess: null,
    transferError: null,
  });

  const [transferForm, setTransferForm] = useState({
    toUserId: '',
    amount: '',
    description: '',
  });

  const [copiedAddress, setCopiedAddress] = useState(false);

  const fetchWalletData = async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }));

      const [account, transactions] = await Promise.all([
        cortexAPI.getWalletAccount(DEFAULT_USER_ID),
        cortexAPI.getWalletTransactions(DEFAULT_USER_ID, 20),
      ]);

      setState(prev => ({
        ...prev,
        account,
        transactions,
        loading: false,
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        loading: false,
        error: error instanceof Error ? error.message : 'Failed to load wallet data',
      }));
    }
  };

  useEffect(() => {
    fetchWalletData();
  }, []);

  const handleCopyAddress = async () => {
    if (state.account?.address) {
      try {
        await navigator.clipboard.writeText(state.account.address);
        setCopiedAddress(true);
        setTimeout(() => setCopiedAddress(false), 2000);
      } catch (error) {
        console.error('Failed to copy address:', error);
      }
    }
  };

  const handleTransfer = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!transferForm.toUserId || !transferForm.amount) {
      setState(prev => ({ ...prev, transferError: 'Please fill in all required fields' }));
      return;
    }

    // Convert amount to smallest unit (assuming 18 decimals like LORE)
    const amountFloat = parseFloat(transferForm.amount);
    if (isNaN(amountFloat) || amountFloat <= 0) {
      setState(prev => ({ ...prev, transferError: 'Please enter a valid amount' }));
      return;
    }

    // Convert to smallest unit (multiply by 10^18)
    const amountBigInt = (BigInt(Math.floor(amountFloat * 1e18))).toString();

    const transferRequest: WalletTransferRequest = {
      from_user_id: DEFAULT_USER_ID,
      to_user_id: transferForm.toUserId,
      amount: amountBigInt,
      description: transferForm.description || 'Wallet transfer',
    };

    try {
      setState(prev => ({ 
        ...prev, 
        transferLoading: true, 
        transferError: null, 
        transferSuccess: null 
      }));

      const response = await cortexAPI.transferTokens(transferRequest);
      
      setState(prev => ({ 
        ...prev, 
        transferLoading: false,
        transferSuccess: `Transfer successful! Transaction ID: ${response.transaction_id}`,
      }));

      // Reset form
      setTransferForm({ toUserId: '', amount: '', description: '' });
      
      // Refresh wallet data
      setTimeout(fetchWalletData, 1000);

    } catch (error) {
      setState(prev => ({
        ...prev,
        transferLoading: false,
        transferError: error instanceof Error ? error.message : 'Transfer failed',
      }));
    }
  };

  const formatTransactionType = (type: string, fromId: string) => {
    if (type === 'transfer') {
      return fromId === DEFAULT_USER_ID ? 'SENT' : 'RECEIVED';
    }
    return type.charAt(0).toUpperCase() + type.slice(1).toUpperCase();
  };

  const getTransactionIcon = (type: string, fromId: string) => {
    if (type === 'transfer') {
      return fromId === DEFAULT_USER_ID ? 
        <ArrowUpRight className="h-4 w-4 text-tesla-accent-red" /> : 
        <ArrowDownLeft className="h-4 w-4 text-tesla-accent-green" />;
    }
    return <Clock className="h-4 w-4 text-tesla-accent-blue" />;
  };

  if (state.loading && !state.account) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-tesla-black">
        <div className="animate-spin h-8 w-8 border-2 border-tesla-white border-t-transparent"></div>
      </div>
    );
  }

  return (
    <div className="max-w-6xl mx-auto px-6 py-8 space-y-8 bg-tesla-black">
      {/* Header */}
      <div className="flex items-center justify-between border-b border-tesla-border pb-6">
        <h1 className="text-2xl font-medium flex items-center gap-3 text-tesla-white uppercase tracking-wide">
          <WalletIcon className="h-6 w-6 text-tesla-white" />
          LORE WALLET
        </h1>
        <button
          onClick={fetchWalletData}
          className="tesla-button text-xs flex items-center gap-2"
          disabled={state.loading}
        >
          <Loader2 className={`h-4 w-4 ${state.loading ? 'animate-spin' : ''}`} />
          REFRESH
        </button>
      </div>

      {state.error && (
        <div className="bg-tesla-dark-gray border border-tesla-accent-red p-4 flex items-center gap-3">
          <AlertCircle className="h-5 w-5 text-tesla-accent-red" />
          <span className="text-tesla-white text-sm">ERROR: {state.error}</span>
        </div>
      )}

      {/* Wallet Overview */}
      {state.account && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <MetricCard
            title="Balance"
            icon={<WalletIcon className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-3xl font-light text-tesla-white">
                {state.account.balance_formatted}
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                RAW: {state.account.balance}
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Total Spent"
            icon={<Send className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-2xl font-light text-tesla-accent-red">
                {/* Format total spent similar to balance */}
                {(parseFloat(state.account.total_spent) / 1e18).toFixed(4)} LORE
              </div>
              <div className="text-xs text-tesla-text-gray uppercase tracking-wider">
                QUERIES: {state.account.queries_submitted}
              </div>
            </div>
          </MetricCard>

          <MetricCard
            title="Account Address"
            icon={<Download className="h-6 w-6" />}
          >
            <div className="space-y-3">
              <div className="text-xs font-mono break-all text-tesla-white">
                {state.account.address}
              </div>
              <button
                onClick={handleCopyAddress}
                className="flex items-center gap-2 text-xs text-tesla-white hover:text-tesla-text-gray transition-colors uppercase tracking-wider"
              >
                {copiedAddress ? (
                  <CheckCircle className="h-4 w-4" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
                {copiedAddress ? 'COPIED!' : 'COPY ADDRESS'}
              </button>
            </div>
          </MetricCard>
        </div>
      )}

      {/* Transfer Form and Transaction History */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Transfer Form */}
        <MetricCard title="Send LORE" icon={<Send className="h-6 w-6" />}>
          <form onSubmit={handleTransfer} className="space-y-6">
            <div>
              <label className="block text-xs font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">
                RECIPIENT USER ID
              </label>
              <input
                type="text"
                value={transferForm.toUserId}
                onChange={(e) => setTransferForm(prev => ({ ...prev, toUserId: e.target.value }))}
                placeholder="ENTER USER ID"
                className="w-full bg-tesla-medium-gray border border-tesla-border px-3 py-3 text-tesla-white placeholder-tesla-text-gray focus:outline-none focus:border-tesla-white text-sm transition-all duration-150"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">
                AMOUNT (LORE)
              </label>
              <input
                type="number"
                step="0.0001"
                min="0"
                value={transferForm.amount}
                onChange={(e) => setTransferForm(prev => ({ ...prev, amount: e.target.value }))}
                placeholder="0.0000"
                className="w-full bg-tesla-medium-gray border border-tesla-border px-3 py-3 text-tesla-white placeholder-tesla-text-gray focus:outline-none focus:border-tesla-white text-sm transition-all duration-150"
                required
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-tesla-text-gray mb-2 uppercase tracking-wider">
                DESCRIPTION (OPTIONAL)
              </label>
              <input
                type="text"
                value={transferForm.description}
                onChange={(e) => setTransferForm(prev => ({ ...prev, description: e.target.value }))}
                placeholder="WHAT'S THIS FOR?"
                className="w-full bg-tesla-medium-gray border border-tesla-border px-3 py-3 text-tesla-white placeholder-tesla-text-gray focus:outline-none focus:border-tesla-white text-sm transition-all duration-150"
              />
            </div>

            {state.transferError && (
              <div className="text-tesla-accent-red text-xs flex items-center gap-2 uppercase tracking-wider">
                <AlertCircle className="h-4 w-4" />
                {state.transferError}
              </div>
            )}

            {state.transferSuccess && (
              <div className="text-tesla-accent-green text-xs flex items-center gap-2 uppercase tracking-wider">
                <CheckCircle className="h-4 w-4" />
                {state.transferSuccess}
              </div>
            )}

            <button
              type="submit"
              disabled={state.transferLoading}
              className="w-full tesla-button text-xs disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
            >
              {state.transferLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              {state.transferLoading ? 'SENDING...' : 'SEND LORE'}
            </button>
          </form>
        </MetricCard>

        {/* Transaction History */}
        <MetricCard title="Recent Transactions" icon={<History className="h-6 w-6" />}>
          <div className="space-y-2 max-h-96 overflow-y-auto">
            {state.transactions.length === 0 ? (
              <div className="text-center text-tesla-text-gray py-8">
                <div className="text-xs uppercase tracking-wider">NO TRANSACTIONS FOUND</div>
              </div>
            ) : (
              state.transactions.map((tx) => (
                <div
                  key={tx.id}
                  className="flex items-center justify-between p-3 border border-tesla-border bg-tesla-medium-gray hover:border-tesla-light-gray transition-all duration-150"
                >
                  <div className="flex items-center gap-3">
                    {getTransactionIcon(tx.type, tx.from_id)}
                    <div>
                      <div className="font-medium text-tesla-white text-sm uppercase tracking-wider">
                        {formatTransactionType(tx.type, tx.from_id)}
                      </div>
                      <div className="text-xs text-tesla-text-gray">
                        {tx.description} • {new Date(tx.created_at).toLocaleString()}
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className={`font-medium text-sm ${
                      tx.from_id === DEFAULT_USER_ID ? 'text-tesla-accent-red' : 'text-tesla-accent-green'
                    }`}>
                      {tx.from_id === DEFAULT_USER_ID ? '-' : '+'}
                      {(parseFloat(tx.amount) / 1e18).toFixed(4)} LORE
                    </div>
                    <div className={`text-xs px-2 py-1 uppercase tracking-wider ${
                      tx.status === 'completed' ? 'bg-tesla-accent-green text-tesla-black' :
                      tx.status === 'pending' ? 'bg-tesla-text-gray text-tesla-white' :
                      'bg-tesla-accent-red text-tesla-white'
                    }`}>
                      {tx.status}
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </MetricCard>
      </div>
    </div>
  );
}