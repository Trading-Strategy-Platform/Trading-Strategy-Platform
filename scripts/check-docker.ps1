# Docker Environment Check Script
# scripts/check-docker.ps1

# Output colors
$GREEN = [ConsoleColor]::Green
$RED = [ConsoleColor]::Red
$YELLOW = [ConsoleColor]::Yellow
$WHITE = [ConsoleColor]::White

function Write-ColorOutput {
    param (
        [string]$Message,
        [ConsoleColor]$Color = [ConsoleColor]::White
    )
    
    Write-Host $Message -ForegroundColor $Color
}

function Test-DockerRunning {
    try {
        $info = docker info 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ Docker is running correctly" $GREEN
            return $true
        } else {
            Write-ColorOutput "✗ Docker is not running or has an issue" $RED
            Write-Host $info
            return $false
        }
    } catch {
        Write-ColorOutput "✗ Docker is not installed or not in PATH" $RED
        return $false
    }
}

function Test-DockerCompose {
    try {
        $version = docker compose version 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ Docker Compose is available" $GREEN
            Write-Host "  $version"
            return $true
        } else {
            Write-ColorOutput "✗ Docker Compose is not available" $RED
            Write-Host $version
            return $false
        }
    } catch {
        Write-ColorOutput "✗ Docker Compose is not installed or not in PATH" $RED
        return $false
    }
}

function Show-RunningContainers {
    try {
        $containers = docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ Currently running containers:" $GREEN
            Write-Host $containers
        } else {
            Write-ColorOutput "✗ Error retrieving running containers" $RED
            Write-Host $containers
        }
    } catch {
        Write-ColorOutput "✗ Error retrieving running containers" $RED
    }
}

function Test-ServicePorts {
    $ports = @(
        @{Port = 8080; Service = "API Gateway"},
        @{Port = 8081; Service = "User Service"},
        @{Port = 8082; Service = "Strategy Service"},
        @{Port = 8083; Service = "Historical Data Service"},
        @{Port = 5432; Service = "User DB"},
        @{Port = 5433; Service = "Strategy DB"},
        @{Port = 5434; Service = "Historical DB"},
        @{Port = 9092; Service = "Kafka"},
        @{Port = 6379; Service = "Redis"}
    )
    
    Write-ColorOutput "Checking service ports..." $WHITE
    
    foreach ($port in $ports) {
        $result = Test-NetConnection -ComputerName localhost -Port $port.Port -WarningAction SilentlyContinue -ErrorAction SilentlyContinue
        if ($result.TcpTestSucceeded) {
            Write-ColorOutput "✓ Port $($port.Port) ($($port.Service)) is open and accessible" $GREEN
        } else {
            Write-ColorOutput "✗ Port $($port.Port) ($($port.Service)) is not accessible" $RED
        }
    }
}

function Test-DockerNetworks {
    try {
        $networks = docker network ls --format "table {{.Name}}\t{{.Driver}}\t{{.Scope}}" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ Docker networks available:" $GREEN
            Write-Host $networks
            
            # Check if our expected networks exist
            $expectedNetworks = @(
                "user-service-network",
                "strategy-service-network",
                "historical-service-network",
                "kafka-network",
                "redis-network",
                "api-gateway-network"
            )
            
            foreach ($network in $expectedNetworks) {
                $exists = docker network ls --format "{{.Name}}" | Where-Object { $_ -eq $network }
                if ($exists) {
                    Write-ColorOutput "✓ Expected network $network exists" $GREEN
                } else {
                    Write-ColorOutput "! Expected network $network does not exist" $YELLOW
                }
            }
        } else {
            Write-ColorOutput "✗ Error retrieving Docker networks" $RED
            Write-Host $networks
        }
    } catch {
        Write-ColorOutput "✗ Error retrieving Docker networks" $RED
    }
}

function Test-DockerVolumes {
    try {
        $volumes = docker volume ls --format "table {{.Name}}\t{{.Driver}}\t{{.Mountpoint}}" 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ Docker volumes available:" $GREEN
            Write-Host $volumes
            
            # Check if our expected volumes exist
            $expectedVolumes = @(
                "user-db-data",
                "strategy-db-data",
                "historical-db-data",
                "kafka-data",
                "zookeeper-data",
                "redis-data"
            )
            
            foreach ($volume in $expectedVolumes) {
                $exists = docker volume ls --format "{{.Name}}" | Where-Object { $_ -eq "trading-strategy-platform_$volume" }
                if ($exists) {
                    Write-ColorOutput "✓ Expected volume $volume exists" $GREEN
                } else {
                    Write-ColorOutput "! Expected volume $volume does not exist" $YELLOW
                }
            }
        } else {
            Write-ColorOutput "✗ Error retrieving Docker volumes" $RED
            Write-Host $volumes
        }
    } catch {
        Write-ColorOutput "✗ Error retrieving Docker volumes" $RED
    }
}

function Test-NetworkConnectivity {
    Write-ColorOutput "Checking network connectivity between containers..." $WHITE
    
    $serviceContainers = @("api-gateway", "user-service", "strategy-service", "historical-service")
    # Define dbContainers for clarity even though it's only used indirectly via dbMappings
    $dbContainers = @("user-service-db", "strategy-service-db", "historical-service-db")
    $infraContainers = @("kafka", "redis")
    
    # Check if API Gateway can reach services
    foreach ($service in $serviceContainers | Where-Object { $_ -ne "api-gateway" }) {
        $result = docker exec api-gateway ping -c 2 $service 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ API Gateway can reach $service" $GREEN
        } else {
            Write-ColorOutput "✗ API Gateway cannot reach $service" $RED
            Write-Host $result
        }
    }
    
    # Check if services can reach their DBs
    $dbMappings = @(
        @{Service = "user-service"; DB = "user-service-db"},
        @{Service = "strategy-service"; DB = "strategy-service-db"},
        @{Service = "historical-service"; DB = "historical-service-db"}
    )
    
    foreach ($mapping in $dbMappings) {
        $result = docker exec $mapping.Service ping -c 2 $mapping.DB 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-ColorOutput "✓ $($mapping.Service) can reach $($mapping.DB)" $GREEN
        } else {
            Write-ColorOutput "✗ $($mapping.Service) cannot reach $($mapping.DB)" $RED
            Write-Host $result
        }
    }
    
    # Check if services can reach Kafka & Redis
    foreach ($service in $serviceContainers) {
        foreach ($infra in $infraContainers) {
            $result = docker exec $service ping -c 2 $infra 2>&1
            if ($LASTEXITCODE -eq 0) {
                Write-ColorOutput "✓ $service can reach $infra" $GREEN
            } else {
                Write-ColorOutput "✗ $service cannot reach $infra" $RED
                Write-Host $result
            }
        }
    }
}

Write-ColorOutput "=======================================================" $WHITE
Write-ColorOutput "        Docker Environment Check for Trading Platform" $WHITE
Write-ColorOutput "=======================================================" $WHITE

# Run all checks
$dockerRunning = Test-DockerRunning
if ($dockerRunning) {
    $composeAvailable = Test-DockerCompose
    
    if ($composeAvailable) {
        Write-ColorOutput "`nChecking container status..." $WHITE
        Show-RunningContainers
        
        Write-ColorOutput "`nChecking container networks..." $WHITE
        Test-DockerNetworks
        
        Write-ColorOutput "`nChecking volumes..." $WHITE
        Test-DockerVolumes
        
        Write-ColorOutput "`nChecking port availability..." $WHITE
        Test-ServicePorts
        
        Write-ColorOutput "`nChecking container connectivity..." $WHITE
        Test-NetworkConnectivity
    }
}

Write-ColorOutput "`n=======================================================" $WHITE
Write-ColorOutput "                     Check Complete" $WHITE
Write-ColorOutput "=======================================================" $WHITE

# Provide recommendations based on results
if (-not $dockerRunning) {
    Write-ColorOutput "`nRecommendation: Start Docker Desktop and try again." $YELLOW
} elseif (-not $composeAvailable) {
    Write-ColorOutput "`nRecommendation: Install Docker Compose or update Docker Desktop." $YELLOW
} else {
    Write-ColorOutput "`nIf issues were found, consider the following actions:" $YELLOW
    Write-ColorOutput "1. Restart Docker Desktop" $YELLOW
    Write-ColorOutput "2. Run 'docker compose down -v' to clean up containers and volumes" $YELLOW
    Write-ColorOutput "3. Run 'docker compose up -d' to recreate all services" $YELLOW
    Write-ColorOutput "4. Check each service's logs with 'docker logs <container_name>'" $YELLOW
}