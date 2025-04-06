-- Insert default symbols
INSERT INTO symbols (symbol, name, asset_type, exchange, is_active, created_at)
VALUES 
('BTCUSD', 'Bitcoin/US Dollar', 'crypto', 'Coinbase', true, CURRENT_TIMESTAMP),
('ETHUSD', 'Ethereum/US Dollar', 'crypto', 'Coinbase', true, CURRENT_TIMESTAMP),
('AAPL', 'Apple Inc.', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('MSFT', 'Microsoft Corporation', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('AMZN', 'Amazon.com, Inc.', 'stock', 'NASDAQ', true, CURRENT_TIMESTAMP),
('BTCUSDT', 'Bitcoin/USDT', 'crypto', 'Binance', true, CURRENT_TIMESTAMP),
('ETHUSDT', 'Ethereum/USDT', 'crypto', 'Binance', true, CURRENT_TIMESTAMP)
ON CONFLICT (symbol) DO NOTHING;