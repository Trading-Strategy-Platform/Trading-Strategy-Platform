# Container Restart Script
# scripts/restart-containers.ps1

Write-Host "===== TRADING PLATFORM CONTAINER RESTART =====" -ForegroundColor Cyan

# Stop and remove all containers and volumes
Write-Host "Stopping all containers and removing volumes..." -ForegroundColor Yellow
docker compose down -v
if ($LASTEXITCODE -ne 0) {
    Write-Host "Error stopping containers. Trying to force remove..." -ForegroundColor Red
    docker compose rm -f -v
}

# Remove any dangling images
Write-Host "Cleaning up Docker system..." -ForegroundColor Yellow
docker system prune -f

# Start the infrastructure containers first and wait
Write-Host "Starting infrastructure containers (databases, Kafka, Redis)..." -ForegroundColor Green
docker compose up -d user-db strategy-db historical-db zookeeper kafka redis
Write-Host "Waiting 30 seconds for infrastructure containers to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

# Start the service containers
Write-Host "Starting service containers..." -ForegroundColor Green
docker compose up -d user-service strategy-service historical-service 
Write-Host "Waiting 15 seconds for services to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# Finally start the API gateway
Write-Host "Starting API gateway..." -ForegroundColor Green
docker compose up -d api-gateway
Write-Host "Waiting 10 seconds for API gateway to initialize..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# Check if all containers are running
Write-Host "Checking container status..." -ForegroundColor Cyan
$containers = docker ps --format "{{.Names}}"
$requiredContainers = @(
    "user-service", 
    "strategy-service", 
    "historical-service", 
    "api-gateway", 
    "user-service-db", 
    "strategy-service-db", 
    "historical-service-db", 
    "kafka", 
    "redis",
    "zookeeper"
)

$allRunning = $true
foreach ($container in $requiredContainers) {
    if ($containers -notcontains $container) {
        Write-Host "❌ Container $container is not running!" -ForegroundColor Red
        $allRunning = $false
    } else {
        Write-Host "✅ Container $container is running" -ForegroundColor Green
    }
}

if ($allRunning) {
    Write-Host "`nAll containers are running successfully!" -ForegroundColor Green
    Write-Host "The trading platform should now be accessible at: http://localhost:8080" -ForegroundColor Cyan
} else {
    Write-Host "`nSome containers failed to start." -ForegroundColor Red
    Write-Host "Check the logs for more information:" -ForegroundColor Yellow
    Write-Host "  docker logs user-service" -ForegroundColor Yellow
    Write-Host "  docker logs api-gateway" -ForegroundColor Yellow
}

Write-Host "`n===== RESTART COMPLETE =====" -ForegroundColor Cyan