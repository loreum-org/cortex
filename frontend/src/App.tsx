import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { NodeDashboard } from './components/NodeDashboard';
import { ChatInterface } from './components/ChatInterface';
import { NetworkOverview } from './components/NetworkOverview';
import { Wallet } from './components/Wallet';
import { BlockExplorer } from './components/BlockExplorer';
import { TransactionDetails } from './components/TransactionDetails';
import { AccountDetails } from './components/AccountDetails';
import { Layout } from './components/Layout';

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/node/*" element={<NodeDashboard />} />
          <Route path="/wallet" element={<Wallet />} />
          <Route path="/explorer" element={<BlockExplorer />} />
          <Route path="/explorer/transaction/:txId" element={<TransactionDetails />} />
          <Route path="/explorer/account/:accountId" element={<AccountDetails />} />
          <Route path="/network" element={<NetworkOverview />} />
          <Route path="/chat" element={<ChatInterface />} />
          <Route path="/dashboard" element={<Navigate to="/node" replace />} />
          <Route path="/" element={<Navigate to="/node" replace />} />
        </Routes>
      </Layout>
    </Router>
  );
}

export default App
