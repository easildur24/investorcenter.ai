'use client';

import { cn } from '@/lib/utils';
import { StatementType, statementTypeLabels } from '@/types/financials';

interface StatementTabsProps {
  activeTab: StatementType;
  onChange: (tab: StatementType) => void;
}

const tabs: StatementType[] = ['income', 'balance_sheet', 'cash_flow', 'ratios'];

export default function StatementTabs({ activeTab, onChange }: StatementTabsProps) {
  return (
    <div className="flex gap-1 p-1 bg-ic-surface rounded-lg overflow-x-auto">
      {tabs.map((tab) => (
        <button
          key={tab}
          onClick={() => onChange(tab)}
          className={cn(
            'px-4 py-2 text-sm font-medium rounded-md whitespace-nowrap transition-all',
            activeTab === tab
              ? 'bg-white text-blue-600 shadow-sm'
              : 'text-ic-text-muted hover:text-ic-text-primary hover:bg-ic-surface-hover'
          )}
        >
          {statementTypeLabels[tab]}
        </button>
      ))}
    </div>
  );
}
