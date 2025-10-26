'use client';

import Link from 'next/link';
import { ChartBarIcon } from '@heroicons/react/24/outline';
import TickerSearch from '@/components/TickerSearch';

export default function Header() {
  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex items-center">
            <div className="flex-shrink-0 flex items-center">
              <Link href="/" className="flex items-center">
                <ChartBarIcon className="h-8 w-8 text-primary-600" />
                <span className="ml-2 text-xl font-bold text-gray-900">InvestorCenter</span>
              </Link>
            </div>
            
            {/* Navigation Links */}
            <div className="hidden md:ml-6 md:flex md:space-x-8">
              <Link
                href="/"
                className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
              >
                Home
              </Link>
              <Link
                href="/crypto"
                className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
              >
                Crypto
              </Link>
              <Link
                href="/reddit"
                className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
              >
                Reddit Trends
              </Link>
            </div>
          </div>

          <div className="flex items-center space-x-4">
            {/* Ticker Search - Always visible */}
            <div className="hidden sm:block">
              <TickerSearch />
            </div>
            
            <button className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium">
              Login
            </button>
            <button className="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-md text-sm font-medium">
              Get Started
            </button>
          </div>
        </div>
      </div>

      {/* Mobile Search */}
      <div className="sm:hidden px-4 pb-3">
        <TickerSearch />
      </div>
    </nav>
  );
}
