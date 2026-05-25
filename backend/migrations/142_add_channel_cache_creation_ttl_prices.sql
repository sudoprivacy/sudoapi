-- Add optional TTL-specific cache creation prices for channel pricing.
-- NULL keeps the legacy behavior: fall back to cache_write_price when present.

ALTER TABLE channel_model_pricing
    ADD COLUMN IF NOT EXISTS cache_creation_5m_price NUMERIC(20,12),
    ADD COLUMN IF NOT EXISTS cache_creation_1h_price NUMERIC(20,12);

ALTER TABLE channel_pricing_intervals
    ADD COLUMN IF NOT EXISTS cache_creation_5m_price NUMERIC(20,12),
    ADD COLUMN IF NOT EXISTS cache_creation_1h_price NUMERIC(20,12);

ALTER TABLE channel_account_stats_model_pricing
    ADD COLUMN IF NOT EXISTS cache_creation_5m_price NUMERIC(20,12),
    ADD COLUMN IF NOT EXISTS cache_creation_1h_price NUMERIC(20,12);

ALTER TABLE channel_account_stats_pricing_intervals
    ADD COLUMN IF NOT EXISTS cache_creation_5m_price NUMERIC(20,12),
    ADD COLUMN IF NOT EXISTS cache_creation_1h_price NUMERIC(20,12);

COMMENT ON COLUMN channel_model_pricing.cache_creation_5m_price IS '5分钟缓存创建每 token 价格，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_model_pricing.cache_creation_1h_price IS '1小时缓存创建每 token 价格，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_pricing_intervals.cache_creation_5m_price IS 'token 模式：5分钟缓存创建价，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_pricing_intervals.cache_creation_1h_price IS 'token 模式：1小时缓存创建价，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_account_stats_model_pricing.cache_creation_5m_price IS '账号统计定价：5分钟缓存创建每 token 价格，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_account_stats_model_pricing.cache_creation_1h_price IS '账号统计定价：1小时缓存创建每 token 价格，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_account_stats_pricing_intervals.cache_creation_5m_price IS '账号统计区间定价：5分钟缓存创建价，NULL 表示回退到 cache_write_price';
COMMENT ON COLUMN channel_account_stats_pricing_intervals.cache_creation_1h_price IS '账号统计区间定价：1小时缓存创建价，NULL 表示回退到 cache_write_price';
