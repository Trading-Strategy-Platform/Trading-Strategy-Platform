#!/bin/bash
# Test script for API Gateway functionality

# Set base URL
BASE_URL=${1:-"http://localhost:8080"}

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to test an endpoint
test_endpoint() {
    local endpoint=$1
    local method=${2:-GET}
    local expected_status=${3:-200}
    local auth_header=${4:-""}
    
    echo -n "Testing $method $endpoint (expecting $expected_status): "
    
    if [ -z "$auth_header" ]; then
        response=$(curl -s -o /dev/null -w "%{http_code}" -X $method $BASE_URL$endpoint)
    else
        response=$(curl -s -o /dev/null -w "%{http_code}" -X $method -H "Authorization: $auth_header" $BASE_URL$endpoint)
    fi
    
    if [ "$response" -eq "$expected_status" ]; then
        echo -e "${GREEN}PASS${NC} ($response)"
        return 0
    else
        echo -e "${RED}FAIL${NC} (got $response, expected $expected_status)"
        return 1
    fi
}

# Test health check endpoint
test_endpoint "/health"

# Test API not found
test_endpoint "/not-found" "GET" 404

# Test auth endpoints (should return 400/401 without valid credentials)
test_endpoint "/api/v1/auth/login" "POST" 400
test_endpoint "/api/v1/auth/register" "POST" 400

# Test protected endpoints without auth (should return 401)
test_endpoint "/api/v1/users/me" "GET" 401
test_endpoint "/api/v1/strategies" "GET" 401

# Test admin endpoints without auth (should return 401)
test_endpoint "/api/v1/admin/users" "GET" 401

# Test rate limiting
echo "Testing rate limiting..."
for i in {1..10}; do
    curl -s -o /dev/null -X GET $BASE_URL/health
done
test_endpoint "/health" "GET" 200

echo -e "\nNote: For complete testing with authentication, you need valid JWT tokens."
echo "To test authenticated endpoints, run with a token:"
echo "./test.sh http://localhost:8080 \"Bearer your_jwt_token\""

# If a token was provided as the second argument, test authenticated endpoints
if [ ! -z "$2" ]; then
    echo -e "\nTesting authenticated endpoints with provided token..."
    test_endpoint "/api/v1/users/me" "GET" 200 "$2"
    test_endpoint "/api/v1/strategies" "GET" 200 "$2"
fi

echo -e "\nAPI Gateway testing completed!"