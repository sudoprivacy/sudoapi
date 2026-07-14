-- sudoapi: Channel TTL-specific cache creation pricing.

-- 已经在 https://github.com/Wei-Shaw/sub2api/pull/6474 中实现, 见 232_channel_cache_write_1h_pricing.sql
-- cache_write_price 即为 5m 缓存价格, 增加 cache_write_1h_price 为 1h 缓存价格,
-- 现在将 9001_add_channel_cache_creation_ttl_prices.sql 回滚

UPDATE channel_model_pricing
SET
    cache_write_price = COALESCE(cache_creation_5m_price, cache_write_price),
    cache_write_1h_price = COALESCE(cache_creation_1h_price, cache_write_1h_price);

UPDATE channel_pricing_intervals
SET
    cache_write_price = COALESCE(cache_creation_5m_price, cache_write_price),
    cache_write_1h_price = COALESCE(cache_creation_1h_price, cache_write_1h_price);

UPDATE channel_account_stats_model_pricing
SET
    cache_write_price = COALESCE(cache_creation_5m_price, cache_write_price),
    cache_write_1h_price = COALESCE(cache_creation_1h_price, cache_write_1h_price);

UPDATE channel_account_stats_pricing_intervals
SET
    cache_write_price = COALESCE(cache_creation_5m_price, cache_write_price),
    cache_write_1h_price = COALESCE(cache_creation_1h_price, cache_write_1h_price);

ALTER TABLE channel_model_pricing
    DROP COLUMN IF EXISTS cache_creation_5m_price,
    DROP COLUMN IF EXISTS cache_creation_1h_price;

ALTER TABLE channel_pricing_intervals
    DROP COLUMN IF EXISTS cache_creation_5m_price,
    DROP COLUMN IF EXISTS cache_creation_1h_price;

ALTER TABLE channel_account_stats_model_pricing
    DROP COLUMN IF EXISTS cache_creation_5m_price,
    DROP COLUMN IF EXISTS cache_creation_1h_price;

ALTER TABLE channel_account_stats_pricing_intervals
    DROP COLUMN IF EXISTS cache_creation_5m_price,
    DROP COLUMN IF EXISTS cache_creation_1h_price;
