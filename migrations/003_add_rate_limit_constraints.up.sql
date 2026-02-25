ALTER TABLE api_keys
    ADD CONSTRAINT chk_api_keys_rate_limit_max_range
        CHECK (rate_limit_max BETWEEN 1 AND 10000),
    ADD CONSTRAINT chk_api_keys_rate_limit_window_range
        CHECK (rate_limit_window BETWEEN 1 AND 86400);
