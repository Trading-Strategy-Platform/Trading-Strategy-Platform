-- Strategy Service Database Indexes and Constraints
-- File: 02-indexes.sql
-- Contains indexes, foreign keys, and other constraints

-- Indexes for strategies table
CREATE INDEX "idx_strategies_user_id" ON "strategies" ("user_id");
CREATE INDEX ON "strategies" ("is_public");
CREATE INDEX ON "strategies" ("is_active");
CREATE INDEX ON "strategies" ("strategy_group_id");
CREATE INDEX ON "strategies" ("strategy_group_id", "version");

-- Indexes for other tables
CREATE UNIQUE INDEX ON "indicator_parameters" ("indicator_id", "parameter_name");
CREATE INDEX ON "strategy_marketplace" ("is_active");
CREATE UNIQUE INDEX ON "strategy_marketplace" ("strategy_id", "version_id");
CREATE UNIQUE INDEX ON "strategy_reviews" ("marketplace_id", "user_id");
CREATE UNIQUE INDEX ON "user_strategy_versions" ("user_id", "strategy_group_id");

-- Add Foreign Keys
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("strategy_group_id") ON DELETE CASCADE;
ALTER TABLE "strategy_tag_mappings" ADD FOREIGN KEY ("tag_id") REFERENCES "strategy_tags" ("id") ON DELETE CASCADE;
ALTER TABLE "user_strategy_versions" ADD FOREIGN KEY ("strategy_group_id") REFERENCES "strategies" ("strategy_group_id") ON DELETE CASCADE;
ALTER TABLE "user_strategy_versions" ADD FOREIGN KEY ("active_version_id") REFERENCES "strategies" ("id") ON DELETE CASCADE;
ALTER TABLE "indicator_parameters" ADD FOREIGN KEY ("indicator_id") REFERENCES "indicators" ("id") ON DELETE CASCADE;
ALTER TABLE "parameter_enum_values" ADD FOREIGN KEY ("parameter_id") REFERENCES "indicator_parameters" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_marketplace" ADD FOREIGN KEY ("strategy_id") REFERENCES "strategies" ("strategy_group_id") ON DELETE CASCADE;
ALTER TABLE "strategy_purchases" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;
ALTER TABLE "strategy_reviews" ADD FOREIGN KEY ("marketplace_id") REFERENCES "strategy_marketplace" ("id") ON DELETE CASCADE;