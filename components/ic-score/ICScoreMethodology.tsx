'use client';

import { useState } from 'react';

interface Section {
  id: string;
  title: string;
  content: React.ReactNode;
}

export default function ICScoreMethodology() {
  const [activeSection, setActiveSection] = useState<string>('overview');

  const sections: Section[] = [
    {
      id: 'overview',
      title: 'Overview',
      content: <OverviewSection />,
    },
    {
      id: 'factors',
      title: 'Scoring Factors',
      content: <FactorsSection />,
    },
    {
      id: 'categories',
      title: 'Categories',
      content: <CategoriesSection />,
    },
    {
      id: 'lifecycle',
      title: 'Lifecycle Classification',
      content: <LifecycleSection />,
    },
    {
      id: 'sector',
      title: 'Sector-Relative Scoring',
      content: <SectorRelativeSection />,
    },
    {
      id: 'confidence',
      title: 'Confidence & Data Quality',
      content: <ConfidenceSection />,
    },
    {
      id: 'stability',
      title: 'Score Stability',
      content: <StabilitySection />,
    },
    {
      id: 'interpretation',
      title: 'Interpretation Guide',
      content: <InterpretationSection />,
    },
  ];

  return (
    <div className="max-w-6xl mx-auto">
      <div className="mb-8">
        <h1 className="text-3xl font-bold">IC Score Methodology</h1>
        <p className="text-gray-600 mt-2">
          Understanding how the Investor Center Score evaluates stocks
        </p>
      </div>

      <div className="flex gap-8">
        {/* Sidebar Navigation */}
        <nav className="w-64 shrink-0">
          <div className="sticky top-4 bg-gray-50 rounded-lg p-4">
            <h3 className="font-semibold text-sm text-gray-500 uppercase mb-3">Contents</h3>
            <ul className="space-y-1">
              {sections.map((section) => (
                <li key={section.id}>
                  <button
                    onClick={() => setActiveSection(section.id)}
                    className={`w-full text-left px-3 py-2 rounded text-sm ${
                      activeSection === section.id
                        ? 'bg-blue-100 text-blue-700 font-medium'
                        : 'text-gray-700 hover:bg-gray-100'
                    }`}
                  >
                    {section.title}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </nav>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="bg-white rounded-lg border p-6">
            {sections.find((s) => s.id === activeSection)?.content}
          </div>
        </div>
      </div>
    </div>
  );
}

function OverviewSection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">What is the IC Score?</h2>

      <p className="text-gray-700">
        The Investor Center Score (IC Score) is a comprehensive stock evaluation metric
        that combines fundamental analysis, market signals, and quantitative factors
        to produce a single score from 0-100.
      </p>

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold text-blue-900 mb-2">Key Features</h3>
        <ul className="list-disc list-inside text-blue-800 space-y-1">
          <li><strong>Sector-Relative:</strong> Compares stocks against peers in the same sector</li>
          <li><strong>Lifecycle-Aware:</strong> Adjusts factor weights based on company stage</li>
          <li><strong>Multi-Factor:</strong> Combines 10+ individual factors across 3 categories</li>
          <li><strong>Transparent:</strong> Shows detailed breakdown of all contributing factors</li>
        </ul>
      </div>

      <h3 className="text-xl font-semibold mt-6">Score Ratings</h3>

      <table className="w-full border-collapse">
        <thead>
          <tr className="border-b">
            <th className="text-left py-2">Score Range</th>
            <th className="text-left py-2">Rating</th>
            <th className="text-left py-2">Interpretation</th>
          </tr>
        </thead>
        <tbody>
          <tr className="border-b">
            <td className="py-2">80-100</td>
            <td className="py-2"><span className="px-2 py-1 bg-emerald-100 text-emerald-800 rounded">Strong Buy</span></td>
            <td className="py-2 text-gray-600">Exceptional fundamentals and momentum</td>
          </tr>
          <tr className="border-b">
            <td className="py-2">65-79</td>
            <td className="py-2"><span className="px-2 py-1 bg-green-100 text-green-800 rounded">Buy</span></td>
            <td className="py-2 text-gray-600">Above-average characteristics</td>
          </tr>
          <tr className="border-b">
            <td className="py-2">50-64</td>
            <td className="py-2"><span className="px-2 py-1 bg-yellow-100 text-yellow-800 rounded">Hold</span></td>
            <td className="py-2 text-gray-600">Mixed signals, wait for clarity</td>
          </tr>
          <tr className="border-b">
            <td className="py-2">35-49</td>
            <td className="py-2"><span className="px-2 py-1 bg-orange-100 text-orange-800 rounded">Sell</span></td>
            <td className="py-2 text-gray-600">Below-average profile</td>
          </tr>
          <tr>
            <td className="py-2">0-34</td>
            <td className="py-2"><span className="px-2 py-1 bg-red-100 text-red-800 rounded">Strong Sell</span></td>
            <td className="py-2 text-gray-600">Significant concerns</td>
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function FactorsSection() {
  const factors = [
    {
      name: 'Growth',
      weight: '12%',
      category: 'Quality',
      description: 'Revenue growth, EPS growth, and growth consistency',
      metrics: ['Revenue Growth YoY', 'EPS Growth YoY', 'Revenue Growth CAGR (3Y)'],
    },
    {
      name: 'Profitability',
      weight: '12%',
      category: 'Quality',
      description: 'Margin quality and return on capital',
      metrics: ['Net Margin', 'ROE', 'ROA', 'Gross Margin'],
    },
    {
      name: 'Financial Health',
      weight: '10%',
      category: 'Quality',
      description: 'Balance sheet strength and debt management',
      metrics: ['Debt/Equity', 'Current Ratio', 'Interest Coverage'],
    },
    {
      name: 'Relative Value',
      weight: '12%',
      category: 'Valuation',
      description: 'Valuation vs sector peers',
      metrics: ['P/E Ratio', 'P/S Ratio', 'P/B Ratio', 'EV/EBITDA'],
    },
    {
      name: 'Intrinsic Value',
      weight: '10%',
      category: 'Valuation',
      description: 'DCF-based fair value assessment',
      metrics: ['DCF Value', 'Margin of Safety'],
    },
    {
      name: 'Historical Value',
      weight: '8%',
      category: 'Valuation',
      description: 'Current valuation vs 5-year history',
      metrics: ['P/E Percentile (5Y)', 'P/S Percentile (5Y)'],
    },
    {
      name: 'Earnings Revisions',
      weight: '8%',
      category: 'Valuation',
      description: 'Analyst estimate changes',
      metrics: ['EPS Revision (90D)', 'Revision Breadth', 'Recency'],
    },
    {
      name: 'Momentum',
      weight: '10%',
      category: 'Signals',
      description: 'Price trend strength',
      metrics: ['12-1 Month Return', '50/200 MA Position'],
    },
    {
      name: 'Smart Money',
      weight: '10%',
      category: 'Signals',
      description: 'Institutional and insider activity',
      metrics: ['Analyst Ratings', 'Insider Transactions', '13F Holdings'],
    },
    {
      name: 'Technical Signals',
      weight: '8%',
      category: 'Signals',
      description: 'Technical indicator composite',
      metrics: ['RSI', 'MACD', 'Volume Trends'],
    },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Scoring Factors</h2>

      <p className="text-gray-700">
        The IC Score combines 10 individual factors, each measuring a specific aspect
        of stock quality, valuation, or market signals.
      </p>

      <div className="space-y-4">
        {factors.map((factor) => (
          <div key={factor.name} className="border rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <h3 className="font-semibold text-lg">{factor.name}</h3>
              <div className="flex items-center gap-2">
                <span className={`px-2 py-1 text-xs rounded ${
                  factor.category === 'Quality' ? 'bg-blue-100 text-blue-800' :
                  factor.category === 'Valuation' ? 'bg-purple-100 text-purple-800' :
                  'bg-green-100 text-green-800'
                }`}>
                  {factor.category}
                </span>
                <span className="text-sm font-medium text-gray-600">{factor.weight}</span>
              </div>
            </div>
            <p className="text-gray-600 text-sm mb-2">{factor.description}</p>
            <div className="flex flex-wrap gap-2">
              {factor.metrics.map((metric) => (
                <span key={metric} className="px-2 py-1 bg-gray-100 text-gray-700 text-xs rounded">
                  {metric}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function CategoriesSection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Score Categories</h2>

      <p className="text-gray-700">
        Factors are grouped into three categories for easier interpretation.
      </p>

      <div className="grid gap-6">
        <CategoryCard
          name="Quality"
          weight="~34%"
          color="blue"
          description="Measures the fundamental quality of the business"
          factors={['Growth', 'Profitability', 'Financial Health']}
          interpretation="High quality scores indicate strong business fundamentals, consistent growth, and healthy finances."
        />

        <CategoryCard
          name="Valuation"
          weight="~38%"
          color="purple"
          description="Assesses whether the stock is fairly priced"
          factors={['Relative Value', 'Intrinsic Value', 'Historical Value', 'Earnings Revisions']}
          interpretation="High valuation scores suggest the stock is undervalued relative to peers, history, and intrinsic worth."
        />

        <CategoryCard
          name="Signals"
          weight="~28%"
          color="green"
          description="Captures market momentum and sentiment"
          factors={['Momentum', 'Smart Money', 'Technical Signals']}
          interpretation="High signal scores indicate positive price momentum and favorable institutional sentiment."
        />
      </div>
    </div>
  );
}

function CategoryCard({
  name,
  weight,
  color,
  description,
  factors,
  interpretation,
}: {
  name: string;
  weight: string;
  color: 'blue' | 'purple' | 'green';
  description: string;
  factors: string[];
  interpretation: string;
}) {
  const colorClasses = {
    blue: 'bg-blue-50 border-blue-200',
    purple: 'bg-purple-50 border-purple-200',
    green: 'bg-green-50 border-green-200',
  };

  return (
    <div className={`border rounded-lg p-5 ${colorClasses[color]}`}>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-xl font-semibold">{name}</h3>
        <span className="text-sm font-medium text-gray-600">{weight} of total</span>
      </div>
      <p className="text-gray-700 mb-3">{description}</p>
      <div className="flex flex-wrap gap-2 mb-3">
        {factors.map((factor) => (
          <span key={factor} className="px-2 py-1 bg-white rounded text-sm">
            {factor}
          </span>
        ))}
      </div>
      <p className="text-sm text-gray-600 italic">{interpretation}</p>
    </div>
  );
}

function LifecycleSection() {
  const stages = [
    {
      name: 'Hypergrowth',
      criteria: 'Revenue growth >50%',
      emphasis: 'Growth factors weighted higher, valuation de-emphasized',
      examples: 'Early-stage tech, disruptive innovators',
    },
    {
      name: 'Growth',
      criteria: 'Revenue growth 20-50%',
      emphasis: 'Balanced with slight growth tilt',
      examples: 'Scaling tech companies, market share gainers',
    },
    {
      name: 'Mature',
      criteria: 'Stable growth, profitable',
      emphasis: 'Profitability and value emphasized',
      examples: 'Established market leaders',
    },
    {
      name: 'Value',
      criteria: 'Low P/E, positive margins',
      emphasis: 'Valuation factors weighted heavily',
      examples: 'Dividend payers, turnaround candidates',
    },
    {
      name: 'Turnaround',
      criteria: 'Declining revenue, restructuring',
      emphasis: 'Financial health and momentum emphasized',
      examples: 'Companies in transition',
    },
  ];

  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Lifecycle Classification</h2>

      <p className="text-gray-700">
        Companies at different stages deserve different evaluation criteria. A hypergrowth
        startup shouldn&apos;t be penalized for low profitability the same way a mature
        company would be.
      </p>

      <div className="space-y-4">
        {stages.map((stage) => (
          <div key={stage.name} className="border rounded-lg p-4">
            <h3 className="font-semibold text-lg mb-2">{stage.name}</h3>
            <div className="grid grid-cols-3 gap-4 text-sm">
              <div>
                <p className="text-gray-500">Classification Criteria</p>
                <p className="text-gray-800">{stage.criteria}</p>
              </div>
              <div>
                <p className="text-gray-500">Factor Emphasis</p>
                <p className="text-gray-800">{stage.emphasis}</p>
              </div>
              <div>
                <p className="text-gray-500">Examples</p>
                <p className="text-gray-800">{stage.examples}</p>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function SectorRelativeSection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Sector-Relative Scoring</h2>

      <p className="text-gray-700">
        Raw financial metrics are converted to sector percentiles before scoring.
        This ensures fair comparison across different industries with varying
        characteristics.
      </p>

      <div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
        <h3 className="font-semibold text-amber-900 mb-2">Why Sector-Relative?</h3>
        <p className="text-amber-800">
          A 15% net margin might be excellent for a retailer but below average for a
          software company. Sector-relative scoring accounts for these differences.
        </p>
      </div>

      <h3 className="text-xl font-semibold mt-6">How It Works</h3>

      <ol className="list-decimal list-inside space-y-3 text-gray-700">
        <li>
          <strong>Calculate sector statistics:</strong> For each metric, we compute
          percentile distributions across all companies in the sector.
        </li>
        <li>
          <strong>Convert to percentile:</strong> A stock&apos;s raw metric is converted
          to its percentile rank within the sector (0-100).
        </li>
        <li>
          <strong>Invert where appropriate:</strong> For metrics where lower is better
          (e.g., P/E ratio), we invert the percentile so higher scores are always better.
        </li>
        <li>
          <strong>Apply to factor calculation:</strong> The sector-relative percentile
          feeds into the factor score calculation.
        </li>
      </ol>

      <div className="mt-6 p-4 bg-gray-100 rounded-lg">
        <h4 className="font-medium mb-2">Example: Valuation Comparison</h4>
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b">
              <th className="text-left py-2">Stock</th>
              <th className="text-left py-2">Sector</th>
              <th className="text-right py-2">P/E</th>
              <th className="text-right py-2">Sector Percentile</th>
              <th className="text-right py-2">Value Score</th>
            </tr>
          </thead>
          <tbody>
            <tr className="border-b">
              <td className="py-2">MSFT</td>
              <td className="py-2">Technology</td>
              <td className="py-2 text-right">32x</td>
              <td className="py-2 text-right">45th</td>
              <td className="py-2 text-right">55</td>
            </tr>
            <tr>
              <td className="py-2">WMT</td>
              <td className="py-2">Consumer Defensive</td>
              <td className="py-2 text-right">25x</td>
              <td className="py-2 text-right">30th</td>
              <td className="py-2 text-right">70</td>
            </tr>
          </tbody>
        </table>
        <p className="text-xs text-gray-600 mt-2">
          Despite having a lower P/E, MSFT scores lower on valuation because Tech
          companies typically trade at higher multiples.
        </p>
      </div>
    </div>
  );
}

function ConfidenceSection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Confidence & Data Quality</h2>

      <p className="text-gray-700">
        Not all IC Scores are created equal. The confidence level indicates how much
        data was available and how recent it is.
      </p>

      <h3 className="text-xl font-semibold mt-6">Confidence Levels</h3>

      <div className="space-y-3">
        <div className="flex items-center gap-4 p-3 bg-green-50 border border-green-200 rounded">
          <span className="px-3 py-1 bg-green-600 text-white rounded font-medium">High</span>
          <p className="text-gray-700">
            80%+ data availability, all factors calculated, recent financials
          </p>
        </div>
        <div className="flex items-center gap-4 p-3 bg-yellow-50 border border-yellow-200 rounded">
          <span className="px-3 py-1 bg-yellow-600 text-white rounded font-medium">Medium</span>
          <p className="text-gray-700">
            60-80% data, some factors estimated or using older data
          </p>
        </div>
        <div className="flex items-center gap-4 p-3 bg-red-50 border border-red-200 rounded">
          <span className="px-3 py-1 bg-red-600 text-white rounded font-medium">Low</span>
          <p className="text-gray-700">
            &lt;60% data, significant factors missing, use with caution
          </p>
        </div>
      </div>

      <h3 className="text-xl font-semibold mt-6">Per-Factor Data Status</h3>

      <p className="text-gray-700 mb-3">
        Each factor shows its individual data availability:
      </p>

      <ul className="list-disc list-inside space-y-2 text-gray-700">
        <li><strong>Available:</strong> All required data present</li>
        <li><strong>Partial:</strong> Some metrics missing, factor calculated with available data</li>
        <li><strong>Unavailable:</strong> Critical data missing, factor excluded from score</li>
      </ul>
    </div>
  );
}

function StabilitySection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Score Stability</h2>

      <p className="text-gray-700">
        To prevent score whipsaw from daily data fluctuations, we apply exponential
        smoothing to the score calculation.
      </p>

      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h3 className="font-semibold text-blue-900 mb-2">Smoothing Formula</h3>
        <code className="block bg-white p-2 rounded text-sm">
          Smoothed Score = 0.7 × New Score + 0.3 × Previous Score
        </code>
        <p className="text-blue-800 mt-2 text-sm">
          70% weight to new calculation, 30% to previous score
        </p>
      </div>

      <h3 className="text-xl font-semibold mt-6">Event-Based Resets</h3>

      <p className="text-gray-700 mb-3">
        Smoothing is bypassed when significant events occur, allowing scores to
        react immediately:
      </p>

      <ul className="list-disc list-inside space-y-1 text-gray-700">
        <li>Earnings releases</li>
        <li>Analyst rating changes</li>
        <li>Large insider transactions (&gt;$100k)</li>
        <li>Dividend announcements</li>
        <li>M&A news</li>
        <li>Guidance updates</li>
      </ul>

      <h3 className="text-xl font-semibold mt-6">Minimum Change Threshold</h3>

      <p className="text-gray-700">
        Score changes less than 0.5 points are not displayed to users. This prevents
        noise from appearing as meaningful movement.
      </p>
    </div>
  );
}

function InterpretationSection() {
  return (
    <div className="space-y-6">
      <h2 className="text-2xl font-bold">Interpretation Guide</h2>

      <div className="space-y-4">
        <div className="border rounded-lg p-4">
          <h3 className="font-semibold text-lg mb-2">DO use IC Score for:</h3>
          <ul className="list-disc list-inside space-y-1 text-gray-700">
            <li>Screening and filtering stocks for further research</li>
            <li>Comparing similar companies within a sector</li>
            <li>Identifying potential opportunities and red flags</li>
            <li>Tracking score trends over time</li>
            <li>Understanding the market&apos;s view of a stock</li>
          </ul>
        </div>

        <div className="border rounded-lg p-4">
          <h3 className="font-semibold text-lg mb-2">DO NOT use IC Score for:</h3>
          <ul className="list-disc list-inside space-y-1 text-gray-700">
            <li>Making buy/sell decisions without additional research</li>
            <li>Comparing stocks across very different sectors</li>
            <li>Short-term trading signals</li>
            <li>Predicting exact price movements</li>
            <li>Replacing professional financial advice</li>
          </ul>
        </div>
      </div>

      <div className="mt-6 p-4 bg-gray-100 rounded-lg">
        <h3 className="font-semibold mb-2">Important Disclaimers</h3>
        <ul className="list-disc list-inside space-y-1 text-sm text-gray-600">
          <li>Past performance does not guarantee future results</li>
          <li>IC Scores are based on historical and current data, not predictions</li>
          <li>Data quality varies by company and may contain errors</li>
          <li>Always conduct your own due diligence before investing</li>
          <li>Consult a qualified financial advisor for personalized advice</li>
        </ul>
      </div>
    </div>
  );
}
