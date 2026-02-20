'use client';

import { useState } from 'react';
import { WatchListItem } from '@/lib/api/watchlist';
import { useModal } from '@/lib/hooks/useModal';

interface EditTickerModalProps {
  symbol: string;
  item: WatchListItem;
  onClose: () => void;
  onUpdate: (
    symbol: string,
    data: { notes?: string; tags?: string[]; target_buy_price?: number; target_sell_price?: number }
  ) => Promise<void>;
}

export default function EditTickerModal({ symbol, item, onClose, onUpdate }: EditTickerModalProps) {
  const modalRef = useModal(onClose);
  const [notes, setNotes] = useState(item.notes || '');
  const [tags, setTags] = useState(item.tags.join(', '));
  const [targetBuy, setTargetBuy] = useState(item.target_buy_price?.toString() || '');
  const [targetSell, setTargetSell] = useState(item.target_sell_price?.toString() || '');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const tagArray = tags
        .split(',')
        .map((t) => t.trim())
        .filter((t) => t);
      await onUpdate(symbol, {
        notes: notes || undefined,
        tags: tagArray,
        target_buy_price: targetBuy ? parseFloat(targetBuy) : undefined,
        target_sell_price: targetSell ? parseFloat(targetSell) : undefined,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-ic-bg-primary bg-opacity-50 flex items-center justify-center z-50">
      <div
        ref={modalRef}
        role="dialog"
        aria-modal="true"
        aria-label={`Edit ${symbol}`}
        className="bg-ic-surface rounded-lg p-6 w-full max-w-md"
      >
        <h2 className="text-2xl font-bold mb-4 text-ic-text-primary">Edit {symbol}</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="notes">
              Notes
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-ic-blue"
              rows={3}
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="tags">
              Tags (comma-separated)
            </label>
            <input
              id="tags"
              type="text"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder="e.g., tech, growth"
              className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-ic-blue"
            />
          </div>

          <div className="grid grid-cols-2 gap-4 mb-6">
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetBuy">
                Target Buy Price
              </label>
              <input
                id="targetBuy"
                type="number"
                step="0.01"
                value={targetBuy}
                onChange={(e) => setTargetBuy(e.target.value)}
                className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-ic-blue"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-2" htmlFor="targetSell">
                Target Sell Price
              </label>
              <input
                id="targetSell"
                type="number"
                step="0.01"
                value={targetSell}
                onChange={(e) => setTargetSell(e.target.value)}
                className="w-full px-3 py-2 border border-ic-border rounded focus:outline-none focus:ring-2 focus:ring-ic-blue"
              />
            </div>
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-ic-border rounded hover:bg-ic-surface-hover"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading}
              className="px-4 py-2 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
            >
              {loading ? 'Saving...' : 'Save'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
