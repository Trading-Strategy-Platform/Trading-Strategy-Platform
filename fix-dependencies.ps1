# fix-dependencies.ps1 - Script to install all required dependencies for Windows environments

# User Service dependencies
Write-Host "Fixing User Service dependencies..."
Set-Location -Path services/user-service
go get github.com/golang-jwt/jwt/v4
go get github.com/gin-gonic/gin
go get github.com/jackc/pgx/v4/stdlib
go get github.com/jmoiron/sqlx
go get go.uber.org/zap
go get golang.org/x/crypto/bcrypt
go get github.com/spf13/viper
go mod tidy
Set-Location -Path ../..

# Strategy Service 
Write-Host "Fixing Strategy Service dependencies..."
Set-Location -Path services/strategy-service
go mod tidy
Set-Location -Path ../..

# Historical Data Service
Write-Host "Fixing Historical Data Service dependencies..."
Set-Location -Path services/historical-data-service
go mod tidy
Set-Location -Path ../..

# API Gateway
Write-Host "Fixing API Gateway dependencies..."
Set-Location -Path services/api-gateway
go mod tidy
Set-Location -Path ../..

Write-Host "Done! All dependencies should be fixed." 