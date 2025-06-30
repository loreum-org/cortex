import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { Dashboard } from './components/Dashboard';
import { ChatInterface } from './components/ChatInterface';
import { NetworkOverview } from './components/NetworkOverview';
import { Wallet } from './components/Wallet';
import Reputation from './components/Reputation';
import Staking from './components/Staking';
import { ServiceManagement } from './components/ServiceManagement';
import { AGIIntelligence } from './components/AGIIntelligence';
import { Layout } from './components/Layout';

function App() {
  return (
    <Router>
      <Layout>
        <Routes>
          <Route path="/network" element={<NetworkOverview />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/services" element={<ServiceManagement />} />
          <Route path="/reputation" element={<Reputation />} />
          <Route path="/staking" element={<Staking />} />
          <Route path="/chat" element={<ChatInterface />} />
          <Route path="/wallet" element={<Wallet />} />
          <Route path="/agi" element={<AGIIntelligence />} />
          <Route path="/" element={<Navigate to="/network" replace />} />
        </Routes>
      </Layout>
    </Router>
  );
}

export default App
