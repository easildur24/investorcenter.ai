# Product Specification: Technical Analysis Features
## InvestorCenter.ai

**Version:** 1.0
**Date:** November 12, 2025
**Status:** Draft
**Priority:** P1 (Must-Have - MVP)

---

## Overview

This document defines technical analysis features including advanced charting, 100+ technical indicators, pattern recognition, and economic data integration to compete with Seeking Alpha's 130+ indicators and YCharts' visualization excellence.

---

## Feature 1: Advanced Charting System

### Business Objectives
- Match Seeking Alpha's charting capabilities
- Provide professional-grade analysis tools
- Support multiple analysis styles (technical, fundamental, macro)
- Enable custom chart configurations and sharing

### Chart Types

**Standard Charts:**
- Line Chart (simple price visualization)
- OHLC (Open, High, Low, Close) bars
- Candlestick (Japanese candlesticks)
- Area Chart (filled line chart)

**Advanced Charts:**
- Heikin-Ashi (smoothed candlesticks)
- Renko (removes time, focuses on price movement)
- Point & Figure (X's and O's, removes time and volume)
- Kagi (thick/thin lines based on reversals)

### Timeframes
- Intraday: 1min, 5min, 15min, 30min, 1hour, 4hour
- Daily: 1day
- Weekly: 1week
- Monthly: 1month
- Custom: User-defined ranges

### Drawing Tools

**Lines & Shapes:**
- Trend lines (auto-extend option)
- Horizontal/Vertical lines
- Channels (parallel trend lines)
- Rectangles (support/resistance zones)
- Circles/Ellipses

**Fibonacci Tools:**
- Fibonacci Retracement (23.6%, 38.2%, 50%, 61.8%, 78.6%)
- Fibonacci Extension
- Fibonacci Fan
- Fibonacci Time Zones
- Fibonacci Arcs

**Pattern Tools:**
- Head & Shoulders marker
- Triangle marker
- Flag/Pennant marker
- Custom annotations
- Text notes with arrows

### Chart Features

**Multi-Chart Layouts:**
- Single chart (default)
- 2x1 (two charts stacked)
- 1x2 (two charts side-by-side)
- 2x2 (four charts)
- 3x2 (six charts)

**Comparison Mode:**
- Overlay up to 5 stocks on one chart
- Percentage or absolute comparison
- Different colors for each stock
- Show relative performance

**Synchronized Cursors:**
- When hovering over one chart, all charts show same timestamp
- Useful for multi-chart analysis
- Display OHLCV data for all charts

**Chart Templates:**
- Save chart configuration (indicators, drawings, layout)
- Load saved templates
- Share templates with other users
- Library of community templates

**Export Options:**
- PNG image (high resolution)
- SVG vector (scalable)
- PDF (for reports)
- Interactive HTML embed

---

## Feature 2: Technical Indicators Library

### Indicator Categories

#### 1. Trend Indicators (15+ indicators)

**Moving Averages:**
- SMA (Simple Moving Average) - periods: 20, 50, 100, 200
- EMA (Exponential Moving Average) - periods: 12, 26, 50, 200
- WMA (Weighted Moving Average)
- HMA (Hull Moving Average)
- DEMA (Double Exponential MA)
- TEMA (Triple Exponential MA)
- VWMA (Volume Weighted MA)
- SMMA (Smoothed MA)

**Trend Following:**
- MACD (Moving Average Convergence Divergence)
  - Default: 12, 26, 9
  - Show histogram, signal line, MACD line
- ADX (Average Directional Index)
  - Shows trend strength (0-100)
  - >25 = trending, <20 = ranging
- Parabolic SAR (Stop and Reverse)
  - Dots above/below price
  - Trailing stop levels
- Supertrend
  - Uses ATR for adaptive stops
- Ichimoku Cloud
  - Tenkan-sen, Kijun-sen, Senkou Span A/B, Chikou Span
  - Cloud support/resistance

#### 2. Momentum Oscillators (20+ indicators)

**Standard Oscillators:**
- RSI (Relative Strength Index)
  - Period: 14 (default), adjustable
  - Overbought: >70, Oversold: <30
  - Divergence detection
- Stochastic Oscillator
  - %K and %D lines
  - Fast, Slow, Full variants
  - Overbought: >80, Oversold: <20
- Williams %R
  - Similar to Stochastic
  - Range: 0 to -100
- CCI (Commodity Channel Index)
  - Overbought: >100, Oversold: <-100
- ROC (Rate of Change)
  - Percentage price change
  - Period: 12 (default)
- Ultimate Oscillator
  - Combines 3 timeframes
  - Reduces false signals

**Advanced Momentum:**
- Money Flow Index (MFI)
  - Volume-weighted RSI
- True Strength Index (TSI)
- Awesome Oscillator
- Momentum Indicator
- Chande Momentum Oscillator (CMO)

#### 3. Volatility Indicators (12+ indicators)

**Bands & Envelopes:**
- Bollinger Bands
  - 20-period SMA ± 2 standard deviations
  - Show %B and Band Width
- Keltner Channels
  - EMA ± ATR
- Donchian Channels
  - High/Low over N periods
- Envelopes (% bands around MA)

**Volatility Measures:**
- ATR (Average True Range)
  - 14-period default
  - Measure of volatility
- Standard Deviation
- Chandelier Exit
- Historical Volatility
- VIX overlay (market volatility)

#### 4. Volume Indicators (15+ indicators)

**Volume Analysis:**
- Volume Bars (standard)
- OBV (On-Balance Volume)
  - Cumulative volume with price direction
- Volume Profile
  - Horizontal volume at price levels
  - Point of Control (POC)
  - Value Area (70% of volume)
- VWAP (Volume Weighted Average Price)
  - Intraday benchmark
- Accumulation/Distribution Line
  - Cumulative money flow
- Chaikin Money Flow
  - Buying/selling pressure
- Money Flow Index (MFI)
- Ease of Movement (EMV)
- Force Index
- Negative Volume Index (NVI)
- Positive Volume Index (PVI)

#### 5. Pivot Points & Support/Resistance (8+ types)

**Pivot Calculators:**
- Standard Pivot Points (S1, S2, S3, R1, R2, R3)
- Fibonacci Pivot Points
- Woodie's Pivot Points
- Camarilla Pivot Points
- DeMark Pivot Points

**Automatic Detection:**
- Support/Resistance levels
- Swing highs/lows
- Psychological levels ($50, $100, etc.)

#### 6. Bill Williams Indicators (5 indicators)

- Accelerator Oscillator
- Awesome Oscillator
- Alligator
- Fractals
- Gator Oscillator

#### 7. Custom Indicators

**User-Created Indicators:**
- Formula builder (simplified scripting)
- Combine existing indicators
- Save and share custom indicators
- Community indicator library

**Indicator Builder Interface:**
```
Create Custom Indicator
─────────────────────────
Name: My Custom RSI
Formula: RSI(14) + SMA(RSI(14), 5)

Plot Options:
☑ Line Chart
□ Histogram
□ Overlay on price

Color: [Blue ▼]
Line Width: [2 ▼]

[Preview] [Save] [Cancel]
```

---

## Feature 3: Pattern Recognition

### Automatic Pattern Detection

**Chart Patterns (20+ patterns):**

*Reversal Patterns:*
- Head and Shoulders (bearish)
- Inverse Head and Shoulders (bullish)
- Double Top (bearish)
- Double Bottom (bullish)
- Triple Top (bearish)
- Triple Bottom (bullish)
- Rounding Top (bearish)
- Rounding Bottom (bullish)

*Continuation Patterns:*
- Ascending Triangle (bullish)
- Descending Triangle (bearish)
- Symmetrical Triangle (neutral)
- Bull Flag
- Bear Flag
- Pennant
- Rising Wedge (bearish)
- Falling Wedge (bullish)
- Rectangle (consolidation)

**Candlestick Patterns (30+ patterns):**

*Bullish Patterns:*
- Hammer
- Inverted Hammer
- Bullish Engulfing
- Morning Star
- Three White Soldiers
- Piercing Line
- Bullish Harami

*Bearish Patterns:*
- Shooting Star
- Hanging Man
- Bearish Engulfing
- Evening Star
- Three Black Crows
- Dark Cloud Cover
- Bearish Harami

*Neutral Patterns:*
- Doji
- Spinning Top
- High Wave

### Pattern Detection Features

**Real-Time Detection:**
- Scan current chart for patterns
- Show pattern overlay on chart
- Pattern details popup
- Confidence score (0-100%)

**Pattern Alerts:**
- Alert when pattern forms
- Email/push notification
- Watchlist pattern scanning
- Portfolio pattern monitoring

**Pattern Performance:**
- Historical success rate
- Expected price target
- Typical duration
- Win/loss statistics

**Pattern Display:**
```
┌────────────────────────────────────┐
│ Pattern Detected                   │
├────────────────────────────────────┤
│ Head and Shoulders (Bearish)       │
│ Confidence: 85%                    │
│                                    │
│ Expected Move: -12% to -18%        │
│ Target Price: $152 - $165          │
│ Stop Loss: $185 (above neckline)   │
│                                    │
│ Historical Success: 68%            │
│ Avg Duration: 15-25 days           │
│                                    │
│ [View Details] [Set Alert]         │
└────────────────────────────────────┘
```

---

## Feature 4: Economic Data Integration

### Economic Indicators to Include

**Macro Economic Data:**
- GDP Growth (quarterly, YoY)
- Unemployment Rate (monthly)
- CPI (Consumer Price Index - inflation)
- PPI (Producer Price Index)
- Fed Funds Rate
- 10-Year Treasury Yield
- 2-Year Treasury Yield
- Yield Curve (10Y-2Y spread)
- Consumer Confidence Index
- PMI (Purchasing Managers' Index)
  - Manufacturing PMI
  - Services PMI
- Housing Starts
- Existing Home Sales
- Retail Sales (monthly)
- Industrial Production
- Dollar Index (DXY)

**Financial Market Indicators:**
- S&P 500 Index
- VIX (Volatility Index)
- Put/Call Ratio
- AAII Sentiment Survey
- Crude Oil Price
- Gold Price

### Overlay on Stock Charts

**Recession Shading (like YCharts):**
- Gray shaded areas during NBER-defined recessions
- Shows how stock performed during downturns
- Historical context for price action

**Correlation Analysis:**
- Calculate correlation between stock and economic indicator
- Display correlation coefficient
- Chart both on same timeline
- Identify leading/lagging relationships

**Event Annotations:**
- Fed rate decisions
- FOMC meetings
- Economic data releases
- Presidential elections
- Major policy changes

### Economic Dashboard

**Macro Overview Page:**
```
┌─────────────────────────────────────────┐
│ Economic Dashboard                      │
├─────────────────────────────────────────┤
│                                         │
│ GDP Growth:     +2.1%  ⬆  (Q3 2025)    │
│ Unemployment:   3.8%   →  (Oct 2025)    │
│ CPI (YoY):      2.4%   ⬇  (Oct 2025)    │
│ Fed Funds:      5.25%  →  (Nov 2025)    │
│ 10Y Treasury:   4.15%  ⬆  (Today)       │
│                                         │
│ [Charts] [Historical] [Forecasts]       │
│                                         │
└─────────────────────────────────────────┘
```

---

## User Interface Requirements

### Chart Page Layout

```
┌──────────────────────────────────────────────────────────┐
│ AAPL - Apple Inc.                     $175.00 ▲ +2.5%   │
├──────────────────────────────────────────────────────────┤
│ [1M][3M][6M][YTD][1Y][5Y][All]  Chart: [Candlestick ▼]  │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  $180 ┤                           ●                      │
│       │                        ●─┘│                      │
│  $170 ┤                     ●─┘   │                      │
│       │                  ●─┘      │                      │
│  $160 ┤               ●─┘         │                      │
│       │            ●─┘            │  ← RSI: 68 (Neutral) │
│  $150 ┤         ●─┘               │                      │
│       │      ●─┘                  │                      │
│  $140 ┤   ●─┘                     │                      │
│       └────────────────────────────────────────────      │
│        J   F   M   A   M   J   J   A   S   O   N   D    │
│                                                          │
│ Volume ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓                               │
│                                                          │
├──────────────────────────────────────────────────────────┤
│ Indicators: [+ Add Indicator ▼]                         │
│ ☑ RSI (14)  ☑ MACD (12,26,9)  ☑ Bollinger Bands        │
│ Drawing Tools: [Line] [Fib] [Text] [⬜] [⚪]           │
├──────────────────────────────────────────────────────────┤
│ Patterns Detected: Head & Shoulders (85% confidence)     │
└──────────────────────────────────────────────────────────┘
```

### Performance Requirements
- Chart render time: <500ms
- Indicator calculation: <200ms per indicator
- Real-time updates: <1s latency
- Smooth panning/zooming (60fps)
- Responsive on mobile devices

---

## Success Metrics
- % users using charting features (target: >70%)
- Average indicators per chart (target: 3-4)
- Pattern detection usage (target: >40% of users)
- Drawing tools usage (target: >30% of users)
- Time spent on charts (target: 5+ min/session)

---

**End of Product Specification: Technical Analysis Features**
