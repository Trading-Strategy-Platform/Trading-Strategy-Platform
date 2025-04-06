-- Strategy Service Database Schema
-- File: 01-schema.sql
-- Contains type definitions and table structures without constraints

-- Type definitions
CREATE TYPE "user_role" AS ENUM (
  'admin',
  'user'
);

CREATE TYPE "timeframe_type" AS ENUM (
  '1m',
  '5m',
  '15m',
  '30m',
  '1h',
  '4h',
  '1d',
  '1w'
);

-- Strategies
CREATE TABLE IF NOT EXISTS "strategies" (
  "id" SERIAL PRIMARY KEY,
  "name" varchar(100) NOT NULL,
  "user_id" int NOT NULL,
  "description" text,
  "thumbnail_url" varchar(255),
  "structure" jsonb NOT NULL,
  "is_public" boolean NOT NULL DEFAULT false,
  "is_active" boolean NOT NULL DEFAULT true,
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
  "is_deleted" boolean NOT NULL DEFAULT false,
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

-- User Strategy Versions
CREATE TABLE IF NOT EXISTS "user_strategy_versions" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "strategy_id" int NOT NULL,
  "active_version" int NOT NULL,
  "updated_at" timestamp NOT NULL DEFAULT (CURRENT_TIMESTAMP)
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
  "version_id" int NOT NULL DEFAULT 1,
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
  "strategy_version" int NOT NULL,
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