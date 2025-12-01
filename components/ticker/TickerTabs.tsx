'use client';

import { useState, Suspense } from 'react';
import { cn } from '@/lib/utils';

interface Tab {
  id: string;
  label: string;
  icon?: React.ReactNode;
}

interface TickerTabsProps {
  symbol: string;
  children: React.ReactNode[];
  tabs: Tab[];
  defaultTab?: string;
}

export default function TickerTabs({ symbol, children, tabs, defaultTab }: TickerTabsProps) {
  const [activeTab, setActiveTab] = useState(defaultTab || tabs[0]?.id || 'overview');

  const activeIndex = tabs.findIndex(tab => tab.id === activeTab);

  return (
    <div className="w-full">
      {/* Tab Navigation */}
      <div className="border-b border-gray-200 bg-white rounded-t-lg">
        <nav className="-mb-px flex space-x-8 px-6" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={cn(
                'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm transition-colors',
                activeTab === tab.id
                  ? 'border-ic-blue text-ic-blue'
                  : 'border-transparent text-ic-text-dim hover:text-ic-text-muted hover:border-ic-border-subtle'
              )}
            >
              <span className="flex items-center gap-2">
                {tab.icon}
                {tab.label}
              </span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="bg-white rounded-b-lg shadow">
        {children[activeIndex]}
      </div>
    </div>
  );
}

// Loading skeleton for tabs
export function TabSkeleton() {
  return (
    <div className="p-6 animate-pulse">
      <div className="h-6 bg-gray-200 rounded w-48 mb-6"></div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-gray-100 rounded-lg p-4">
            <div className="h-4 bg-gray-200 rounded w-20 mb-2"></div>
            <div className="h-6 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
      <div className="h-64 bg-gray-200 rounded"></div>
    </div>
  );
}
