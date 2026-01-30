'use client';

import { useState, useEffect } from 'react';
import { Sparkles, TrendingUp, TrendingDown, AlertTriangle, Target, Loader2, RefreshCw } from 'lucide-react';
import { ICScoreData } from '@/lib/api/ic-score';

interface AIAnalysis {
  ticker: string;
  analysis: string;
  key_strengths: string[];
  key_concerns: string[];
  investment_thesis: string;
  risk_factors: string[];
  generated_at: string;
}

interface ICScoreAIAnalysisProps {
  icScore: ICScoreData;
  companyName?: string;
  sector?: string;
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export default function ICScoreAIAnalysis({ icScore, companyName, sector }: ICScoreAIAnalysisProps) {
  const [analysis, setAnalysis] = useState<AIAnalysis | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);

  const fetchAnalysis = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/stocks/${icScore.ticker}/ic-score/ai-analysis`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ticker: icScore.ticker,
          company_name: companyName,
          sector: sector,
        }),
      });

      if (!response.ok) {
        if (response.status === 503) {
          throw new Error('AI analysis is currently unavailable');
        }
        throw new Error('Failed to fetch AI analysis');
      }

      const data = await response.json();
      setAnalysis(data);
      setExpanded(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  if (!expanded && !analysis) {
    return (
      <div className="mt-4">
        <button
          onClick={fetchAnalysis}
          disabled={loading}
          className="w-full flex items-center justify-center gap-2 px-4 py-3 bg-gradient-to-r from-purple-500 to-indigo-500 text-white rounded-lg hover:from-purple-600 hover:to-indigo-600 transition-all disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {loading ? (
            <>
              <Loader2 className="w-5 h-5 animate-spin" />
              <span>Generating AI Analysis...</span>
            </>
          ) : (
            <>
              <Sparkles className="w-5 h-5" />
              <span>Get AI Analysis</span>
            </>
          )}
        </button>

        {error && (
          <div className="mt-2 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
            {error}
          </div>
        )}
      </div>
    );
  }

  if (!analysis) {
    return null;
  }

  return (
    <div className="mt-4 bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg border border-purple-200 overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 bg-gradient-to-r from-purple-500 to-indigo-500 flex items-center justify-between">
        <div className="flex items-center gap-2 text-white">
          <Sparkles className="w-5 h-5" />
          <span className="font-semibold">AI Analysis</span>
        </div>
        <button
          onClick={fetchAnalysis}
          disabled={loading}
          className="p-1.5 bg-white/20 rounded hover:bg-white/30 transition-colors text-white disabled:opacity-50"
          title="Refresh analysis"
        >
          <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
        </button>
      </div>

      {/* Content */}
      <div className="p-4 space-y-4">
        {/* Summary */}
        <div>
          <p className="text-sm text-ic-text-secondary leading-relaxed">
            {analysis.analysis}
          </p>
        </div>

        {/* Strengths & Concerns Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {/* Strengths */}
          <div className="bg-green-50 rounded-lg p-3 border border-green-100">
            <div className="flex items-center gap-2 text-green-700 font-medium mb-2">
              <TrendingUp className="w-4 h-4" />
              <span>Key Strengths</span>
            </div>
            <ul className="space-y-1">
              {analysis.key_strengths.map((strength, idx) => (
                <li key={idx} className="text-sm text-green-800 flex items-start gap-2">
                  <span className="text-green-500 mt-1">+</span>
                  <span>{strength}</span>
                </li>
              ))}
            </ul>
          </div>

          {/* Concerns */}
          <div className="bg-amber-50 rounded-lg p-3 border border-amber-100">
            <div className="flex items-center gap-2 text-amber-700 font-medium mb-2">
              <TrendingDown className="w-4 h-4" />
              <span>Key Concerns</span>
            </div>
            <ul className="space-y-1">
              {analysis.key_concerns.map((concern, idx) => (
                <li key={idx} className="text-sm text-amber-800 flex items-start gap-2">
                  <span className="text-amber-500 mt-1">-</span>
                  <span>{concern}</span>
                </li>
              ))}
            </ul>
          </div>
        </div>

        {/* Investment Thesis */}
        <div className="bg-blue-50 rounded-lg p-3 border border-blue-100">
          <div className="flex items-center gap-2 text-blue-700 font-medium mb-2">
            <Target className="w-4 h-4" />
            <span>Investment Thesis</span>
          </div>
          <p className="text-sm text-blue-800">
            {analysis.investment_thesis}
          </p>
        </div>

        {/* Risk Factors */}
        <div className="bg-red-50 rounded-lg p-3 border border-red-100">
          <div className="flex items-center gap-2 text-red-700 font-medium mb-2">
            <AlertTriangle className="w-4 h-4" />
            <span>Risk Factors</span>
          </div>
          <ul className="space-y-1">
            {analysis.risk_factors.map((risk, idx) => (
              <li key={idx} className="text-sm text-red-800 flex items-start gap-2">
                <span className="text-red-500 mt-1">!</span>
                <span>{risk}</span>
              </li>
            ))}
          </ul>
        </div>

        {/* Disclaimer */}
        <div className="pt-3 border-t border-purple-200">
          <p className="text-xs text-ic-text-dim">
            AI-generated analysis for informational purposes only. Not investment advice.
            Generated {new Date(analysis.generated_at).toLocaleString()}.
          </p>
        </div>
      </div>
    </div>
  );
}
