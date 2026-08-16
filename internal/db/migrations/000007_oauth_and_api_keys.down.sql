DROP TABLE IF EXISTS oauth_refresh_tokens;
DROP TABLE IF EXISTS oauth_authorization_codes;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS oauth_clients;

ALTER TABLE users ADD COLUMN api_key_hash TEXT;
