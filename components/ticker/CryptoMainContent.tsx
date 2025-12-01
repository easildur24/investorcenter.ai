'use client';

interface CryptoMainContentProps {
  symbol: string;
  cryptoName: string;
}

export default function CryptoMainContent({ symbol, cryptoName }: CryptoMainContentProps) {
  const ticker = symbol.replace('X:', '');

  return (
    <div className="space-y-8">
      {/* Markets section - like CMC */}
      <div className="bg-ic-surface rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-6">{cryptoName} Markets</h2>

        {/* Market type filters */}
        <div className="flex space-x-4 mb-6">
          <button className="bg-ic-blue text-ic-text-primary px-4 py-2 rounded font-medium">ALL</button>
          <button className="text-ic-text-muted hover:text-ic-text-primary px-4 py-2">CEX</button>
          <button className="text-ic-text-muted hover:text-ic-text-primary px-4 py-2">DEX</button>
        </div>

        <div className="flex space-x-4 mb-6">
          <button className="bg-ic-bg-secondary text-ic-text-secondary px-4 py-2 rounded font-medium">Spot</button>
          <button className="text-ic-text-muted hover:text-ic-text-primary px-4 py-2">Perpetual</button>
          <button className="text-ic-text-muted hover:text-ic-text-primary px-4 py-2">Futures</button>
        </div>

        {/* Markets table placeholder */}
        <div className="bg-ic-bg-secondary rounded-lg p-8 text-center">
          <div className="text-ic-text-muted">Loading data...</div>
        </div>
      </div>

      {/* News section - like CMC */}
      <div className="bg-ic-surface rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-6">{cryptoName} News</h2>

        <div className="flex space-x-4 mb-6">
          <button className="bg-ic-blue text-ic-text-primary px-4 py-2 rounded font-medium">Top</button>
          <button className="text-ic-text-muted hover:text-ic-text-primary px-4 py-2">Latest</button>
        </div>

        <div className="space-y-4">
          <div className="border-b border-ic-border pb-4">
            <h3 className="font-semibold text-ic-text-primary mb-2">CMC Daily Analysis</h3>
            <p className="text-ic-text-muted text-sm">Latest market analysis and insights...</p>
          </div>

          <div className="bg-ic-bg-secondary rounded-lg p-4 text-center">
            <div className="text-ic-text-muted">Loading news...</div>
          </div>
        </div>
      </div>

      {/* Community section - like CMC */}
      <div className="bg-ic-surface rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-6">{cryptoName} community</h2>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
          <div>
            <h3 className="font-semibold text-ic-text-primary mb-3">{cryptoName} Yield</h3>
            <div className="bg-ic-bg-secondary rounded p-4 text-center">
              <div className="text-ic-text-muted text-sm">Loading...</div>
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-ic-text-primary mb-3">{cryptoName} Market Cycles</h3>
            <div className="bg-ic-bg-secondary rounded p-4 text-center">
              <div className="text-ic-text-muted text-sm">Loading...</div>
            </div>
          </div>

          <div>
            <h3 className="font-semibold text-ic-text-primary mb-3">NFTs on {cryptoName}</h3>
            <div className="bg-ic-bg-secondary rounded p-4 text-center">
              <div className="text-ic-text-muted text-sm">Loading...</div>
            </div>
          </div>
        </div>
      </div>

      {/* About section - like CMC */}
      <div className="bg-ic-surface rounded-lg shadow p-6">
        <h2 className="text-xl font-semibold text-ic-text-primary mb-6">About {cryptoName}</h2>

        <div className="space-y-6">
          <div>
            <h3 className="text-lg font-semibold text-ic-text-primary mb-3">What Is {cryptoName} ({ticker})?</h3>
            <p className="text-ic-text-secondary leading-relaxed">
              {cryptoName} is a decentralized cryptocurrency originally described in a 2008 whitepaper by a person,
              or group of people, using the alias Satoshi Nakamoto. It was launched soon after, in January 2009.
            </p>
            <p className="text-ic-text-secondary leading-relaxed mt-4">
              {cryptoName} is a peer-to-peer online currency, meaning that all transactions happen directly between
              equal, independent network participants, without the need for any intermediary to permit or facilitate them.
            </p>
          </div>

          <div>
            <h3 className="text-lg font-semibold text-ic-text-primary mb-3">Who Are the Founders of {cryptoName}?</h3>
            <p className="text-ic-text-secondary leading-relaxed">
              {cryptoName}'s original inventor is known under a pseudonym, Satoshi Nakamoto. As of 2021, the true identity
              of the person — or organization — that is behind the alias remains unknown.
            </p>
          </div>

          <div>
            <h3 className="text-lg font-semibold text-ic-text-primary mb-3">What Makes {cryptoName} Unique?</h3>
            <p className="text-ic-text-secondary leading-relaxed">
              {cryptoName}'s most unique advantage comes from the fact that it was the very first cryptocurrency to appear on the market.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
