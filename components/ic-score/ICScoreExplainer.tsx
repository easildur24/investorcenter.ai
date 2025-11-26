'use client';

import { useState } from 'react';
import { XMarkIcon, QuestionMarkCircleIcon } from '@heroicons/react/24/outline';
import { getFactorDetails, ICScoreData, getScoreColor } from '@/lib/api/ic-score';

interface ICScoreExplainerProps {
  icScore?: ICScoreData;
  onClose?: () => void;
}

// Factor definitions with detailed explanations
const factorDefinitions = [
  {
    name: 'value',
    displayName: 'Value',
    weight: 12,
    description: 'Evaluates how attractively priced the stock is relative to its fundamentals.',
    metrics: ['Price-to-Earnings (P/E)', 'Price-to-Book (P/B)', 'Price-to-Sales (P/S)'],
    interpretation: 'Lower ratios compared to sector peers indicate better value.',
  },
  {
    name: 'growth',
    displayName: 'Growth',
    weight: 15,
    description: 'Measures the company\'s historical and projected growth trajectory.',
    metrics: ['Revenue Growth (YoY)', 'Earnings Growth (YoY)', 'Forward EPS Growth'],
    interpretation: 'Higher growth rates earn better scores, adjusted for consistency.',
  },
  {
    name: 'profitability',
    displayName: 'Profitability',
    weight: 12,
    description: 'Assesses how efficiently the company generates profits.',
    metrics: ['Return on Equity (ROE)', 'Return on Assets (ROA)', 'Net Profit Margin'],
    interpretation: 'Higher margins and returns indicate better operational efficiency.',
  },
  {
    name: 'financial_health',
    displayName: 'Financial Health',
    weight: 10,
    description: 'Evaluates the company\'s balance sheet strength and liquidity.',
    metrics: ['Debt-to-Equity Ratio', 'Current Ratio', 'Interest Coverage'],
    interpretation: 'Lower debt and higher liquidity ratios indicate financial stability.',
  },
  {
    name: 'momentum',
    displayName: 'Momentum',
    weight: 8,
    description: 'Tracks price trends and relative strength over time.',
    metrics: ['Price vs 50-day MA', 'Price vs 200-day MA', 'Relative Strength'],
    interpretation: 'Positive momentum suggests continued price appreciation.',
  },
  {
    name: 'analyst_consensus',
    displayName: 'Analyst Consensus',
    weight: 10,
    description: 'Aggregates professional analyst opinions and price targets.',
    metrics: ['Buy/Hold/Sell Ratings', 'Price Target vs Current', 'Rating Changes'],
    interpretation: 'Strong buy consensus and upside to targets earn higher scores.',
  },
  {
    name: 'insider_activity',
    displayName: 'Insider Activity',
    weight: 8,
    description: 'Monitors buying and selling by company executives and directors.',
    metrics: ['Net Insider Purchases', 'Transaction Value', 'Buyer/Seller Ratio'],
    interpretation: 'Net insider buying suggests confidence in the company.',
  },
  {
    name: 'institutional',
    displayName: 'Institutional Ownership',
    weight: 10,
    description: 'Tracks ownership changes by large institutional investors.',
    metrics: ['Ownership %', 'Quarterly Changes', 'New Positions'],
    interpretation: 'Increasing institutional ownership indicates smart money confidence.',
  },
  {
    name: 'news_sentiment',
    displayName: 'News Sentiment',
    weight: 7,
    description: 'Analyzes sentiment from news articles and press releases.',
    metrics: ['Sentiment Score', 'News Volume', 'Headline Analysis'],
    interpretation: 'Positive sentiment across news sources improves the score.',
  },
  {
    name: 'technical',
    displayName: 'Technical Indicators',
    weight: 8,
    description: 'Evaluates chart patterns and technical trading signals.',
    metrics: ['RSI', 'MACD', 'Bollinger Bands', 'Volume Trends'],
    interpretation: 'Bullish technical signals contribute to higher scores.',
  },
];

export function ICScoreExplainerButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="inline-flex items-center gap-1 text-sm text-gray-500 hover:text-primary-600 transition-colors"
      title="Learn how IC Score works"
    >
      <QuestionMarkCircleIcon className="h-4 w-4" />
      <span className="hidden sm:inline">How it works</span>
    </button>
  );
}

export default function ICScoreExplainer({ icScore, onClose }: ICScoreExplainerProps) {
  const [selectedFactor, setSelectedFactor] = useState<string | null>(null);

  const factorScores = icScore ? getFactorDetails(icScore) : null;

  const getFactorScore = (factorName: string) => {
    if (!factorScores) return null;
    const factor = factorScores.find(f => f.name === factorName);
    return factor?.score ?? null;
  };

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto" aria-labelledby="ic-score-explainer" role="dialog" aria-modal="true">
      <div className="flex items-center justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
        {/* Background overlay */}
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={onClose}></div>

        {/* Modal panel */}
        <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-4xl sm:w-full">
          {/* Header */}
          <div className="bg-gradient-to-r from-primary-600 to-primary-700 px-6 py-4 flex justify-between items-center">
            <div>
              <h2 className="text-xl font-semibold text-white">Understanding IC Score</h2>
              <p className="text-primary-100 text-sm">Our proprietary 10-factor investment analysis</p>
            </div>
            <button
              onClick={onClose}
              className="text-white hover:text-primary-100 transition-colors"
            >
              <XMarkIcon className="h-6 w-6" />
            </button>
          </div>

          {/* Content */}
          <div className="px-6 py-6 max-h-[70vh] overflow-y-auto">
            {/* Overview */}
            <div className="mb-8">
              <h3 className="text-lg font-semibold text-gray-900 mb-3">What is IC Score?</h3>
              <p className="text-gray-600 mb-4">
                IC Score is InvestorCenter&apos;s proprietary investment rating that combines 10 key factors
                to provide a comprehensive view of a stock&apos;s investment potential. Each factor is weighted
                based on its historical predictive power and contribution to investment returns.
              </p>
              <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                <p className="text-sm text-blue-800">
                  <strong>Score Range:</strong> 0-100 |
                  <strong className="ml-2">80+:</strong> Strong Buy |
                  <strong className="ml-2">65-79:</strong> Buy |
                  <strong className="ml-2">50-64:</strong> Hold |
                  <strong className="ml-2">&lt;50:</strong> Caution
                </p>
              </div>
            </div>

            {/* Factor Weight Visualization */}
            <div className="mb-8">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Factor Weights</h3>
              <div className="space-y-2">
                {factorDefinitions.map((factor) => {
                  const score = getFactorScore(factor.name);
                  const scoreColor = score !== null ? getScoreColor(score) : 'text-gray-400';

                  return (
                    <div
                      key={factor.name}
                      className={`flex items-center gap-3 p-2 rounded-lg cursor-pointer transition-colors ${
                        selectedFactor === factor.name ? 'bg-primary-50 ring-1 ring-primary-200' : 'hover:bg-gray-50'
                      }`}
                      onClick={() => setSelectedFactor(selectedFactor === factor.name ? null : factor.name)}
                    >
                      <div className="w-32 sm:w-40 flex-shrink-0">
                        <span className="text-sm font-medium text-gray-700">{factor.displayName}</span>
                      </div>
                      <div className="flex-grow">
                        <div className="h-6 bg-gray-100 rounded-full overflow-hidden">
                          <div
                            className="h-full bg-primary-500 rounded-full flex items-center justify-end pr-2"
                            style={{ width: `${factor.weight * 5}%` }}
                          >
                            <span className="text-xs font-medium text-white">{factor.weight}%</span>
                          </div>
                        </div>
                      </div>
                      {icScore && (
                        <div className={`w-12 text-right text-sm font-semibold ${scoreColor}`}>
                          {score !== null ? Math.round(score) : 'N/A'}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>

            {/* Selected Factor Detail */}
            {selectedFactor && (
              <div className="mb-8 bg-gray-50 rounded-lg p-4 animate-fadeIn">
                {(() => {
                  const factor = factorDefinitions.find(f => f.name === selectedFactor);
                  if (!factor) return null;
                  return (
                    <>
                      <h4 className="font-semibold text-gray-900 mb-2">{factor.displayName}</h4>
                      <p className="text-gray-600 text-sm mb-3">{factor.description}</p>
                      <div className="mb-3">
                        <span className="text-xs font-medium text-gray-500 uppercase">Key Metrics:</span>
                        <div className="flex flex-wrap gap-2 mt-1">
                          {factor.metrics.map((metric) => (
                            <span key={metric} className="text-xs bg-white border border-gray-200 rounded px-2 py-1">
                              {metric}
                            </span>
                          ))}
                        </div>
                      </div>
                      <p className="text-sm text-gray-500 italic">{factor.interpretation}</p>
                    </>
                  );
                })()}
              </div>
            )}

            {/* Disclaimer */}
            <div className="text-xs text-gray-400 border-t border-gray-200 pt-4">
              <p>
                <strong>Disclaimer:</strong> IC Score is for informational purposes only and should not be considered
                investment advice. Past performance does not guarantee future results. Always conduct your own
                research and consult a financial advisor before making investment decisions.
              </p>
            </div>
          </div>

          {/* Footer */}
          <div className="bg-gray-50 px-6 py-4 flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
            >
              Got it
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
