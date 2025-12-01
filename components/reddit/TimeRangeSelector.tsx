'use client';

interface TimeRangeSelectorProps {
  value: '1' | '7' | '14' | '30';
  onChange: (value: '1' | '7' | '14' | '30') => void;
}

export default function TimeRangeSelector({ value, onChange }: TimeRangeSelectorProps) {
  const options = [
    { value: '1' as const, label: 'Today' },
    { value: '7' as const, label: '7 Days' },
    { value: '14' as const, label: '14 Days' },
    { value: '30' as const, label: '30 Days' },
  ];

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium text-ic-text-secondary">Time Range:</span>
      <div className="inline-flex rounded-md shadow-sm" role="group">
        {options.map((option, index) => (
          <button
            key={option.value}
            type="button"
            onClick={() => onChange(option.value)}
            className={`
              px-4 py-2 text-sm font-medium
              ${index === 0 ? 'rounded-l-lg' : ''}
              ${index === options.length - 1 ? 'rounded-r-lg' : ''}
              ${value === option.value
                ? 'bg-primary-600 text-ic-text-primary hover:bg-primary-700'
                : 'bg-ic-surface text-ic-text-secondary hover:bg-ic-surface-hover border border-ic-border'
              }
              ${index > 0 && value !== option.value ? '-ml-px' : ''}
              transition-colors duration-150
            `}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}
