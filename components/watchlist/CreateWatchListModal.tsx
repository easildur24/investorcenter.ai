'use client';

import { useState } from 'react';

interface CreateWatchListModalProps {
  onClose: () => void;
  onCreate: (name: string, description?: string) => Promise<void>;
}

export default function CreateWatchListModal({ onClose, onCreate }: CreateWatchListModalProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await onCreate(name, description || undefined);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-ic-bg-primary bg-opacity-50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4 text-gray-900">Create Watch List</h2>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium mb-2 text-gray-700" htmlFor="name">
              Name *
            </label>
            <input
              id="name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-gray-900"
              placeholder="e.g., Tech Stocks, Growth Portfolio"
              required
              maxLength={255}
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium mb-2 text-gray-700" htmlFor="description">
              Description (optional)
            </label>
            <textarea
              id="description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-blue-500 text-gray-900"
              placeholder="Add notes about this watch list..."
              rows={3}
            />
          </div>

          <div className="flex gap-2 justify-end">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !name}
              className="px-4 py-2 bg-ic-blue text-ic-text-primary rounded hover:bg-ic-blue-hover disabled:bg-ic-bg-tertiary disabled:cursor-not-allowed"
            >
              {loading ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
