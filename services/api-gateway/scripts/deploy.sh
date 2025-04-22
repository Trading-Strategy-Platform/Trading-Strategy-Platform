#!/bin/bash
# Deployment script for Nginx API Gateway

# Set environment variables
export COMPOSE_PROJECT_NAME=api-gateway

# Check if the setup has been run
if [ ! -d "nginx/logs" ] || [ ! -d "nginx/cache" ]; then
    echo "Setup has not been run. Running setup script..."
    ./scripts/setup.sh
fi

# Bring down existing services (if any)
docker-compose down

# Pull the latest images
docker-compose pull

# Build Kafka integration service
docker-compose build kafka-integration

# Start services in detached mode
docker-compose up -d

# Check if services are running
echo "Checking service status..."
sleep 5

if docker-compose ps | grep -q "Up"; then
    echo "API Gateway services started successfully!"
    
    # Display service endpoints
    echo "API Gateway is available at: http://localhost:8080"
    echo "Secure API Gateway is available at: https://localhost:8443"
else
    echo "Error: Failed to start API Gateway services."
    docker-compose logs
    exit 1
fi