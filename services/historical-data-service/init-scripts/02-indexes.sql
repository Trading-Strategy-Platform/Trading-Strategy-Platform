-- Indexes
CREATE INDEX "idx_backtest_trades_backtest_run_id" ON "backtest_trades" ("backtest_run_id");
CREATE INDEX "idx_backtests_user_id" ON "backtests" ("user_id");
CREATE INDEX "idx_backtests_strategy_id" ON "backtests" ("strategy_id");
CREATE INDEX "idx_backtest_runs_backtest_id" ON "backtest_runs" ("backtest_id");
CREATE UNIQUE INDEX ON "backtest_runs" ("backtest_id", "symbol_id");
CREATE INDEX "idx_market_data_download_jobs_status" ON "market_data_download_jobs" ("status");
CREATE INDEX "idx_market_data_download_jobs_symbol_id" ON "market_data_download_jobs" ("symbol_id");
CREATE INDEX "idx_market_data_download_jobs_source" ON "market_data_download_jobs" ("source");
CREATE INDEX "idx_symbols_asset_type" ON "symbols" ("asset_type");
CREATE INDEX "idx_symbols_exchange" ON "symbols" ("exchange");
CREATE INDEX "idx_symbols_symbol" ON "symbols" ("symbol");

-- Foreign Keys
ALTER TABLE "candles" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_runs" ADD FOREIGN KEY ("backtest_id") REFERENCES "backtests" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_runs" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_results" ADD FOREIGN KEY ("backtest_run_id") REFERENCES "backtest_runs" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_trades" ADD FOREIGN KEY ("backtest_run_id") REFERENCES "backtest_runs" ("id") ON DELETE CASCADE;
ALTER TABLE "backtest_trades" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;
ALTER TABLE "market_data_download_jobs" ADD FOREIGN KEY ("symbol_id") REFERENCES "symbols" ("id") ON DELETE CASCADE;

-- Convert candles to hypertable for time series optimization
SELECT create_hypertable('candles', 'candle_time', chunk_time_interval => INTERVAL '1 week');