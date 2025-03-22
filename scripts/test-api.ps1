# API Testing Script
# scripts/test-api.ps1

$LogFile = "api-test-results.log"

function Write-Log {
    param (
        [string]$Message,
        [string]$Level = "INFO"
    )
    
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $logMessage = "[$timestamp] [$Level] $Message"
    Write-Host $logMessage
    
    # Also append to log file
    $logMessage | Out-File -FilePath $LogFile -Append
}

# Clear previous log file
if (Test-Path $LogFile) {
    Remove-Item $LogFile
}

Write-Log "Starting API tests..."

# Register a new user
try {
    Write-Log "Test 1: Registering a new user..."
    $username = "testuser" + (Get-Random -Minimum 1000 -Maximum 9999)
    $email = "$username@example.com"
    $password = "SecurePass" + (Get-Random -Minimum 1000 -Maximum 9999)
    
    Write-Log "Registering user: $username / $email"
    
    $body = @{
        username = $username
        email = $email
        password = $password
    } | ConvertTo-Json
    
    $registrationResponse = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/auth/register" `
        -ContentType "application/json" `
        -Body $body -ErrorAction Stop
    
    Write-Log "Registration successful! Response:" "SUCCESS"
    Write-Log ($registrationResponse | ConvertTo-Json) "SUCCESS"
} catch {
    Write-Log "Registration failed: $_" "ERROR"
    
    if ($_.Exception.Response) {
        Write-Log "Error details: $($_.Exception.Response.StatusCode.value__) $($_.Exception.Response.StatusDescription)" "ERROR"
    }
    
    if ($_.ErrorDetails) {
        try {
            $errorBody = $_.ErrorDetails.Message | ConvertFrom-Json
            Write-Log "Error body: $($errorBody | ConvertTo-Json)" "ERROR"
        } catch {
            Write-Log "Raw error details: $($_.ErrorDetails.Message)" "ERROR"
        }
    }
    
    # Let's try to access the swagger docs to see if the API is accessible at all
    try {
        $swaggerResponse = Invoke-WebRequest -Uri "http://localhost:8080/swagger/index.html" -ErrorAction Stop
        Write-Log "Swagger docs accessible: $($swaggerResponse.StatusCode)" "INFO"
    } catch {
        Write-Log "Swagger docs not accessible: $_" "ERROR"
    }
}

# Try to log in
try {
    Write-Log "Test 2: Logging in with the new user..."
    
    $body = @{
        email = $email
        password = $password
    } | ConvertTo-Json
    
    $loginResponse = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/auth/login" `
        -ContentType "application/json" `
        -Body $body -ErrorAction Stop
    
    Write-Log "Login successful! Token received." "SUCCESS"
    $token = $loginResponse.token
    Write-Log "Token: $token" "SUCCESS"
    
    # Save token to file for manual testing
    $token | Out-File -FilePath "auth-token.txt"
    
    # Continue with authenticated requests
    if ($token) {
        # Test 3: Get user profile
        try {
            Write-Log "Test 3: Getting user profile..."
            
            $profileResponse = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/users/me" `
                -Headers @{Authorization = "Bearer $token"} -ErrorAction Stop
            
            Write-Log "Profile retrieved successfully!" "SUCCESS"
            Write-Log ($profileResponse | ConvertTo-Json) "SUCCESS"
        } catch {
            Write-Log "Profile retrieval failed: $_" "ERROR"
        }
        
        # Test 4: Create a strategy
        try {
            Write-Log "Test 4: Creating a trading strategy..."
            
            $strategyBody = @{
                name = "Test Strategy " + (Get-Random -Minimum 1000 -Maximum 9999)
                description = "A test strategy created by the API test script"
                structure = @{
                    buyRules = @(
                        @{
                            type = "rule"
                            indicator = @{
                                id = "2"
                                name = "RSI"
                            }
                            condition = @{
                                symbol = "<="
                            }
                            value = "30"
                            indicatorSettings = @{
                                period = 14
                            }
                            operator = "AND"
                        }
                    )
                    sellRules = @(
                        @{
                            type = "rule"
                            indicator = @{
                                id = "2"
                                name = "RSI"
                            }
                            condition = @{
                                symbol = ">="
                            }
                            value = "70"
                            indicatorSettings = @{
                                period = 14
                            }
                            operator = "AND"
                        }
                    )
                }
                is_public = $false
            } | ConvertTo-Json -Depth 10
            
            $strategyResponse = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/strategies" `
                -ContentType "application/json" `
                -Headers @{Authorization = "Bearer $token"} `
                -Body $strategyBody -ErrorAction Stop
            
            Write-Log "Strategy created successfully!" "SUCCESS"
            Write-Log ($strategyResponse | ConvertTo-Json) "SUCCESS"
            
            # Save strategy ID
            $strategyId = $strategyResponse.id
        } catch {
            Write-Log "Strategy creation failed: $_" "ERROR"
        }
        
        # Test 5: Get all strategies
        try {
            Write-Log "Test 5: Getting all strategies..."
            
            $strategiesResponse = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/strategies" `
                -Headers @{Authorization = "Bearer $token"} -ErrorAction Stop
            
            Write-Log "Strategies retrieved successfully!" "SUCCESS"
            Write-Log "Found $($strategiesResponse.Length) strategies" "SUCCESS"
            
            if ($strategiesResponse.Length -gt 0) {
                Write-Log "First strategy: $($strategiesResponse[0] | ConvertTo-Json)" "SUCCESS"
            }
        } catch {
            Write-Log "Strategies retrieval failed: $_" "ERROR"
        }
        
        # Test 6: Get available symbols
        try {
            Write-Log "Test 6: Getting available symbols..."
            
            $symbolsResponse = Invoke-RestMethod -Method Get -Uri "http://localhost:8080/api/v1/symbols" `
                -Headers @{Authorization = "Bearer $token"} -ErrorAction Stop
            
            Write-Log "Symbols retrieved successfully!" "SUCCESS"
            Write-Log "Found $($symbolsResponse.Length) symbols" "SUCCESS"
            
            # Save first symbol ID for backtest
            if ($symbolsResponse.Length -gt 0) {
                $symbolId = $symbolsResponse[0].id
                Write-Log "First symbol: $($symbolsResponse[0] | ConvertTo-Json)" "SUCCESS"
            } else {
                Write-Log "No symbols found. Cannot proceed with backtest." "WARN"
            }
        } catch {
            Write-Log "Symbols retrieval failed: $_" "ERROR"
        }
        
        # Test 7: Create a backtest (if we have a strategy and symbol)
        if ($strategyId -and $symbolId) {
            try {
                Write-Log "Test 7: Creating a backtest..."
                
                $backtestBody = @{
                    strategy_id = $strategyId
                    symbol_id = $symbolId
                    timeframe_minutes = 15
                    start_date = "2023-01-01T00:00:00Z"
                    end_date = "2023-12-31T23:59:59Z"
                    initial_capital = 10000
                } | ConvertTo-Json
                
                $backtestResponse = Invoke-RestMethod -Method Post -Uri "http://localhost:8080/api/v1/backtests" `
                    -ContentType "application/json" `
                    -Headers @{Authorization = "Bearer $token"} `
                    -Body $backtestBody -ErrorAction Stop
                
                Write-Log "Backtest created successfully!" "SUCCESS"
                Write-Log ($backtestResponse | ConvertTo-Json) "SUCCESS"
            } catch {
                Write-Log "Backtest creation failed: $_" "ERROR"
            }
        }
    }
} catch {
    Write-Log "Login failed: $_" "ERROR"
}

Write-Log "API tests completed. Check $LogFile for results." "INFO"

# Return test status
if ((Select-String -Path $LogFile -Pattern "ERROR").Count -eq 0) {
    Write-Log "All tests completed successfully!" "SUCCESS"
    exit 0
} else {
    Write-Log "Some tests failed. Review the log for details." "ERROR"
    exit 1
}