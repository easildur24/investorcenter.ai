'use client';

import Link from 'next/link';
import { WatchListItem } from '@/lib/api/watchlist';

interface WatchListTableProps {
  items: WatchListItem[];
  onRemove: (symbol: string) => void;
  onEdit: (symbol: string) => void;
}

export default function WatchListTable({ items, onRemove, onEdit }: WatchListTableProps) {
  const formatPrice = (price?: number) => {
    if (price === undefined || price === null) return '-';
    return `$${price.toFixed(2)}`;
  };

  const formatChange = (change?: number, changePct?: number) => {
    if (change === undefined || changePct === undefined) return '-';
    const color = change >= 0 ? 'text-green-600' : 'text-red-600';
    const sign = change >= 0 ? '+' : '';
    return (
      <span className={color}>
        {sign}{change.toFixed(2)} ({sign}{changePct.toFixed(2)}%)
      </span>
    );
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full bg-white rounded-lg shadow">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold">Symbol</th>
            <th className="px-4 py-3 text-left text-sm font-semibold">Name</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Price</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Change</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Target Buy</th>
            <th className="px-4 py-3 text-right text-sm font-semibold">Target Sell</th>
            <th className="px-4 py-3 text-center text-sm font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {items.map((item) => (
            <tr key={item.symbol} className="hover:bg-gray-50">
              <td className="px-4 py-3">
                <Link href={`/ticker/${item.symbol}`} className="text-blue-600 hover:underline font-medium">
                  {item.symbol}
                </Link>
              </td>
              <td className="px-4 py-3 text-sm">{item.name}</td>
              <td className="px-4 py-3 text-right font-medium">{formatPrice(item.current_price)}</td>
              <td className="px-4 py-3 text-right">{formatChange(item.price_change, item.price_change_pct)}</td>
              <td className="px-4 py-3 text-right text-sm">{formatPrice(item.target_buy_price)}</td>
              <td className="px-4 py-3 text-right text-sm">{formatPrice(item.target_sell_price)}</td>
              <td className="px-4 py-3 text-center">
                <button
                  onClick={() => onEdit(item.symbol)}
                  className="text-blue-600 hover:underline text-sm mr-3"
                >
                  Edit
                </button>
                <button
                  onClick={() => onRemove(item.symbol)}
                  className="text-red-600 hover:underline text-sm"
                >
                  Remove
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
