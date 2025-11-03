'use client';

import { useState } from 'react';

interface AddTickerModalProps {
  onClose: () => void;
  onAdd: (symbol: string, notes?: string, tags?: string[], targetBuy?: number, targetSell?: number) => Promise<void>;
}

export default function AddTickerModal({ onClose, onAdd }: AddTickerModalProps) {
  const [symbol, setSymbol] = useState('');
  const [notes, setNotes] = useState('');
  const [tags, setTags] = useState('');
  const [targetBuy, setTargetBuy] = useState('');
  const [targetSell, setTargetSell] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const tagArray = tags.split(',').map(t => t.trim()).filter(t => t);
      await onAdd(
        symbol.toUpperCase(),
        notes || undefined,
        tagArray.length > 0 ? tagArray : undefined,
        targetBuy ? parseFloat(targetBuy) : undefined,
        targetSell ? parseFloat(targetSell) : undefined
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4">Add Ticker</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="symbol">
              Symbol *
            </label>
            <input
              id="symbol"
              type="text"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
              placeholder="e.g., AAPL, X:BTCUSD"
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              required
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-2" htmlFor="notes">
              Notes
            </label>
            <textarea
              id="notes"
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows={2}
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
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
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
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
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
                className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !symbol}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed"
            >
              {loading ? 'Adding...' : 'Add'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
