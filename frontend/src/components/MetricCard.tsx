import type { ReactNode } from 'react';

interface MetricCardProps {
  title: string;
  icon: ReactNode;
  children: ReactNode;
  className?: string;
}

export function MetricCard({ title, icon, children, className = '' }: MetricCardProps) {
  return (
    <div className={`bg-tesla-dark-gray border border-tesla-border p-6 hover:border-tesla-light-gray transition-all duration-150 animate-fade-in ${className}`}>
      <div className="flex items-center justify-between mb-6">
        <h3 className="text-sm font-medium text-tesla-white uppercase tracking-wider">{title}</h3>
        <div className="text-tesla-white opacity-80">{icon}</div>
      </div>
      <div className="text-tesla-white">{children}</div>
    </div>
  );
}