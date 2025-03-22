# Platform Diagnostic and Test Script (Simplified Version)
# scripts/test-platform.ps1

# Create output directory for logs
$LogDir = ".\diagnostic-logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Write-Log {
    param (
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    Write-Host $logMessage
    
    # Also append to log file
    $logMessage | Out-File -FilePath "$LogDir\diagnostic.log" -Append
}

function Test-ContainersRunning {
    Write-Log "Checking if all containers are running..."
    
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
        "redis"
    )
    
    $allRunning = $true
    foreach ($container in $requiredContainers) {
        if ($containers -notcontains $container) {
            Write-Log "Container $container is not running!" "ERROR"
            $allRunning = $false
        } else {
            Write-Log "Container $container is running." "INFO"
        }
    }
    
    return $allRunning
}

function Get-ContainerLogs {
    param (
        [string]$ContainerName
    )
    
    Write-Log "Getting logs for $ContainerName..."
    docker logs $ContainerName > "$LogDir\$ContainerName.log" 2>&1
    
    # Return the last 10 lines for quick diagnosis
    $lastLines = Get-Content "$LogDir\$ContainerName.log" -Tail 10
    return $lastLines
}

function Test-ServiceHealth {
    param (
        [string]$ServiceName,
        [string]$Url
    )
    
    try {
        Write-Log "Checking health of $ServiceName at $Url..."
        # Don't store the response since we don't use it
        Invoke-RestMethod -Method Get -Uri $Url -TimeoutSec 5 | Out-Null
        Write-Log "Health check successful for $ServiceName" "INFO"
        return $true
    } catch {
        Write-Log "Testing direct connection to database $DbName on ${DbHost}:${DbPort}..."
        return $false
    }
}

function Wait-ForServices {
    Write-Log "Waiting for services to be ready..."
    $maxRetries = 10
    $retryCount = 0
    $allHealthy = $false
    
    while (-not $allHealthy -and $retryCount -lt $maxRetries) {
        $retryCount++
        Write-Log "Attempt $retryCount of $maxRetries..."
        
        $apiGatewayHealth = Test-ServiceHealth "API Gateway" "http://localhost:8080/health"
        $userServiceHealth = Test-ServiceHealth "User Service" "http://localhost:8081/health"
        $strategyServiceHealth = Test-ServiceHealth "Strategy Service" "http://localhost:8082/health"
        $historicalServiceHealth = Test-ServiceHealth "Historical Service" "http://localhost:8083/health"
        
        $allHealthy = $apiGatewayHealth -and $userServiceHealth -and $strategyServiceHealth -and $historicalServiceHealth
        
        if (-not $allHealthy) {
            Write-Log "Not all services are healthy, waiting 10 seconds before retry..." "WARN"
            Start-Sleep -Seconds 10
        }
    }
    
    return $allHealthy
}

# Note: Using plain text passwords for simplicity in this diagnostic script
# In production, these should be SecureString objects
function Test-RegisterUser {
    param (
        [string]$Username,
        [string]$Email,
        [string]$Password  # Using string for simplicity in this diagnostic script
    )
    
    try {
        Write-Log "Attempting to register user $Username..."
        $body = @{
            username = $Username
            email = $Email
            password = $Password
        } | ConvertTo-Json
        
        $response = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/auth/register" `
            -ContentType "application/json" `
            -Body $body
        
        Write-Log "User registration successful" "INFO"
        return $response
    } catch {
        Write-Log "User registration failed: $_" "ERROR"
        # Try to get more details about the error
        try {
            $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
            Write-Log "Error response: $($errorResponse | ConvertTo-Json)" "ERROR"
        } catch {
            Write-Log "Could not parse error response" "ERROR"
        }
        return $null
    }
}

# Note: Using plain text passwords for simplicity in this diagnostic script
# In production, these should be SecureString objects
function Test-LoginUser {
    param (
        [string]$Email,
        [string]$Password  # Using string for simplicity in this diagnostic script
    )
    
    try {
        Write-Log "Attempting to login user $Email..."
        $body = @{
            email = $Email
            password = $Password
        } | ConvertTo-Json
        
        $response = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/auth/login" `
            -ContentType "application/json" `
            -Body $body
        
        Write-Log "User login successful" "INFO"
        return $response
    } catch {
        Write-Log "User login failed: $_" "ERROR"
        # Try to get more details about the error
        try {
            $errorResponse = $_.ErrorDetails.Message | ConvertFrom-Json
            Write-Log "Error response: $($errorResponse | ConvertTo-Json)" "ERROR"
        } catch {
            Write-Log "Could not parse error response" "ERROR"
        }
        return $null
    }
}

# Note: Using plain text passwords for simplicity in this diagnostic script
# In production, these should be SecureString objects
function Test-DirectDatabaseConnection {
    param (
        [string]$DbHost,
        [string]$DbPort,
        [string]$DbName,
        [string]$DbUser,
        [string]$DbPassword  # Using string for simplicity in this diagnostic script
    )
    
    # Using ${} to properly format the host:port string
    Write-Log "Testing direct connection to database $DbName on ${DbHost}:$DbPort..."
    
    # Check if psql is available
    try {
        $psqlVersion = docker exec user-service-db psql --version
        Write-Log "PostgreSQL client available: $psqlVersion" "INFO"
        
        # Try connecting to the database
        $command = "PGPASSWORD='$DbPassword' psql -h $DbHost -p $DbPort -U $DbUser -d $DbName -c 'SELECT 1'"
        $result = docker exec user-service-db bash -c $command 2>&1
        
        if ($result -match "1 row") {
            Write-Log "Database connection successful" "INFO"
            return $true
        } else {
            Write-Log "Database connection failed: $result" "ERROR"
            return $false
        }
    } catch {
        Write-Log "PostgreSQL client not available or error: $_" "ERROR"
        return $false
    }
}

function Test-DatabaseInitialization {
    Write-Log "Checking if databases are properly initialized..."
    
    # Check User DB
    $userDbConnection = Test-DirectDatabaseConnection "user-db" "5432" "user_service" "user_service_user" "user_service_password"
    
    # Check Strategy DB
    $strategyDbConnection = Test-DirectDatabaseConnection "strategy-db" "5432" "strategy_service" "strategy_service_user" "strategy_service_password"
    
    # Check Historical DB
    $historicalDbConnection = Test-DirectDatabaseConnection "historical-db" "5432" "historical_service" "historical_service_user" "historical_service_password"
    
    return $userDbConnection -and $strategyDbConnection -and $historicalDbConnection
}

function Test-ServiceConfigurations {
    Write-Log "Checking service configurations..."
    
    # Get and save config files
    docker cp user-service:/app/config/config.yaml "$LogDir\user-service-config.yaml"
    docker cp strategy-service:/app/config/config.yaml "$LogDir\strategy-service-config.yaml"
    docker cp historical-service:/app/config/config.yaml "$LogDir\historical-service-config.yaml"
    docker cp api-gateway:/app/config/config.yaml "$LogDir\api-gateway-config.yaml"
    
    Write-Log "Service configurations saved to $LogDir" "INFO"
}

function Restart-Services {
    Write-Log "Restarting all services..."
    
    # Restart services in the correct order
    docker compose restart user-db strategy-db historical-db
    Start-Sleep -Seconds 10
    docker compose restart user-service strategy-service historical-service
    Start-Sleep -Seconds 10
    docker compose restart api-gateway
    
    Write-Log "All services restarted, waiting for them to be ready..." "INFO"
    Start-Sleep -Seconds 20
}

# Main diagnostic flow
Write-Log "Starting platform diagnostics..." "INFO"

$containersRunning = Test-ContainersRunning

if (-not $containersRunning) {
    Write-Log "Some required containers are not running. Starting them..." "WARN"
    docker compose up -d
    Start-Sleep -Seconds 30
    $containersRunning = Test-ContainersRunning
}

if ($containersRunning) {
    Write-Log "All required containers are running, getting logs..." "INFO"
    
    # Get logs for key services
    $userServiceLogs = Get-ContainerLogs "user-service"
    Get-ContainerLogs "strategy-service"  # Just save to file
    Get-ContainerLogs "historical-service"  # Just save to file
    $apiGatewayLogs = Get-ContainerLogs "api-gateway"
    
    Write-Log "Last 10 lines of user-service logs:" "INFO"
    $userServiceLogs | ForEach-Object { Write-Log $_ "LOG" }
    
    Write-Log "Last 10 lines of api-gateway logs:" "INFO"
    $apiGatewayLogs | ForEach-Object { Write-Log $_ "LOG" }
    
    # Check service configurations
    Test-ServiceConfigurations
    
    # Check if services are healthy
    $servicesHealthy = Wait-ForServices
    
    if (-not $servicesHealthy) {
        Write-Log "Services are not healthy, checking database initialization..." "WARN"
        $dbsInitialized = Test-DatabaseInitialization
        
        if (-not $dbsInitialized) {
            Write-Log "Database initialization issues detected. This may be caused by missing schema or incorrect credentials." "ERROR"
        }
        
        Write-Log "Attempting to restart services..." "WARN"
        Restart-Services
        $servicesHealthy = Wait-ForServices
    }
    
    if ($servicesHealthy) {
        Write-Log "All services are healthy, testing API..." "INFO"
        
        # Register a test user
        $username = "testuser" + (Get-Random -Minimum 1000 -Maximum 9999)
        $email = "$username@example.com"
        $password = "SecurePass" + (Get-Random -Minimum 1000 -Maximum 9999)
        
        $registration = Test-RegisterUser $username $email $password
        
        if ($registration) {
            Write-Log "User registration successful, attempting login..." "INFO"
            $login = Test-LoginUser $email $password
            
            if ($login) {
                Write-Log "Login successful! Here's your token:" "INFO"
                Write-Log $login.token "INFO"
                
                # Save token for future use
                $login | ConvertTo-Json | Out-File -FilePath "$LogDir\login-response.json"
                
                Write-Log "Platform is working correctly!" "INFO"
            }
        }
    } else {
        Write-Log "Services are not healthy after restart. Manual intervention required." "ERROR"
    }
} else {
    Write-Log "Some required containers are still not running. Manual intervention required." "ERROR"
}

Write-Log "Diagnostic complete. Check $LogDir for detailed logs." "INFO"