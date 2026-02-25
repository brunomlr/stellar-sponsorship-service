ALTER TABLE api_keys
    DROP CONSTRAINT IF EXISTS chk_api_keys_rate_limit_max_range,
    DROP CONSTRAINT IF EXISTS chk_api_keys_rate_limit_window_range;
