#!/bin/bash

# Create necessary directories
mkdir -p infrastructure/kafka/config
mkdir -p infrastructure/postgres
mkdir -p infrastructure/redis
mkdir -p infrastructure/timescaledb

# Set up Kafka configuration
echo "Setting up Kafka configuration..."
cat > infrastructure/kafka/config/server.properties << 'EOF'
# Kafka Broker Configuration
broker.id=0
listeners=PLAINTEXT://:9092
advertised.listeners=PLAINTEXT://kafka:9092
listener.security.protocol.map=PLAINTEXT:PLAINTEXT
num.network.threads=3
num.io.threads=8
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600
log.dirs=/var/lib/kafka/data
num.partitions=3
num.recovery.threads.per.data.dir=1
log.retention.hours=168
zookeeper.connect=zookeeper:2181
auto.create.topics.enable=false
delete.topic.enable=true
default.replication.factor=1
offsets.topic.replication.factor=1
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1
EOF

# Create Kafka topics script
cat > infrastructure/kafka/config/topics.sh << 'EOF'
#!/bin/bash
# Wait for Kafka to be ready
echo "Waiting for Kafka to be ready..."
until kafka-topics.sh --bootstrap-server kafka:9092 --list > /dev/null 2>&1; do
    sleep 5
    echo "Waiting for Kafka..."
done

echo "Creating Kafka topics..."

# Create topics with specified partitions and replication factor
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic strategy-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic user-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic user-notifications --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic backtest-events --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic backtest-completions --partitions 3 --replication-factor 1
kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic marketplace-events --partitions 3 --replication-factor 1

echo "List of topics:"
kafka-topics.sh --bootstrap-server kafka:9092 --list

echo "Topic creation completed!"
EOF

# Make the topics script executable
chmod +x infrastructure/kafka/config/topics.sh

# Set up PostgreSQL configuration
echo "Setting up PostgreSQL configuration..."
cat > infrastructure/postgres/postgres.conf << 'EOF'
# PostgreSQL Configuration File
max_connections = 100
listen_addresses = '*'
shared_buffers = 256MB
work_mem = 16MB
maintenance_work_mem = 64MB
wal_level = replica
max_wal_size = 1GB
min_wal_size = 80MB
checkpoint_timeout = 5min
random_page_cost = 1.1
effective_cache_size = 768MB
autovacuum = on
log_destination = 'stderr'
logging_collector = on
log_directory = 'log'
timezone = 'UTC'
EOF

# Set up Redis configuration
echo "Setting up Redis configuration..."
cat > infrastructure/redis/redis.conf << 'EOF'
# Redis configuration file
bind 0.0.0.0
protected-mode yes
port 6379
tcp-backlog 511
timeout 0
tcp-keepalive 300
daemonize no
supervised no
pidfile /var/run/redis_6379.pid
loglevel notice
logfile ""
databases 16
save 900 1
save 300 10
save 60 10000
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /data
maxmemory 256mb
maxmemory-policy allkeys-lru
appendonly no
appendfsync everysec
EOF

# Set up TimescaleDB configuration
echo "Setting up TimescaleDB configuration..."
cat > infrastructure/timescaledb/timescaledb.conf << 'EOF'
# TimescaleDB specific configuration
shared_preload_libraries = 'timescaledb'
timescaledb.max_background_workers = 8
timescaledb.telemetry_level = 'basic'
timescaledb.max_chunks_per_insert = 8
timescaledb.max_insert_batch_size = 1000
timescaledb.enable_chunk_append = 'on'
timescaledb.enable_ordered_append = 'on'
timescaledb.enable_constraint_aware_append = 'on'
EOF

echo "Infrastructure configuration files created successfully!"
echo "To start the infrastructure, run: docker compose up -d"