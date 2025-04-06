-- Strategy Service Default Data
-- File: 03-default-data.sql
-- Contains initial data for the database

-- Insert default tags
INSERT INTO strategy_tags (name) VALUES 
('Trend Following'),
('Mean Reversion'),
('Momentum'),
('Breakout'),
('Volatility'),
('Swing Trading'),
('Scalping'),
('Day Trading'),
('Algorithmic'),
('Machine Learning')
ON CONFLICT (name) DO NOTHING;