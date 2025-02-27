#!/bin/bash
# fix-dependencies.sh - Script to install all required dependencies

# User Service dependencies
echo "Fixing User Service dependencies..."
cd services/user-service
go get github.com/golang-jwt/jwt/v4
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v4/stdlib
go get github.com/jmoiron/sqlx
go get go.uber.org/zap
go get golang.org/x/crypto/bcrypt
go get github.com/spf13/viper
go mod tidy
cd ../..

# Strategy Service 
echo "Fixing Strategy Service dependencies..."
cd services/strategy-service
go mod tidy || true
cd ../..

# Historical Data Service
echo "Fixing Historical Data Service dependencies..."
cd services/historical-data-service
go mod tidy || true
cd ../..

# API Gateway
echo "Fixing API Gateway dependencies..."
cd services/api-gateway
go mod tidy || true
cd ../..

echo "Done! All dependencies should be fixed."