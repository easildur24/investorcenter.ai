#!/usr/bin/env python3
"""
Import all ticker types from Polygon.io to the database
"""
import os
import sys
import time
import psycopg2
from datetime import datetime

# Add the parent directory to the path to import us_tickers module
sys.path.insert(0, '/app/scripts')

from us_tickers.polygon_client import PolygonClient

def import_tickers(conn, asset_type):
    """Import tickers of a specific type"""
    print(f"\n📦 Importing {asset_type} tickers...")
    
    api_key = os.environ.get('POLYGON_API_KEY')
    if not api_key:
        print("ERROR: POLYGON_API_KEY environment variable not set")
        return False
    
    client = PolygonClient(api_key)
    
    # Map our types to Polygon market types
    market_map = {
        'stocks': 'stocks',
        'etf': 'stocks',  # ETFs are in stocks market with type 'ETF'
        'crypto': 'crypto',
        'indices': 'indices'
    }
    
    market = market_map.get(asset_type, asset_type)
    
    try:
        # Fetch tickers from Polygon
        if asset_type == 'etf':
            # For ETFs, fetch stocks and filter by type
            tickers = client.fetch_tickers(market='stocks', ticker_type='ETF', active=True, limit=1000)
        else:
            tickers = client.fetch_tickers(market=market, active=True, limit=1000)
        
        print(f"Fetched {len(tickers)} {asset_type} tickers from Polygon")
        
        # Insert into database
        cur = conn.cursor()
        inserted = 0
        updated = 0
        
        for ticker in tickers:
            symbol = ticker.get('ticker', '')
            if not symbol:
                continue
            
            # Determine asset type
            if asset_type == 'crypto':
                db_asset_type = 'crypto'
            elif asset_type == 'indices':
                db_asset_type = 'index'
            elif ticker.get('type') == 'ETF' or asset_type == 'etf':
                db_asset_type = 'etf'
            else:
                db_asset_type = 'stock'
            
            try:
                cur.execute("""
                    INSERT INTO tickers (
                        symbol, name, exchange, ticker_type, asset_type,
                        currency_name, market, locale, primary_exchange,
                        active, cik, composite_figi, share_class_figi,
                        last_updated_utc, delisted_utc, created_at, updated_at
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
                    )
                    ON CONFLICT (symbol) DO UPDATE SET
                        name = EXCLUDED.name,
                        exchange = EXCLUDED.exchange,
                        ticker_type = EXCLUDED.ticker_type,
                        asset_type = EXCLUDED.asset_type,
                        active = EXCLUDED.active,
                        updated_at = EXCLUDED.updated_at
                    RETURNING (xmax = 0) AS inserted
                """, (
                    symbol,
                    ticker.get('name', ''),
                    ticker.get('exchange', ''),
                    ticker.get('type', ''),
                    db_asset_type,
                    ticker.get('currency_name', 'USD'),
                    ticker.get('market', market),
                    ticker.get('locale', 'us'),
                    ticker.get('primary_exchange', ''),
                    ticker.get('active', True),
                    ticker.get('cik', ''),
                    ticker.get('composite_figi', ''),
                    ticker.get('share_class_figi', ''),
                    ticker.get('last_updated_utc'),
                    ticker.get('delisted_utc'),
                    datetime.now(),
                    datetime.now()
                ))
                
                result = cur.fetchone()
                if result[0]:
                    inserted += 1
                else:
                    updated += 1
                    
            except Exception as e:
                print(f"Error inserting {symbol}: {e}")
                continue
        
        conn.commit()
        print(f"✅ {asset_type}: Inserted {inserted}, Updated {updated}")
        return True
        
    except Exception as e:
        print(f"❌ Error importing {asset_type}: {e}")
        conn.rollback()
        return False
    finally:
        cur.close()

def main():
    # Database connection
    db_config = {
        'host': os.environ.get('DB_HOST', 'localhost'),
        'port': os.environ.get('DB_PORT', '5432'),
        'database': os.environ.get('DB_NAME', 'investorcenter_db'),
        'user': os.environ.get('DB_USER', 'investorcenter'),
        'password': os.environ.get('DB_PASSWORD', '')
    }
    
    print("Connecting to database...")
    conn = psycopg2.connect(**db_config)
    
    # Import all asset types
    asset_types = ['stocks', 'etf', 'crypto', 'indices']
    
    for asset_type in asset_types:
        import_tickers(conn, asset_type)
        # Wait between types to avoid rate limiting
        if asset_type != asset_types[-1]:
            print("⏳ Waiting 2 seconds before next asset type...")
            time.sleep(2)
    
    # Print summary
    cur = conn.cursor()
    cur.execute("""
        SELECT asset_type, COUNT(*) 
        FROM tickers 
        GROUP BY asset_type 
        ORDER BY asset_type
    """)
    
    print("\n📊 Final Summary:")
    print("-" * 30)
    total = 0
    for row in cur.fetchall():
        print(f"{row[0]:10} {row[1]:,}")
        total += row[1]
    print("-" * 30)
    print(f"{'Total':10} {total:,}")
    
    cur.close()
    conn.close()

if __name__ == '__main__':
    main()