import { useState, useEffect } from 'react';
import { useNavigate, useLocation, Routes, Route, Navigate } from 'react-router-dom';
import { BarChart3, Bot, Brain, Shield, Lock, Download } from 'lucide-react';
import { Dashboard } from './Dashboard';
import { AgentManagement } from './AgentManagement';
import { AGIIntelligence } from './AGIIntelligence';
import Reputation from './Reputation';
import Staking from './Staking';
import { ModelManagement } from './ModelManagement';

type NodeTab = 'overview' | 'agents' | 'models' | 'agi' | 'reputation' | 'staking';

export function NodeDashboard() {
  const navigate = useNavigate();
  const location = useLocation();
  const [activeTab, setActiveTab] = useState<NodeTab>('overview');

  const tabs = [
    { id: 'overview' as const, label: 'Overview', icon: BarChart3, path: '' },
    { id: 'agents' as const, label: 'Agents', icon: Bot, path: 'agents' },
    { id: 'models' as const, label: 'Models', icon: Download, path: 'models' },
    { id: 'agi' as const, label: 'AGI', icon: Brain, path: 'agi' },
    { id: 'reputation' as const, label: 'Reputation', icon: Shield, path: 'reputation' },
    { id: 'staking' as const, label: 'Staking', icon: Lock, path: 'staking' },
  ];

  useEffect(() => {
    const path = location.pathname.replace('/node', '').replace('/', '');
    const tab = tabs.find(t => t.path === path);
    if (tab) {
      setActiveTab(tab.id);
    } else {
      setActiveTab('overview');
    }
  }, [location.pathname]);

  const handleTabClick = (tab: typeof tabs[0]) => {
    setActiveTab(tab.id);
    navigate(`/node${tab.path ? `/${tab.path}` : ''}`);
  };

  return (
    <div className="min-h-screen bg-tesla-black">
      {/* Tab Navigation */}
      <div className="border-b border-tesla-border bg-tesla-dark-gray">
        <div className="max-w-7xl mx-auto px-6">
          <div className="flex space-x-1">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => handleTabClick(tab)}
                className={`px-6 py-4 text-sm font-medium flex items-center gap-2 transition-all duration-150 uppercase tracking-wider border-b-2 ${
                  activeTab === tab.id
                    ? 'border-tesla-white text-tesla-white bg-tesla-medium-gray'
                    : 'border-transparent text-tesla-text-gray hover:text-tesla-white hover:bg-tesla-medium-gray'
                }`}
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Tab Content */}
      <div className="flex-1">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/agents" element={<AgentManagement />} />
          <Route path="/models" element={<ModelManagement />} />
          <Route path="/agi" element={<AGIIntelligence />} />
          <Route path="/reputation" element={<Reputation />} />
          <Route path="/staking" element={<Staking />} />
        </Routes>
      </div>
    </div>
  );
}