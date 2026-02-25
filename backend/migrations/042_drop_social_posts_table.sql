-- Migration: 042_drop_social_posts_table.sql
-- Description: Drop social_posts table — replaced by querying
--              reddit_posts_raw JOIN reddit_post_tickers directly.
-- Date: 2026-02-24

DROP TABLE IF EXISTS social_posts;
