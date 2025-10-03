#!/usr/bin/env python3
"""
Complete CoinGecko Cryptocurrency Service

Fetches ALL cryptocurrencies from CoinGecko with intelligent caching.
Respects rate limits and provides on-demand updates.
"""

import asyncio
import aiohttp
import json
import redis
import logging
from datetime import datetime, timedelta
import time
from typing import Dict, List, Optional, Tuple
import os

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class CoinGeckoCompleteService:
    def __init__(self):
        self.redis_client = redis.Redis(
            host=os.getenv('REDIS_HOST', 'localhost'),
            port=int(os.getenv('REDIS_PORT', 6379)),
            decode_responses=True
        )

        # CoinGecko API endpoints
        self.base_url = 'https://api.coingecko.com/api/v3'

        # Rate limiting for free tier
        self.calls_per_minute = 10  # Conservative for free tier
        self.seconds_between_calls = 60 / self.calls_per_minute  # 6 seconds
        self.last_api_call = 0

        # Track API usage
        self.api_calls_count = 0
        self.api_calls_reset_time = time.time() + 60

        # Cache configuration
        self.default_cache_ttl = 300  # 5 minutes default
        self.on_demand_cache_ttl = 120  # 2 minutes for on-demand updates

        # Store metadata about all coins
        self.all_coins_list = []
        self.coin_id_to_symbol_map = {}

    async def rate_limit(self):
        """Enforce rate limiting"""
        current_time = time.time()

        # Reset counter every minute
        if current_time >= self.api_calls_reset_time:
            self.api_calls_count = 0
            self.api_calls_reset_time = current_time + 60

        # If we've hit the limit, wait
        if self.api_calls_count >= self.calls_per_minute:
            wait_time = self.api_calls_reset_time - current_time
            if wait_time > 0:
                logger.warning(f"Rate limit reached. Waiting {wait_time:.1f} seconds...")
                await asyncio.sleep(wait_time)
                self.api_calls_count = 0
                self.api_calls_reset_time = time.time() + 60

        # Ensure minimum time between calls
        time_since_last = current_time - self.last_api_call
        if time_since_last < self.seconds_between_calls:
            await asyncio.sleep(self.seconds_between_calls - time_since_last)

        self.last_api_call = time.time()
        self.api_calls_count += 1

    async def fetch_all_coins_list(self, session):
        """Fetch the complete list of all available coins"""
        try:
            await self.rate_limit()

            url = f"{self.base_url}/coins/list"
            params = {'include_platform': 'false'}

            async with session.get(url, params=params) as response:
                if response.status == 200:
                    data = await response.json()
                    self.all_coins_list = data

                    # Build symbol to ID mapping
                    for coin in data:
                        symbol = coin.get('symbol', '').upper()
                        coin_id = coin.get('id', '')
                        self.coin_id_to_symbol_map[symbol] = coin_id

                    # Store the list in Redis for reference
                    self.redis_client.setex(
                        'crypto:metadata:all_coins',
                        86400,  # 24 hours
                        json.dumps(data)
                    )

                    logger.info(f"📋 Fetched complete list of {len(data)} coins from CoinGecko")
                    return data
                elif response.status == 429:
                    logger.error("Rate limit hit on coins list")
                    return []
                else:
                    logger.error(f"Failed to fetch coins list: {response.status}")
                    return []
        except Exception as e:
            logger.error(f"Error fetching coins list: {e}")
            return []

    async def fetch_market_data(self, session, page=1, per_page=250):
        """Fetch market data for coins"""
        try:
            await self.rate_limit()

            url = f"{self.base_url}/coins/markets"
            params = {
                'vs_currency': 'usd',
                'order': 'market_cap_desc',
                'per_page': per_page,
                'page': page,
                'sparkline': 'false',
                'price_change_percentage': '1h,24h,7d,30d'
            }

            async with session.get(url, params=params) as response:
                if response.status == 200:
                    data = await response.json()
                    logger.info(f"📊 Fetched page {page} with {len(data)} coins")
                    return data
                elif response.status == 429:
                    logger.warning(f"Rate limit hit on page {page}")
                    await asyncio.sleep(60)  # Wait a minute
                    return []
                else:
                    logger.error(f"Failed to fetch page {page}: {response.status}")
                    return []
        except Exception as e:
            logger.error(f"Error fetching market data page {page}: {e}")
            return []

    async def fetch_specific_coins(self, session, coin_ids: List[str]):
        """Fetch specific coins by their IDs"""
        try:
            await self.rate_limit()

            url = f"{self.base_url}/coins/markets"
            params = {
                'vs_currency': 'usd',
                'ids': ','.join(coin_ids[:250]),  # Max 250 per request
                'order': 'market_cap_desc',
                'sparkline': 'false',
                'price_change_percentage': '1h,24h,7d,30d'
            }

            async with session.get(url, params=params) as response:
                if response.status == 200:
                    data = await response.json()
                    return data
                elif response.status == 429:
                    logger.warning("Rate limit hit on specific coins fetch")
                    return []
                else:
                    logger.error(f"Failed to fetch specific coins: {response.status}")
                    return []
        except Exception as e:
            logger.error(f"Error fetching specific coins: {e}")
            return []

    def store_coin_data(self, coins: List[Dict], ttl_override: Optional[int] = None):
        """Store coin data in Redis with appropriate TTL"""
        stored_count = 0

        for coin in coins:
            try:
                symbol = coin.get('symbol', '').upper()
                if not symbol:
                    continue

                # Prepare the data with timestamp
                redis_data = {
                    'symbol': symbol,
                    'id': coin.get('id', ''),
                    'name': coin.get('name', ''),
                    'image': coin.get('image', ''),
                    'current_price': coin.get('current_price', 0),
                    'market_cap': coin.get('market_cap', 0),
                    'market_cap_rank': coin.get('market_cap_rank'),
                    'fully_diluted_valuation': coin.get('fully_diluted_valuation'),
                    'total_volume': coin.get('total_volume', 0),
                    'high_24h': coin.get('high_24h', 0),
                    'low_24h': coin.get('low_24h', 0),
                    'price_change_24h': coin.get('price_change_24h', 0),
                    'price_change_percentage_24h': coin.get('price_change_percentage_24h', 0),
                    'price_change_percentage_1h': coin.get('price_change_percentage_1h_in_currency', 0),
                    'price_change_percentage_7d': coin.get('price_change_percentage_7d_in_currency', 0),
                    'price_change_percentage_30d': coin.get('price_change_percentage_30d_in_currency', 0),
                    'circulating_supply': coin.get('circulating_supply', 0),
                    'total_supply': coin.get('total_supply', 0),
                    'max_supply': coin.get('max_supply'),
                    'ath': coin.get('ath', 0),
                    'ath_change_percentage': coin.get('ath_change_percentage', 0),
                    'ath_date': coin.get('ath_date', ''),
                    'atl': coin.get('atl', 0),
                    'atl_change_percentage': coin.get('atl_change_percentage', 0),
                    'atl_date': coin.get('atl_date', ''),
                    'last_updated': coin.get('last_updated', datetime.utcnow().isoformat()),
                    'fetched_at': datetime.utcnow().isoformat(),  # When we fetched it
                    'source': 'coingecko'
                }

                # Determine TTL based on market cap rank
                if ttl_override:
                    ttl = ttl_override
                else:
                    rank = redis_data.get('market_cap_rank')
                    if rank and rank <= 10:
                        ttl = 60  # 1 minute for top 10
                    elif rank and rank <= 50:
                        ttl = 120  # 2 minutes for top 50
                    elif rank and rank <= 100:
                        ttl = 180  # 3 minutes for top 100
                    elif rank and rank <= 500:
                        ttl = 300  # 5 minutes for top 500
                    else:
                        ttl = 600  # 10 minutes for others

                # Store in Redis - but check if a better coin already exists
                key = f"crypto:quote:{symbol}"

                # Check if this symbol already exists
                existing = self.redis_client.get(key)
                should_store = True

                if existing:
                    try:
                        existing_data = json.loads(existing)
                        existing_rank = existing_data.get('market_cap_rank')
                        new_rank = redis_data.get('market_cap_rank')

                        # Only overwrite if new coin has better rank
                        if (existing_rank and new_rank and
                            existing_rank < new_rank):
                            should_store = False
                            logger.debug(
                                f"Skipping {symbol} (rank {new_rank}) "
                                f"- better coin exists (rank {existing_rank})"
                            )
                    except:
                        pass  # If parsing fails, store anyway

                if should_store:
                    self.redis_client.setex(
                        key,
                        ttl,
                        json.dumps(redis_data)
                    )

                # Also store by CoinGecko ID for quick lookups
                id_key = f"crypto:id_map:{coin.get('id', '')}"
                self.redis_client.setex(
                    id_key,
                    86400,  # 24 hours
                    symbol
                )

                stored_count += 1

            except Exception as e:
                logger.error(f"Error storing coin {coin.get('symbol', 'UNKNOWN')}: {e}")

        logger.info(f"💾 Stored {stored_count} coins in Redis")
        return stored_count

    async def check_and_update_on_demand(self, session, symbol: str) -> Tuple[bool, Optional[Dict]]:
        """
        Check if a coin needs updating and fetch if necessary.
        Returns (was_updated, coin_data)
        """
        try:
            # Check if we have the coin in cache
            key = f"crypto:quote:{symbol.upper()}"
            cached_data = self.redis_client.get(key)

            if cached_data:
                data = json.loads(cached_data)
                fetched_at = datetime.fromisoformat(data.get('fetched_at', datetime.utcnow().isoformat()))
                age_seconds = (datetime.utcnow() - fetched_at).total_seconds()

                # If data is less than 1 minute old, don't update
                if age_seconds < 60:
                    return False, data

            # Try to find the coin ID
            coin_id = self.coin_id_to_symbol_map.get(symbol.upper())
            if not coin_id:
                # Try to get from Redis mapping
                stored_list = self.redis_client.get('crypto:metadata:all_coins')
                if stored_list:
                    coins = json.loads(stored_list)
                    for coin in coins:
                        if coin.get('symbol', '').upper() == symbol.upper():
                            coin_id = coin.get('id')
                            break

            if coin_id:
                # Fetch fresh data for this specific coin
                coins = await self.fetch_specific_coins(session, [coin_id])
                if coins:
                    self.store_coin_data(coins, ttl_override=self.on_demand_cache_ttl)
                    return True, coins[0] if coins else None

            return False, None

        except Exception as e:
            logger.error(f"Error in on-demand update for {symbol}: {e}")
            return False, None

    async def fetch_all_market_data(self, session):
        """Fetch ALL coins from CoinGecko (paginated)"""
        all_coins = []
        page = 1
        max_pages = 60  # ~15,000 coins (250 per page)

        logger.info("🔄 Starting comprehensive market data fetch...")

        while page <= max_pages:
            try:
                coins = await self.fetch_market_data(session, page=page)
                if not coins:
                    break  # No more data

                all_coins.extend(coins)
                self.store_coin_data(coins)

                logger.info(f"Progress: {len(all_coins)} total coins fetched")

                # If we got less than full page, we're done
                if len(coins) < 250:
                    break

                page += 1

            except Exception as e:
                logger.error(f"Error fetching page {page}: {e}")
                await asyncio.sleep(60)  # Wait before retry
                continue

        logger.info(f"✅ Comprehensive fetch complete: {len(all_coins)} total coins")

        # Store summary statistics
        stats = {
            'total_coins': len(all_coins),
            'last_full_update': datetime.utcnow().isoformat(),
            'pages_fetched': page - 1
        }
        self.redis_client.setex(
            'crypto:stats:last_update',
            3600,
            json.dumps(stats)
        )

        return all_coins

    async def run(self):
        """Main service loop"""
        logger.info("🚀 Complete CoinGecko Service Starting")
        logger.info("=" * 60)
        logger.info("📊 Will fetch ALL cryptocurrencies from CoinGecko")
        logger.info("⏱️ Rate limit: 10 calls/minute (free tier)")
        logger.info("💾 Smart caching with TTL based on market cap rank")
        logger.info("🔄 On-demand updates for requested coins")
        logger.info("=" * 60)

        async with aiohttp.ClientSession() as session:
            # Initial fetch of all coins metadata
            await self.fetch_all_coins_list(session)

            # Main service loop
            while True:
                try:
                    # Full market data update every 30 minutes
                    logger.info("\n" + "=" * 60)
                    logger.info("🔄 Starting full market update cycle")
                    logger.info("=" * 60)

                    await self.fetch_all_market_data(session)

                    # After full update, do targeted updates for 30 minutes
                    targeted_update_end = time.time() + 1800  # 30 minutes

                    while time.time() < targeted_update_end:
                        # Update top 100 coins more frequently
                        logger.info("📈 Updating top 100 coins...")
                        top_coins = await self.fetch_market_data(session, page=1, per_page=100)
                        if top_coins:
                            self.store_coin_data(top_coins)

                        # Wait 3 minutes before next top 100 update
                        await asyncio.sleep(180)

                except Exception as e:
                    logger.error(f"Error in main loop: {e}")
                    await asyncio.sleep(60)  # Wait before retry

    async def handle_on_demand_request(self, symbol: str):
        """
        Handle on-demand update request from API
        This would be called by the backend when needed
        """
        async with aiohttp.ClientSession() as session:
            was_updated, coin_data = await self.check_and_update_on_demand(session, symbol)
            return {
                'updated': was_updated,
                'data': coin_data,
                'timestamp': datetime.utcnow().isoformat()
            }


if __name__ == "__main__":
    try:
        service = CoinGeckoCompleteService()
        asyncio.run(service.run())
    except KeyboardInterrupt:
        logger.info("Service stopped by user")
    except Exception as e:
        logger.error(f"Service error: {e}")