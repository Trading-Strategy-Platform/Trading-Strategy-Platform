-- services/strategy-service/init-scripts/01-init-schema.sql
-- Strategy Service Database Schema

-- Strategies
CREATE TABLE IF NOT EXISTS "strategies" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(100) NOT NULL,
  "user_id" int NOT NULL,
  "description" text,
  "structure" jsonb NOT NULL,
  "is_public" boolean NOT NULL DEFAULT false,
  "version" int NOT NULL DEFAULT 1,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Strategy Versions
CREATE TABLE IF NOT EXISTS "strategy_versions" (
  "id" SERIAL PRIMARY KEY,
  "strategy_id" int NOT NULL,
  "version" int NOT NULL,
  "structure" jsonb NOT NULL,
  "change_notes" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Strategy Tags
CREATE TABLE IF NOT EXISTS "strategy_tags" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(50) UNIQUE NOT NULL
);

-- Strategy to Tag Mappings
CREATE TABLE IF NOT EXISTS "strategy_tag_mappings" (
  "strategy_id" int NOT NULL,
  "tag_id" int NOT NULL,
  PRIMARY KEY ("strategy_id", "tag_id")
);

-- Technical Indicators
CREATE TABLE IF NOT EXISTS "indicators" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(50) UNIQUE NOT NULL,
  "description" text,
  "category" varchar(50),
  "formula" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Indicator Parameters
CREATE TABLE IF NOT EXISTS "indicator_parameters" (
  "id" SERIAL PRIMARY KEY,
  "indicator_id" int NOT NULL,
  "parameter_name" varchar(50) NOT NULL,
  "parameter_type" varchar(20) NOT NULL,
  "is_required" boolean NOT NULL DEFAULT true,
  "min_value" float,
  "max_value" float,
  "default_value" varchar(50),
  "description" text
);

-- Parameter Enum Values
CREATE TABLE IF NOT EXISTS "parameter_enum_values" (
  "id" SERIAL PRIMARY KEY,
  "parameter_id" int NOT NULL,
  "enum_value" varchar(50) NOT NULL,
  "display_name" varchar(100)
);

-- Marketplace Listings
CREATE TABLE IF NOT EXISTS "strategy_marketplace" (
  "id" SERIAL PRIMARY KEY,
  "strategy_id" int NOT NULL,
  "user_id" int NOT NULL,
  "price" numeric(10,2) NOT NULL DEFAULT 0,
  "is_subscription" boolean NOT NULL DEFAULT false,
  "subscription_period" varchar(20),
  "is_active" boolean NOT NULL DEFAULT true,
  "description" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Strategy Purchases
CREATE TABLE IF NOT EXISTS "strategy_purchases" (
  "id" SERIAL PRIMARY KEY,
  "marketplace_id" int NOT NULL,
  "buyer_id" int NOT NULL,
  "purchase_price" numeric(10,2) NOT NULL,
  "subscription_end" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

-- Strategy Reviews
CREATE TABLE IF NOT EXISTS "strategy_reviews" (
  "id" SERIAL PRIMARY KEY,
  "marketplace_id" int NOT NULL,
  "user_id" int NOT NULL,
  "rating" int NOT NULL,
  "comment" text,
  "created_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamp
);

-- Indexes
CREATE INDEX IF NOT EXISTS "idx_strategies_user_id" ON "strategies" ("user_id");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_strategy_versions" ON "strategy_versions" ("strategy_id", "version");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_indicator_parameters" ON "indicator_parameters" ("indicator_id", "parameter_name");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_strategy_reviews" ON "strategy_reviews" ("marketplace_id", "user_id");

-- Foreign Keys
ALTER TABLE "strategy_versions" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("tag_id") REFERENCES "strategy_tags" ("id") ON DELETE CASCADE;
ALTER TABLE "indicator_parameters" ADD FOREIGN KEY ("indicator_id") REFERENCES "indicators" ("id") ON DELETE CASCADE;
ALTER TABLE "parameter_enum_values" ADD FOREIGN KEY ("parameter_id") REFERENCES "indicator_parameters" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_marketplace" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_purchases" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_reviews" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;

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
('Machine Learning');

-- Insert default indicators
INSERT INTO indicators (name, description, category, formula, created_at) VALUES
('RSI', 'Relative Strength Index - measures the magnitude of price changes to evaluate oversold or overbought conditions', 'Momentum', 'RSI = 100 - (100 / (1 + RS))', CURRENT_TIMESTAMP),
('Bollinger Bands', 'Volatility bands placed above and below a moving average', 'Volatility', 'Middle Band = 20-day SMA, Upper Band = Middle Band + (20-day SD * 2), Lower Band = Middle Band - (20-day SD * 2)', CURRENT_TIMESTAMP),
('MACD', 'Moving Average Convergence Divergence - shows the relationship between two moving averages of a security\'s price', 'Trend', 'MACD Line = 12-day EMA - 26-day EMA, Signal Line = 9-day EMA of MACD Line', CURRENT_TIMESTAMP),
('Moving Average', 'Average of price over a specific period', 'Trend', 'SMA = Sum of prices for n periods / n', CURRENT_TIMESTAMP),
('Stochastic', 'Momentum indicator comparing closing price to price range over a specific period', 'Momentum', '%K = 100 * (Current Close - Lowest Low) / (Highest High - Lowest Low), %D = 3-day SMA of %K', CURRENT_TIMESTAMP);

-- Insert indicator parameters
-- RSI Parameters
INSERT INTO indicator_parameters (indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description) VALUES
(1, 'period', 'integer', true, 2, 100, '14', 'Number of periods for RSI calculation');

-- Bollinger Bands Parameters
INSERT INTO indicator_parameters (indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description) VALUES
(2, 'period', 'integer', true, 5, 100, '20', 'Number of periods for the moving average'),
(2, 'deviations', 'float', true, 0.1, 5, '2', 'Number of standard deviations for the upper and lower bands');

-- MACD Parameters
INSERT INTO indicator_parameters (indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description) VALUES
(3, 'fastPeriod', 'integer', true, 2, 100, '12', 'Number of periods for the fast EMA'),
(3, 'slowPeriod', 'integer', true, 2, 100, '26', 'Number of periods for the slow EMA'),
(3, 'signalPeriod', 'integer', true, 2, 100, '9', 'Number of periods for the signal line');

-- Moving Average Parameters
INSERT INTO indicator_parameters (indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description) VALUES
(4, 'period', 'integer', true, 2, 200, '20', 'Number of periods for the moving average'),
(4, 'type', 'enum', true, NULL, NULL, 'sma', 'Type of moving average');

-- Moving Average Enum Values
INSERT INTO parameter_enum_values (parameter_id, enum_value, display_name) VALUES
(7, 'sma', 'Simple Moving Average'),
(7, 'ema', 'Exponential Moving Average'),
(7, 'wma', 'Weighted Moving Average'),
(7, 'dema', 'Double Exponential Moving Average'),
(7, 'tema', 'Triple Exponential Moving Average');

-- Stochastic Parameters
INSERT INTO indicator_parameters (indicator_id, parameter_name, parameter_type, is_required, min_value, max_value, default_value, description) VALUES
(5, 'kPeriod', 'integer', true, 1, 100, '14', 'Number of periods for %K'),
(5, 'dPeriod', 'integer', true, 1, 100, '3', 'Number of periods for %D'),
(5, 'slowing', 'integer', true, 1, 100, '3', 'Slowing period');