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

  const formatVolume = (volume?: number) => {
    if (!volume) return '-';
    if (volume >= 1e9) return `${(volume / 1e9).toFixed(2)}B`;
    if (volume >= 1e6) return `${(volume / 1e6).toFixed(2)}M`;
    if (volume >= 1e3) return `${(volume / 1e3).toFixed(2)}K`;
    return volume.toLocaleString();
  };

  const formatMarketCap = (marketCap?: number) => {
    if (!marketCap) return '-';
    if (marketCap >= 1e12) return `$${(marketCap / 1e12).toFixed(2)}T`;
    if (marketCap >= 1e9) return `$${(marketCap / 1e9).toFixed(2)}B`;
    if (marketCap >= 1e6) return `$${(marketCap / 1e6).toFixed(2)}M`;
    return `$${marketCap.toLocaleString()}`;
  };

  const getAssetBadge = (assetType: string, exchange: string) => {
    const isCrypto = assetType.toLowerCase() === 'crypto' || assetType.toLowerCase() === 'cryptocurrency';
    if (isCrypto) {
      return <span className="inline-block px-2 py-1 text-xs font-semibold bg-purple-100 text-purple-800 rounded">CRYPTO</span>;
    }
    return <span className="inline-block px-2 py-1 text-xs font-semibold bg-gray-100 text-gray-700 rounded">{exchange}</span>;
  };

  // Check if price meets target conditions
  const checkTargetAlert = (item: WatchListItem) => {
    if (!item.current_price) return null;

    if (item.target_buy_price && item.current_price <= item.target_buy_price) {
      return {
        type: 'buy',
        message: 'At buy target',
        bgClass: 'bg-green-50 border-l-4 border-green-500',
      };
    }

    if (item.target_sell_price && item.current_price >= item.target_sell_price) {
      return {
        type: 'sell',
        message: 'At sell target',
        bgClass: 'bg-blue-50 border-l-4 border-blue-500',
      };
    }

    return null;
  };

  return (
    <div className="overflow-x-auto">
      <table className="w-full bg-white rounded-lg shadow">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-3 text-left text-sm font-semibold text-gray-900">Symbol</th>
            <th className="px-4 py-3 text-left text-sm font-semibold text-gray-900">Name</th>
            <th className="px-4 py-3 text-left text-sm font-semibold text-gray-900">Type</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Price</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Prev Close</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Change</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Volume</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Market Cap</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Target Buy</th>
            <th className="px-4 py-3 text-right text-sm font-semibold text-gray-900">Target Sell</th>
            <th className="px-4 py-3 text-center text-sm font-semibold text-gray-900">Alert</th>
            <th className="px-4 py-3 text-center text-sm font-semibold text-gray-900">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y">
          {items.map((item) => {
            const alert = checkTargetAlert(item);
            return (
              <tr key={item.symbol} className={`hover:bg-gray-50 ${alert ? alert.bgClass : ''}`}>
                <td className="px-4 py-3">
                  <Link href={`/ticker/${item.symbol}`} className="text-blue-600 hover:underline font-medium">
                    {item.symbol}
                  </Link>
                </td>
                <td className="px-4 py-3 text-sm text-gray-900">{item.name}</td>
                <td className="px-4 py-3">{getAssetBadge(item.asset_type, item.exchange)}</td>
                <td className="px-4 py-3 text-right font-medium text-gray-900">{formatPrice(item.current_price)}</td>
                <td className="px-4 py-3 text-right text-sm text-gray-600">{formatPrice(item.prev_close)}</td>
                <td className="px-4 py-3 text-right">{formatChange(item.price_change, item.price_change_pct)}</td>
                <td className="px-4 py-3 text-right text-sm text-gray-900">{formatVolume(item.volume)}</td>
                <td className="px-4 py-3 text-right text-sm text-gray-900">{formatMarketCap(item.market_cap)}</td>
                <td className="px-4 py-3 text-right text-sm">
                  {item.target_buy_price ? (
                    <span className={alert?.type === 'buy' ? 'font-bold text-green-700' : 'text-gray-900'}>
                      {formatPrice(item.target_buy_price)}
                    </span>
                  ) : '-'}
                </td>
                <td className="px-4 py-3 text-right text-sm">
                  {item.target_sell_price ? (
                    <span className={alert?.type === 'sell' ? 'font-bold text-blue-700' : 'text-gray-900'}>
                      {formatPrice(item.target_sell_price)}
                    </span>
                  ) : '-'}
                </td>
                <td className="px-4 py-3 text-center">
                  {alert && (
                    <span className={`inline-block px-2 py-1 text-xs font-semibold rounded ${
                      alert.type === 'buy' ? 'bg-green-100 text-green-800' : 'bg-blue-100 text-blue-800'
                    }`}>
                      {alert.message}
                    </span>
                  )}
                </td>
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
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
