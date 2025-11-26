'use client';

import Link from 'next/link';
import { ChartBarIcon, BellIcon } from '@heroicons/react/24/outline';
import TickerSearch from '@/components/TickerSearch';
import { useAuth } from '@/lib/auth/AuthContext';
import { useState, useEffect } from 'react';

export default function Header() {
  const { user, logout } = useAuth();
  const [showDropdown, setShowDropdown] = useState(false);

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
                href="/screener"
                className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
              >
                Screener
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
              {user && (
                <>
                  <Link
                    href="/watchlist"
                    className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
                  >
                    Watch Lists
                  </Link>
                  <Link
                    href="/alerts"
                    className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium"
                  >
                    Alerts
                  </Link>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center space-x-2 sm:space-x-4">
            {/* Ticker Search - Always visible on all screen sizes */}
            <div className="w-36 sm:w-auto">
              <TickerSearch />
            </div>

            {/* Alerts Bell Icon - Only when logged in */}
            {user && (
              <Link
                href="/alerts"
                className="relative p-2 text-gray-500 hover:text-gray-700 rounded-full hover:bg-gray-100"
                title="Alerts"
              >
                <BellIcon className="h-6 w-6" />
                {/* TODO: Add notification count badge here */}
              </Link>
            )}

            {user ? (
              <div className="relative">
                <button
                  onClick={() => setShowDropdown(!showDropdown)}
                  className="flex items-center gap-2 text-gray-700 hover:text-gray-900 px-3 py-2 rounded-md text-sm font-medium"
                >
                  <span>{user.full_name}</span>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                  </svg>
                </button>

                {showDropdown && (
                  <div className="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg py-1 z-10 border border-gray-200">
                    <Link
                      href="/watchlist"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      My Watch Lists
                    </Link>
                    <Link
                      href="/alerts"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      My Alerts
                    </Link>
                    <Link
                      href="/settings/profile"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      Settings
                    </Link>
                    <div className="border-t border-gray-200 my-1"></div>
                    <div className="px-4 py-2 text-xs font-semibold text-gray-500 uppercase">
                      Admin
                    </div>
                    <Link
                      href="/admin/dashboard"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      Dashboard
                    </Link>
                    <Link
                      href="/admin/cronjobs"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      Cronjob Monitoring
                    </Link>
                    <Link
                      href="/admin/ic-scores"
                      className="block px-4 py-2 text-gray-700 hover:bg-gray-100"
                      onClick={() => setShowDropdown(false)}
                    >
                      IC Score Admin
                    </Link>
                    <div className="border-t border-gray-200 my-1"></div>
                    <button
                      onClick={() => {
                        setShowDropdown(false);
                        logout();
                      }}
                      className="block w-full text-left px-4 py-2 text-gray-700 hover:bg-gray-100"
                    >
                      Logout
                    </button>
                  </div>
                )}
              </div>
            ) : (
              <>
                <Link href="/auth/login" className="text-gray-500 hover:text-gray-700 px-3 py-2 rounded-md text-sm font-medium">
                  Login
                </Link>
                <Link href="/auth/signup" className="bg-primary-600 hover:bg-primary-700 text-white px-4 py-2 rounded-md text-sm font-medium">
                  Get Started
                </Link>
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
