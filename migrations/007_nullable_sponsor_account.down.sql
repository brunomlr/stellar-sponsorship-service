UPDATE api_keys SET sponsor_account = '' WHERE sponsor_account IS NULL;
ALTER TABLE api_keys ALTER COLUMN sponsor_account SET NOT NULL;
