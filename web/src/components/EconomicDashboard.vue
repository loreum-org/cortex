<template>
  <div class="economic-dashboard">
    <h1>Economic Monitoring Dashboard</h1>
    
    <!-- Network Statistics Section -->
    <div class="dashboard-section">
      <h2>Network Statistics</h2>
      <div v-if="loading.networkStats" class="loading">Loading network statistics...</div>
      <div v-else class="stats-grid">
        <div class="stat-card">
          <div class="stat-title">Total Supply</div>
          <div class="stat-value">{{ formatTokenAmount(networkStats.total_supply) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Active Nodes</div>
          <div class="stat-value">{{ networkStats.active_nodes }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Total Staked</div>
          <div class="stat-value">{{ formatTokenAmount(networkStats.total_staked) }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Avg. Reputation</div>
          <div class="stat-value">{{ networkStats.avg_reputation ? networkStats.avg_reputation.toFixed(2) : '0.00' }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Total Accounts</div>
          <div class="stat-value">{{ networkStats.total_accounts }}</div>
        </div>
        <div class="stat-card">
          <div class="stat-title">Total Transactions</div>
          <div class="stat-value">{{ networkStats.total_transactions }}</div>
        </div>
      </div>

      <!-- Transaction Types Chart -->
      <div class="chart-container" v-if="!loading.networkStats && chartData.transactionTypes.labels.length > 0">
        <h3>Transaction Distribution</h3>
        <canvas ref="transactionTypesChart"></canvas>
      </div>
    </div>

    <!-- Node Staking Section -->
    <div class="dashboard-section">
      <h2>Node Staking</h2>
      <div v-if="loading.nodeStakes" class="loading">Loading node staking information...</div>
      <div v-else>
        <!-- Node Stake Distribution Chart -->
        <div class="chart-container" v-if="chartData.nodeStakes.labels.length > 0">
          <h3>Stake Distribution</h3>
          <canvas ref="nodeStakesChart"></canvas>
        </div>
        
        <!-- Node Staking Table -->
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Node ID</th>
                <th>Total Stake</th>
                <th>Stakers</th>
                <th>Reputation Boost</th>
                <th>Performance</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(stake, nodeId) in nodeStakes" :key="nodeId">
                <td>{{ formatNodeId(nodeId) }}</td>
                <td>{{ formatTokenAmount(stake.total_user_stake) }}</td>
                <td>{{ stake.stake_count }}</td>
                <td>{{ stake.reputation_boost.toFixed(2) }}</td>
                <td>
                  <div v-if="stake.performance_data" class="performance-indicator">
                    <div class="success-rate" :style="{ width: `${stake.performance_data.success_rate * 100}%` }"></div>
                  </div>
                  <span v-else>No data</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- User Account Section -->
    <div class="dashboard-section">
      <h2>User Account</h2>
      
      <!-- Account Selection -->
      <div class="account-selector">
        <label for="account-select">Select Account:</label>
        <select id="account-select" v-model="selectedAccount" @change="loadUserData">
          <option value="default_user">Default User</option>
          <option value="user-1">User 1</option>
          <option value="user-2">User 2</option>
          <option value="custom">Custom Account ID</option>
        </select>
        <div v-if="selectedAccount === 'custom'" class="custom-account-input">
          <input type="text" v-model="customAccountId" placeholder="Enter account ID">
          <button @click="loadUserData">Load</button>
        </div>
      </div>

      <!-- Account Balance -->
      <div class="account-balance" v-if="userAccount">
        <h3>Account Balance</h3>
        <div class="balance-display">
          <span class="balance-amount">{{ userAccount.balance_formatted }}</span>
          <span class="balance-token">{{ userAccount.token_symbol }}</span>
        </div>
        <div class="account-details">
          <div><strong>Account ID:</strong> {{ userAccount.account }}</div>
          <div><strong>Address:</strong> {{ userAccount.address }}</div>
          <div><strong>Total Spent:</strong> {{ formatTokenAmount(userAccount.total_spent) }}</div>
          <div><strong>Queries Submitted:</strong> {{ userAccount.queries_submitted }}</div>
          <div><strong>Created:</strong> {{ formatDate(userAccount.created_at) }}</div>
        </div>
      </div>

      <!-- User Stakes -->
      <div class="user-stakes" v-if="userStakes.length > 0">
        <h3>User Stakes</h3>
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Node ID</th>
                <th>Amount</th>
                <th>Staked At</th>
                <th>Status</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="stake in userStakes" :key="stake.id">
                <td>{{ formatNodeId(stake.node_id) }}</td>
                <td>{{ formatTokenAmount(stake.amount) }}</td>
                <td>{{ formatDate(stake.staked_at) }}</td>
                <td>
                  <span v-if="stake.withdrawal_requested" class="status withdrawal">Withdrawal Pending</span>
                  <span v-else class="status active">Active</span>
                </td>
                <td>
                  <button v-if="!stake.withdrawal_requested" @click="requestWithdrawal(stake)" class="btn-small">Withdraw</button>
                  <button v-else @click="executeWithdrawal(stake)" class="btn-small">Execute</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- User Transactions -->
      <div class="user-transactions">
        <h3>Recent Transactions</h3>
        <div v-if="loading.transactions" class="loading">Loading transactions...</div>
        <div v-else-if="userTransactions.length === 0" class="no-data">No transactions found</div>
        <div v-else class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Type</th>
                <th>Amount</th>
                <th>From/To</th>
                <th>Status</th>
                <th>Date</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="tx in userTransactions" :key="tx.id">
                <td>
                  <span class="tx-type" :class="tx.type">{{ formatTransactionType(tx.type) }}</span>
                </td>
                <td>{{ formatTokenAmount(tx.amount) }}</td>
                <td>
                  <span v-if="tx.from_id === selectedAccount">
                    To: {{ formatAccountId(tx.to_id) }}
                  </span>
                  <span v-else>
                    From: {{ formatAccountId(tx.from_id) }}
                  </span>
                </td>
                <td>{{ tx.status }}</td>
                <td>{{ formatDate(tx.timestamp) }}</td>
                <td>{{ tx.description }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Economic Actions Section -->
    <div class="dashboard-section">
      <h2>Economic Actions</h2>
      
      <div class="action-cards">
        <!-- Transfer Form -->
        <div class="action-card">
          <h3>Transfer Tokens</h3>
          <form @submit.prevent="performTransfer">
            <div class="form-group">
              <label for="transfer-from">From Account:</label>
              <input id="transfer-from" type="text" v-model="transferForm.fromAccount" required>
            </div>
            <div class="form-group">
              <label for="transfer-to">To Account:</label>
              <input id="transfer-to" type="text" v-model="transferForm.toAccount" required>
            </div>
            <div class="form-group">
              <label for="transfer-amount">Amount (LORE):</label>
              <input id="transfer-amount" type="number" v-model="transferForm.amount" min="0.01" step="0.01" required>
            </div>
            <div class="form-group">
              <label for="transfer-desc">Description:</label>
              <input id="transfer-desc" type="text" v-model="transferForm.description">
            </div>
            <button type="submit" class="btn-primary" :disabled="transferForm.processing">
              {{ transferForm.processing ? 'Processing...' : 'Transfer' }}
            </button>
            <div v-if="transferForm.error" class="form-error">{{ transferForm.error }}</div>
            <div v-if="transferForm.success" class="form-success">{{ transferForm.success }}</div>
          </form>
        </div>

        <!-- Staking Form -->
        <div class="action-card">
          <h3>Stake Tokens</h3>
          <form @submit.prevent="performStake">
            <div class="form-group">
              <label for="stake-account">Account:</label>
              <input id="stake-account" type="text" v-model="stakeForm.account" required>
            </div>
            <div class="form-group">
              <label for="stake-node">Node ID:</label>
              <input id="stake-node" type="text" v-model="stakeForm.nodeId" required>
            </div>
            <div class="form-group">
              <label for="stake-amount">Amount (LORE):</label>
              <input id="stake-amount" type="number" v-model="stakeForm.amount" min="10" step="1" required>
            </div>
            <button type="submit" class="btn-primary" :disabled="stakeForm.processing">
              {{ stakeForm.processing ? 'Processing...' : 'Stake' }}
            </button>
            <div v-if="stakeForm.error" class="form-error">{{ stakeForm.error }}</div>
            <div v-if="stakeForm.success" class="form-success">{{ stakeForm.success }}</div>
          </form>
        </div>
      </div>
    </div>

    <!-- Pricing Section -->
    <div class="dashboard-section">
      <h2>Model Pricing</h2>
      
      <div class="pricing-tiers">
        <div class="pricing-tier" v-for="(tier, index) in pricingTiers" :key="index">
          <div class="tier-header" :class="tier.id">
            <h3>{{ tier.name }}</h3>
          </div>
          <div class="tier-body">
            <div class="tier-price">{{ tier.basePrice }} LORE</div>
            <div class="tier-description">{{ tier.description }}</div>
            <ul class="tier-features">
              <li v-for="(feature, i) in tier.features" :key="i">{{ feature }}</li>
            </ul>
            <div class="tier-query-types">
              <div class="query-type" v-for="(price, type) in tier.queryTypes" :key="type">
                <span class="query-type-name">{{ formatQueryType(type) }}</span>
                <span class="query-type-price">{{ price }} LORE</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Reward Distribution Section -->
    <div class="dashboard-section">
      <h2>Reward Distribution</h2>
      
      <div v-if="loading.rewards" class="loading">Loading reward distribution data...</div>
      <div v-else>
        <!-- Reward Distribution Chart -->
        <div class="chart-container" v-if="chartData.rewards.labels.length > 0">
          <h3>Recent Rewards</h3>
          <canvas ref="rewardsChart"></canvas>
        </div>
        
        <!-- Reward Distribution Table -->
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Node ID</th>
                <th>Amount</th>
                <th>Performance Score</th>
                <th>Timestamp</th>
                <th>Reason</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="reward in rewardEvents" :key="reward.id">
                <td>{{ formatNodeId(reward.node_id) }}</td>
                <td>{{ formatTokenAmount(reward.amount) }}</td>
                <td>{{ reward.performance_data?.performance_score?.toFixed(2) || 'N/A' }}</td>
                <td>{{ formatDate(reward.timestamp) }}</td>
                <td>{{ reward.reason }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, reactive, onMounted, watch, nextTick } from 'vue';
import Chart from 'chart.js/auto';

export default {
  name: 'EconomicDashboard',
  setup() {
    // Charts references
    const transactionTypesChart = ref(null);
    const nodeStakesChart = ref(null);
    const rewardsChart = ref(null);
    
    // Charts instances
    let txTypesChartInstance = null;
    let nodeStakesChartInstance = null;
    let rewardsChartInstance = null;

    // Loading states
    const loading = reactive({
      networkStats: true,
      nodeStakes: true,
      userAccount: true,
      transactions: true,
      rewards: true
    });

    // Data states
    const networkStats = ref({});
    const nodeStakes = ref({});
    const userAccount = ref(null);
    const userStakes = ref([]);
    const userTransactions = ref([]);
    const rewardEvents = ref([]);

    // Chart data
    const chartData = reactive({
      transactionTypes: {
        labels: [],
        data: []
      },
      nodeStakes: {
        labels: [],
        data: []
      },
      rewards: {
        labels: [],
        data: []
      }
    });

    // User selection
    const selectedAccount = ref('default_user');
    const customAccountId = ref('');

    // Forms
    const transferForm = reactive({
      fromAccount: '',
      toAccount: '',
      amount: '',
      description: '',
      processing: false,
      error: null,
      success: null
    });

    const stakeForm = reactive({
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
    onMounted(async () => {
      await Promise.all([
        fetchNetworkStats(),
        fetchNodeStakes(),
        fetchRewardDistribution()
      ]);
      
      // Set default account in forms
      transferForm.fromAccount = selectedAccount.value;
      stakeForm.account = selectedAccount.value;
      
      // Load user data
      loadUserData();
    });

    // Watch for chart data changes to update charts
    watch(() => chartData.transactionTypes, updateTransactionTypesChart, { deep: true });
    watch(() => chartData.nodeStakes, updateNodeStakesChart, { deep: true });
    watch(() => chartData.rewards, updateRewardsChart, { deep: true });

    // Fetch network statistics
    async function fetchNetworkStats() {
      loading.networkStats = true;
      try {
        const response = await fetch('/api/metrics/economy');
        const data = await response.json();
        networkStats.value = data;
        
        // Prepare transaction types chart data
        if (data.transaction_counts) {
          const labels = [];
          const values = [];
          
          for (const [type, count] of Object.entries(data.transaction_counts)) {
            labels.push(formatTransactionType(type));
            values.push(count);
          }
          
          chartData.transactionTypes.labels = labels;
          chartData.transactionTypes.data = values;
        }
      } catch (error) {
        console.error('Failed to fetch network statistics:', error);
      } finally {
        loading.networkStats = false;
      }
    }

    // Fetch node staking information
    async function fetchNodeStakes() {
      loading.nodeStakes = true;
      try {
        const response = await fetch('/api/staking/nodes');
        const data = await response.json();
        nodeStakes.value = data.nodes || {};
        
        // Prepare node stakes chart data
        const labels = [];
        const values = [];
        
        for (const [nodeId, stake] of Object.entries(nodeStakes.value)) {
          if (stake.total_user_stake) {
            labels.push(formatNodeId(nodeId));
            // Convert from string to number for chart
            const stakeValue = parseFloat(stake.total_user_stake) / Math.pow(10, 18);
            values.push(stakeValue);
          }
        }
        
        chartData.nodeStakes.labels = labels;
        chartData.nodeStakes.data = values;
      } catch (error) {
        console.error('Failed to fetch node stakes:', error);
      } finally {
        loading.nodeStakes = false;
      }
    }

    // Load user data (account, stakes, transactions)
    async function loadUserData() {
      const accountId = selectedAccount.value === 'custom' ? customAccountId.value : selectedAccount.value;
      if (!accountId) return;
      
      loading.userAccount = true;
      loading.transactions = true;
      
      // Fetch account details
      try {
        const accountResponse = await fetch(`/api/wallet/account/${accountId}`);
        if (accountResponse.ok) {
          userAccount.value = await accountResponse.json();
        } else {
          userAccount.value = null;
          console.error('Failed to fetch user account:', await accountResponse.text());
        }
      } catch (error) {
        console.error('Error fetching user account:', error);
        userAccount.value = null;
      } finally {
        loading.userAccount = false;
      }
      
      // Fetch user stakes
      try {
        const stakesResponse = await fetch(`/api/staking/user/${accountId}`);
        if (stakesResponse.ok) {
          const data = await stakesResponse.json();
          userStakes.value = data.stakes || [];
        } else {
          userStakes.value = [];
        }
      } catch (error) {
        console.error('Error fetching user stakes:', error);
        userStakes.value = [];
      }
      
      // Fetch user transactions
      try {
        const txResponse = await fetch(`/api/wallet/transactions/${accountId}?limit=20`);
        if (txResponse.ok) {
          userTransactions.value = await txResponse.json();
        } else {
          userTransactions.value = [];
        }
      } catch (error) {
        console.error('Error fetching transactions:', error);
        userTransactions.value = [];
      } finally {
        loading.transactions = false;
      }
      
      // Update form values
      transferForm.fromAccount = accountId;
    }

    // Fetch reward distribution data
    async function fetchRewardDistribution() {
      loading.rewards = true;
      try {
        // This would be a real endpoint in production
        // For now we'll simulate some reward data
        // const response = await fetch('/api/rewards/distribution');
        // rewardEvents.value = await response.json();
        
        // Simulated reward data
        rewardEvents.value = [
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
        
        // Prepare rewards chart data
        const nodeRewards = {};
        
        for (const reward of rewardEvents.value) {
          const nodeId = formatNodeId(reward.node_id);
          if (!nodeRewards[nodeId]) {
            nodeRewards[nodeId] = 0;
          }
          // Convert from string to number
          const amount = parseFloat(reward.amount) / Math.pow(10, 18);
          nodeRewards[nodeId] += amount;
        }
        
        chartData.rewards.labels = Object.keys(nodeRewards);
        chartData.rewards.data = Object.values(nodeRewards);
      } catch (error) {
        console.error('Failed to fetch reward distribution:', error);
      } finally {
        loading.rewards = false;
      }
    }

    // Update transaction types chart
    async function updateTransactionTypesChart() {
      if (txTypesChartInstance) {
        txTypesChartInstance.destroy();
      }
      
      await nextTick();
      const ctx = transactionTypesChart.value?.getContext('2d');
      if (ctx && chartData.transactionTypes.labels.length > 0) {
        txTypesChartInstance = new Chart(ctx, {
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
                position: 'right'
              },
              title: {
                display: true,
                text: 'Transaction Types'
              }
            }
          }
        });
      }
    }

    // Update node stakes chart
    async function updateNodeStakesChart() {
      if (nodeStakesChartInstance) {
        nodeStakesChartInstance.destroy();
      }
      
      await nextTick();
      const ctx = nodeStakesChart.value?.getContext('2d');
      if (ctx && chartData.nodeStakes.labels.length > 0) {
        nodeStakesChartInstance = new Chart(ctx, {
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
              y: {
                beginAtZero: true,
                title: {
                  display: true,
                  text: 'LORE Tokens'
                }
              }
            },
            plugins: {
              title: {
                display: true,
                text: 'Node Stake Distribution'
              }
            }
          }
        });
      }
    }

    // Update rewards chart
    async function updateRewardsChart() {
      if (rewardsChartInstance) {
        rewardsChartInstance.destroy();
      }
      
      await nextTick();
      const ctx = rewardsChart.value?.getContext('2d');
      if (ctx && chartData.rewards.labels.length > 0) {
        rewardsChartInstance = new Chart(ctx, {
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
              y: {
                beginAtZero: true,
                title: {
                  display: true,
                  text: 'LORE Tokens'
                }
              }
            },
            plugins: {
              title: {
                display: true,
                text: 'Reward Distribution by Node'
              }
            }
          }
        });
      }
    }

    // Perform token transfer
    async function performTransfer() {
      transferForm.processing = true;
      transferForm.error = null;
      transferForm.success = null;
      
      try {
        // Convert LORE amount to smallest unit
        const amountInSmallestUnit = BigInt(Math.floor(parseFloat(transferForm.amount) * Math.pow(10, 18))).toString();
        
        const response = await fetch('/api/wallet/transfer', {
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
          const result = await response.json();
          transferForm.success = `Successfully transferred ${transferForm.amount} LORE to ${transferForm.toAccount}`;
          
          // Reset form
          transferForm.amount = '';
          transferForm.description = '';
          
          // Refresh data
          await Promise.all([
            fetchNetworkStats(),
            loadUserData()
          ]);
        } else {
          const error = await response.json();
          transferForm.error = error.error || 'Transfer failed';
        }
      } catch (error) {
        console.error('Transfer error:', error);
        transferForm.error = 'An unexpected error occurred';
      } finally {
        transferForm.processing = false;
      }
    }

    // Perform token staking
    async function performStake() {
      stakeForm.processing = true;
      stakeForm.error = null;
      stakeForm.success = null;
      
      try {
        // Convert LORE amount to smallest unit
        const amountInSmallestUnit = BigInt(Math.floor(parseFloat(stakeForm.amount) * Math.pow(10, 18))).toString();
        
        const response = await fetch('/api/staking/stake', {
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
          const result = await response.json();
          stakeForm.success = `Successfully staked ${stakeForm.amount} LORE to node ${stakeForm.nodeId}`;
          
          // Reset form
          stakeForm.amount = '';
          
          // Refresh data
          await Promise.all([
            fetchNetworkStats(),
            fetchNodeStakes(),
            loadUserData()
          ]);
        } else {
          const error = await response.json();
          stakeForm.error = error.error || 'Staking failed';
        }
      } catch (error) {
        console.error('Staking error:', error);
        stakeForm.error = 'An unexpected error occurred';
      } finally {
        stakeForm.processing = false;
      }
    }

    // Request withdrawal of staked tokens
    async function requestWithdrawal(stake) {
      try {
        const response = await fetch('/api/staking/withdraw/request', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            account: selectedAccount.value === 'custom' ? customAccountId.value : selectedAccount.value,
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
    }

    // Execute withdrawal of staked tokens
    async function executeWithdrawal(stake) {
      try {
        const response = await fetch('/api/staking/withdraw/execute', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            account: selectedAccount.value === 'custom' ? customAccountId.value : selectedAccount.value,
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
    }

    // Helper functions
    function formatTokenAmount(amount) {
      if (!amount) return '0 LORE';
      
      // Convert from smallest unit (10^18) to LORE
      const amountBigInt = BigInt(amount);
      const loreAmount = Number(amountBigInt) / Math.pow(10, 18);
      
      return `${loreAmount.toLocaleString(undefined, { maximumFractionDigits: 4 })} LORE`;
    }

    function formatDate(dateString) {
      if (!dateString) return '';
      return new Date(dateString).toLocaleString();
    }

    function formatNodeId(nodeId) {
      if (!nodeId) return '';
      if (nodeId.length <= 12) return nodeId;
      return `${nodeId.substring(0, 6)}...${nodeId.substring(nodeId.length - 6)}`;
    }

    function formatAccountId(accountId) {
      if (!accountId) return '';
      if (accountId === 'network') return 'Network';
      if (accountId === 'staking_pool') return 'Staking Pool';
      return accountId;
    }

    function formatTransactionType(type) {
      if (!type) return '';
      
      // Remove "TransactionType" prefix if present
      type = type.replace('TransactionType', '');
      
      // Convert snake_case to Title Case
      return type.split('_')
        .map(word => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
        .join(' ');
    }

    function formatQueryType(type) {
      if (!type) return '';
      
      // Convert camelCase to Title Case
      return type.replace(/([A-Z])/g, ' $1')
        .replace(/^./, str => str.toUpperCase());
    }

    return {
      // Refs
      transactionTypesChart,
      nodeStakesChart,
      rewardsChart,
      
      // State
      loading,
      networkStats,
      nodeStakes,
      userAccount,
      userStakes,
      userTransactions,
      rewardEvents,
      chartData,
      selectedAccount,
      customAccountId,
      transferForm,
      stakeForm,
      pricingTiers,
      
      // Methods
      loadUserData,
      performTransfer,
      performStake,
      requestWithdrawal,
      executeWithdrawal,
      formatTokenAmount,
      formatDate,
      formatNodeId,
      formatAccountId,
      formatTransactionType,
      formatQueryType
    };
  }
};
</script>

<style scoped>
.economic-dashboard {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
  font-family: 'Inter', system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

h1 {
  font-size: 2rem;
  margin-bottom: 1.5rem;
  color: #333;
  border-bottom: 2px solid #eee;
  padding-bottom: 0.5rem;
}

h2 {
  font-size: 1.5rem;
  margin-top: 2rem;
  margin-bottom: 1rem;
  color: #444;
}

h3 {
  font-size: 1.2rem;
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  color: #555;
}

.dashboard-section {
  background-color: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.05);
  padding: 1.5rem;
  margin-bottom: 2rem;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.stat-card {
  background-color: #f9f9f9;
  border-radius: 6px;
  padding: 1rem;
  text-align: center;
}

.stat-title {
  font-size: 0.9rem;
  color: #666;
  margin-bottom: 0.5rem;
}

.stat-value {
  font-size: 1.2rem;
  font-weight: 600;
  color: #333;
}

.chart-container {
  height: 300px;
  margin: 1.5rem 0;
}

.loading {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100px;
  color: #666;
  font-style: italic;
}

.no-data {
  text-align: center;
  padding: 2rem;
  color: #666;
  font-style: italic;
}

.table-container {
  overflow-x: auto;
  margin: 1rem 0;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.data-table th {
  background-color: #f5f5f5;
  padding: 0.75rem;
  text-align: left;
  font-weight: 600;
  color: #444;
}

.data-table td {
  padding: 0.75rem;
  border-top: 1px solid #eee;
}

.data-table tr:hover {
  background-color: #f9f9f9;
}

.performance-indicator {
  width: 100px;
  height: 10px;
  background-color: #eee;
  border-radius: 5px;
  overflow: hidden;
}

.success-rate {
  height: 100%;
  background-color: #4BC0C0;
}

.account-selector {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.account-selector label {
  font-weight: 500;
}

.account-selector select {
  padding: 0.5rem;
  border-radius: 4px;
  border: 1px solid #ddd;
}

.custom-account-input {
  display: flex;
  gap: 0.5rem;
}

.custom-account-input input {
  padding: 0.5rem;
  border-radius: 4px;
  border: 1px solid #ddd;
  width: 200px;
}

.account-balance {
  background-color: #f9f9f9;
  border-radius: 8px;
  padding: 1.5rem;
  margin-bottom: 1.5rem;
}

.balance-display {
  font-size: 2rem;
  font-weight: 700;
  margin-bottom: 1rem;
  color: #333;
}

.balance-token {
  font-size: 1.2rem;
  color: #666;
  margin-left: 0.5rem;
}

.account-details {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 1rem;
}

.account-details div {
  font-size: 0.9rem;
  color: #555;
}

.action-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.action-card {
  background-color: #f9f9f9;
  border-radius: 8px;
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-group input {
  width: 100%;
  padding: 0.75rem;
  border-radius: 4px;
  border: 1px solid #ddd;
}

.btn-primary {
  background-color: #36A2EB;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 0.75rem 1.5rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.btn-primary:hover {
  background-color: #2a8ed3;
}

.btn-primary:disabled {
  background-color: #a0d0f0;
  cursor: not-allowed;
}

.btn-small {
  background-color: #36A2EB;
  color: white;
  border: none;
  border-radius: 4px;
  padding: 0.4rem 0.75rem;
  font-size: 0.8rem;
  cursor: pointer;
}

.form-error {
  color: #d9534f;
  margin-top: 0.75rem;
  font-size: 0.9rem;
}

.form-success {
  color: #5cb85c;
  margin-top: 0.75rem;
  font-size: 0.9rem;
}

.tx-type {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  color: white;
}

.tx-type.query_payment {
  background-color: #FF6384;
}

.tx-type.reward {
  background-color: #4BC0C0;
}

.tx-type.stake {
  background-color: #FFCE56;
}

.tx-type.unstake {
  background-color: #9966FF;
}

.tx-type.transfer {
  background-color: #36A2EB;
}

.tx-type.network_fee {
  background-color: #FF9F40;
}

.status {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  font-size: 0.8rem;
  color: white;
}

.status.active {
  background-color: #5cb85c;
}

.status.withdrawal {
  background-color: #f0ad4e;
}

.pricing-tiers {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-top: 1.5rem;
}

.pricing-tier {
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
}

.tier-header {
  padding: 1rem;
  color: white;
  text-align: center;
}

.tier-header.basic {
  background-color: #36A2EB;
}

.tier-header.standard {
  background-color: #4BC0C0;
}

.tier-header.premium {
  background-color: #9966FF;
}

.tier-body {
  padding: 1.5rem;
  background-color: white;
}

.tier-price {
  font-size: 1.5rem;
  font-weight: 700;
  text-align: center;
  margin-bottom: 1rem;
}

.tier-description {
  text-align: center;
  color: #666;
  margin-bottom: 1.5rem;
}

.tier-features {
  list-style-type: none;
  padding: 0;
  margin-bottom: 1.5rem;
}

.tier-features li {
  padding: 0.5rem 0;
  border-bottom: 1px solid #eee;
  position: relative;
  padding-left: 1.5rem;
}

.tier-features li:before {
  content: "✓";
  position: absolute;
  left: 0;
  color: #5cb85c;
}

.query-type {
  display: flex;
  justify-content: space-between;
  padding: 0.5rem 0;
  border-bottom: 1px solid #eee;
}

.query-type-name {
  color: #555;
}

.query-type-price {
  font-weight: 500;
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  
  .action-cards {
    grid-template-columns: 1fr;
  }
  
  .pricing-tiers {
    grid-template-columns: 1fr;
  }
  
  .account-details {
    grid-template-columns: 1fr;
  }
}
</style>
