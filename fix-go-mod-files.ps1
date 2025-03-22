# PowerShell script to downgrade problematic dependencies to Go 1.21 compatible versions

# Array of service directories
$services = @(
    "services/user-service",
    "services/strategy-service", 
    "services/historical-data-service",
    "services/api-gateway"
)

# Loop through each service
foreach ($service in $services) {
    Write-Host "Processing $service..."
    
    # Navigate to the service directory
    Push-Location $service
    
    # First set Go version to 1.21
    go mod edit -go=1.21
    Write-Host "  - Set Go version to 1.21"
    
    # Explicitly downgrade problematic dependencies
    go mod edit -require golang.org/x/crypto@v0.14.0
    Write-Host "  - Downgraded golang.org/x/crypto to v0.14.0"
    
    go mod edit -require github.com/klauspost/cpuid/v2@v2.0.9
    Write-Host "  - Downgraded github.com/klauspost/cpuid/v2 to v2.0.9"
    
    # Check if bytedance/sonic is used and downgrade it
    if (Select-String -Path "go.mod" -Pattern "github.com/bytedance/sonic" -Quiet) {
        go mod edit -require github.com/bytedance/sonic@v1.9.1
        Write-Host "  - Downgraded github.com/bytedance/sonic to v1.9.1"
    }
    
    # Downgrade other dependencies known to require newer Go versions
    go mod edit -require golang.org/x/net@v0.10.0
    Write-Host "  - Downgraded golang.org/x/net to v0.10.0"
    
    go mod edit -require golang.org/x/sys@v0.8.0
    Write-Host "  - Downgraded golang.org/x/sys to v0.8.0"
    
    go mod edit -require golang.org/x/text@v0.9.0
    Write-Host "  - Downgraded golang.org/x/text to v0.9.0"
    
    go mod edit -require google.golang.org/protobuf@v1.30.0
    Write-Host "  - Downgraded google.golang.org/protobuf to v1.30.0"
    
    # Run go mod tidy
    go mod tidy
    Write-Host "  - Ran go mod tidy"
    
    # Create empty go.sum if it doesn't exist
    if (-not (Test-Path "go.sum")) {
        New-Item -ItemType File -Path "go.sum" -Force | Out-Null
        Write-Host "  - Created empty go.sum file"
    }
    
    # Return to the original directory
    Pop-Location
}

Write-Host "All services dependencies have been downgraded for Go 1.21 compatibility!"