#!/bin/bash
# Setup script for first-time deployment of Nginx API Gateway

# Ensure we're in the right directory
cd "$(dirname "$0")/.."

# Create necessary directories
mkdir -p nginx/logs nginx/cache/auth_cache nginx/cache/api_cache nginx/cache/media_cache nginx/ssl

# Ensure the log files exist
touch nginx/logs/audit_events.log
touch nginx/logs/access.log
touch nginx/logs/error.log

# Set proper permissions
chmod -R 755 nginx/logs nginx/cache
chmod 644 nginx/logs/audit_events.log
chmod 644 nginx/logs/access.log
chmod 644 nginx/logs/error.log

# Use this if running as root, otherwise skip this step
if [ $(id -u) -eq 0 ]; then
    chown -R 101:101 nginx/logs nginx/cache  # 101 is the nginx user in the container
fi

# Clean up any old config files to avoid conflicts
rm -f nginx/conf/conf.d/auth.conf
rm -f nginx/conf/conf.d/rate_limiting.conf

# Create self-signed SSL certificate if not exists (for testing)
if [ ! -f "nginx/ssl/server.crt" ]; then
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