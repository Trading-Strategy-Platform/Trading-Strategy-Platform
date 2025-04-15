-- Historical Data Service Database Schema

-- Create timeframe_type enum
CREATE TYPE "timeframe_type" AS ENUM (
  '1m', '5m', '15m', '30m', '1h', '4h', '1d', '1w'
);

-- Extension for time series data
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Symbols table
CREATE TABLE IF NOT EXISTS "symbols" (
  "id" SERIAL PRIMARY KEY,
  "symbol" varchar(20) UNIQUE NOT NULL,
  "name" varchar(100) NOT NULL,
  "asset_type" varchar(20) NOT NULL,
  "exchange" varchar(50),
  "is_active" boolean NOT NULL DEFAULT true,
  "data_available" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz
);

-- Candles table - Using "candle_time" as column name to avoid reserved keywords
CREATE TABLE IF NOT EXISTS "candles" (
  "symbol_id" int NOT NULL,
  "candle_time" timestamptz NOT NULL,
  "open" numeric(20,8) NOT NULL,
  "high" numeric(20,8) NOT NULL,
  "low" numeric(20,8) NOT NULL,
  "close" numeric(20,8) NOT NULL,
  "volume" numeric(20,8) NOT NULL,
  PRIMARY KEY ("symbol_id", "candle_time")
);

-- Backtests table
CREATE TABLE IF NOT EXISTS "backtests" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "strategy_id" int NOT NULL,
  "strategy_version" int NOT NULL,
  "name" varchar(100),
  "description" text,
  "timeframe" timeframe_type NOT NULL,
  "start_date" timestamptz NOT NULL,
  "end_date" timestamptz NOT NULL,
  "initial_capital" numeric(20,8) NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "error_message" text,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz,
  "completed_at" timestamptz
);

-- Backtest runs table
CREATE TABLE IF NOT EXISTS "backtest_runs" (
  "id" SERIAL PRIMARY KEY,
  "backtest_id" int NOT NULL,
  "symbol_id" int NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "completed_at" timestamptz
);

-- Backtest results table
CREATE TABLE IF NOT EXISTS "backtest_results" (
  "id" SERIAL PRIMARY KEY,
  "backtest_run_id" int NOT NULL,
  "total_trades" int NOT NULL DEFAULT 0,
  "winning_trades" int NOT NULL DEFAULT 0,
  "losing_trades" int NOT NULL DEFAULT 0,
  "profit_factor" numeric(10,4),
  "sharpe_ratio" numeric(10,4),
  "max_drawdown" numeric(10,4),
  "final_capital" numeric(20,8),
  "total_return" numeric(10,4),
  "annualized_return" numeric(10,4),
  "results_json" jsonb
);

-- Backtest trades table
CREATE TABLE IF NOT EXISTS "backtest_trades" (
  "id" SERIAL PRIMARY KEY,
  "backtest_run_id" int NOT NULL,
  "symbol_id" int NOT NULL,
  "entry_time" timestamptz NOT NULL,
  "exit_time" timestamptz,
  "position_type" varchar(10) NOT NULL,
  "entry_price" numeric(20,8) NOT NULL,
  "exit_price" numeric(20,8),
  "quantity" numeric(20,8) NOT NULL,
  "profit_loss" numeric(20,8),
  "profit_loss_percent" numeric(10,4),
  "exit_reason" varchar(50)
);

-- Market data download jobs table
CREATE TABLE IF NOT EXISTS "market_data_download_jobs" (
  "id" SERIAL PRIMARY KEY,
  "symbol_id" int NOT NULL,
  "symbol" varchar(20) NOT NULL,
  "source" varchar(50) NOT NULL,
  "timeframe" timeframe_type NOT NULL,
  "start_date" timestamptz NOT NULL,
  "end_date" timestamptz NOT NULL,
  "status" varchar(20) NOT NULL DEFAULT 'pending',
  "progress" numeric(5,2) NOT NULL DEFAULT 0,
  "total_candles" int NOT NULL DEFAULT 0,
  "processed_candles" int NOT NULL DEFAULT 0,
  "retries" int NOT NULL DEFAULT 0,
  "error" text,
  "created_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "updated_at" timestamptz NOT NULL DEFAULT (CURRENT_TIMESTAMP),
  "last_processed_time" timestamptz
);