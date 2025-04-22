#!/bin/bash
# Setup script for first-time deployment of Nginx API Gateway

# Create necessary directories
mkdir -p nginx/logs nginx/cache

# Set proper permissions
chmod -R 755 nginx/logs nginx/cache
chown -R 101:101 nginx/logs nginx/cache  # 101 is the nginx user in the container

# Create self-signed SSL certificate if not exists (for testing)
if [ ! -f "nginx/ssl/server.crt" ]; then
    mkdir -p nginx/ssl
    echo "Creating self-signed SSL certificate for testing..."
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
        -keyout nginx/ssl/server.key -out nginx/ssl/server.crt \
        -subj "/C=US/ST=State/L=City/O=Organization/CN=api.yourdomain.com"
    chmod 600 nginx/ssl/server.key
fi

# Create docker networks if they don't exist
for network in api-gateway-network user-service-network strategy-service-network historical-service-network media-service-network backtest-service-network kafka-network; do
    if ! docker network inspect $network >/dev/null 2>&1; then
        echo "Creating Docker network: $network"
        docker network create $network
    fi
done

echo "Setup completed successfully!"