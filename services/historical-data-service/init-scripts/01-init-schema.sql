-- Historical Data Service Database Schema
-- Uses TimescaleDB for time series data

-- Extension for time series data
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Symbols table
CREATE TABLE IF NOT EXISTS symbols (
  id SERIAL PRIMARY KEY,
  symbol VARCHAR(20) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL,
  exchange VARCHAR(50) NOT NULL,
  asset_type VARCHAR(20) NOT NULL DEFAULT 'stock',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  data_available BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP
);

-- Timeframes table
CREATE TABLE IF NOT EXISTS timeframes (
  id SERIAL PRIMARY KEY,
  name VARCHAR(10) NOT NULL UNIQUE,
  minutes INTEGER NOT NULL,
  display_name VARCHAR(20) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP
);

-- Market data table
CREATE TABLE IF NOT EXISTS market_data (
  id BIGSERIAL PRIMARY KEY,
  symbol_id INTEGER NOT NULL,
  timeframe_id INTEGER NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  open NUMERIC(16, 8) NOT NULL,
  high NUMERIC(16, 8) NOT NULL,
  low NUMERIC(16, 8) NOT NULL,
  close NUMERIC(16, 8) NOT NULL,
  volume NUMERIC(24, 8) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP,
  UNIQUE (symbol_id, timeframe_id, timestamp)
);

-- Convert market_data to hypertable for time series optimization
SELECT create_hypertable('market_data', 'timestamp', chunk_time_interval => INTERVAL '1 week');

-- Backtests table
CREATE TABLE IF NOT EXISTS backtests (
  id SERIAL PRIMARY KEY,
  user_id INTEGER NOT NULL,
  strategy_id INTEGER NOT NULL,
  strategy_name VARCHAR(100) NOT NULL,
  strategy_version INTEGER NOT NULL DEFAULT 1,
  symbol_id INTEGER NOT NULL,
  timeframe_id INTEGER NOT NULL,
  start_date TIMESTAMP NOT NULL,
  end_date TIMESTAMP NOT NULL,
  initial_capital NUMERIC(16, 2) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  results JSONB,
  error_message TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP,
  completed_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_market_data_symbol_timeframe ON market_data (symbol_id, timeframe_id);
CREATE INDEX idx_backtests_user_id ON backtests (user_id);
CREATE INDEX idx_backtests_strategy_id ON backtests (strategy_id);
CREATE INDEX idx_backtests_status ON backtests (status);
CREATE INDEX idx_backtests_created_at ON backtests (created_at);

-- Foreign Keys
ALTER TABLE market_data ADD CONSTRAINT fk_market_data_symbol
  FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE;
  
ALTER TABLE market_data ADD CONSTRAINT fk_market_data_timeframe
  FOREIGN KEY (timeframe_id) REFERENCES timeframes(id) ON DELETE CASCADE;
  
ALTER TABLE backtests ADD CONSTRAINT fk_backtests_symbol
  FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE;
  
ALTER TABLE backtests ADD CONSTRAINT fk_backtests_timeframe
  FOREIGN KEY (timeframe_id) REFERENCES timeframes(id) ON DELETE CASCADE;

-- Insert default timeframes
INSERT INTO timeframes (name, minutes, display_name, created_at) VALUES
('1m', 1, '1 Minute', CURRENT_TIMESTAMP),
('5m', 5, '5 Minutes', CURRENT_TIMESTAMP),
('15m', 15, '15 Minutes', CURRENT_TIMESTAMP),
('30m', 30, '30 Minutes', CURRENT_TIMESTAMP),
('1h', 60, '1 Hour', CURRENT_TIMESTAMP),
('4h', 240, '4 Hours', CURRENT_TIMESTAMP),
('1d', 1440, '1 Day', CURRENT_TIMESTAMP),
('1w', 10080, '1 Week', CURRENT_TIMESTAMP)
ON CONFLICT (name) DO NOTHING;

-- Insert some sample symbols
INSERT INTO symbols (symbol, name, exchange, asset_type, is_active, data_available, created_at) VALUES
('AAPL', 'Apple Inc.', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('MSFT', 'Microsoft Corporation', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('GOOGL', 'Alphabet Inc.', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('AMZN', 'Amazon.com, Inc.', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('META', 'Meta Platforms, Inc.', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('TSLA', 'Tesla, Inc.', 'NASDAQ', 'stock', TRUE, FALSE, CURRENT_TIMESTAMP),
('BTC-USD', 'Bitcoin / US Dollar', 'CRYPTO', 'crypto', TRUE, FALSE, CURRENT_TIMESTAMP),
('ETH-USD', 'Ethereum / US Dollar', 'CRYPTO', 'crypto', TRUE, FALSE, CURRENT_TIMESTAMP)
ON CONFLICT (symbol) DO NOTHING;