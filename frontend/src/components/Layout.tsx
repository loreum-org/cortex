import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { BarChart3, MessageCircle, Network, Wallet, RefreshCw, Search } from 'lucide-react';

interface LayoutProps {
  children: ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const location = useLocation();

  const navItems = [
    { path: '/network', label: 'Network', icon: Network },
    { path: '/dashboard', label: 'Node', icon: BarChart3 },
    { path: '/explorer', label: 'Explorer', icon: Search },
    { path: '/chat', label: 'Chat', icon: MessageCircle },
    { path: '/wallet', label: 'Wallet', icon: Wallet },
  ];

  return (
    <div className="min-h-screen bg-tesla-black text-tesla-white font-tesla">
      {/* Navigation */}
      <nav className="bg-tesla-dark-gray border-b border-tesla-border">
        <div className="max-w-7xl mx-auto px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex items-center">
              <div className="flex-shrink-0 flex items-center gap-3">
                <img src="https://cdn.loreum.org/logos/white.svg" alt="Loreum Logo" className="h-20 w-20" />
                <span className="text-xl font-medium tracking-wide">LOREUM</span>
              </div>
              <div className="ml-12 flex items-center space-x-1">
                {navItems.map(({ path, label, icon: Icon }) => (
                  <Link
                    key={path}
                    to={path}
                    className={`px-4 py-2 text-sm font-medium flex items-center gap-2 transition-all duration-150 uppercase tracking-wider ${
                      location.pathname === path || (path === '/network' && location.pathname === '/')
                        ? 'bg-tesla-white text-tesla-black'
                        : 'text-tesla-text-gray hover:bg-tesla-medium-gray hover:text-tesla-white'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {label}
                  </Link>
                ))}
              </div>
            </div>
            <div className="flex items-center">
              <button
                onClick={() => window.location.reload()}
                className="p-2 text-tesla-text-gray hover:text-tesla-white hover:bg-tesla-medium-gray transition-all duration-150"
                title="Refresh"
              >
                <RefreshCw className="h-5 w-5" />
              </button>
            </div>
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main className="flex-1 bg-tesla-black">
        {children}
      </main>
    </div>
  );
}